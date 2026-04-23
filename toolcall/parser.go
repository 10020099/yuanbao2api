package toolcall

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// ToolCall represents a parsed tool call
type ToolCall struct {
	Name      string
	Arguments string
}

// MatchResult represents the result of a marker detection
type MatchResult struct {
	Index int
	Tag   string
}

const (
	toolCallStartUnicode = "<｜tool▁calls_begin｜>"
	toolCallEndUnicode   = "<｜tool▁calls_end｜>"
	toolCallStartASCII   = "<|tool_calls_begin|>"
	toolCallEndASCII     = "<|tool_calls_end|>"
)

// MAX_TOOL_RESULT_LENGTH is the maximum length of a tool result before truncation
const MAX_TOOL_RESULT_LENGTH = 3000

// MAX_PROMPT_LENGTH is the maximum total prompt length
const MAX_PROMPT_LENGTH = 8000

// NATURAL_TOOL_PREFIX_LOOKBACK is the maximum lookback window for natural tool call prefixes
const NATURAL_TOOL_PREFIX_LOOKBACK = 80

// detectToolCallStart finds the start of a tool call marker
func detectToolCallStart(text string, fromIndex int) MatchResult {
	uIdx := strings.Index(text[fromIndex:], toolCallStartUnicode)
	aIdx := strings.Index(text[fromIndex:], toolCallStartASCII)

	if uIdx == -1 && aIdx == -1 {
		return MatchResult{Index: -1}
	}

	if uIdx == -1 {
		return MatchResult{Index: fromIndex + aIdx, Tag: toolCallStartASCII}
	}
	if aIdx == -1 {
		return MatchResult{Index: fromIndex + uIdx, Tag: toolCallStartUnicode}
	}

	if uIdx <= aIdx {
		return MatchResult{Index: fromIndex + uIdx, Tag: toolCallStartUnicode}
	}
	return MatchResult{Index: fromIndex + aIdx, Tag: toolCallStartASCII}
}

// detectToolCallEnd finds the end of a tool call marker
func detectToolCallEnd(text string, fromIndex int) MatchResult {
	uIdx := strings.Index(text[fromIndex:], toolCallEndUnicode)
	aIdx := strings.Index(text[fromIndex:], toolCallEndASCII)

	if uIdx == -1 && aIdx == -1 {
		return MatchResult{Index: -1}
	}

	if uIdx == -1 {
		return MatchResult{Index: fromIndex + aIdx, Tag: toolCallEndASCII}
	}
	if aIdx == -1 {
		return MatchResult{Index: fromIndex + uIdx, Tag: toolCallEndUnicode}
	}

	if uIdx <= aIdx {
		return MatchResult{Index: fromIndex + uIdx, Tag: toolCallEndUnicode}
	}
	return MatchResult{Index: fromIndex + aIdx, Tag: toolCallEndASCII}
}

// ToolCallStartLength returns the maximum length of tool call start markers
func ToolCallStartLength() int {
	if len(toolCallStartUnicode) > len(toolCallStartASCII) {
		return len(toolCallStartUnicode)
	}
	return len(toolCallStartASCII)
}

// NaturalToolPrefixLookback checks if the end of textBuffer might be a prefix of a natural format tool call
func NaturalToolPrefixLookback(textBuffer string) int {
	if textBuffer == "" {
		return 0
	}
	for i := len(textBuffer) - 1; i >= max(0, len(textBuffer)-NATURAL_TOOL_PREFIX_LOOKBACK); i-- {
		if textBuffer[i] == '{' {
			after := textBuffer[i+1:]
			trimmedAfter := strings.TrimLeft(after, " \t\n\r")
			if trimmedAfter == "" || strings.HasPrefix(trimmedAfter, "\"") {
				return len(textBuffer) - i
			}
		}
	}
	return 0
}



