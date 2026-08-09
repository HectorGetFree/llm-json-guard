# llm-json-guard

一个 Go 工具包，用于将 LLM 的非稳定结构化输出安全地转换为目标 Go 类型。它遵循：**严格解码优先 → 默认安全修复 → 最多一次 LLM 兜底 → Schema 与业务规则统一验收**。

项目当前能力、处理链路和成熟度边界见 [`docs/current_implementation.md`](docs/current_implementation.md)。

## 功能

- 从原始 JSON、Markdown 代码块和普通文本中提取 JSON 候选；
- 处理全角结构标点，并保留不完整的最外层对象 / 数组供 repair 使用；
- 默认使用安全模式修复裸键、单引号、尾逗号和缺失闭合符号，拒绝补值、删值和根结构重组；
- 使用 `encoding/json` 严格解码，拒绝未知字段和多个连续 JSON 值；
- 使用同一份 JSON Schema 完成本地结构校验和 LLM 修复约束；
- 通过可选业务回调补充跨字段等领域规则；
- 在本地无法处理时，支持一次 OpenAI-compatible LLM 修复请求；
- 核心链路和 LLM 兜底均支持对象与数组根类型；
- 默认拒绝语义修复，只有显式开启后才允许 LLM 修复合法 JSON 的 Schema / 业务错误。

## 目录结构

```text
.
├── parser.go                # 解析与恢复流程调度
├── extractor.go             # JSON 候选提取和全角标点标准化
├── decoder.go               # 严格解码和业务校验
├── schema.go                # JSON Schema 编译和本地校验
├── repair.go                # 安全与宽松本地修复策略
├── errors.go                # 结构化错误码和阶段
├── types.go                 # 公共类型、选项和默认限制
├── llm.go                   # OpenAI-compatible Chat Completions 修复器
├── parser_test.go           # 本地测试，调用真实 jsonrepair，但不访问网络
├── llm_integration_test.go  # 配置环境变量后调用真实 LLM
├── evaluation/              # 内置 JSONL 数据集、评测 Runner 和指标说明
├── cmd/guard-eval/          # 离线评测命令入口
├── go.mod                   # Go module 与依赖
└── README.md
```

## 依赖

```text
Go:          1.24.2
JSON repair:  github.com/kaptinlin/jsonrepair v0.1.1
LLM endpoint: OpenAI-compatible Chat Completions
```

## 使用方式

### 定义目标结构、Schema 和业务校验

```go
type Person struct {
    Name   *string  `json:"name"`
    Age    *int     `json:"age"`
    Tags   []string `json:"tags"`
    Active *bool    `json:"active"`
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
    return nil
}

const personSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["name", "age", "tags", "active"],
  "properties": {
    "name": {"type": "string", "minLength": 1},
    "age": {"type": "integer", "minimum": 0, "maximum": 150},
    "tags": {"type": "array", "items": {"type": "string"}},
    "active": {"type": "boolean"}
  }
}`
```

Schema 负责必填字段、类型、范围和数组等结构规则；`Validate` 只补充跨字段或外部状态相关的业务规则。Schema 会在一次 `Parse` 开始时编译一次，并由所有恢复路径复用。外部 `$ref` 默认禁止，避免不可信 Schema 读取文件或访问网络。

### 仅使用本地提取、repair 和校验

```go
result, err := llmjsonguard.Parse[Person](ctx, rawModelOutput, llmjsonguard.ParseOptions[Person]{
    Schema:   personSchema,
    Validate: validatePerson,
})
if err != nil {
    return err
}

person := result.Value
fmt.Println(result.Path) // direct、extracted 或 local_repair
```

### 配置一次 LLM 兜底

```go
repairer, err := llmjsonguard.NewLLMRepairer(llmjsonguard.LLMConfig{
    BaseURL: os.Getenv("MODELJSON_LLM_BASE_URL"),
    APIKey:  os.Getenv("MODELJSON_LLM_API_KEY"),
    Model:   os.Getenv("MODELJSON_LLM_MODEL"),
})
if err != nil {
    return err
}

