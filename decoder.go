package jsonguard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// decodeStrict 是所有成功路径共享的最终解析边界。
// 它只保证 JSON 能严格反序列化为 T，不承担字段完整性或业务规则校验。
func decodeStrict[T any](input []byte) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("JSON 解码失败: %w", err)
	}

	// 一个候选只能包含一个根 JSON 值，避免前一个合法值掩盖后续冲突内容。
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return value, errors.New("JSON 后存在额外的 JSON 值")
		}
		return value, fmt.Errorf("JSON 尾随内容解码失败: %w", err)
	}
	return value, nil
}
