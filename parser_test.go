package modeljson

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

func TestParseResult(t *testing.T) {
	valid := `{"name":"Alice","age":18,"tags":["go","llm"],"active":true}`
	tests := []struct {
		name    string
		raw     string
		repair  JSONRepairer
		path    ParsePath
		wantErr string
	}{
		{"raw valid JSON", valid, nil, ParsePathRaw, ""},
		{"markdown code block", "结果：\n```json\n" + valid + "\n```", nil, ParsePathRaw, ""},
		{"bare keys single quotes trailing commas", `{name: 'Alice', age: 18, tags: ['go', 'llm',], active: true,}`, LocalJSONRepair, ParsePathLocalFix, ""},
		{"full width structural punctuation", `｛name:"Alice"，age:18，tags:["go"]，active:true｝`, LocalJSONRepair, ParsePathLocalFix, ""},
		{"incomplete outer JSON repaired locally", `{"name":"Alice","age":18,"tags":["go","llm"],"active":true`, LocalJSONRepair, ParsePathLocalFix, ""},
		{"unknown field rejected", valid[:len(valid)-1] + `,"role":"admin"}`, nil, "", "schema or business rules"},
		{"empty required business field rejected", `{"name":"  ","age":18,"tags":["go"],"active":true}`, nil, "", `field "name" is required`},
		{"wrong type rejected without LLM", `{"name":"Alice","age":"18","tags":["go"],"active":true}`, nil, "", "schema or business rules"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseResult[Person](context.Background(), tt.raw, ParseOptions[Person]{LocalRepair: tt.repair, Validate: validatePerson})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
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

func TestDecodeAndValidateRejectsMultipleJSONValues(t *testing.T) {
	_, err := decodeAndValidate[Person]([]byte(`{"name":"Alice","age":18,"tags":[],"active":true} {"name":"Bob","age":20,"tags":[],"active":false}`), validatePerson)
	if err == nil {
		t.Fatal("expected multiple JSON values to be rejected")
	}
}

func TestExtractJSONCandidates(t *testing.T) {
	candidates := extractJSONCandidates("说明\n```json\n{\"name\":\"Alice\"}\n```\n其他 {\"ignored\":true}")
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2: %#v", len(candidates), candidates)
	}
}
