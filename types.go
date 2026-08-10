package llmjsonguard

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

// ChatMessage 是 Guard 构造的模型消息。调用方只负责映射到自身 ChatModel 的消息类型。
type ChatMessage struct {
	Role    string
	Content string
}

// ChatModelCaller 定义黑盒模型调用边界。
// ChatModel 可以是任意项目的具体模型实例，Guard 不依赖其类型或 Generate 方法。
type ChatModelCaller func(ctx context.Context, chatModel any, messages []ChatMessage) (string, error)

// Validator 校验仅靠 Go 类型无法表达的领域规则。
type Validator[T any] func(value T) error

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
	RepairChatModel        any
	CallRepairChatModel    ChatModelCaller
	Validate               Validator[T]
	AllowLLMSemanticRepair bool
	Limits                 ParseLimits
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
