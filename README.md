# llm-json-guard

独立测试环境：验证模型 JSON 输出的提取、严格解码、真实 `jsonrepair` 本地修复和一次真实 LLM 兜底。

## 本地测试

```bash
go test ./... -v -race
```

常规单元测试会调用真实 `github.com/kaptinlin/jsonrepair`，但不访问网络，也不会产生模型费用。真实 LLM 调用只存在于 `llm_integration_test.go`，不再使用 LLM mock。

## 真实 LLM 集成测试

该模块通过 OpenAI-compatible Chat Completions 接口调用修复模型。先填入自己的值：

```bash
export MODELJSON_LLM_BASE_URL='https://your-host/v1'
export MODELJSON_LLM_API_KEY='replace-me'
export MODELJSON_LLM_MODEL='replace-me'

go test ./... -run '^TestRealLLMRepair$' -v -count=1
```

`MODELJSON_LLM_BASE_URL` 必须包含 `/v1`，代码会请求：

```text
${MODELJSON_LLM_BASE_URL}/chat/completions
```

该集成测试包含两条路径：一条是合法 JSON 因未知字段而未通过 Schema 校验，并显式允许语义修复，因此会真实调用一次 LLM；另一条是缺少最外层闭合符号的 JSON，提取器会保留其不完整候选并交给真实 `jsonrepair` 修复，预期不会调用 LLM。修复器会把 JSON Schema、原始输出和“不得编造字段”的约束传给模型；其输出仍会经过 Go 的严格解码和业务校验。
