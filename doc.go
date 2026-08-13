// Package jsonguard 将不可信的模型文本转换为经过统一验收的 JSON 业务对象。
//
// 默认链路依次尝试严格解析、候选提取和保守本地修复；调用方可按需提供
// JSON Schema、业务 Validator 与外部 RepairFunc。无论结果来自哪条路径，
// 都必须重新经过相同的解码和校验边界。
package jsonguard
