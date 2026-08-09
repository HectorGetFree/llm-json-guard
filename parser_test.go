package llmjsonguard

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type Person struct {
	Name    *string  `json:"name"`
	Age     *int     `json:"age"`
	Tags    []string `json:"tags"`
	Active  *bool    `json:"active"`
	Profile *Profile `json:"profile,omitempty"`
}

type Profile struct {
	City string `json:"city"`
}

func validatePerson(value Person) error {
	if value.Name == nil || strings.TrimSpace(*value.Name) == "" {
		return errors.New(`field "name" is required`)
	}
	if value.Age == nil || *value.Age < 0 || *value.Age > 150 {
		return errors.New(`field "age" must be between 0 and 150`)
	}
	if value.Tags == nil {
		return errors.New(`field "tags" is required`)
	}
	if value.Active == nil {
		return errors.New(`field "active" is required`)
	}
	if value.Profile != nil && strings.TrimSpace(value.Profile.City) == "" {
		return errors.New(`field "profile.city" must not be empty`)
	}
	return nil
}

const personSchema = `{"type":"object","additionalProperties":false,"required":["name","age","tags","active"],"properties":{"name":{"type":"string","minLength":1},"age":{"type":"integer","minimum":0,"maximum":150},"tags":{"type":"array","items":{"type":"string"}},"active":{"type":"boolean"},"profile":{"type":"object","additionalProperties":false,"required":["city"],"properties":{"city":{"type":"string","minLength":1}}}}}`

func TestParseResult(t *testing.T) {
	valid := `{"name":"Alice","age":18,"tags":["go","llm"],"active":true}`
	tests := []struct {
		name     string
		raw      string
		repair   JSONRepairer
		path     ParsePath
		wantCode ErrorCode
	}{
		{"raw valid JSON", valid, nil, ParsePathDirect, ""},
		{"markdown code block", "结果：\n```json\n" + valid + "\n```", nil, ParsePathExtracted, ""},
		{"bare keys single quotes trailing commas", `{name: 'Alice', age: 18, tags: ['go', 'llm',], active: true,}`, LocalJSONRepair, ParsePathLocalFix, ""},
		{"full width structural punctuation", `｛name:"Alice"，age:18，tags:["go"]，active:true｝`, LocalJSONRepair, ParsePathLocalFix, ""},
		{"incomplete outer JSON repaired locally", `{"name":"Alice","age":18,"tags":["go","llm"],"active":true`, LocalJSONRepair, ParsePathLocalFix, ""},
		{"unknown field rejected", valid[:len(valid)-1] + `,"role":"admin"}`, nil, "", ErrorCodeValidationFailed},
		{"empty required business field rejected", `{"name":"  ","age":18,"tags":["go"],"active":true}`, nil, "", ErrorCodeValidationFailed},
		{"wrong type rejected without LLM", `{"name":"Alice","age":"18","tags":["go"],"active":true}`, nil, "", ErrorCodeValidationFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseResult[Person](context.Background(), tt.raw, ParseOptions[Person]{LocalRepair: tt.repair, Validate: validatePerson})
			if tt.wantCode != "" {
				assertParseError(t, err, tt.wantCode)
			} else {
				if err != nil {
					t.Fatalf("parseResult error: %v", err)
				}
				if result.Path != tt.path {
					t.Fatalf("path = %q, want %q", result.Path, tt.path)
				}
				if result.Value.Name == nil || *result.Value.Name != "Alice" {
					t.Fatalf("unexpected value: %#v", result.Value)
				}
			}
		})
	}
}

func assertParseError(t *testing.T, err error, code ErrorCode) *ParseError {
	t.Helper()
	if err == nil {
		t.Fatalf("expected parse error %q", code)
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error type = %T, want *ParseError: %v", err, err)
	}
	if parseErr.Code != code {
		t.Fatalf("error code = %q, want %q: %v", parseErr.Code, code, err)
	}
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("error must match ErrInvalidOutput: %v", err)
	}
	return parseErr
}

func TestDecodeAndValidateRejectsMultipleJSONValues(t *testing.T) {
	_, err := decodeAndValidate[Person]([]byte(`{"name":"Alice","age":18,"tags":[],"active":true} {"name":"Bob","age":20,"tags":[],"active":false}`), nil, validatePerson)
	if err == nil {
		t.Fatal("expected multiple JSON values to be rejected")
	}
}

