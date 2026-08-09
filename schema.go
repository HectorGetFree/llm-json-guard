package llmjsonguard

import (
	"bytes"
	"fmt"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const schemaResourceURL = "urn:llm-json-guard:request-schema"

// compiledSchema 是一次 Parse 调用共享的结构契约。
// Schema 只编译一次，所有恢复路径复用同一实例，避免重复开销和规则漂移。
type compiledSchema struct {
	schema    *jsonschema.Schema
	rootTypes map[string]struct{}
}

// localOnlySchemaLoader 阻断根 Schema 之外的资源加载，避免解析配置时产生隐式 I/O。
type localOnlySchemaLoader struct{}

func (localOnlySchemaLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external Schema reference is not allowed: %s", url)
}

// compileJSONSchema 在处理模型输出前编译调用方契约。
// 空 Schema 保留 V1.1 兼容行为；一旦提供，非法 Schema 会作为配置错误立即终止链路。
func compileJSONSchema(source string) (*compiledSchema, error) {
	if strings.TrimSpace(source) == "" {
		return nil, nil
	}

	document, err := jsonschema.UnmarshalJSON(bytes.NewBufferString(source))
	if err != nil {
		return nil, fmt.Errorf("decode JSON Schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	// Schema 属于调用方提供的不可信配置，禁止编译器通过外部引用读取文件或访问网络。
	compiler.UseLoader(localOnlySchemaLoader{})
	if err := compiler.AddResource(schemaResourceURL, document); err != nil {
		return nil, fmt.Errorf("register JSON Schema: %w", err)
	}
	schema, err := compiler.Compile(schemaResourceURL)
	if err != nil {
		return nil, fmt.Errorf("compile JSON Schema: %w", err)
	}
	return &compiledSchema{schema: schema, rootTypes: schemaRootTypes(document)}, nil
}

// validate 校验原始 JSON 值，而不是 Go 结构体的序列化结果。
// 这样可以保留数字类型和字段存在性，确保 required、additionalProperties 等规则准确生效。
func (schema *compiledSchema) validate(input []byte) error {
	if schema == nil {
		return nil
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(input))
	if err != nil {
		return fmt.Errorf("decode JSON for Schema validation: %w", err)
	}
	if err := schema.schema.Validate(instance); err != nil {
		return fmt.Errorf("validate JSON Schema: %w", err)
	}
	return nil
}

// matchesRootType 判断候选是否可能属于目标 Schema 的根类型。
// 嵌套数组等合法 JSON 片段如果根类型不符，不应被误记为业务语义错误并阻断后续恢复。
func (schema *compiledSchema) matchesRootType(input string) bool {
	if schema == nil || len(schema.rootTypes) == 0 {
		return true
	}
	_, matches := schema.rootTypes[jsonRootType(input)]
	return matches
}

func schemaRootTypes(document any) map[string]struct{} {
	object, ok := document.(map[string]any)
	if !ok {
		return nil
	}
	types := make(map[string]struct{})
	switch value := object["type"].(type) {
	case string:
		types[value] = struct{}{}
	case []any:
		for _, item := range value {
			if typeName, ok := item.(string); ok {
				types[typeName] = struct{}{}
			}
		}
	}
	return types
}

func jsonRootType(input string) string {
	switch firstNonSpace(input) {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}
