package llmjsonguard

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRealLLMRepair(t *testing.T) {
	config := LLMConfig{
		BaseURL: os.Getenv("MODELJSON_LLM_BASE_URL"),
		APIKey:  os.Getenv("MODELJSON_LLM_API_KEY"),
		Model:   os.Getenv("MODELJSON_LLM_MODEL"),
		Timeout: 30 * time.Second,
	}
	if config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		t.Skip("set MODELJSON_LLM_BASE_URL, MODELJSON_LLM_API_KEY, and MODELJSON_LLM_MODEL to run")
	}

	repairer, err := NewLLMRepairer(config)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name              string
		raw               string
		allowSemanticFix  bool
		expectedParsePath ParsePath
	}{
		{
			name:              "schema-invalid JSON uses the real LLM semantic-repair path",
			raw:               `{"name":"Alice","age":18,"tags":["go","llm"],"active":true,"role":"admin"}`,
			allowSemanticFix:  true,
			expectedParsePath: ParsePathLLMFix,
		},
		{
			name:              "incomplete outer JSON becomes a local-repair candidate",
			raw:               `{"name":"Alice","age":18,"tags":["go","llm"],"active":true`,
			expectedParsePath: ParsePathLocalFix,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			calls := 0
			countedRepairer := func(ctx context.Context, request LLMRepairRequest) (string, error) {
				calls++
				return repairer(ctx, request)
			}

			result, err := Parse[Person](context.Background(), testCase.raw, ParseOptions[Person]{
				Schema:                 personSchema,
				LocalRepair:            LocalJSONRepair,
				LLMRepair:              countedRepairer,
				Validate:               validatePerson,
				AllowLLMSemanticRepair: testCase.allowSemanticFix,
			})
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if testCase.expectedParsePath == ParsePathLLMFix && calls != 1 {
				t.Fatalf("real LLM calls = %d, want 1", calls)
			}
			if testCase.expectedParsePath == ParsePathLocalFix && calls != 0 {
				t.Fatalf("real LLM calls = %d, want 0", calls)
			}
			if result.Path != testCase.expectedParsePath {
				t.Fatalf("path = %q, want %q", result.Path, testCase.expectedParsePath)
			}
		})
	}
}
