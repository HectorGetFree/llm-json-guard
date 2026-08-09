package evaluation

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"

	llmjsonguard "github.com/HectorGetFree/llm-json-guard"
)

// defaultCases 保存可复现的离线评测集，避免运行评测时依赖网络或公司内部数据。
//
//go:embed cases.jsonl
var defaultCases []byte

// Case 描述一个带标准答案的模型输出样本。
// ExpectedJSON 只在应当接受时使用；拒绝样本通过 ExpectedError 固定安全边界。
type Case struct {
	ID                 string   `json:"id"`
	Category           string   `json:"category"`
	Input              string   `json:"input"`
	Schema             string   `json:"schema"`
	ExpectedAccept     bool     `json:"expected_accept"`
	ExpectedJSON       string   `json:"expected_json,omitempty"`
	ExpectedPaths      []string `json:"expected_paths,omitempty"`
	ExpectedError      string   `json:"expected_error,omitempty"`
	DisableLocalRepair bool     `json:"disable_local_repair,omitempty"`
}

// CaseResult 记录单个样本的行为与标准答案差异，便于定位指标变化来源。
type CaseResult struct {
	ID             string                 `json:"id"`
	Category       string                 `json:"category"`
	Passed         bool                   `json:"passed"`
	ExpectedAccept bool                   `json:"expected_accept"`
	Accepted       bool                   `json:"accepted"`
	Path           llmjsonguard.ParsePath `json:"path,omitempty"`
	ErrorCode      llmjsonguard.ErrorCode `json:"error_code,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
	Duration       time.Duration          `json:"duration"`
}

// Report 汇总 Guard 的正确性、恢复收益、安全性和耗时。
// Accuracy 是总指标；FalseAcceptanceRate 应优先保持为零，避免用误修复换取恢复率。
type Report struct {
	Total               int                `json:"total"`
	Passed              int                `json:"passed"`
	Accuracy            float64            `json:"accuracy"`
	ExpectedAccepts     int                `json:"expected_accepts"`
	ExpectedRejects     int                `json:"expected_rejects"`
	DirectCorrect       int                `json:"direct_correct"`
	RecoveredCorrect    int                `json:"recovered_correct"`
	RecoveryEligible    int                `json:"recovery_eligible"`
	RecoverySuccessRate float64            `json:"recovery_success_rate"`
	RecoveryUplift      float64            `json:"recovery_uplift"`
	FalseAcceptances    int                `json:"false_acceptances"`
	FalseAcceptanceRate float64            `json:"false_acceptance_rate"`
	FalseRejections     int                `json:"false_rejections"`
	WrongValues         int                `json:"wrong_values"`
	PathMismatches      int                `json:"path_mismatches"`
	PathCounts          map[string]int     `json:"path_counts"`
	ErrorCounts         map[string]int     `json:"error_counts"`
	CategoryAccuracy    map[string]float64 `json:"category_accuracy"`
	MeanLatency         time.Duration      `json:"mean_latency"`
	P50Latency          time.Duration      `json:"p50_latency"`
	P95Latency          time.Duration      `json:"p95_latency"`
	Iterations          int                `json:"iterations"`
	Results             []CaseResult       `json:"results"`
}

// LoadDefaultCases 返回仓库内置评测集。
func LoadDefaultCases() ([]Case, error) {
	return LoadCases(bytes.NewReader(defaultCases))
}

// LoadCases 按 JSONL 读取样本，错误中包含行号，便于维护较大的评测集。
func LoadCases(reader io.Reader) ([]Case, error) {
	var cases []Case
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 2<<20)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var testCase Case
		if err := json.Unmarshal(line, &testCase); err != nil {
			return nil, fmt.Errorf("decode evaluation case at line %d: %w", lineNumber, err)
		}
		if err := validateCase(testCase); err != nil {
			return nil, fmt.Errorf("validate evaluation case at line %d: %w", lineNumber, err)
		}
		cases = append(cases, testCase)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read evaluation cases: %w", err)
	}
	if len(cases) == 0 {
		return nil, errors.New("evaluation dataset is empty")
	}
	return cases, nil
}

// Run 重复执行同一批样本并汇总指标。
// 正确性只取每个样本第一次结果，额外迭代仅用于降低延迟测量的偶然波动。
func Run(cases []Case, iterations int) Report {
	if iterations <= 0 {
		iterations = 1
	}

	report := Report{
		Total:            len(cases),
		PathCounts:       make(map[string]int),
		ErrorCounts:      make(map[string]int),
		CategoryAccuracy: make(map[string]float64),
		Iterations:       iterations,
	}
	categoryTotals := make(map[string]int)
	categoryPassed := make(map[string]int)
	latencies := make([]time.Duration, 0, len(cases)*iterations)

	// 1. 每轮使用完全相同的样本，首轮评判行为，所有轮次共同统计耗时。
	for iteration := 0; iteration < iterations; iteration++ {
		for _, testCase := range cases {
			startedAt := time.Now()
			result, err := llmjsonguard.Parse[any](context.Background(), testCase.Input, llmjsonguard.ParseOptions[any]{
				Schema:             testCase.Schema,
				DisableLocalRepair: testCase.DisableLocalRepair,
			})
			duration := time.Since(startedAt)
			latencies = append(latencies, duration)
			if iteration > 0 {
				continue
			}

			caseResult := evaluateResult(testCase, result, err)
			caseResult.Duration = duration
			report.Results = append(report.Results, caseResult)
			categoryTotals[testCase.Category]++
			if caseResult.Passed {
				report.Passed++
				categoryPassed[testCase.Category]++
			}
			updateBehaviorMetrics(&report, testCase, caseResult)
		}
	}

	// 2. 比例统一在样本执行结束后计算，避免分母为零和逐步累积误差。
	report.Accuracy = ratio(report.Passed, report.Total)
	report.RecoverySuccessRate = ratio(report.RecoveredCorrect, report.RecoveryEligible)
	report.RecoveryUplift = ratio(report.RecoveredCorrect, report.Total)
	report.FalseAcceptanceRate = ratio(report.FalseAcceptances, report.ExpectedRejects)
	for category, total := range categoryTotals {
		report.CategoryAccuracy[category] = ratio(categoryPassed[category], total)
	}
	setLatencyMetrics(&report, latencies)
	return report
}

func validateCase(testCase Case) error {
	if strings.TrimSpace(testCase.ID) == "" || strings.TrimSpace(testCase.Category) == "" {
		return errors.New("id and category are required")
	}
	if strings.TrimSpace(testCase.Schema) == "" {
		return errors.New("schema is required")
	}
	if testCase.ExpectedAccept && strings.TrimSpace(testCase.ExpectedJSON) == "" {
		return errors.New("expected_json is required for accepted case")
	}
	if !testCase.ExpectedAccept && strings.TrimSpace(testCase.ExpectedError) == "" {
		return errors.New("expected_error is required for rejected case")
	}
	return nil
}

func evaluateResult(testCase Case, result llmjsonguard.ParseResult[any], err error) CaseResult {
	caseResult := CaseResult{
		ID:             testCase.ID,
		Category:       testCase.Category,
		ExpectedAccept: testCase.ExpectedAccept,
		Accepted:       err == nil,
		Path:           result.Path,
	}
	if err != nil {
		var parseError *llmjsonguard.ParseError
		if errors.As(err, &parseError) {
			caseResult.ErrorCode = parseError.Code
		}
	}

	if testCase.ExpectedAccept {
		if err != nil {
			caseResult.Reason = "expected acceptance but guard rejected the input"
			return caseResult
		}
		if !equalJSON(result.JSON, testCase.ExpectedJSON) {
			caseResult.Reason = "accepted JSON differs from expected value"
			return caseResult
		}
		if len(testCase.ExpectedPaths) > 0 && !contains(testCase.ExpectedPaths, string(result.Path)) {
			caseResult.Reason = fmt.Sprintf("path %q is outside expected paths %v", result.Path, testCase.ExpectedPaths)
			return caseResult
		}
		caseResult.Passed = true
		return caseResult
	}

	if err == nil {
		caseResult.Reason = "expected rejection but guard accepted the input"
		return caseResult
	}
	if string(caseResult.ErrorCode) != testCase.ExpectedError {
		caseResult.Reason = fmt.Sprintf("error %q differs from expected %q", caseResult.ErrorCode, testCase.ExpectedError)
		return caseResult
	}
	caseResult.Passed = true
	return caseResult
}

func updateBehaviorMetrics(report *Report, testCase Case, result CaseResult) {
	if testCase.ExpectedAccept {
		report.ExpectedAccepts++
		if !result.Accepted {
			report.FalseRejections++
		} else if strings.HasPrefix(result.Reason, "accepted JSON differs") {
			report.WrongValues++
		} else if strings.HasPrefix(result.Reason, "path ") {
			report.PathMismatches++
		}
		if len(testCase.ExpectedPaths) > 0 && !contains(testCase.ExpectedPaths, string(llmjsonguard.ParsePathDirect)) {
			report.RecoveryEligible++
		}
		if result.Passed {
			if result.Path == llmjsonguard.ParsePathDirect {
				report.DirectCorrect++
			} else {
				report.RecoveredCorrect++
			}
		}
	} else {
		report.ExpectedRejects++
		if result.Accepted {
			report.FalseAcceptances++
		}
	}
	if result.Accepted {
		report.PathCounts[string(result.Path)]++
	} else if result.ErrorCode != "" {
		report.ErrorCounts[string(result.ErrorCode)]++
	}
}

func equalJSON(actual, expected string) bool {
	decode := func(input string) (any, error) {
		decoder := json.NewDecoder(strings.NewReader(input))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		return value, nil
	}
	actualValue, actualError := decode(actual)
	expectedValue, expectedError := decode(expected)
	return actualError == nil && expectedError == nil && reflect.DeepEqual(actualValue, expectedValue)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func setLatencyMetrics(report *Report, latencies []time.Duration) {
	if len(latencies) == 0 {
		return
	}
	sort.Slice(latencies, func(left, right int) bool { return latencies[left] < latencies[right] })
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	report.MeanLatency = total / time.Duration(len(latencies))
	report.P50Latency = percentile(latencies, 50)
	report.P95Latency = percentile(latencies, 95)
}

func percentile(sorted []time.Duration, value int) time.Duration {
	index := (len(sorted)*value + 99) / 100
	if index < 1 {
		index = 1
	}
	return sorted[index-1]
}
