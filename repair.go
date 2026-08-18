package jsonguard

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	jsonrepair "github.com/kaptinlin/jsonrepair"
)

// localJSONRepair 在大模型兜底前提供确定性语法恢复。
// 默认拒绝补值、删除占位内容和根结构重组，防止通用修复库静默改变业务事实。
func localJSONRepair(input string) (string, error) {
	if reason := detectLossyRepairRisk(input); reason != "" {
		return "", fmt.Errorf("%w: %s", ErrLossyRepair, reason)
	}

	repaired, err := jsonrepair.JSONRepair(input)
	if err != nil {
		return "", fmt.Errorf("JSON 修复失败: %w", err)
	}
	if rootChanged(input, repaired) {
		return "", fmt.Errorf("%w: JSON 根类型发生变化", ErrLossyRepair)
	}
	return repaired, nil
}

// detectLossyRepairRisk 在修复前识别第三方库会主动补值或删除内容的输入。
// 扫描时保护单双引号字符串，避免业务文本中的 undefined、... 等字样触发误判。
func detectLossyRepairRisk(input string) string {
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	for index := 0; index < len(input); index++ {
		current := input[index]
		if escaped {
			escaped = false
			continue
		}
		if (inSingleQuote || inDoubleQuote) && current == '\\' {
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
		if inSingleQuote || inDoubleQuote {
			continue
		}

		switch {
		case current == ':' && missingValueAfter(input, index+1):
			return "对象字段缺少原始值"
		case strings.HasPrefix(input[index:], "..."):
			return "省略号内容会被删除"
		case hasStandaloneToken(input, index, "undefined"):
			return "undefined 会被替换为 null"
		case current == '+':
			return "拼接表达式会被合并为新值"
		}
	}
	return ""
}

func missingValueAfter(input string, start int) bool {
	index := skipWhitespaceAndComments(input, start)
	return index >= len(input) || input[index] == ',' || input[index] == '}'
}

func skipWhitespaceAndComments(input string, start int) int {
	for index := start; index < len(input); {
		current, size := utf8.DecodeRuneInString(input[index:])
		if unicode.IsSpace(current) {
			index += size
			continue
		}
		if strings.HasPrefix(input[index:], "//") {
			if newline := strings.IndexByte(input[index+2:], '\n'); newline >= 0 {
				index += newline + 3
				continue
			}
			return len(input)
		}
		if strings.HasPrefix(input[index:], "/*") {
			if end := strings.Index(input[index+2:], "*/"); end >= 0 {
				index += end + 4
				continue
			}
			return len(input)
		}
		return index
	}
	return len(input)
}

func hasStandaloneToken(input string, start int, token string) bool {
	if !strings.HasPrefix(input[start:], token) {
		return false
	}
	end := start + len(token)
	leftBoundary := start == 0 || !isIdentifierByte(input[start-1])
	rightBoundary := end == len(input) || !isIdentifierByte(input[end])
	return leftBoundary && rightBoundary
}

func isIdentifierByte(current byte) bool {
	return current == '_' || current >= 'a' && current <= 'z' || current >= 'A' && current <= 'Z' || current >= '0' && current <= '9'
}

func rootChanged(input, repaired string) bool {
	originalRoot := firstNonSpace(input)
	repairedRoot := firstNonSpace(repaired)
	return originalRoot != 0 && repairedRoot != 0 && originalRoot != repairedRoot
}

func firstNonSpace(input string) rune {
	for _, current := range input {
		if !unicode.IsSpace(current) {
			return current
		}
	}
	return 0
}