result, err := llmjsonguard.Parse[Person](ctx, rawModelOutput, llmjsonguard.ParseOptions[Person]{
    Schema:                  personSchema,
    LLMRepair:               repairer,
    Validate:                validatePerson,
    AllowLLMSemanticRepair: true,
})
```

自定义公司修复服务时实现同一个函数类型即可：

```go
companyRepairer := func(ctx context.Context, request llmjsonguard.LLMRepairRequest) (string, error) {
    // request 同时包含 RawOutput、Schema 和本地 ValidationFailure。
    return companyClient.Repair(ctx, request)
}
```

`ParseResult.Path` 的可能值：

```text
direct        原始文本直接通过严格校验
extracted     从 Markdown 或普通文本提取出的候选通过严格校验
local_repair  真实 jsonrepair 修复后通过严格校验
llm_repair    一次 LLM 修复后通过严格校验
```

旧名称 `ParsePathRaw` 保留为 `ParsePathDirect` 的兼容别名。

### 输入限制与结构化错误

解析器默认限制输入、Schema 和单个候选为 1 MiB、候选数量为 32，可通过
`ParseOptions.Limits` 调整：

```go
Limits: llmjsonguard.ParseLimits{
    MaxInputBytes:     2 << 20,
    MaxSchemaBytes:    1 << 20,
    MaxCandidateBytes: 1 << 20,
    MaxCandidates:     16,
}
```

失败会返回 `*llmjsonguard.ParseError`，调用方可通过 `errors.As` 读取稳定的
`Code` 和 `Stage`；所有解析失败仍可通过 `errors.Is(err,
llmjsonguard.ErrInvalidOutput)` 判断。

## 设计逻辑

```text
模型原始输出
  → 编译一次 JSON Schema
  → 原样严格解码 + Schema + 业务校验
  → 标准化全角结构标点
  → 提取完整或不完整 JSON 候选
  → 逐个严格解码 + Schema + 业务校验
  → 仅对语法错误候选执行安全本地修复
  → 再次严格解码 + Schema + 业务校验
  → 必要时最多调用一次 LLM
  → 严格解码 + Schema + 业务校验
  → 返回结果或错误
