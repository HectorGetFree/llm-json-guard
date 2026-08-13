package jsonguard

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	jsonrepair "github.com/kaptinlin/jsonrepair"
)

// LocalJSONRepair 在外部兜底前提供确定性语法恢复。
// 默认拒绝补值、删除占位内容和根结构重组，防止通用修复库静默改变业务事实。
func LocalJSONRepair(input string) (string, error) {
	if reason := detectLossyRepairRisk(input); reason != "" {
		return "", fmt.Errorf("%w: %s", ErrLossyRepair, reason)
	}

	repaired, err := jsonrepair.JSONRepair(input)
	if err != nil {
		return "", fmt.Errorf("repair JSON: %w", err)
	}
	if rootChanged(input, repaired) {
		return "", fmt.Errorf("%w: root JSON type changed", ErrLossyRepair)
	}
	return repaired, nil
}

// PermissiveLocalJSONRepair 暴露第三方库的完整恢复能力。
// 调用方只有在接受补 null、删除省略内容或重组根结构时才应显式启用。
func PermissiveLocalJSONRepair(input string) (string, error) {
	repaired, err := jsonrepair.JSONRepair(input)
	if err != nil {
		return "", fmt.Errorf("repair JSON: %w", err)
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
			return "object field has no source value"
		case strings.HasPrefix(input[index:], "..."):
			return "ellipsis would be deleted"
		case hasStandaloneToken(input, index, "undefined"):
			return "undefined would be replaced with null"
		case current == '+':
			return "concatenated values would be merged"
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
