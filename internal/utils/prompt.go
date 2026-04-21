package utils

import (
	"fmt"
	"strings"

	"yuanbao2api/internal/models"
	"yuanbao2api/toolcall"
)

// BuildToolSystemPrompt generates the tool system prompt from tools definitions
func BuildToolSystemPrompt(tools []models.Tool) string {
	if len(tools) == 0 {
		return ""
	}

	var toolDescriptions strings.Builder
	for _, tool := range tools {
		params := "{}"
		if tool.Function.Parameters != nil {
			switch v := tool.Function.Parameters.(type) {
			case string:
				params = v
			default:
				params = fmt.Sprintf("%v", v)
			}
		}
		toolDescriptions.WriteString(fmt.Sprintf("### %s\n%s\n参数:\n```json\n%s\n```\n\n",
			tool.Function.Name,
			tool.Function.Description,
			params,
		))
	}

	return strings.Join([]string{
		"",
		"# 可用工具",
		"你可以调用以下工具来完成任务。",
		"",
		"**重要：当你需要调用工具时，必须严格按照以下格式输出：**",
		"",
		"<|tool_calls_begin|>",
		`{"name": "函数名", "arguments": {"参数名": "参数值"}}`,
		"<|tool_calls_end|>",
		"",
		"注意事项：",
		"1. 工具调用必须用上述标记包裹，不要遗漏标记",
		"2. 标记内只能包含 JSON 格式的工具调用，不要添加其他文字",
		"3. 你可以同时调用多个工具，每个工具调用使用单独的标记对",
		"4. 如果不需要调用工具，直接回复用户即可，不要输出标记",
		"",
		"可用工具列表：",
		toolDescriptions.String(),
		"",
	}, "\n")
}

// ConvertMessagesToYuanbaoPrompt converts OpenAI/Anthropic messages to Yuanbao prompt format
func ConvertMessagesToYuanbaoPrompt(messages []models.Message, tools []models.Tool) (string, string) {
	var prompt strings.Builder
	toolSystemPrompt := BuildToolSystemPrompt(tools)
	systemInjected := false

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			content := fmt.Sprintf("%v", msg.Content)
			if toolSystemPrompt != "" && !strings.Contains(content, "[系统提示:") {
				prompt.WriteString(fmt.Sprintf("[系统提示: %s%s]\n\n", content, toolSystemPrompt))
			} else {
				prompt.WriteString(fmt.Sprintf("[系统提示: %s]\n\n", content))
			}
			systemInjected = true
		case "user":
			content := fmt.Sprintf("%v", msg.Content)
			prompt.WriteString(fmt.Sprintf("用户: %s\n", content))
		case "assistant":
			content := fmt.Sprintf("%v", msg.Content)
			if msg.ToolCalls != nil && len(msg.ToolCalls) > 0 {
				var calls []string
				for _, tc := range msg.ToolCalls {
					calls = append(calls, fmt.Sprintf("调用工具 %s，参数: %s", tc.Function.Name, tc.Function.Arguments))
				}
				prompt.WriteString(fmt.Sprintf("助手: 我需要调用工具来完成任务。\n%s\n", strings.Join(calls, "\n")))
			} else {
				prompt.WriteString(fmt.Sprintf("助手: %s\n", content))
			}
		case "tool":
			toolName := msg.Name
			if toolName == "" {
				toolName = "unknown"
			}
			content := fmt.Sprintf("%v", msg.Content)
			truncatedContent := toolcall.TruncateToolResult(content)
			prompt.WriteString(fmt.Sprintf("工具 %s 的执行结果:\n%s\n\n", toolName, truncatedContent))
		}
	}

	prompt.WriteString("\n请作为助手继续回复：")

	if toolSystemPrompt != "" && !systemInjected {
		prompt.WriteString("\n\n" + toolSystemPrompt)
	}

	return prompt.String(), toolSystemPrompt
}

// TruncatePrompt truncates the prompt if it exceeds MAX_PROMPT_LENGTH
func TruncatePrompt(prompt string, messages []models.Message, toolSystemPrompt string) string {
	if len(prompt) <= toolcall.MAX_PROMPT_LENGTH || len(messages) <= 1 {
		return prompt
	}

	// First pass: compress all tool results
	for i := range messages {
		if messages[i].Role == "tool" && messages[i].Content != nil {
			content := fmt.Sprintf("%v", messages[i].Content)
			compressed := toolcall.TruncateToolResult(content, 1000)
			messages[i].Content = compressed
		}
	}

	// Rebuild prompt with compressed tool results
	newPrompt, _ := ConvertMessagesToYuanbaoPrompt(messages, nil)
	if toolSystemPrompt != "" && !strings.Contains(newPrompt, toolSystemPrompt) {
		newPrompt += "\n\n" + toolSystemPrompt
	}

	if len(newPrompt) <= toolcall.MAX_PROMPT_LENGTH {
		return newPrompt
	}

	// Hard truncate if still too long
	if len(newPrompt) > toolcall.MAX_PROMPT_LENGTH*15/10 {
		systemPart := strings.Index(newPrompt, "]\n\n")
		if systemPart != -1 {
			systemEnd := systemPart + 4
			remaining := toolcall.MAX_PROMPT_LENGTH*15/10 - systemEnd
			if remaining > 0 {
				newPrompt = newPrompt[:systemEnd] + newPrompt[len(newPrompt)-remaining:]
				newPrompt += "\n[...历史消息已截断...]"
			}
		} else {
			newPrompt = newPrompt[:toolcall.MAX_PROMPT_LENGTH*15/10] + "\n[...已截断...]"
		}
	}

	return newPrompt
}
