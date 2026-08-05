# llm-json-guard

一个 Go 工具包，用于将 LLM 的非稳定结构化输出安全地转换为目标 Go 类型。它遵循：**严格解码优先 → 本地确定性修复 → 最多一次 LLM 兜底 → 最终本地验证**。

## 功能

- 从原始 JSON、Markdown 代码块和普通文本中提取 JSON 候选；
- 处理全角结构标点，并保留不完整的最外层对象 / 数组供 repair 使用；
- 使用真实 `jsonrepair` 修复裸键、单引号、尾逗号和缺失闭合符号等格式问题；
- 使用 `encoding/json` 严格解码，拒绝未知字段和多个连续 JSON 值；
- 通过业务回调验证必填字段、空值、范围、枚举和跨字段约束；
- 在本地无法处理时，支持一次 OpenAI-compatible LLM 修复请求；
- 默认拒绝语义修复，只有显式开启后才允许 LLM 修复合法 JSON 的 Schema / 业务错误。

## 目录结构

```text
.
├── parser.go                # 提取、严格解码、本地 repair 与 LLM 调度
├── llm.go                   # OpenAI-compatible Chat Completions 修复器
├── parser_test.go           # 本地测试，调用真实 jsonrepair，但不访问网络
├── llm_integration_test.go  # 配置环境变量后调用真实 LLM
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

### 定义目标结构和业务校验

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
```

### 仅使用本地提取、repair 和校验

```go
result, err := modeljson.Parse[Person](ctx, rawModelOutput, modeljson.ParseOptions[Person]{
    LocalRepair: modeljson.LocalJSONRepair,
    Validate:    validatePerson,
})
if err != nil {
    return err
}

person := result.Value
fmt.Println(result.Path) // raw 或 local_repair
```

### 配置一次 LLM 兜底

```go
schema := `{
  "type": "object",
  "additionalProperties": false,
  "required": ["name", "age", "tags", "active"]
}`

repairer, err := modeljson.NewLLMRepairer(modeljson.LLMConfig{
    BaseURL: os.Getenv("MODELJSON_LLM_BASE_URL"),
    APIKey:  os.Getenv("MODELJSON_LLM_API_KEY"),
    Model:   os.Getenv("MODELJSON_LLM_MODEL"),
}, schema)
if err != nil {
    return err
}

result, err := modeljson.Parse[Person](ctx, rawModelOutput, modeljson.ParseOptions[Person]{
    LocalRepair:             modeljson.LocalJSONRepair,
    LLMRepair:               repairer,
    Validate:                validatePerson,
    AllowLLMSemanticRepair: true,
})
```

`ParseResult.Path` 的可能值：

```text
raw           原始文本或提取出的候选直接通过严格校验
local_repair  真实 jsonrepair 修复后通过严格校验
llm_repair    一次 LLM 修复后通过严格校验
```

## 设计逻辑

```text
模型原始输出
  → 原样严格解码 + 业务校验
  → 标准化全角结构标点
  → 提取完整或不完整 JSON 候选
  → 逐个严格解码 + 业务校验
  → 仅对语法错误候选执行 jsonrepair
  → 再次严格解码 + 业务校验
  → 必要时最多调用一次 LLM
  → 严格解码 + 业务校验
  → 返回结果或错误
```

### 本地 repair 的边界

`LocalJSONRepair` 只处理 JSON **语法和格式**问题，例如：

```js
{name: 'Alice', tags: ['go',], active: true,}
```

它不会处理语义问题。以下输入虽然是合法 JSON，但会在严格解码或业务校验时失败：

```json
{"name":"Alice","age":"18","tags":["go"],"active":true}
```

默认不会为这类问题调用 LLM。若确实需要由模型按 Schema 修复未知字段、空键或无歧义类型问题，才设置：

```go
AllowLLMSemanticRepair: true
```

无论 LLM 输出什么，都必须重新通过本地严格解码和 `Validate`。

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
- 未知字段；
- 必填业务字段为空；
- 字段类型错误；
- 多个连续 JSON 值；
- 多候选 JSON 提取。

运行真实 LLM 集成测试：

```bash
GOTOOLCHAIN=local go test ./... -run '^TestRealLLMRepair$' -v -count=1
```

该测试会在环境变量已配置时真实调用模型：

- 合法 JSON 含未知字段、并开启 `AllowLLMSemanticRepair` 时，预期调用 LLM 一次；
- 截断 JSON 预期由本地 `jsonrepair` 修复，断言 LLM 调用次数为 `0`。

## 安全说明

- 上传 GitHub 前请运行 `git status --short`，确认没有 `.env`、`.idea`、密钥或本地配置文件；
- 一旦 API Key 曾进入 Git 历史，应立即在服务商侧撤销并轮换；
- 不要把真实 LLM 集成测试放进持续运行的无条件 CI，以免产生非预期费用。