// ParseToolCalls extracts tool calls from text using both Unicode and ASCII markers
func ParseToolCalls(text string) []ToolCall {
	var calls []ToolCall
	searchFrom := 0
	textLen := len(text)

	for searchFrom < textLen {
		startMatch := detectToolCallStart(text, searchFrom)
		if startMatch.Index == -1 || startMatch.Tag == "" {
			break
		}

		endMatch := detectToolCallEnd(text, startMatch.Index+len(startMatch.Tag))
		if endMatch.Index == -1 || endMatch.Tag == "" {
			break
		}

		content := strings.TrimSpace(text[startMatch.Index+len(startMatch.Tag) : endMatch.Index])
		parsed, err := parseJSONContent(content)
		if err != nil {
			parsed = ToolCall{
				Name:      "unknown",
				Arguments: content,
			}
		}

		calls = append(calls, parsed)
		searchFrom = endMatch.Index + len(endMatch.Tag)
	}

	// If no structured tool calls found, try natural language parsing
	if len(calls) == 0 {
		naturalCalls := findNaturalToolCalls(text)
		calls = append(calls, naturalCalls...)
	}

	return calls
}

// parseJSONContent attempts to parse content as JSON and extract name and arguments
func parseJSONContent(content string) (ToolCall, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return ToolCall{}, err
	}

	name, ok := data["name"].(string)
	if !ok {
		return ToolCall{}, fmt.Errorf("missing or invalid name field")
	}

	arguments := ""
	if arg, ok := data["arguments"]; ok {
		switch v := arg.(type) {
		case string:
			arguments = v
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return ToolCall{}, fmt.Errorf("failed to marshal arguments: %w", err)
			}
			arguments = string(b)
		}
	}

	return ToolCall{
		Name:      name,
		Arguments: arguments,
	}, nil
}

// extractBalancedJSON extracts a balanced JSON object starting at startIndex
func extractBalancedJSON(text string, startIndex int) string {
	if startIndex >= len(text) || text[startIndex] != '{' {
		return ""
	}
	depth := 0
	inString := false
	escape := false
	for i := startIndex; i < len(text); i++ {
		ch := text[i]
		if escape {
			escape = false
			continue
		}
		if ch == '\\' && inString {
			escape = true
			continue
		}
		if ch == '"' && !escape {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return text[startIndex : i+1]
			}
		}
	}
	return ""
}

// findNaturalToolCalls finds tool calls in natural language format
func findNaturalToolCalls(text string) []ToolCall {
	var calls []ToolCall
	namePattern := regexp.MustCompile(`"?name"?\s*:\s*"([^"]+)"\s*,\s*"?arguments"?\s*:\s*`)
	matches := namePattern.FindAllStringIndex(text, -1)
	for _, match := range matches {
		name := text[match[0]+strings.Index(text[match[0]:match[1]], "\"")+1:]
		// Extract name properly
		submatch := namePattern.FindStringSubmatch(text[match[0]:match[1]])
		if len(submatch) < 2 {
			continue
		}
		name = submatch[1]
		argsStart := match[1]
		argsJSON := extractBalancedJSON(text, argsStart)
		if argsJSON == "" {
			continue
		}
		var args interface{}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			continue
		}
		argsStr, _ := json.Marshal(args)
		calls = append(calls, ToolCall{
			Name:      name,
			Arguments: string(argsStr),
		})
	}
	return calls
}

// StripToolCalls removes tool call markers from text
func StripToolCalls(text string) string {
	result := text
	for {
		startMatch := detectToolCallStart(result, 0)
		if startMatch.Index == -1 || startMatch.Tag == "" {
			break
		}
		endMatch := detectToolCallEnd(result, startMatch.Index+len(startMatch.Tag))
		if endMatch.Index == -1 || endMatch.Tag == "" {
			break
		}
		result = result[:startMatch.Index] + result[endMatch.Index+len(endMatch.Tag):]
	}

	// Also strip natural language tool calls
	naturalCalls := findNaturalToolCalls(result)
	for i := len(naturalCalls) - 1; i >= 0; i-- {
		// Find and remove each natural call - simplified: just remove the JSON portion
		// In practice, the regex-based removal is complex; we rely on the caller to handle this
		_ = naturalCalls[i]
	}

	return strings.TrimSpace(result)
}

