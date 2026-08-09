package llmjsonguard

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildChatCompletionRequestSelectsResponseFormatBySchemaRoot(t *testing.T) {
	tests := []struct {
		name               string
		schema             string
		wantResponseFormat bool
	}{
		{name: "object", schema: `{"type":"object"}`, wantResponseFormat: true},
		{name: "array", schema: `{"type":"array"}`, wantResponseFormat: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			payload := buildChatCompletionRequest("test", LLMRepairRequest{Schema: testCase.schema, RawOutput: "invalid"})
			if (payload.ResponseFormat != nil) != testCase.wantResponseFormat {
				t.Fatalf("response format = %#v, want enabled %v", payload.ResponseFormat, testCase.wantResponseFormat)
			}
		})
	}
}

func TestBuildRepairPromptUsesRequestContract(t *testing.T) {
	request := LLMRepairRequest{
		RawOutput:         "name: Alice",
		Schema:            `{"type":"object"}`,
		ValidationFailure: "no JSON candidate",
	}
	prompt := buildRepairPrompt(request)
	for _, expected := range []string{request.RawOutput, request.Schema, request.ValidationFailure, "Do not invent missing values"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt does not contain %q: %s", expected, prompt)
		}
	}
}

func TestSchemaRootIsObject(t *testing.T) {
	if !schemaRootIsObject(`{"type":"object"}`) {
		t.Fatal("object Schema must use json_object response format")
	}
	if schemaRootIsObject(`{"type":"array"}`) {
		t.Fatal("array Schema must not use json_object response format")
	}
}

func TestParseLLMRepairRefusal(t *testing.T) {
	reason, refused := parseLLMRepairRefusal(`{"_repair_error":"missing age"}`)
	if !refused || reason != "missing age" {
		t.Fatalf("reason = %q, refused = %v", reason, refused)
	}
	err := errors.Join(ErrLLMRepairRefused, errors.New(reason))
	if !errors.Is(err, ErrLLMRepairRefused) {
		t.Fatal("refusal must preserve sentinel error")
	}
}
