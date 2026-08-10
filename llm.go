package llmjsonguard

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const repairSystemPrompt = "You are a strict JSON repairer. Return exactly one JSON value and no Markdown or explanation. Never invent information that is absent from the input."

func buildRepairMessages(rawOutput string, schema string, validationFailure string) []ChatMessage {
	return []ChatMessage{
		{Role: "system", Content: repairSystemPrompt},
		{Role: "user", Content: buildRepairPrompt(rawOutput, schema, validationFailure)},
	}
}

func buildRepairPrompt(rawOutput string, schema string, validationFailure string) string {
	return "Repair the input into exactly one JSON value matching this schema.\n" +
		"Only repair syntax, normalize unambiguous formatting, or recover facts explicitly present in the input.\n" +
		"Do not invent missing values, resolve conflicting facts, or change business meaning.\n" +
		"If safe repair is impossible, return {\"_repair_error\":\"reason\"}.\n\n" +
		"Local validation failure:\n" + validationFailure + "\n\n" +
		"Schema:\n" + schema + "\n\nInput:\n" + rawOutput
}

func normalizeLLMRepairContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", errors.New("LLM response has no message content")
	}
	if reason, refused := parseLLMRepairRefusal(content); refused {
		return "", fmt.Errorf("%w: %s", ErrLLMRepairRefused, reason)
	}
	return content, nil
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
