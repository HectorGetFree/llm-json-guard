# jsonguard

`jsonguard` 将大模型原始文本转换为可严格反序列化的 Go 值。它不依赖具体模型或 RPC，也不承担业务规则校验。

```text
原始解析 → JSON 候选提取 → 内置安全修复
                              ↓ 失败
                         可选 LLM 修复
                              ↓
                         再走本地链路
```

## 本地解析

```go
value, jsonText, err := jsonguard.Parse[Target](ctx, rawOutput, "", nil)
```

## LLM 兜底

```go
llmRepair := func(ctx context.Context, req jsonguard.LLMRepairRequest) (string, error) {
    resp, err := llmClient.Repair(ctx, &llm.LLMJSONRepairReq{
        RawOutput:     req.RawOutput,
        JsonFormat:    req.JSONFormat,
        FailureReason: req.FailureReason,
    })
    if err != nil {
        return "", err
    }
    return resp.GetRepairedJson(), nil
}

value, jsonText, err := jsonguard.Parse[Target](
    ctx,
    rawOutput,
    originalPromptJSONFormat,
    llmRepair,
)
```

- 本地链路成功时不会调用 LLM。
- LLM 每次 `Parse` 最多调用一次，其输出仍须通过本地链路。
- `JSONFormat` 是自然语言格式说明，可直接复用原 Prompt 的输出格式部分。
- `T` 支持结构体、map、数组和基础类型。
- 缺失字段、零值和跨字段关系由业务方在成功返回后自行校验。
