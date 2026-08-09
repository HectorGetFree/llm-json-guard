package llmjsonguard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMConfig 配置可选的网络兜底。
// 它与解析器分离，使本地恢复无需密钥或网络也能独立使用。
type LLMConfig struct {
	BaseURL string        // BaseURL 是供应商 API 根地址，通常以 /v1 结尾。
	APIKey  string        // APIKey 用于认证修复请求。
	Model   string        // Model 标识执行修复的模型。
	Timeout time.Duration // Timeout 限制完整 HTTP 请求耗时。
}

// NewLLMRepairer 创建 OpenAI-compatible 修复器，请求模型返回一个满足 Schema 的 JSON 值。
// 解析器负责限制调用次数并执行最终校验；该适配器每次调用只发送一个 HTTP 请求。
func NewLLMRepairer(config LLMConfig) (LLMRepairer, error) {
	// 1. 创建修复器时校验固定配置，避免运行时把配置缺失误判为解析失败。
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" || strings.TrimSpace(config.APIKey) == "" || strings.TrimSpace(config.Model) == "" {
		return nil, errors.New("LLM BaseURL, APIKey, and Model are required")
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	return func(ctx context.Context, repairRequest LLMRepairRequest) (string, error) {
		if strings.TrimSpace(repairRequest.Schema) == "" {
			return "", errors.New("LLM repair schema is required")
		}

		// 2. 限制模型只返回一个 JSON 值且不得编造数据；复用 Parse 的 Schema，避免双重契约。
		payload := buildChatCompletionRequest(config.Model, repairRequest)

		// 3. 发送一次有界请求；上下文取消和客户端超时优先，修复层不能延长调用方截止时间。
		body, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("marshal LLM request: %w", err)
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("create LLM request: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+config.APIKey)
		request.Header.Set("Content-Type", "application/json")

		response, err := client.Do(request)
		if err != nil {
			return "", fmt.Errorf("call LLM: %w", err)
		}
		defer response.Body.Close()

		// 4. 解码前限制响应读取规模，防止修复供应商通过超大响应绕过资源保护。
		const maxResponseBytes = 2 << 20
		responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
		if err != nil {
			return "", fmt.Errorf("read LLM response: %w", err)
		}
		if len(responseBody) > maxResponseBytes {
			return "", fmt.Errorf("LLM response exceeds %d bytes", maxResponseBytes)
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return "", fmt.Errorf("LLM returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
		}

		// 5. 只返回消息正文；修复结果是否可信仍由主解析器统一判断。
		var decoded chatCompletionResponse
		if err := json.Unmarshal(responseBody, &decoded); err != nil {
			return "", fmt.Errorf("decode LLM response: %w", err)
		}
		if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
			return "", errors.New("LLM response has no message content")
		}

		content := strings.TrimSpace(decoded.Choices[0].Message.Content)
		if reason, refused := parseLLMRepairRefusal(content); refused {
			return "", fmt.Errorf("%w: %s", ErrLLMRepairRefused, reason)
		}
		return content, nil
	}, nil
}

func buildChatCompletionRequest(model string, request LLMRepairRequest) chatCompletionRequest {
	payload := chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: "You are a strict JSON repairer. Return exactly one JSON value and no Markdown or explanation. Never invent information that is absent from the input."},
			{Role: "user", Content: buildRepairPrompt(request)},
		},
		Temperature: 0,
	}
	if schemaRootIsObject(request.Schema) {
		payload.ResponseFormat = map[string]string{"type": "json_object"}
	}
	return payload
}

func buildRepairPrompt(request LLMRepairRequest) string {
	return "Repair the input into exactly one JSON value matching this schema.\n" +
		"Only repair syntax, normalize unambiguous formatting, or recover facts explicitly present in the input.\n" +
		"Do not invent missing values, resolve conflicting facts, or change business meaning.\n" +
		"If safe repair is impossible, return {\"_repair_error\":\"reason\"}.\n\n" +
		"Local validation failure:\n" + request.ValidationFailure + "\n\n" +
		"Schema:\n" + request.Schema + "\n\nInput:\n" + request.RawOutput
}

func schemaRootIsObject(schema string) bool {
	var document struct {
		Type any `json:"type"`
	}
	if err := json.Unmarshal([]byte(schema), &document); err != nil {
		return false
	}
	typeName, ok := document.Type.(string)
	return ok && typeName == "object"
}

func parseLLMRepairRefusal(content string) (string, bool) {
	var refusal struct {
		Reason string `json:"_repair_error"`
	}
	if err := json.Unmarshal([]byte(content), &refusal); err != nil {
		return "", false
	}
	reason := strings.TrimSpace(refusal.Reason)
	return reason, reason != ""
}

type chatCompletionRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
	Temperature    int               `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}
