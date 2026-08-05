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

type LLMConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// NewLLMRepairer creates a one-shot JSON repairer for an OpenAI-compatible
// Chat Completions endpoint. schema is a JSON Schema or concise schema contract.
func NewLLMRepairer(config LLMConfig, schema string) (LLMRepairer, error) {
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
		payload := chatCompletionRequest{
			Model: config.Model,
			Messages: []chatMessage{
				{Role: "system", Content: "You are a strict JSON repairer. Return exactly one JSON object and no Markdown or explanation."},
				{Role: "user", Content: buildRepairPrompt(schema, rawOutput)},
			},
			ResponseFormat: map[string]string{"type": "json_object"},
			Temperature:    0,
		}

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

		responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		if err != nil {
			return "", fmt.Errorf("read LLM response: %w", err)
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return "", fmt.Errorf("LLM returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
		}

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
