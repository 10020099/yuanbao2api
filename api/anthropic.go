package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"yuanbao2api/internal/models"
	"yuanbao2api/internal/utils"
	"yuanbao2api/session"
	"yuanbao2api/toolcall"
	"yuanbao2api/yuanbao"
)

// HandleAnthropicMessages handles Anthropic Messages API requests
func HandleAnthropicMessages(c *gin.Context) {
	var req models.AnthropicMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": map[string]string{"type": "invalid_request_error", "message": err.Error()},
		})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": map[string]string{"type": "invalid_request_error", "message": "messages is required and must be a non-empty array"},
		})
		return
	}

	cfg := GetServerConfig()
	model := req.Model
	if model == "" {
		model = cfg.DefaultModel
	}

	useDeepThinking := req.Thinking != nil || req.DeepThinking
	useInternetSearch := req.InternetSearch

	// Convert messages
	rawPrompt, toolSystemPrompt, _ := anthropicMessagesToPrompt(req.Messages, req.Tools)

	// Handle system prompt
	prompt := ""
	sysPart := buildAnthropicSystem(req.System, toolSystemPrompt)
	if sysPart != "" {
		prompt = fmt.Sprintf("[绯荤粺鎻愮ず: %s]\n\n", sysPart)
	}
	prompt += rawPrompt

	// Prompt length limit
	if len(prompt) > toolcall.MAX_PROMPT_LENGTH {
		sysPrefix := ""
		sysEnd := strings.Index(prompt, "]\n\n")
		if sysEnd != -1 {
			sysPrefix = prompt[:sysEnd+4]
		}
		maxRawLen := toolcall.MAX_PROMPT_LENGTH - len(sysPrefix)
		if maxRawLen > 500 {
			prompt = sysPrefix + rawPrompt[len(rawPrompt)-maxRawLen:] + "\n[...鍘嗗彶娑堟伅宸叉埅鏂?..]"
		} else {
			prompt = prompt[:toolcall.MAX_PROMPT_LENGTH] + "\n[...宸叉埅鏂?..]"
		}
	}

	modelConfig := GetModelConfig(model)
	agentID := getAgentID()
	conversationID := session.GenerateConversationID()

	yuanbaoReq := buildYuanbaoRequest(prompt, modelConfig, useDeepThinking, useInternetSearch, agentID)

	log.Printf("=== Anthropic API -> 鍏冨疂璇锋眰 ===")
	log.Printf("Model: %s -> chatModelId: %s", model, modelConfig.ChatModelID)
	log.Printf("Tools: %d", len(req.Tools))
	log.Printf("Stream: %v", req.Stream)

	client := yuanbao.NewClient()
	resp, err := client.SendRequestWithID(yuanbaoReq, agentID, conversationID)
	if err != nil {
		log.Printf("Error sending request to Yuanbao: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": map[string]string{"type": "api_error", "message": err.Error()},
		})
		return
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		errMsg := fmt.Sprintf("Yuanbao API error: %d", resp.StatusCode)
		log.Printf("%s, body: %s", errMsg, string(body[:min(500, len(body))]))
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": map[string]string{"type": "api_error", "message": errMsg},
		})
		return
	}

	msgID := fmt.Sprintf("msg_%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:24])

	if req.Stream {
		handleAnthropicStream(c, resp, model, req.Tools, msgID)
	} else {
		handleAnthropicNonStream(c, resp, model, req.Tools, msgID)
	}
}

