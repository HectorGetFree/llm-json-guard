package jsonguard

import (
	"regexp"
	"strings"
)

// fencedCodeBlockPattern 识别模型常用的 Markdown 代码块边界。
// 即使模型没有直接返回纯 JSON，代码块通常仍能表达明确的载荷范围。
var fencedCodeBlockPattern = regexp.MustCompile("(?s)```(?:json|JSON)?\\s*(.*?)\\s*```")

// extractJSONCandidates 按模型意图强弱生成候选。
// Markdown 代码块优先于普通文本中的括号；内部固定限制避免异常输出产生无界候选。
func extractJSONCandidates(input string) []string {
	seen := make(map[string]struct{})
	var candidates []string
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if len(candidate) > maxCandidateBytes {
			return
		}
		if _, exists := seen[candidate]; exists {
			return
		}
		if len(candidates) >= maxCandidates {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	// 1. 优先检查代码块，因为模型显式标记的载荷边界比普通文本中的括号更可靠。
	for _, match := range fencedCodeBlockPattern.FindAllStringSubmatch(input, -1) {
		if len(match) < 2 {
			continue
		}
		block := strings.TrimSpace(match[1])
		appendCandidate(block)
		if len(candidates) >= maxCandidates {
			return candidates
		}
		for _, candidate := range extractBalancedCandidates(block, maxCandidates-len(candidates)) {
			appendCandidate(candidate)
		}
		appendCandidate(extractIncompleteCandidate(block))
		if len(candidates) >= maxCandidates {
			return candidates
		}
	}

	// 2. 扫描完整响应，兼容在自然语言中直接夹带 JSON 的模型；去重时保留先前优先级。
	for _, candidate := range extractBalancedCandidates(input, maxCandidates-len(candidates)) {
		appendCandidate(candidate)
	}
	if len(candidates) >= maxCandidates {
		return candidates
	}
	appendCandidate(extractIncompleteCandidate(input))
	return candidates
}

// extractIncompleteCandidate 只将未闭合对象或数组保留为修复输入。
// 在修复并通过校验前，该内容不会被视为有效输出。
func extractIncompleteCandidate(input string) string {
	for start := 0; start < len(input); start++ {
		if input[start] != '{' && input[start] != '[' {
			continue
		}
		if _, complete := findBalancedEnd(input, start); !complete {
			return strings.TrimSpace(input[start:])
		}
	}
	return ""
}

func extractBalancedCandidates(input string, maxCandidates int) []string {
	if maxCandidates <= 0 {
		return nil
	}
	var candidates []string
	for start := 0; start < len(input); start++ {
		if input[start] != '{' && input[start] != '[' {
			continue
		}
		end, complete := findBalancedEnd(input, start)
		if complete {
			candidates = append(candidates, input[start:end+1])
			if len(candidates) >= maxCandidates {
				break
			}
		}
	}
	return candidates
}

func findBalancedEnd(input string, start int) (int, bool) {
	stack := make([]byte, 0, 8)
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false
	for index := start; index < len(input); index++ {
		current := input[index]
		if escaped {
			escaped = false
			continue
		}
		if (inDoubleQuote || inSingleQuote) && current == '\\' {
			escaped = true
			continue
		}
		if !inSingleQuote && current == '"' {
			inDoubleQuote = !inDoubleQuote
			continue
		}
		if !inDoubleQuote && current == '\'' {
			inSingleQuote = !inSingleQuote
			continue
		}
		if inDoubleQuote || inSingleQuote {
			continue
		}
		switch current {
		case '{', '[':
			stack = append(stack, current)
		case '}':
			if len(stack) == 0 || stack[len(stack)-1] != '{' {
				return 0, false
			}
			stack = stack[:len(stack)-1]
		case ']':
			if len(stack) == 0 || stack[len(stack)-1] != '[' {
				return 0, false
			}
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			return index, true
		}
	}
	return 0, false
}

// normalizeStructurePunctuation 只转换引号外的全角 JSON 结构标点。
// 引号内文本保持原样，避免修改用户可见内容或业务标识。
func normalizeStructurePunctuation(input string) string {
	var output strings.Builder
	output.Grow(len(input))
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false
	for _, current := range input {
		if escaped {
			output.WriteRune(current)
			escaped = false
			continue
		}
		if (inDoubleQuote || inSingleQuote) && current == '\\' {
			output.WriteRune(current)
			escaped = true
			continue
		}
		if !inSingleQuote && current == '"' {
			inDoubleQuote = !inDoubleQuote
			output.WriteRune(current)
			continue
		}
		if !inDoubleQuote && current == '\'' {
			inSingleQuote = !inSingleQuote
			output.WriteRune(current)
			continue
		}
		if inDoubleQuote || inSingleQuote {
			output.WriteRune(current)
			continue
		}
		switch current {
		case '｛':
			output.WriteRune('{')
		case '｝':
			output.WriteRune('}')
		case '［':
			output.WriteRune('[')
		case '］':
			output.WriteRune(']')
		case '：':
			output.WriteRune(':')
		case '，':
			output.WriteRune(',')
		default:
			output.WriteRune(current)
		}
	}
	return output.String()
}