func TestExtractJSONCandidates(t *testing.T) {
	candidates, limitReached := extractJSONCandidatesWithLimits(
		"说明\n```json\n{\"name\":\"Alice\"}\n```\n其他 {\"ignored\":true}",
		normalizeLimits(ParseLimits{}),
	)
	if limitReached {
		t.Fatal("default candidate limit should not be reached")
	}
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2: %#v", len(candidates), candidates)
	}
}

func TestParseTopLevelArray(t *testing.T) {
	raw := `[{"name":"Alice","age":18,"tags":[],"active":true}]`
	result, err := Parse[[]Person](context.Background(), raw, ParseOptions[[]Person]{
		Validate: func(value []Person) error {
			if len(value) != 1 {
				return errors.New("expected one person")
			}
			return validatePerson(value[0])
		},
	})
	if err != nil {
		t.Fatalf("Parse array: %v", err)
	}
	if result.Path != ParsePathDirect || len(result.Value) != 1 {
		t.Fatalf("unexpected array result: %#v", result)
	}
}

func TestParseUsesSchemaAsStructuralContract(t *testing.T) {
	t.Run("valid value", func(t *testing.T) {
		result, err := Parse[Person](context.Background(), `{"name":"Alice","age":18,"tags":[],"active":true}`, ParseOptions[Person]{
			Schema: personSchema,
		})
		if err != nil {
			t.Fatalf("Parse with Schema: %v", err)
		}
		if result.Path != ParsePathDirect {
			t.Fatalf("path = %q, want %q", result.Path, ParsePathDirect)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		_, err := Parse[Person](context.Background(), `{"name":"Alice","tags":[],"active":true}`, ParseOptions[Person]{
			Schema: personSchema,
		})
		assertParseError(t, err, ErrorCodeValidationFailed)
	})

	t.Run("invalid Schema", func(t *testing.T) {
		_, err := Parse[Person](context.Background(), `{}`, ParseOptions[Person]{
			Schema: `{"type":`,
		})
		parseErr := assertParseError(t, err, ErrorCodeInvalidSchema)
		if parseErr.Stage != ParseStageSchema {
			t.Fatalf("stage = %q, want %q", parseErr.Stage, ParseStageSchema)
		}
	})

	t.Run("external Schema reference", func(t *testing.T) {
		_, err := Parse[Person](context.Background(), `{}`, ParseOptions[Person]{
			Schema: `{"$ref":"https://example.com/person.schema.json"}`,
		})
		assertParseError(t, err, ErrorCodeInvalidSchema)
	})
}

func TestParseUsesSafeLocalRepairByDefault(t *testing.T) {
	raw := `{name:'Alice',age:18,tags:[],active:true,}`
	result, err := Parse[Person](context.Background(), raw, ParseOptions[Person]{
		Schema: personSchema,
	})
	if err != nil {
		t.Fatalf("Parse with default local repair: %v", err)
	}
	if result.Path != ParsePathLocalFix {
		t.Fatalf("path = %q, want %q", result.Path, ParsePathLocalFix)
	}

	_, err = Parse[Person](context.Background(), raw, ParseOptions[Person]{
		Schema:             personSchema,
		DisableLocalRepair: true,
	})
	assertParseError(t, err, ErrorCodeInvalidJSON)
}

func TestParsePassesSchemaAndFailureToLLMRepair(t *testing.T) {
	raw := `{"name":"Alice","age":"wrong","tags":[],"active":true}`
	var received LLMRepairRequest
	result, err := Parse[Person](context.Background(), raw, ParseOptions[Person]{
		Schema: personSchema,
		LLMRepair: func(_ context.Context, request LLMRepairRequest) (string, error) {
			received = request
			return `{"name":"Alice","age":18,"tags":[],"active":true}`, nil
		},
		AllowLLMSemanticRepair: true,
	})
	if err != nil {
		t.Fatalf("Parse with LLM repair: %v", err)
	}
	if result.Path != ParsePathLLMFix {
		t.Fatalf("path = %q, want %q", result.Path, ParsePathLLMFix)
	}
	if received.RawOutput != raw || received.Schema != personSchema || received.ValidationFailure == "" {
		t.Fatalf("unexpected repair request: %#v", received)
	}
}

func TestParseTopLevelArrayThroughLLMRepair(t *testing.T) {
	schema := `{"type":"array","minItems":1,"items":{"type":"integer"}}`
	result, err := Parse[[]int](context.Background(), "numbers are one and two", ParseOptions[[]int]{
		Schema: schema,
		LLMRepair: func(_ context.Context, request LLMRepairRequest) (string, error) {
			if request.Schema != schema {
				t.Fatalf("Schema = %q, want %q", request.Schema, schema)
			}
			return `[1,2]`, nil
		},
	})
	if err != nil {
		t.Fatalf("Parse array with LLM repair: %v", err)
	}
	if result.Path != ParsePathLLMFix || len(result.Value) != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseSelectsValidCandidate(t *testing.T) {
	raw := `first {"name":"","age":18,"tags":[],"active":true} second {"name":"Bob","age":20,"tags":[],"active":false}`
	result, err := Parse[Person](context.Background(), raw, ParseOptions[Person]{Validate: validatePerson})
	if err != nil {
		t.Fatalf("Parse multiple candidates: %v", err)
	}
	if result.Path != ParsePathExtracted || result.Value.Name == nil || *result.Value.Name != "Bob" {
		t.Fatalf("unexpected selected candidate: %#v", result)
	}
}

func TestParseLimits(t *testing.T) {
	t.Run("input bytes", func(t *testing.T) {
		_, err := Parse[Person](context.Background(), `{}`, ParseOptions[Person]{
			Limits: ParseLimits{MaxInputBytes: 1},
		})
		parseErr := assertParseError(t, err, ErrorCodeInputTooLarge)
		if parseErr.Stage != ParseStageInput {
			t.Fatalf("stage = %q, want %q", parseErr.Stage, ParseStageInput)
		}
	})

	t.Run("candidate count", func(t *testing.T) {
		raw := `broken {bad} valid {"name":"Alice","age":18,"tags":[],"active":true}`
		_, err := Parse[Person](context.Background(), raw, ParseOptions[Person]{
			Validate: validatePerson,
			Limits:   ParseLimits{MaxCandidates: 1},
		})
		assertParseError(t, err, ErrorCodeCandidateLimit)
	})
}

func TestParseLLMRepairFailureIsStructuredAndWrapped(t *testing.T) {
	repairFailure := errors.New("provider unavailable")
	_, err := Parse[Person](context.Background(), "not json", ParseOptions[Person]{
		Schema: personSchema,
		LLMRepair: func(context.Context, LLMRepairRequest) (string, error) {
			return "", repairFailure
		},
	})
	parseErr := assertParseError(t, err, ErrorCodeLLMRepairFailed)
	if parseErr.Stage != ParseStageLLMRepair {
		t.Fatalf("stage = %q, want %q", parseErr.Stage, ParseStageLLMRepair)
	}
	if !errors.Is(err, repairFailure) {
		t.Fatalf("error must wrap repair failure: %v", err)
	}
}

func TestParseRejectsInvalidLLMRepairOutput(t *testing.T) {
	_, err := Parse[Person](context.Background(), "not json", ParseOptions[Person]{
		Schema: personSchema,
		LLMRepair: func(context.Context, LLMRepairRequest) (string, error) {
			return `{"name":"Alice","age":"wrong"}`, nil
		},
	})
	parseErr := assertParseError(t, err, ErrorCodeLLMRepairFailed)
	if parseErr.Stage != ParseStageValidation {
		t.Fatalf("stage = %q, want %q", parseErr.Stage, ParseStageValidation)
	}
}

func TestParseTruncatedStringFailsWithoutInventingContent(t *testing.T) {
	raw := `{"name":"Alice","age":18,"tags":["unfinished]`
	_, err := Parse[Person](context.Background(), raw, ParseOptions[Person]{
		LocalRepair: LocalJSONRepair,
		Validate:    validatePerson,
	})
	if err == nil {
		t.Fatal("expected truncated string to fail")
	}
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("error must match ErrInvalidOutput: %v", err)
	}
}

func TestLocalJSONRepairRejectsLossyChanges(t *testing.T) {
	_, err := LocalJSONRepair(`{"name":}`)
	if !errors.Is(err, ErrLossyRepair) {
		t.Fatalf("error = %v, want ErrLossyRepair", err)
	}

	repaired, err := PermissiveLocalJSONRepair(`{"name":}`)
	if err != nil {
		t.Fatalf("PermissiveLocalJSONRepair: %v", err)
	}
	if repaired != `{"name":null}` {
		t.Fatalf("repaired = %s, want null insertion", repaired)
	}
}

func TestParseReportsRejectedLossyRepair(t *testing.T) {
	_, err := Parse[map[string]any](context.Background(), `{"name":}`, ParseOptions[map[string]any]{
		LocalRepair: LocalJSONRepair,
	})
	assertParseError(t, err, ErrorCodeLossyRepair)
}