// anthropicMessagesToPrompt converts Anthropic messages to prompt format
func anthropicMessagesToPrompt(messages []models.Message, tools []models.Tool) (string, string, bool) {
	var prompt strings.Builder
	toolSystemPrompt := utils.BuildToolSystemPrompt(tools)
	systemInjected := false

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			text := contentToString(msg.Content)
			// Check for tool_result blocks
			if blocks, ok := msg.Content.([]interface{}); ok {
				var parts []string
				for _, block := range blocks {
					if blockMap, ok := block.(map[string]interface{}); ok {
						blockType, _ := blockMap["type"].(string)
						switch blockType {
						case "text":
							if t, ok := blockMap["text"].(string); ok {
								parts = append(parts, t)
							}
						case "tool_result":
							toolUseID, _ := blockMap["tool_use_id"].(string)
							rawContent := ""
							if c, ok := blockMap["content"]; ok {
								switch v := c.(type) {
								case string:
									rawContent = v
								case []interface{}:
									var cs []string
									for _, item := range v {
										if m, ok := item.(map[string]interface{}); ok {
											if t, ok := m["text"].(string); ok {
												cs = append(cs, t)
											}
										}
									}
									rawContent = strings.Join(cs, "\n")
								default:
									rawContent = fmt.Sprintf("%v", v)
								}
							}
							truncated := toolcall.TruncateToolResult(rawContent)
							parts = append(parts, fmt.Sprintf("宸ュ叿 %s 鐨勬墽琛岀粨鏋?\n%s", toolUseID, truncated))
						}
					}
				}
				if len(parts) > 0 {
					text = strings.Join(parts, "\n")
				}
			}
			prompt.WriteString(fmt.Sprintf("鐢ㄦ埛: %s\n", text))

		case "assistant":
			if str, ok := msg.Content.(string); ok {
				prompt.WriteString(fmt.Sprintf("鍔╂墜: %s\n", str))
			} else if blocks, ok := msg.Content.([]interface{}); ok {
				var parts []string
				for _, block := range blocks {
					if blockMap, ok := block.(map[string]interface{}); ok {
						blockType, _ := blockMap["type"].(string)
						switch blockType {
						case "text":
							if t, ok := blockMap["text"].(string); ok {
								parts = append(parts, t)
							}
						case "tool_use":
							name, _ := blockMap["name"].(string)
							input, _ := blockMap["input"]
							inputJSON, _ := json.Marshal(input)
							parts = append(parts, fmt.Sprintf("璋冪敤宸ュ叿 %s锛屽弬鏁? %s", name, string(inputJSON)))
						}
					}
				}
				prompt.WriteString(fmt.Sprintf("鍔╂墜: %s\n", strings.Join(parts, "\n")))
			} else if msg.ToolCalls != nil && len(msg.ToolCalls) > 0 {
				var calls []string
				for _, tc := range msg.ToolCalls {
					calls = append(calls, fmt.Sprintf("璋冪敤宸ュ叿 %s锛屽弬鏁? %s", tc.Function.Name, tc.Function.Arguments))
				}
				prompt.WriteString(fmt.Sprintf("鍔╂墜: 鎴戦渶瑕佽皟鐢ㄥ伐鍏锋潵瀹屾垚浠诲姟銆俓n%s\n", strings.Join(calls, "\n")))
			}
		}
	}

	prompt.WriteString("\n璇蜂綔涓哄姪鎵嬬户缁洖澶嶏細")

	return prompt.String(), toolSystemPrompt, systemInjected
}

// contentToString converts message content to string
func contentToString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// buildAnthropicSystem builds the system prompt for Anthropic
func buildAnthropicSystem(system interface{}, toolSystemPrompt string) string {
	if system == nil && toolSystemPrompt == "" {
		return ""
	}
	var parts []string
	if system != nil {
		switch v := system.(type) {
		case string:
			parts = append(parts, v)
		case []interface{}:
			var texts []string
			for _, block := range v {
				if m, ok := block.(map[string]interface{}); ok {
					if t, ok := m["text"].(string); ok {
						texts = append(texts, t)
					}
				}
			}
			parts = append(parts, strings.Join(texts, "\n"))
		}
	}
	if toolSystemPrompt != "" {
		parts = append(parts, strings.TrimSpace(toolSystemPrompt))
	}
	return strings.Join(parts, "\n\n")
}

