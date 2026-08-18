package jsonguard

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type testPerson struct {
	Name   string   `json:"name"`
	Age    int      `json:"age"`
	Tags   []string `json:"tags"`
	Active bool     `json:"active"`
}

type testIdea struct {
	ID    string `json:"id"`
	Ideas string `json:"ideas"`
}

// TestParseDirectTypes 验证结构体、map、数组和基础类型都可作为目标类型。
func TestParseDirectTypes(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{"结构体", func(t *testing.T) {
			value, jsonText, err := Parse[testPerson](context.Background(), `{"name":"Alice","age":18,"tags":[],"active":true}`, "", nil)
			if err != nil || value.Name != "Alice" || jsonText == "" {
				t.Fatalf("value=%#v json=%q err=%v", value, jsonText, err)
			}
		}},
		{"map", func(t *testing.T) {
			value, _, err := Parse[map[string]any](context.Background(), `{"name":"Alice"}`, "", nil)
			if err != nil || value["name"] != "Alice" {
				t.Fatalf("value=%#v err=%v", value, err)
			}
		}},
		{"数组", func(t *testing.T) {
			value, _, err := Parse[[]int](context.Background(), `[1,2,3]`, "", nil)
			if err != nil || len(value) != 3 {
				t.Fatalf("value=%#v err=%v", value, err)
			}
		}},
		{"基础类型", func(t *testing.T) {
			value, _, err := Parse[bool](context.Background(), `true`, "", nil)
			if err != nil || !value {
				t.Fatalf("value=%v err=%v", value, err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

// TestParseExtractsMarkdownCandidate 验证 Markdown 包裹不会阻断 JSON 解析。
func TestParseExtractsMarkdownCandidate(t *testing.T) {
	value, jsonText, err := Parse[[]int](context.Background(), "结果如下：\n```json\n[1,2]\n```", "", nil)
	if err != nil || len(value) != 2 || jsonText != `[1,2]` {
		t.Fatalf("value=%#v json=%q err=%v", value, jsonText, err)
	}
}

// TestParseUsesDefaultLocalRepair 验证未闭合 JSON 会自动进入内置安全修复器。
func TestParseUsesDefaultLocalRepair(t *testing.T) {
	value, jsonText, err := Parse[map[string]string](context.Background(), `{"name":"Alice"`, "", nil)
	if err != nil || value["name"] != "Alice" || jsonText != `{"name":"Alice"}` {
		t.Fatalf("value=%#v json=%q err=%v", value, jsonText, err)
	}
}

// TestParseEscapesRawNewlineInString 验证模型将真实换行写入字符串时，本地修复器会转义换行且不改变文本内容。
func TestParseEscapesRawNewlineInString(t *testing.T) {
	raw := "[{\"id\":\"1\",\"ideas\":\"第一段内容\n第二段内容\"}]"

	value, jsonText, err := Parse[[]testIdea](context.Background(), raw, "", nil)
	if err != nil {
		t.Fatalf("parse raw newline: %v", err)
	}
	if len(value) != 1 || value[0].ID != "1" || value[0].Ideas != "第一段内容\n第二段内容" {
		t.Fatalf("value=%#v", value)
	}
	if !strings.Contains(jsonText, `第一段内容\n第二段内容`) {
		t.Fatalf("json=%q must contain escaped newline", jsonText)
	}
}

// TestParseDoesNotCallLLMWhenLocalSucceeds 验证 LLM 只作为本地链路失败后的兜底。
func TestParseDoesNotCallLLMWhenLocalSucceeds(t *testing.T) {
	called := 0
	value, _, err := Parse[map[string]string](context.Background(), `{"name":"Alice"`, "输出对象", func(context.Context, LLMRepairRequest) (string, error) {
		called++
		return "", errors.New("不应调用")
	})
	if err != nil || value["name"] != "Alice" || called != 0 {
		t.Fatalf("value=%#v called=%d err=%v", value, called, err)
	}
}

// TestParseRepairsBooleanStringWithLLM 验证类型纠正只在本地失败后交给 LLM。
func TestParseRepairsBooleanStringWithLLM(t *testing.T) {
	raw := `{"name":"Alice","age":18,"tags":[],"active":"true"}`
	format := `输出对象，其中 active 必须为布尔值`
	var received LLMRepairRequest
	value, jsonText, err := Parse[testPerson](context.Background(), raw, format, func(_ context.Context, request LLMRepairRequest) (string, error) {
		received = request
		return `{"name":"Alice","age":18,"tags":[],"active":true}`, nil
	})
	if err != nil || !value.Active || !strings.Contains(jsonText, `"active":true`) {
		t.Fatalf("value=%#v json=%q err=%v", value, jsonText, err)
	}
	if received.RawOutput != raw || received.JSONFormat != format || received.FailureReason == "" {
		t.Fatalf("request=%#v", received)
	}
}

// TestParseReprocessesLLMOutputLocally 验证 LLM 返回的 Markdown 和简单语法错误仍会经过本地链路。
func TestParseReprocessesLLMOutputLocally(t *testing.T) {
	value, jsonText, err := Parse[map[string]bool](context.Background(), `{"active":"true"}`, "active 为布尔值", func(context.Context, LLMRepairRequest) (string, error) {
		return "```json\n{\"active\":true\n```", nil
	})
	if err != nil || !value["active"] || jsonText != `{"active":true}` {
		t.Fatalf("value=%#v json=%q err=%v", value, jsonText, err)
	}
}

// TestParseCallsLLMAtMostOnce 验证 LLM 结果失败后快速失败，不形成修复循环。
func TestParseCallsLLMAtMostOnce(t *testing.T) {
	called := 0
	_, _, err := Parse[map[string]bool](context.Background(), `{"active":"true"}`, "active 为布尔值", func(context.Context, LLMRepairRequest) (string, error) {
		called++
		return `{"active":"still wrong"}`, nil
	})
	if err == nil || called != 1 {
		t.Fatalf("called=%d err=%v", called, err)
	}
}

// TestParseRequiresFormatForLLM 验证启用 LLM 时必须提供自然语言格式说明。
func TestParseRequiresFormatForLLM(t *testing.T) {
	_, _, err := Parse[map[string]any](context.Background(), `{}`, "", func(context.Context, LLMRepairRequest) (string, error) {
		return `{}`, nil
	})
	assertParseErrorCode(t, err, ErrorCodeInvalidConfig)
}

// TestParseReturnsLLMFailure 验证模型调用错误会保留为可识别的修复失败。
func TestParseReturnsLLMFailure(t *testing.T) {
	want := errors.New("模型不可用")
	_, _, err := Parse[map[string]bool](context.Background(), `{"active":"true"}`, "active 为布尔值", func(context.Context, LLMRepairRequest) (string, error) {
		return "", want
	})
	assertParseErrorCode(t, err, ErrorCodeLLMRepairFailed)
	if !errors.Is(err, want) {
		t.Fatalf("error must wrap model failure: %v", err)
	}
}

// TestParseRejectsUnknownFields 验证结构体目标仍拒绝模型输出中的额外字段。
func TestParseRejectsUnknownFields(t *testing.T) {
	_, _, err := Parse[testPerson](context.Background(), `{"name":"Alice","age":18,"tags":[],"active":true,"extra":1}`, "", nil)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
}

// TestDecodeStrictRejectsMultipleRootValues 验证单个候选不能携带多个根 JSON 值。
func TestDecodeStrictRejectsMultipleRootValues(t *testing.T) {
	_, err := decodeStrict[map[string]any]([]byte(`{"name":"Alice"} {}`))
	if err == nil {
		t.Fatal("expected multiple root values error")
	}
}

func assertParseErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	var parseErr *ParseError
	if !errors.As(err, &parseErr) || parseErr.Code != want {
		t.Fatalf("error=%v want code=%s", err, want)
	}
}
