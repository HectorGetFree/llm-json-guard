package modeljson

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

// NewLLMRepairer 创建 OpenAI-compatible 修复器，请求模型返回一个满足 Schema 的 JSON 对象。
// 解析器负责限制调用次数并执行最终校验；该适配器每次调用只发送一个 HTTP 请求。
func NewLLMRepairer(config LLMConfig, schema string) (LLMRepairer, error) {
	// 1. 创建修复器时校验固定配置，避免运行时把配置缺失误判为解析失败。
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" || strings.TrimSpace(config.APIKey) == "" || strings.TrimSpace(config.Model) == "" {
		return nil, errors.New("LLM BaseURL, APIKey, and Model are required")
	}
	if strings.TrimSpace(schema) == "" {
		return nil, errors.New("LLM repair schema is required")
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	return func(ctx context.Context, rawOutput string) (string, error) {
		// 2. 限制模型只返回一个 JSON 对象且不得编造数据；复用调用方 Schema，避免双重契约。
		payload := chatCompletionRequest{
			Model: config.Model,
			Messages: []chatMessage{
				{Role: "system", Content: "You are a strict JSON repairer. Return exactly one JSON object and no Markdown or explanation."},
				{Role: "user", Content: buildRepairPrompt(schema, rawOutput)},
			},
			ResponseFormat: map[string]string{"type": "json_object"},
			Temperature:    0,
		}

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
		responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		if err != nil {
			return "", fmt.Errorf("read LLM response: %w", err)
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

		return strings.TrimSpace(decoded.Choices[0].Message.Content), nil
	}, nil
}

func buildRepairPrompt(schema, rawOutput string) string {
	return "Repair the input into exactly one JSON object matching this schema.\n" +
		"Only repair syntax or unambiguous formatting. Do not invent missing values.\n" +
		"If safe repair is impossible, return {\"_repair_error\":\"reason\"}.\n\n" +
		"Schema:\n" + schema + "\n\nInput:\n" + rawOutput
}

type chatCompletionRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format"`
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