// handleAnthropicStream handles streaming Anthropic response
func handleAnthropicStream(c *gin.Context, resp *http.Response, model string, tools []models.Tool, msgID string) {
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var fullText strings.Builder
	var thinkingText strings.Builder
	var textBuffer string
	inToolCall := false
	inNaturalToolCall := false
	textBlockStarted := false
	thinkingBlockStarted := false
	hasTools := len(tools) > 0

	// UTF-8 安全的字符串切片辅助函数
	substringRune := func(s string, start, end int) string {
		runes := []rune(s)
		if start < 0 { start = 0 }
		if end > len(runes) { end = len(runes) }
		if start >= end { return "" }
		return string(runes[start:end])
	}

	streamTimeout := time.NewTimer(120 * time.Second)
	defer streamTimeout.Stop()

	resetTimeout := func() {
		streamTimeout.Reset(120 * time.Second)
	}

	sendSSE := func(event string, data interface{}) {
		dataJSON, _ := json.Marshal(data)
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(dataJSON))
		safeFlush(c.Writer)
	}

	// Send message_start
	sendSSE("message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            msgID,
			"type":          "message",
			"role":          "assistant",
			"content":       []interface{}{},
			"model":         model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage":         map[string]int{"input_tokens": 0, "output_tokens": 0},
		},
	})

	// Send ping
	sendSSE("ping", map[string]interface{}{})

	flushTextAsContent := func() {
		if textBuffer != "" && !inToolCall && !inNaturalToolCall {
			if !textBlockStarted {
				blockIdx := 0
				if thinkingBlockStarted {
					blockIdx = 1
				}
				sendSSE("content_block_start", map[string]interface{}{
					"type":  "content_block_start",
					"index": blockIdx,
					"content_block": map[string]string{
						"type": "text",
						"text": "",
					},
				})
				textBlockStarted = true
			}
			blockIdx := 0
			if thinkingBlockStarted {
				blockIdx = 1
			}
			sendSSE("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": blockIdx,
				"delta": map[string]string{
					"type": "text_delta",
					"text": textBuffer,
				},
			})
			textBuffer = ""
		}
	}

	processLine := func(line string) {
		chunk, err := yuanbao.ParseStreamLine(line)
		if err != nil || chunk == nil {
			return
		}

		if chunk.Type == "think" && chunk.Content != "" {
			thinkingText.WriteString(chunk.Content)
			if !thinkingBlockStarted {
				sendSSE("content_block_start", map[string]interface{}{
					"type":  "content_block_start",
					"index": 0,
					"content_block": map[string]string{
						"type":     "thinking",
						"thinking": "",
					},
				})
				thinkingBlockStarted = true
			}
			sendSSE("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]string{
					"type":     "thinking_delta",
					"thinking": chunk.Content,
				},
			})
		}

		if chunk.Type == "text" && chunk.Msg != "" {
			fullText.WriteString(chunk.Msg)
			textBuffer += chunk.Msg

			if hasTools {
				startMatch := toolcall.DetectToolCallStartPublic(textBuffer, 0)
				if startMatch.Index != -1 && !inToolCall {
					if inNaturalToolCall {
						inNaturalToolCall = false
					}
					beforeTag := substringRune(textBuffer, 0, startMatch.Index)
				textBuffer = substringRune(textBuffer, startMatch.Index, len([]rune(textBuffer)))
					if beforeTag != "" {
						if !textBlockStarted {
							blockIdx := 0
							if thinkingBlockStarted {
								blockIdx = 1
							}
							sendSSE("content_block_start", map[string]interface{}{
								"type":  "content_block_start",
								"index": blockIdx,
								"content_block": map[string]string{
									"type": "text",
									"text": "",
								},
							})
							textBlockStarted = true
						}
						blockIdx := 0
						if thinkingBlockStarted {
							blockIdx = 1
						}
						sendSSE("content_block_delta", map[string]interface{}{
							"type":  "content_block_delta",
							"index": blockIdx,
							"delta": map[string]string{
								"type": "text_delta",
								"text": beforeTag,
							},
						})
					}
					inToolCall = true
					textBuffer = ""
				}

				if inToolCall {
					fullTextStr := fullText.String()
					endMatch := toolcall.DetectToolCallEndPublic(fullTextStr, 0)
					if endMatch.Index != -1 {
						inToolCall = false
						textBuffer = substringRune(fullTextStr, endMatch.Index+len(endMatch.Tag), len([]rune(fullTextStr)))
					}
				}

				if !inToolCall && !inNaturalToolCall {
					if strings.Contains(textBuffer, `"name"`) && strings.Contains(textBuffer, `"arguments"`) {
						natIdx := findNaturalToolStart(textBuffer)
						if natIdx != -1 {
							beforeNat := substringRune(textBuffer, 0, natIdx)
							textBuffer = substringRune(textBuffer, natIdx, len([]rune(textBuffer)))
				if beforeNat != "" {
								if !textBlockStarted {
									blockIdx := 0
									if thinkingBlockStarted {
										blockIdx = 1
									}
									sendSSE("content_block_start", map[string]interface{}{
										"type":  "content_block_start",
										"index": blockIdx,
										"content_block": map[string]string{
											"type": "text",
											"text": "",
										},
									})
									textBlockStarted = true
								}
								blockIdx := 0
								if thinkingBlockStarted {
									blockIdx = 1
								}
								sendSSE("content_block_delta", map[string]interface{}{
									"type":  "content_block_delta",
									"index": blockIdx,
									"delta": map[string]string{
										"type": "text_delta",
										"text": beforeNat,
									},
								})
							}
							inNaturalToolCall = true
						}
					}
				}

				if inNaturalToolCall {
					fullTextStr := fullText.String()
					fromNatStart := len(fullTextStr) - len(textBuffer)
					subText := substringRune(fullTextStr, fromNatStart, len([]rune(fullTextStr)))
						if balanced := toolcall.ExtractBalancedJSONPublic(subText, 0); balanced != "" {
							inNaturalToolCall = false
							textBuffer = substringRune(fullTextStr, fromNatStart+len(balanced), len([]rune(fullTextStr)))
					}
				}

				if !inToolCall && !inNaturalToolCall {
					tagLookback := toolcall.ToolCallStartLength()
					natLookback := toolcall.NaturalToolPrefixLookback(textBuffer)
					lookback := max(tagLookback, natLookback)
					// 使用 rune 切片确保 UTF-8 安全（按字符而非字节）
					runes := []rune(textBuffer)
					runeLen := len(runes) - lookback
					if runeLen > 0 {
						safeText := string(runes[:runeLen])
						textBuffer = string(runes[runeLen:])
						if !textBlockStarted {
							blockIdx := 0
							if thinkingBlockStarted {
								blockIdx = 1
							}
							sendSSE("content_block_start", map[string]interface{}{
								"type":  "content_block_start",
								"index": blockIdx,
								"content_block": map[string]string{
									"type": "text",
									"text": "",
								},
							})
							textBlockStarted = true
						}
						blockIdx := 0
						if thinkingBlockStarted {
							blockIdx = 1
						}
						sendSSE("content_block_delta", map[string]interface{}{
							"type":  "content_block_delta",
							"index": blockIdx,
							"delta": map[string]string{
								"type": "text_delta",
								"text": safeText,
							},
						})
					}
				}
			} else {
				flushTextAsContent()
			}
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	done := make(chan bool, 1)
	go func() {
		for scanner.Scan() {
			resetTimeout()
			line := scanner.Text()
			processLine(line)
		}
		done <- true
	}()

	select {
	case <-done:
	case <-streamTimeout.C:
		log.Printf("Stream timeout (Anthropic): no data for 120s, forcing end")
	}

	resp.Body.Close()

	log.Printf("[Anthropic Stream End] fullText length=%d, thinkingText length=%d", fullText.Len(), thinkingText.Len())

	fullTextStr := fullText.String()
	toolCalls := []toolcall.ToolCall{}
	if hasTools {
		toolCalls = toolcall.ParseToolCalls(fullTextStr)
	}
	hasToolCalls := len(toolCalls) > 0

	nextIndex := 0
	if thinkingBlockStarted {
		nextIndex++
	}
	if textBlockStarted {
		nextIndex++
	}

	// Close text block
	if textBlockStarted {
		blockIdx := 0
		if thinkingBlockStarted {
			blockIdx = 1
		}
		sendSSE("content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": blockIdx,
		})
		textBlockStarted = false
	}

	// Send tool_use blocks
	if hasToolCalls {
		formattedCalls := toolcall.FormatToolCalls(toolCalls, 0)
		for i, fc := range formattedCalls {
			blockIdx := nextIndex + i
			fn := fc["function"].(map[string]interface{})
			sendSSE("content_block_start", map[string]interface{}{
				"type":  "content_block_start",
				"index": blockIdx,
				"content_block": map[string]interface{}{
					"type":  "tool_use",
					"id":    fc["id"],
					"name":  fn["name"],
					"input": map[string]interface{}{},
				},
			})

			sendSSE("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": blockIdx,
				"delta": map[string]interface{}{
					"type":          "input_json_delta",
					"partial_json":  fn["arguments"],
				},
			})

			sendSSE("content_block_stop", map[string]interface{}{
				"type":  "content_block_stop",
				"index": blockIdx,
			})
		}
	}

	// Close thinking block
	if thinkingBlockStarted {
		sendSSE("content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": 0,
		})
	}

	// If no block was started, send a text block
	if !textBlockStarted && !thinkingBlockStarted && !hasToolCalls {
		sendSSE("content_block_start", map[string]interface{}{
			"type":  "content_block_start",
			"index": 0,
			"content_block": map[string]string{
				"type": "text",
				"text": "",
			},
		})
		if textBuffer != "" && !inNaturalToolCall {
			sendSSE("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]string{
					"type": "text_delta",
					"text": textBuffer,
				},
			})
		}
		sendSSE("content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": 0,
		})
	}

	// message_delta + message_stop
	stopReason := "end_turn"
	if hasToolCalls {
		stopReason = "tool_use"
	}
	sendSSE("message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]int{"output_tokens": 0},
	})

	sendSSE("message_stop", map[string]interface{}{
		"type": "message_stop",
	})
}

