package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/HectorGetFree/llm-json-guard/evaluation"
)

func main() {
	iterations := flag.Int("iterations", 100, "每个样本执行次数，首轮评判正确性，全部轮次统计耗时")
	dataset := flag.String("dataset", "", "自定义 JSONL 数据集路径，留空时使用内置数据集")
	jsonOutput := flag.Bool("json", false, "以 JSON 输出完整报告")
	minimumAccuracy := flag.Float64("min-accuracy", 0, "低于该准确率时返回非零退出码")
	maximumFalseAcceptance := flag.Float64("max-false-acceptance", 1, "高于该误接收率时返回非零退出码")
	flag.Parse()

	cases, err := loadCases(*dataset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载评测集失败: %v\n", err)
		os.Exit(2)
	}
	report := evaluation.Run(cases, *iterations)
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "输出评测报告失败: %v\n", err)
			os.Exit(2)
		}
	} else {
		printReport(report)
	}

	if report.Accuracy < *minimumAccuracy || report.FalseAcceptanceRate > *maximumFalseAcceptance {
		os.Exit(1)
	}
}

func loadCases(path string) ([]evaluation.Case, error) {
	if path == "" {
		return evaluation.LoadDefaultCases()
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return evaluation.LoadCases(file)
}

func printReport(report evaluation.Report) {
	fmt.Println("LLM JSON Guard 离线评测")
	fmt.Printf("样本: %d，通过: %d，准确率: %.2f%%\n", report.Total, report.Passed, report.Accuracy*100)
	fmt.Printf("直接正确: %d，恢复正确: %d，恢复成功率: %.2f%%，恢复增益: %.2f 个百分点\n",
		report.DirectCorrect, report.RecoveredCorrect, report.RecoverySuccessRate*100, report.RecoveryUplift*100)
	fmt.Printf("误接收: %d/%d (%.2f%%)，误拒绝: %d，错误值: %d，路径偏差: %d\n",
		report.FalseAcceptances, report.ExpectedRejects, report.FalseAcceptanceRate*100,
		report.FalseRejections, report.WrongValues, report.PathMismatches)
	fmt.Printf("延迟: mean=%s, p50=%s, p95=%s，迭代: %d\n",
		report.MeanLatency, report.P50Latency, report.P95Latency, report.Iterations)
	printCounts("成功路径", report.PathCounts)
	printCounts("错误分类", report.ErrorCounts)
	printAccuracy("分类准确率", report.CategoryAccuracy)

	var failures []evaluation.CaseResult
	for _, result := range report.Results {
		if !result.Passed {
			failures = append(failures, result)
		}
	}
	if len(failures) == 0 {
		return
	}
	fmt.Println("未通过样本:")
	for _, result := range failures {
		fmt.Printf("  - %s [%s]: %s\n", result.ID, result.Category, result.Reason)
	}
}

func printCounts(title string, values map[string]int) {
	keys := sortedKeys(values)
	if len(keys) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	for _, key := range keys {
		fmt.Printf("  - %s: %d\n", key, values[key])
	}
}

func printAccuracy(title string, values map[string]float64) {
	keys := sortedKeys(values)
	fmt.Printf("%s:\n", title)
	for _, key := range keys {
		fmt.Printf("  - %s: %.2f%%\n", key, values[key]*100)
	}
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
