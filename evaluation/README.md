# 离线评测

这套评测使用带标准答案的 JSONL 数据集量化 Guard 的确定性能力，不访问 LLM，也不产生模型费用。

## 指标口径

- `accuracy`：行为和标准答案完全一致的样本比例，包括正确接受与正确拒绝。
- `direct_correct`：原始输出不经过恢复即可正确返回的样本数。
- `recovered_correct`：通过提取、标准化或本地修复正确返回的样本数。
- `recovery_success_rate`：需要恢复的可接受样本中，最终正确恢复的比例。
- `recovery_uplift`：恢复路径相对只接受直接结果，为总样本增加的正确结果百分点。
- `false_acceptance_rate`：本应拒绝却被 Guard 接受的比例，这是安全性优先指标。
- `false_rejections`：本应接受却被 Guard 拒绝的数量。
- `wrong_values`：虽然接受，但最终 JSON 与标准答案不同的数量。
- `p50/p95 latency`：包含 Schema 编译、候选提取、本地修复和校验的端到端耗时。

## 数据集规则

每行是一个独立 JSON 对象，主要字段如下：

````json
{
  "id": "markdown_object",
  "category": "extraction",
  "input": "```json\n{\"status\":\"ok\"}\n```",
  "schema": "{\"type\":\"object\"}",
  "expected_accept": true,
  "expected_json": "{\"status\":\"ok\"}",
  "expected_paths": ["extracted"]
}
````

拒绝样本使用：

```json
{
  "id": "missing_value",
  "category": "lossy_rejection",
  "input": "{\"name\":}",
  "schema": "{\"type\":\"object\"}",
  "expected_accept": false,
  "expected_error": "lossy_repair_rejected"
}
```

新增修复能力时必须同时增加正向样本和可能误修复的反向样本，避免只提高恢复率而扩大误接收范围。

## 当前已知缺口

`ambiguous_multiple_objects` 被标记为应当拒绝，但当前实现会选择第一个合法候选，因此基线不会是 100%。这个样本用于量化后续歧义检测带来的真实改进。

当前内置数据集基线：

| 指标 | 结果 |
|---|---:|
| 样本数 | 21 |
| 总体准确率 | 95.24% |
| 恢复成功率 | 100.00% |
| 恢复增益 | 42.86 个百分点 |
| 误接收率 | 14.29% |
| 误拒绝数 | 0 |
| 错误值数 | 0 |

延迟与运行环境相关，不写入固定基线，应在相同机器和 Go 版本下比较。

## 使用边界

内置数据集是用于建立工程基线的人工样本，能够发现功能回归，但不能代表公司的真实模型分布。接入实际服务后，应持续加入脱敏失败样本，并分别统计不同模型、输出契约和业务 Schema 的结果。在线 LLM 修复还需要单独评测调用成功率、最终验收率、P95 延迟和单次成本。
