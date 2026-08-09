package llmjsonguard

import (
	"context"
	"time"
)

// ParsePath 记录模型输出通过哪条路径成为可信结果，用于统计模型可靠性和恢复成本。
type ParsePath string

// 成功路径区分直接解析、提取、本地修复和 LLM 修复，避免恢复过程不可观测。
const (
	ParsePathDirect    ParsePath = "direct"
	ParsePathExtracted ParsePath = "extracted"
	ParsePathLocalFix  ParsePath = "local_repair"
	ParsePathLLMFix    ParsePath = "llm_repair"

	// ParsePathRaw 是 ParsePathDirect 的源码兼容别名。
	ParsePathRaw ParsePath = ParsePathDirect
)

// 默认限制用于保护共享服务，避免异常模型响应消耗过多资源。
const (
	DefaultMaxInputBytes     = 1 << 20
	DefaultMaxSchemaBytes    = 1 << 20
	DefaultMaxCandidateBytes = 1 << 20
	DefaultMaxCandidates     = 32
)

// ParseResult 保存可信值、最终采用的 JSON 和恢复路径。
// RawOutput 仅用于诊断；其中可能包含敏感内容，调用方不应默认写入日志。
type ParseResult[T any] struct {
	Value      T
	JSON       string
	Path       ParsePath
	RawOutput  string
	UsedRepair bool
	UsedLLMFix bool
}

// JSONRepairer 定义确定性语法恢复边界，实现方不能补造业务值或改写合法载荷。
type JSONRepairer func(input string) (string, error)

// LLMRepairRequest 汇总模型修复所需的唯一契约和失败上下文。
// Schema 与本地校验共用同一来源，避免提示词约束和最终验收规则漂移。
type LLMRepairRequest struct {
	RawOutput         string
	Schema            string
	ValidationFailure string
}

// LLMRepairer 定义高成本兜底边界；Parse 最多调用一次，结果仍须通过本地校验。
type LLMRepairer func(ctx context.Context, request LLMRepairRequest) (string, error)

// Validator 校验仅靠 Go 类型无法表达的领域规则。
type Validator[T any] func(value T) error

// ParseObserver 在一次公开 Parse 调用完成后接收基础运行数据。
// Observer 只负责观察，不能修改解析结果，也不应在回调中执行高延迟阻塞操作。
type ParseObserver func(ctx context.Context, observation ParseObservation)

// ParseObservation 描述一次 Parse 的结果、资源规模和恢复策略使用情况。
// 该结构不包含原始输出、Schema、候选或错误详情，避免监控链路默认采集业务内容。
type ParseObservation struct {
	Success               bool
	Path                  ParsePath
	ErrorCode             ErrorCode
	Stage                 ParseStage
	Duration              time.Duration
	InputBytes            int
	SchemaBytes           int
	CandidateCount        int
	LocalRepairAttempts   int
	LLMRepairCalls        int
	CandidateLimitReached bool
}

// ParseLimits 限制模型输入规模和候选提取开销。
// 字段小于等于零时使用安全默认值，而不是关闭保护。
type ParseLimits struct {
	MaxInputBytes     int
	MaxSchemaBytes    int
	MaxCandidateBytes int
	MaxCandidates     int
}

// ParseOptions 声明调用方接受的恢复策略。
// 安全本地修复默认启用；LLM 修复只有显式传入后才会产生网络调用。
type ParseOptions[T any] struct {
	Schema                 string
	LocalRepair            JSONRepairer
	DisableLocalRepair     bool
	LLMRepair              LLMRepairer
	Validate               Validator[T]
	AllowLLMSemanticRepair bool
	Limits                 ParseLimits
	Observer               ParseObserver
}

func normalizeLimits(limits ParseLimits) ParseLimits {
	if limits.MaxInputBytes <= 0 {
		limits.MaxInputBytes = DefaultMaxInputBytes
	}
	if limits.MaxSchemaBytes <= 0 {
		limits.MaxSchemaBytes = DefaultMaxSchemaBytes
	}
	if limits.MaxCandidateBytes <= 0 {
		limits.MaxCandidateBytes = DefaultMaxCandidateBytes
	}
	if limits.MaxCandidates <= 0 {
		limits.MaxCandidates = DefaultMaxCandidates
	}
	return limits
}