// handleAnthropicNonStream handles non-streaming Anthropic response
func handleAnthropicNonStream(c *gin.Context, resp *http.Response, model string, tools []models.Tool, msgID string) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": map[string]string{"type": "api_error", "message": "Failed to read response"},
		})
		return
	}

	var fullText strings.Builder
	var thinkingText strings.Builder

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		chunk, err := yuanbao.ParseStreamLine(line)
		if err != nil || chunk == nil {
			continue
		}
		if chunk.Type == "think" && chunk.Content != "" {
			thinkingText.WriteString(chunk.Content)
		}
		if chunk.Type == "text" && chunk.Msg != "" {
			fullText.WriteString(chunk.Msg)
		}
	}

	fullTextStr := fullText.String()
	thinkingStr := thinkingText.String()

	hasTools := len(tools) > 0
	toolCalls := []toolcall.ToolCall{}
	if hasTools {
		toolCalls = toolcall.ParseToolCalls(fullTextStr)
	}
	hasToolCalls := len(toolCalls) > 0
	cleanText := fullTextStr
	if hasToolCalls {
		cleanText = toolcall.StripToolCalls(fullTextStr)
	}

	// Build content blocks
	content := []models.AnthropicContentBlock{}

	if thinkingStr != "" {
		content = append(content, models.AnthropicContentBlock{
			Type:     "thinking",
			Thinking: thinkingStr,
		})
	}

	if hasToolCalls {
		if cleanText != "" {
			content = append(content, models.AnthropicContentBlock{
				Type: "text",
				Text: cleanText,
			})
		}
		formattedCalls := toolcall.FormatToolCalls(toolCalls, 0)
		for _, fc := range formattedCalls {
			fn := fc["function"].(map[string]interface{})
			input := map[string]interface{}{}
			if args, ok := fn["arguments"].(string); ok {
				json.Unmarshal([]byte(args), &input)
			}
			content = append(content, models.AnthropicContentBlock{
				Type:  "tool_use",
				ToolUseID: fc["id"].(string),
				Name:  fn["name"].(string),
				Input: input,
			})
		}
	} else {
		content = append(content, models.AnthropicContentBlock{
			Type: "text",
			Text: fullTextStr,
		})
	}

	stopReason := "end_turn"
	if hasToolCalls {
		stopReason = "tool_use"
	}

	response := models.AnthropicMessageResponse{
		ID:           msgID,
		Type:         "message",
		Role:         "assistant",
		Content:      content,
		Model:        model,
		StopReason:   stopReason,
		StopSequence: nil,
		Usage:        models.AnthropicUsage{InputTokens: 0, OutputTokens: 0},
	}

	c.JSON(http.StatusOK, response)
}