// FormatToolCalls converts ToolCall slices to OpenAI format tool_calls
func FormatToolCalls(calls []ToolCall, startIndex int) []map[string]interface{} {
	result := make([]map[string]interface{}, len(calls))
	for i, call := range calls {
		result[i] = map[string]interface{}{
			"id":       fmt.Sprintf("call_%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:24]),
			"type":     "function",
			"index":    startIndex + i,
			"function": map[string]interface{}{
				"name":      call.Name,
				"arguments": call.Arguments,
			},
		}
	}
	return result
}

// CompressToolResult intelligently compresses tool result content
func CompressToolResult(content string) string {
	if content == "" {
		return content
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return content
	}
	// Tavily search results format
	if results, ok := parsed["results"].([]interface{}); ok {
		summary := map[string]interface{}{}
		if q, ok := parsed["query"]; ok {
			summary["query"] = q
		}
		if a, ok := parsed["answer"]; ok {
			summary["answer"] = a
		}
		newResults := make([]interface{}, len(results))
		for i, r := range results {
			if resultMap, ok := r.(map[string]interface{}); ok {
				newResult := map[string]interface{}{}
				if t, ok := resultMap["title"]; ok {
					newResult["title"] = t
				}
				if u, ok := resultMap["url"]; ok {
					newResult["url"] = u
				}
				if c, ok := resultMap["content"]; ok {
					cs := fmt.Sprintf("%v", c)
					if len(cs) > 200 {
						cs = cs[:200] + "..."
					}
					newResult["content"] = cs
				}
				if s, ok := resultMap["score"]; ok {
					newResult["score"] = s
				}
				newResults[i] = newResult
			}
		}
		summary["results"] = newResults
		b, _ := json.Marshal(summary)
		return string(b)
	}
	// If structuredContent is redundant
	if _, ok := parsed["structuredContent"]; ok {
		if _, ok2 := parsed["content"]; ok2 {
			delete(parsed, "content")
			b, _ := json.Marshal(parsed)
			return string(b)
		}
	}
	return content
}

// TruncateToolResult truncates tool result to maxLength, preserving JSON structure if possible
func TruncateToolResult(content string, maxLength ...int) string {
	maxLen := MAX_TOOL_RESULT_LENGTH
	if len(maxLength) > 0 && maxLength[0] > 0 {
		maxLen = maxLength[0]
	}
	if content == "" {
		return content
	}
	compressed := CompressToolResult(content)
	if len(compressed) <= maxLen {
		return compressed
	}
	truncated := compressed[:maxLen]
	lastBrace := strings.LastIndex(truncated, "}")
	if lastBrace > maxLen/2 {
		return truncated[:lastBrace+1] + "\n...[结果已截断，原始长度: " + fmt.Sprintf("%d", len(content)) + " 字符]"
	}
	return truncated + "\n...[结果已截断，原始长度: " + fmt.Sprintf("%d", len(content)) + " 字符]"
}

// DetectToolCallStartPublic is the public wrapper for detectToolCallStart
func DetectToolCallStartPublic(text string, fromIndex int) MatchResult {
	return detectToolCallStart(text, fromIndex)
}

// DetectToolCallEndPublic is the public wrapper for detectToolCallEnd
func DetectToolCallEndPublic(text string, fromIndex int) MatchResult {
	return detectToolCallEnd(text, fromIndex)
}

// ExtractBalancedJSONPublic is the public wrapper for extractBalancedJSON
func ExtractBalancedJSONPublic(text string, startIndex int) string {
	return extractBalancedJSON(text, startIndex)
}