```

### 本地 repair 的边界

`LocalJSONRepair` 默认启用，只处理 JSON **语法和无损格式**问题，例如：

```js
{name: 'Alice', tags: ['go',], active: true,}
```

以下可能改变业务事实的修复会返回 `ErrLossyRepair`：

```text
{"name":}          缺少值时补 null
[1, 2, ...]        删除省略内容
"hello" + "world" 合并值
多个根值           自动重组为数组
```

确实接受这些行为时，调用方必须显式传入：

```go
LocalRepair: llmjsonguard.PermissiveLocalJSONRepair
```

完全关闭本地修复：

```go
DisableLocalRepair: true
```

本地修复不会处理语义问题。以下输入虽然是合法 JSON，但会在严格解码、Schema 或业务校验时失败：

```json
{"name":"Alice","age":"18","tags":["go"],"active":true}
```

默认不会为这类问题调用 LLM。若确实需要由模型按 Schema 修复未知字段、空键或无歧义类型问题，才设置：

```go
AllowLLMSemanticRepair: true
```

无论 LLM 输出什么，都必须重新通过本地严格解码、同一份 Schema 和 `Validate`。

## LLM 配置

真实 LLM 修复器请求：

```text
POST ${MODELJSON_LLM_BASE_URL}/chat/completions
```

环境变量示例：

```bash
export MODELJSON_LLM_BASE_URL="https://your-host/v1"
export MODELJSON_LLM_API_KEY="replace-me"
export MODELJSON_LLM_MODEL="replace-me"
```

不要将真实 API Key 写入代码、`.env` 文件或 IDE 工作区配置；这些文件已经由 `.gitignore` 排除。

## 测试

运行本地测试：

```bash
GOTOOLCHAIN=local go test ./... -v -race
```

本地测试使用真实 `github.com/kaptinlin/jsonrepair`，覆盖：

- 原始合法 JSON；
- Markdown JSON 代码块；
- 裸键、单引号和尾逗号；
- 全角结构标点；
- 最外层 JSON 缺少闭合符号；
- Schema 必填字段、类型和外部引用限制；
- 默认安全修复与显式宽松修复；
- 未知字段；
- 必填业务字段为空；
- 字段类型错误；
- 多个连续 JSON 值；
- 多候选 JSON 提取；
- 顶层数组的本地解析与 LLM 兜底。

运行真实 LLM 集成测试：

```bash
GOTOOLCHAIN=local go test ./... -run '^TestRealLLMRepair$' -v -count=1
```

该测试会在环境变量已配置时真实调用模型：

- 合法 JSON 含未知字段、并开启 `AllowLLMSemanticRepair` 时，预期调用 LLM 一次；
- 截断 JSON 预期由本地 `jsonrepair` 修复，断言 LLM 调用次数为 `0`。

## 能力量化评测

运行不访问网络的默认评测集：

```bash
GOTOOLCHAIN=local go run ./cmd/guard-eval -iterations 100
```

输出包括：

- 总体准确率；
- 直接成功和恢复成功数量；
- 恢复成功率与恢复增益；
- 误接收率、误拒绝数和错误值数量；
- `direct`、`extracted`、`local_repair` 路径分布；
- 错误码和分类准确率；
- 端到端 mean、P50 和 P95 延迟。

用于 CI 阈值检查：

```bash
GOTOOLCHAIN=local go run ./cmd/guard-eval \
  -iterations 100 \
  -min-accuracy 0.95 \
  -max-false-acceptance 0.15
```

输出机器可读报告：

```bash
GOTOOLCHAIN=local go run ./cmd/guard-eval -json
```

数据集格式、指标定义和已知缺口见 [`evaluation/README.md`](evaluation/README.md)。

## 运行时统计

`ParseOptions.Observer` 会在每次公开 `Parse` 调用完成后同步接收一条基础观测记录：

```go
observer := func(ctx context.Context, observation llmjsonguard.ParseObservation) {
    metrics.Total.Add(1)
    metrics.Duration.Observe(observation.Duration.Seconds())
    metrics.Paths.Add(string(observation.Path), 1)
    metrics.Errors.Add(string(observation.ErrorCode), 1)
}

result, err := llmjsonguard.Parse[Person](ctx, rawModelOutput, llmjsonguard.ParseOptions[Person]{
    Schema:   personSchema,
    Observer: observer,
})
```

`ParseObservation` 只包含：

- 成功状态、最终路径、错误码和失败阶段；
- Guard 自身总耗时；
- 输入和 Schema 字节数；
- 候选数量与候选限制状态；
- 本地修复尝试次数和 LLM 调用次数。

它不会记录原始输出、Schema、候选、最终 JSON 或错误详情。Observer 每次调用最多执行一次，不返回错误；其 panic 会被隔离，不会改变解析结果。回调是同步的，但 `Duration` 在调用 Observer 前完成计算，因此不包含监控写入耗时。

同一个 Observer 可能被并发调用，实现方应保证线程安全，并把日志、数据库或网络写入放进自有的有界异步队列，避免阻塞 Guard 链路。

## 安全说明

- 上传 GitHub 前请运行 `git status --short`，确认没有 `.env`、`.idea`、密钥或本地配置文件；
- 一旦 API Key 曾进入 Git 历史，应立即在服务商侧撤销并轮换；
- 不要把真实 LLM 集成测试放进持续运行的无条件 CI，以免产生非预期费用。
