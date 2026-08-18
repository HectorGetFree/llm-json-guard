// Package jsonguard 将不可信的大模型输出转换为可严格反序列化的 Go 值。
// 默认执行 JSON 提取和安全本地修复；本地失败时可通过 LLMRepairFunc 进行一次外部修复。
package jsonguard
