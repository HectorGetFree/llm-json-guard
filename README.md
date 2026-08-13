# llm-json-guard

`llm-json-guard` 是一个纯 Go 工具包，用于将不可信的大模型文本转换为经过统一验收的业务对象。

它不依赖模型 SDK、RPC 协议或具体服务，默认执行：

```text
严格解析 → JSON 候选提取 → 安全本地修复 → Schema → Validator
       → 可选 RepairFunc → 再次执行完整验收
```

## 安装

```bash
go get github.com/HectorGetFree/llm-json-guard
```

包名为 `jsonguard`：

```go
import jsonguard "github.com/HectorGetFree/llm-json-guard"
```

## 基础使用

```go
result, err := jsonguard.Parse[Target](ctx, rawOutput, jsonguard.ParseOptions[Target]{
    Schema:   targetSchema,
    Validate: validateTarget,
})
if err != nil {
    return err
}

value := result.Value
trustedJSON := result.JSON
```

`Schema` 和 `Validate` 均可选。不传时仍会执行严格解析、候选提取和默认安全本地修复。

## 外部修复

仓库不内置 LLM 或网络调用，只提供通用扩展点：

```go
type RepairRequest struct {
    RawOutput         string
    Schema            string
    ValidationFailure string
}

type RepairFunc func(
    ctx context.Context,
    request RepairRequest,
) (string, error)
```

调用方可以自由接入 RPC、内部模型或其他修复服务：

```go
repairFunc := func(ctx context.Context, req jsonguard.RepairRequest) (string, error) {
    return repairService.Repair(ctx, req)
}

result, err := jsonguard.Parse[Target](ctx, rawOutput, jsonguard.ParseOptions[Target]{
    Schema:              targetSchema,
    Validate:            validateTarget,
    Repair:              repairFunc,
    AllowSemanticRepair: true,
})
```

约束：

- 配置 `Repair` 时必须同时提供 Schema；
- 每次 `Parse` 最多调用一次外部修复；
- Schema 或业务语义错误默认不会交给外部修复；
- 只有显式启用 `AllowSemanticRepair` 才允许尝试语义修复；
- 外部修复结果仍须通过严格解码、同一份 Schema 和 Validator。

## Schema 与 Validator

```text
Schema
→ 校验 required、type、范围、数组元素和额外字段等 JSON 结构契约。

Validator
→ 校验跨字段关系、业务状态和外部数据等领域规则。
```

Schema 直接校验原始 JSON，避免缺失字段被 Go 零值掩盖。所有成功路径均按以下顺序验收：

```text
严格解码为 T
→ 拒绝未知字段
→ 拒绝尾随内容和多个根 JSON 值
→ Schema
→ Validator
```

## 本地修复边界

默认 `LocalJSONRepair` 只接受可以确定的语法恢复，例如裸键、单引号、尾逗号和缺失闭合符号。

可能补值、删值或重组根结构的输入会返回 `ErrLossyRepair`。确实接受第三方库的完整恢复行为时，可以显式传入：

```go
LocalRepair: jsonguard.PermissiveLocalJSONRepair
```

完全关闭本地修复：

```go
DisableLocalRepair: true
```

## 结果与错误

成功路径：

```text
direct           原始输出直接通过。
extracted        提取出的 JSON 候选通过。
local_repair     安全本地修复结果通过。
external_repair  RepairFunc 返回结果通过。
```

失败返回 `*jsonguard.ParseError`，其中包含稳定的 `Code`、`Stage` 和根因。所有解析失败也支持：

```go
errors.Is(err, jsonguard.ErrInvalidOutput)
```

`ParseOptions.Limits` 可以限制输入、Schema、候选大小和候选数量，避免异常模型输出消耗过多资源。

## 测试

```bash
GOTOOLCHAIN=local go test ./... -race
```

测试覆盖对象与数组、Markdown 提取、严格解码、Schema、Validator、安全本地修复、外部修复及各类限制。测试不会访问模型或网络。
