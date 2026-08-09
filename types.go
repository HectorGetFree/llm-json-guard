package modeljson

import "context"

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

// LLMRepairer 定义高成本兜底边界；Parse 最多调用一次，结果仍须通过本地校验。
type LLMRepairer func(ctx context.Context, rawOutput string) (string, error)

// Validator 校验仅靠 Go 类型无法表达的领域规则。
type Validator[T any] func(value T) error

// ParseLimits 限制模型输入规模和候选提取开销。
// 字段小于等于零时使用安全默认值，而不是关闭保护。
type ParseLimits struct {
	MaxInputBytes     int
	MaxCandidateBytes int
	MaxCandidates     int
}

// ParseOptions 声明调用方接受的恢复策略。
// 本地修复和 LLM 修复只有在显式传入对应函数后才会启用。
type ParseOptions[T any] struct {
	LocalRepair            JSONRepairer
	LLMRepair              LLMRepairer
	Validate               Validator[T]
	AllowLLMSemanticRepair bool
	Limits                 ParseLimits
}

func normalizeLimits(limits ParseLimits) ParseLimits {
	if limits.MaxInputBytes <= 0 {
		limits.MaxInputBytes = DefaultMaxInputBytes
	}
	if limits.MaxCandidateBytes <= 0 {
		limits.MaxCandidateBytes = DefaultMaxCandidateBytes
	}
	if limits.MaxCandidates <= 0 {
		limits.MaxCandidates = DefaultMaxCandidates
	}
	return limits
}
