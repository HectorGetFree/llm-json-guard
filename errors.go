package jsonguard

import (
	"errors"
	"fmt"
)

var (
	// ErrNoJSONCandidate 用于判断文本中不存在可恢复 JSON 边界的情况。
	ErrNoJSONCandidate = errors.New("未找到完整的 JSON 候选")
	// ErrInvalidOutput 是所有模型输出解析失败共享的哨兵错误。
	ErrInvalidOutput = errors.New("模型输出无法通过本地解析与修复")
	// ErrLossyRepair 用于识别本地修复可能新增、删除或重组业务数据的情况。
	ErrLossyRepair = errors.New("已拒绝可能改变业务语义的 JSON 修复")
)

// ErrorCode 提供稳定的失败分类，供指标统计和重试策略使用。
type ErrorCode string

// 稳定错误码用于区分可重试的恢复错误和终止性的校验错误。
const (
	ErrorCodeEmptyOutput     ErrorCode = "empty_output"
	ErrorCodeInputTooLarge   ErrorCode = "input_too_large"
	ErrorCodeInvalidConfig   ErrorCode = "invalid_config"
	ErrorCodeNoCandidate     ErrorCode = "no_candidate"
	ErrorCodeInvalidJSON     ErrorCode = "invalid_json"
	ErrorCodeLossyRepair     ErrorCode = "lossy_repair_rejected"
	ErrorCodeLLMRepairFailed ErrorCode = "llm_repair_failed"
)

// ParseStage 标识输出在可信处理链路中停止的阶段。
type ParseStage string

// 阶段值与错误文案解耦，避免文案调整破坏监控面板。
const (
	ParseStageInput       ParseStage = "input"
	ParseStageExtraction  ParseStage = "extraction"
	ParseStageLocalRepair ParseStage = "local_repair"
	ParseStageLLMRepair   ParseStage = "llm_repair"
)

// ParseError 提供错误码、阶段和根因，使调用方无需解析文案即可决定重试、降级或拒绝。
// 同时保留旧版 errors.Is 判断能力。
type ParseError struct {
	Code  ErrorCode
	Stage ParseStage
	Cause error
}

// Error 返回简洁诊断信息，不包含可能敏感的原始模型输出。
func (e *ParseError) Error() string {
	if e == nil {
		return "空的解析错误"
	}
	if e.Cause == nil {
		return fmt.Sprintf("%s，阶段：%s", e.Code, e.Stage)
	}
	return fmt.Sprintf("%s，阶段：%s，原因：%v", e.Code, e.Stage, e.Cause)
}

// Unwrap 同时暴露包级哨兵错误和根因，兼容不同版本调用方的错误判断方式。
func (e *ParseError) Unwrap() []error {
	if e == nil {
		return nil
	}
	if e.Cause == nil {
		return []error{ErrInvalidOutput}
	}
	return []error{ErrInvalidOutput, e.Cause}
}

func newParseError(code ErrorCode, stage ParseStage, cause error) error {
	return &ParseError{Code: code, Stage: stage, Cause: cause}
}
