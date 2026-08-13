package jsonguard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// decodeAndValidate 是所有解析路径共享的最终信任边界。
// 只有通过该函数，恢复结果才会被视为成功。
func decodeAndValidate[T any](input []byte, schema *compiledSchema, validate Validator[T]) (T, error) {
	// 1. 解码为调用方类型并拒绝未知字段，防止合法 JSON 中的额外数据被静默丢弃。
	var value T
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode JSON: %w", err)
	}

	// 2. 限制输入只能包含一个 JSON 值，避免首个合法值掩盖尾部说明或冲突数据。
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return value, errors.New("unexpected additional JSON value")
		}
		return value, fmt.Errorf("decode trailing JSON content: %w", err)
	}

	// 3. 使用调用方 Schema 校验原始 JSON 值，使必填字段、数组和组合约束保持统一来源。
	if err := schema.validate(input); err != nil {
		return value, err
	}

	// 4. Schema 通过后再执行领域规则，补充跨字段等 JSON Schema 难以表达的业务约束。
	if validate != nil {
		if err := validate(value); err != nil {
			return value, fmt.Errorf("validate business value: %w", err)
		}
	}
	return value, nil
}
