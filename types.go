package jsonguard

import "context"

const (
	maxInputBytes     = 1 << 20
	maxCandidateBytes = 1 << 20
	maxCandidates     = 32
)

// LLMRepairRequest 描述一次大模型修复所需的上下文。
// JSONFormat 是调用方原 Prompt 中的自然语言格式说明，不是 JSON Schema。
type LLMRepairRequest struct {
	RawOutput     string
	JSONFormat    string
	FailureReason string
}

// LLMRepairFunc 定义 Guard 与模型服务之间的调用边界。
// Parse 最多调用一次，返回内容仍须通过本地解析链路。
type LLMRepairFunc func(ctx context.Context, request LLMRepairRequest) (string, error)
