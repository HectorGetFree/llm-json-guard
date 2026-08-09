package modeljson

import jsonrepair "github.com/kaptinlin/jsonrepair"

// LocalJSONRepair 在 LLM 兜底前提供确定性语法恢复。
// 该层只处理语法非法的 JSON，不负责判断 Schema 或业务语义。
func LocalJSONRepair(input string) (string, error) {
	return jsonrepair.JSONRepair(input)
}
