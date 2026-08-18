package jsonguard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Parse 将不可信的模型文本转换为 T。
// 本地链路失败且传入 llmRepair 时才会调用一次大模型；模型结果只会重新走本地链路，避免循环修复。
func Parse[T any](ctx context.Context, rawOutput, jsonFormat string, llmRepair LLMRepairFunc) (T, string, error) {
	var zero T
	if llmRepair != nil && strings.TrimSpace(jsonFormat) == "" {
		return zero, "", newParseError(ErrorCodeInvalidConfig, ParseStageInput, errors.New("启用大模型修复时必须提供 JSON 格式说明"))
	}

	value, jsonText, localErr := parseLocal[T](rawOutput)
	if localErr == nil {
		return value, jsonText, nil
	}
	if llmRepair == nil {
		return zero, "", localErr
	}

	// 大模型只处理本地无法恢复的格式或明确类型问题，失败原因用于缩小修复范围。
	repaired, err := llmRepair(ctx, LLMRepairRequest{
		RawOutput:     strings.TrimSpace(rawOutput),
		JSONFormat:    jsonFormat,
		FailureReason: localErr.Error(),
	})
	if err != nil {
		return zero, "", newParseError(ErrorCodeLLMRepairFailed, ParseStageLLMRepair, err)
	}
	if strings.TrimSpace(repaired) == "" {
		return zero, "", newParseError(ErrorCodeLLMRepairFailed, ParseStageLLMRepair, errors.New("大模型修复结果为空"))
	}

	value, jsonText, err = parseLocal[T](repaired)
	if err != nil {
		return zero, "", newParseError(ErrorCodeLLMRepairFailed, ParseStageLLMRepair, err)
	}
	return value, jsonText, nil
}

// parseLocal 统一执行无网络副作用的解析、提取和安全语法修复。
func parseLocal[T any](rawOutput string) (T, string, error) {
	var zero T
	if len(rawOutput) > maxInputBytes {
		return zero, "", newParseError(
			ErrorCodeInputTooLarge,
			ParseStageInput,
			fmt.Errorf("模型输出为 %d 字节，超过内部限制 %d 字节", len(rawOutput), maxInputBytes),
		)
	}

	rawOutput = strings.TrimSpace(rawOutput)
	if rawOutput == "" {
		return zero, "", newParseError(ErrorCodeEmptyOutput, ParseStageInput, errors.New("模型输出为空"))
	}

	// 1. 优先保留合法原始输出，避免对正常结果执行不必要的转换。
	if value, err := decodeStrict[T]([]byte(rawOutput)); err == nil {
		return value, rawOutput, nil
	}

	// 2. 标准化字符串外的结构标点并提取候选，兼容 Markdown 和自然语言包裹。
	normalizedOutput := normalizeStructurePunctuation(rawOutput)
	candidates := extractJSONCandidates(normalizedOutput)
	var lastFailure error
	for _, candidate := range candidates {
		value, err := decodeStrict[T]([]byte(candidate))
		if err == nil {
			return value, candidate, nil
		}
		lastFailure = err
	}

	// 3. 本地修复只处理语法非法候选；类型不匹配等问题保留给可选的大模型修复。
	for _, candidate := range candidates {
		if json.Valid([]byte(candidate)) {
			continue
		}
		repaired, err := localJSONRepair(candidate)
		if err != nil {
			lastFailure = err
			continue
		}
		value, err := decodeStrict[T]([]byte(repaired))
		if err == nil {
			return value, repaired, nil
		}
		lastFailure = err
	}

	if len(candidates) == 0 {
		return zero, "", newParseError(ErrorCodeNoCandidate, ParseStageExtraction, ErrNoJSONCandidate)
	}
	if lastFailure == nil {
		lastFailure = errors.New("本地解析与修复失败")
	}
	return zero, "", newParseError(ErrorCodeInvalidJSON, ParseStageLocalRepair, lastFailure)
}
