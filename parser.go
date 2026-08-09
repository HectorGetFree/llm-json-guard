package modeljson

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// parseResult 按成本和风险从低到高调度恢复链路。
// 无论通过哪条路径恢复，结果都必须经过相同的本地解码和业务校验。
func parseResult[T any](ctx context.Context, rawOutput string, options ParseOptions[T]) (ParseResult[T], error) {
	// 1. 提取前校验输入规模，避免异常模型输出占用过多内存和扫描时间。
	limits := normalizeLimits(options.Limits)
	if len(rawOutput) > limits.MaxInputBytes {
		return ParseResult[T]{}, newParseError(
			ErrorCodeInputTooLarge,
			ParseStageInput,
			fmt.Errorf("input is %d bytes, limit is %d", len(rawOutput), limits.MaxInputBytes),
		)
	}

	rawOutput = strings.TrimSpace(rawOutput)
	if rawOutput == "" {
		return ParseResult[T]{}, newParseError(ErrorCodeEmptyOutput, ParseStageInput, errors.New("empty output"))
	}

	// 2. 优先解析原始输出，保留模型原意，并避免对正常输出执行不必要的恢复。
	if len(rawOutput) <= limits.MaxCandidateBytes {
		if value, err := decodeAndValidate[T]([]byte(rawOutput), options.Validate); err == nil {
			return ParseResult[T]{Value: value, JSON: rawOutput, Path: ParsePathDirect, RawOutput: rawOutput}, nil
		}
	}

	// 3. 只标准化字符串外的结构标点，再按限制提取候选，避免修改业务字段内容。
	normalizedOutput := normalizeStructurePunctuation(rawOutput)
	candidates, limitReached := extractJSONCandidatesWithLimits(normalizedOutput, limits)
	sawSemanticError := false
	var lastSemanticError error

	// 4. 优先选择无需修复且通过本地规则的候选。
	// 合法 JSON 如果校验失败，应归类为语义错误，不能交给语法修复器静默改写。
	for _, candidate := range candidates {
		value, err := decodeAndValidate[T]([]byte(candidate), options.Validate)
		if err == nil {
			return ParseResult[T]{Value: value, JSON: candidate, Path: ParsePathExtracted, RawOutput: rawOutput}, nil
		}
		if json.Valid([]byte(candidate)) {
			sawSemanticError = true
			if lastSemanticError == nil {
				lastSemanticError = err
			}
		}
	}

	// 5. 仅修复语法非法的候选，避免把 Schema 或业务错误伪装成格式问题。
	if options.LocalRepair != nil {
		for _, candidate := range candidates {
			if json.Valid([]byte(candidate)) {
				continue
			}

			repaired, err := options.LocalRepair(candidate)
			if err != nil {
				continue
			}

			value, err := decodeAndValidate[T]([]byte(repaired), options.Validate)
			if err != nil {
				continue
			}

			return ParseResult[T]{
				Value: value, JSON: repaired, Path: ParsePathLocalFix,
				RawOutput: rawOutput, UsedRepair: true,
			}, nil
		}
	}

	// 6. LLM 兜底最多调用一次；语义修复必须显式开启，防止模型为满足目标而编造数据。
	if options.LLMRepair != nil && (!sawSemanticError || options.AllowLLMSemanticRepair) {
		fixed, err := options.LLMRepair(ctx, rawOutput)
		if err != nil {
			return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeLLMRepairFailed, ParseStageLLMRepair, err)
		}

		fixed = strings.TrimSpace(fixed)
		if len(fixed) > limits.MaxCandidateBytes {
			return ParseResult[T]{RawOutput: rawOutput}, newParseError(
				ErrorCodeLLMRepairFailed,
				ParseStageLLMRepair,
				fmt.Errorf("LLM output is %d bytes, candidate limit is %d", len(fixed), limits.MaxCandidateBytes),
			)
		}
		value, err := decodeAndValidate[T]([]byte(fixed), options.Validate)
		if err != nil {
			return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeLLMRepairFailed, ParseStageValidation, err)
		}

		return ParseResult[T]{
			Value: value, JSON: fixed, Path: ParsePathLLMFix,
			RawOutput: rawOutput, UsedLLMFix: true,
		}, nil
	}

	// 7. 对最终失败分类，供调用方决策和指标统计。
	// 限制类错误单独返回，因为不调整限制时重试相同输入没有意义。
	if len(candidates) == 0 {
		if limitReached {
			return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeCandidateLimit, ParseStageExtraction, errors.New("candidate limit reached"))
		}
		return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeNoCandidate, ParseStageExtraction, ErrNoJSONCandidate)
	}
	if sawSemanticError {
		return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeValidationFailed, ParseStageValidation, lastSemanticError)
	}
	if limitReached {
		return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeCandidateLimit, ParseStageExtraction, errors.New("candidate limit reached"))
	}
	return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeInvalidJSON, ParseStageLocalRepair, errors.New("local parsing and repair failed"))
}

// Parse 将不可信的模型文本转换为 T。
// 返回值一定通过严格本地解码和可选业务校验，任何恢复路径都不能绕过该边界。
func Parse[T any](ctx context.Context, rawOutput string, options ParseOptions[T]) (ParseResult[T], error) {
	return parseResult(ctx, rawOutput, options)
}
