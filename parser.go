package jsonguard

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
	if len(options.Schema) > limits.MaxSchemaBytes {
		return ParseResult[T]{RawOutput: rawOutput}, newParseError(
			ErrorCodeInvalidSchema,
			ParseStageSchema,
			fmt.Errorf("schema is %d bytes, limit is %d", len(options.Schema), limits.MaxSchemaBytes),
		)
	}
	if options.Repair != nil && strings.TrimSpace(options.Schema) == "" {
		return ParseResult[T]{RawOutput: rawOutput}, newParseError(
			ErrorCodeInvalidSchema,
			ParseStageSchema,
			errors.New("Schema is required when external repair is enabled"),
		)
	}
	schema, err := compileJSONSchema(options.Schema)
	if err != nil {
		return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeInvalidSchema, ParseStageSchema, err)
	}

	// 2. 优先解析原始输出，保留模型原意，并避免对正常输出执行不必要的恢复。
	var lastFailure error
	if len(rawOutput) <= limits.MaxCandidateBytes {
		if value, decodeErr := decodeAndValidate[T]([]byte(rawOutput), schema, options.Validate); decodeErr == nil {
			return ParseResult[T]{Value: value, JSON: rawOutput, Path: ParsePathDirect, RawOutput: rawOutput}, nil
		} else {
			lastFailure = decodeErr
		}
	}

	// 3. 只标准化字符串外的结构标点，再按限制提取候选，避免修改业务字段内容。
	normalizedOutput := normalizeStructurePunctuation(rawOutput)
	candidates, limitReached := extractJSONCandidatesWithLimits(normalizedOutput, limits)
	sawSemanticError := false
	sawLossyRepair := false
	var lastSemanticError error
	var lastLossyRepairError error

	// 4. 优先选择无需修复且通过本地规则的候选。
	// 合法 JSON 如果校验失败，应归类为语义错误，不能交给语法修复器静默改写。
	for _, candidate := range candidates {
		value, err := decodeAndValidate[T]([]byte(candidate), schema, options.Validate)
		if err == nil {
			return ParseResult[T]{Value: value, JSON: candidate, Path: ParsePathExtracted, RawOutput: rawOutput}, nil
		}
		lastFailure = err
		if json.Valid([]byte(candidate)) && schema.matchesRootType(candidate) {
			sawSemanticError = true
			if lastSemanticError == nil {
				lastSemanticError = err
			}
		}
	}

	// 5. 仅修复语法非法的候选，避免把 Schema 或业务错误伪装成格式问题。
	localRepair := options.LocalRepair
	if localRepair == nil && !options.DisableLocalRepair {
		localRepair = LocalJSONRepair
	}
	if localRepair != nil {
		for _, candidate := range candidates {
			if json.Valid([]byte(candidate)) {
				continue
			}

			repaired, err := localRepair(candidate)
			if err != nil {
				lastFailure = err
				if errors.Is(err, ErrLossyRepair) {
					sawLossyRepair = true
					lastLossyRepairError = err
				}
				continue
			}

			value, err := decodeAndValidate[T]([]byte(repaired), schema, options.Validate)
			if err != nil {
				lastFailure = err
				continue
			}

			return ParseResult[T]{
				Value: value, JSON: repaired, Path: ParsePathLocalFix,
				RawOutput: rawOutput, UsedRepair: true,
			}, nil
		}
	}

	// 6. 外部兜底最多调用一次；语义修复必须显式开启，防止修复器改写业务事实。
	if options.Repair != nil && (!sawSemanticError || options.AllowSemanticRepair) {
		validationFailure := "no JSON candidate passed local decoding and validation"
		if lastFailure != nil {
			validationFailure = lastFailure.Error()
		}
		fixed, err := options.Repair(ctx, RepairRequest{
			RawOutput:         rawOutput,
			Schema:            options.Schema,
			ValidationFailure: validationFailure,
		})
		if err != nil {
			return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeRepairFailed, ParseStageRepair, err)
		}

		fixed = strings.TrimSpace(fixed)
		if len(fixed) > limits.MaxCandidateBytes {
			return ParseResult[T]{RawOutput: rawOutput}, newParseError(
				ErrorCodeRepairFailed,
				ParseStageRepair,
				fmt.Errorf("external repair output is %d bytes, candidate limit is %d", len(fixed), limits.MaxCandidateBytes),
			)
		}
		value, err := decodeAndValidate[T]([]byte(fixed), schema, options.Validate)
		if err != nil {
			return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeRepairFailed, ParseStageValidation, err)
		}

		return ParseResult[T]{
			Value: value, JSON: fixed, Path: ParsePathExternalRepair,
			RawOutput: rawOutput, UsedRepair: true,
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
	if sawLossyRepair {
		return ParseResult[T]{RawOutput: rawOutput}, newParseError(ErrorCodeLossyRepair, ParseStageLocalRepair, lastLossyRepairError)
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
