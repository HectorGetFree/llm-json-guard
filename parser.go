package modeljson

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	jsonrepair "github.com/kaptinlin/jsonrepair"
)

var (
	ErrNoJSONCandidate = errors.New("no complete JSON candidate found")
	ErrInvalidOutput   = errors.New("model output could not be parsed and validated")
)

type ParsePath string

const (
	ParsePathRaw      ParsePath = "raw"
	ParsePathLocalFix ParsePath = "local_repair"
	ParsePathLLMFix   ParsePath = "llm_repair"
)

type ParseResult[T any] struct {
	Value      T
	JSON       string
	Path       ParsePath
	RawOutput  string
	UsedRepair bool
	UsedLLMFix bool
}

type JSONRepairer func(input string) (string, error)
type LLMRepairer func(ctx context.Context, rawOutput string) (string, error)
type Validator[T any] func(value T) error

type ParseOptions[T any] struct {
	LocalRepair            JSONRepairer
	LLMRepair              LLMRepairer
	Validate               Validator[T]
	AllowLLMSemanticRepair bool
}

// LocalJSONRepair adapts the real jsonrepair library to the parser pipeline.
func LocalJSONRepair(input string) (string, error) {
	return jsonrepair.JSONRepair(input)
}

// parseResult parses model output through the normal, local-repair, and one-shot LLM paths.
func parseResult[T any](ctx context.Context, rawOutput string, options ParseOptions[T]) (ParseResult[T], error) {
	rawOutput = strings.TrimSpace(rawOutput)
	if rawOutput == "" {
		return ParseResult[T]{}, fmt.Errorf("%w: empty output", ErrInvalidOutput)
	}

	if value, err := decodeAndValidate[T]([]byte(rawOutput), options.Validate); err == nil {
		return ParseResult[T]{Value: value, JSON: rawOutput, Path: ParsePathRaw, RawOutput: rawOutput}, nil
	}

	normalizedOutput := normalizeStructurePunctuation(rawOutput)
	candidates := extractJSONCandidates(normalizedOutput)
	sawSemanticError := false
	var lastSemanticError error

	for _, candidate := range candidates {
		value, err := decodeAndValidate[T]([]byte(candidate), options.Validate)
		if err == nil {
			return ParseResult[T]{Value: value, JSON: candidate, Path: ParsePathRaw, RawOutput: rawOutput}, nil
		}
		if json.Valid([]byte(candidate)) {
			sawSemanticError = true
			if lastSemanticError == nil {
				lastSemanticError = err
			}
		}
	}

	if options.LocalRepair != nil {
		for _, candidate := range candidates {
			if json.Valid([]byte(candidate)) {
				continue
			}

			repaired, err := options.LocalRepair(candidate)
			if err != nil {
				continue
			}

			value, err := decodeAndValidate[T]([]byte(repaired), options.Validate)
			if err != nil {
				continue
			}

			return ParseResult[T]{
				Value: value, JSON: repaired, Path: ParsePathLocalFix,
				RawOutput: rawOutput, UsedRepair: true,
			}, nil
		}
	}

	if options.LLMRepair != nil && (!sawSemanticError || options.AllowLLMSemanticRepair) {
		fixed, err := options.LLMRepair(ctx, rawOutput)
		if err != nil {
			return ParseResult[T]{RawOutput: rawOutput}, fmt.Errorf("%w: LLM repair request failed: %v", ErrInvalidOutput, err)
		}

		fixed = strings.TrimSpace(fixed)
		value, err := decodeAndValidate[T]([]byte(fixed), options.Validate)
		if err != nil {
			return ParseResult[T]{RawOutput: rawOutput}, fmt.Errorf("%w: LLM-repaired JSON failed strict validation: %v", ErrInvalidOutput, err)
		}

		return ParseResult[T]{
			Value: value, JSON: fixed, Path: ParsePathLLMFix,
			RawOutput: rawOutput, UsedLLMFix: true,
		}, nil
	}

	if len(candidates) == 0 {
		return ParseResult[T]{RawOutput: rawOutput}, fmt.Errorf("%w: %w", ErrInvalidOutput, ErrNoJSONCandidate)
	}
	if sawSemanticError {
		if lastSemanticError != nil {
			return ParseResult[T]{RawOutput: rawOutput}, fmt.Errorf(
				"%w: JSON is syntactically valid but violates the target schema or business rules: %v",
				ErrInvalidOutput,
				lastSemanticError,
			)
		}
		return ParseResult[T]{RawOutput: rawOutput}, fmt.Errorf("%w: JSON is syntactically valid but violates the target schema or business rules", ErrInvalidOutput)
	}
	return ParseResult[T]{RawOutput: rawOutput}, fmt.Errorf("%w: local parsing and repair failed", ErrInvalidOutput)
}

// Parse is the exported entry point for callers outside this package.
func Parse[T any](ctx context.Context, rawOutput string, options ParseOptions[T]) (ParseResult[T], error) {
	return parseResult(ctx, rawOutput, options)
}

func decodeAndValidate[T any](input []byte, validate Validator[T]) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode JSON: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return value, errors.New("unexpected additional JSON value")
		}
		return value, fmt.Errorf("unexpected trailing content: %w", err)
	}

	if validate != nil {
		if err := validate(value); err != nil {
			return value, fmt.Errorf("business validation: %w", err)
		}
	}
	return value, nil
}

var fencedCodeBlockPattern = regexp.MustCompile("(?s)```(?:json|JSON)?\\s*(.*?)\\s*```")

func extractJSONCandidates(input string) []string {
	seen := make(map[string]struct{})
	var candidates []string
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if _, exists := seen[candidate]; exists {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	for _, match := range fencedCodeBlockPattern.FindAllStringSubmatch(input, -1) {
		if len(match) < 2 {
			continue
		}
		block := strings.TrimSpace(match[1])
		appendCandidate(block)
		for _, candidate := range extractBalancedCandidates(block) {
			appendCandidate(candidate)
		}
		appendCandidate(extractIncompleteCandidate(block))
	}
	for _, candidate := range extractBalancedCandidates(input) {
		appendCandidate(candidate)
	}
	appendCandidate(extractIncompleteCandidate(input))
	return candidates
}

// extractIncompleteCandidate returns the first object or array that starts in
// input but does not close. It is only useful as a fallback input for a JSON
// repairer; it is still rejected unless repair and strict validation succeed.
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

func extractBalancedCandidates(input string) []string {
	var candidates []string
	for start := 0; start < len(input); start++ {
		if input[start] != '{' && input[start] != '[' {
			continue
		}
		end, ok := findBalancedEnd(input, start)
		if ok {
			candidates = append(candidates, input[start:end+1])
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
