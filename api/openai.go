package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"yuanbao2api/internal/models"
	"yuanbao2api/internal/utils"
	"yuanbao2api/session"
	"yuanbao2api/toolcall"
	"yuanbao2api/yuanbao"
)

// getAgentID returns the Yuanbao agent ID from environment or default
func getAgentID() string {
	agentID := os.Getenv("YUANBAO_AGENT_ID")
	if agentID == "" {
		agentID = "naQivTmsDa"
	}
	return agentID
}

// HandleOpenAIChatCompletion processes OpenAI-compatible chat completion requests
func HandleOpenAIChatCompletion(c *gin.Context) {
	var req models.OpenAIChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Received OpenAI chat completion request: model=%s, stream=%v", req.Model, req.Stream)

	cfg := GetServerConfig()

	// Apply global defaults if not specified
	model := req.Model
	if model == "" {
		model = cfg.DefaultModel
	}
	useDeepThinking := req.DeepThinking
	if !req.DeepThinking {
		useDeepThinking = cfg.DeepThinking
	}
	useInternetSearch := req.InternetSearch
	if !req.InternetSearch {
		useInternetSearch = cfg.InternetSearch
	}

	// Build prompt
	prompt, toolSystemPrompt := utils.ConvertMessagesToYuanbaoPrompt(req.Messages, req.Tools)
	prompt = utils.TruncatePrompt(prompt, req.Messages, toolSystemPrompt)

	// Send request to Yuanbao API
	modelConfig := GetModelConfig(model)
	agentID := getAgentID()
	conversationID := session.GenerateConversationID()

	yuanbaoReq := buildYuanbaoRequest(prompt, modelConfig, useDeepThinking, useInternetSearch, agentID)

	client := yuanbao.NewClient()
	resp, err := client.SendRequestWithID(yuanbaoReq, agentID, conversationID)
	if err != nil {
		log.Printf("Error sending request to Yuanbao: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("Yuanbao API error: status=%d, body=%s", resp.StatusCode, string(body[:min(500, len(body))]))
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Yuanbao API error: %d", resp.StatusCode)})
		return
	}

	if req.Stream {
		handleOpenAIStream(c, resp, model, req.Tools)
	} else {
		handleOpenAINonStream(c, resp, model, req.Tools)
	}
}

// handleOpenAIStream handles streaming OpenAI response
func handleOpenAIStream(c *gin.Context, resp *http.Response, model string, tools []models.Tool) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var fullText strings.Builder
	var thinkingText strings.Builder
	var textBuffer string
	isFirstThinkChunk := true
	isFirstTextChunk := true
	inToolCall := false
	inNaturalToolCall := false
	hasTools := len(tools) > 0

	streamTimeout := time.NewTimer(120 * time.Second)
	defer streamTimeout.Stop()

	resetTimeout := func() {
		streamTimeout.Reset(120 * time.Second)
	}

	sendChunk := func(delta map[string]interface{}) {
		chunkID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
		chunk := map[string]interface{}{
			"id":      chunkID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         delta,
					"finish_reason": nil,
				},
			},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.(http.Flusher).Flush()
	}

	flushTextBuffer := func() {
		if textBuffer != "" && !inToolCall && !inNaturalToolCall {
			delta := map[string]interface{}{"content": textBuffer}
			if isFirstTextChunk && isFirstThinkChunk {
				delta["role"] = "assistant"
			}
			isFirstTextChunk = false
			sendChunk(delta)
		}
		textBuffer = ""
	}

	sendTextChunk := func(text string) {
		if text == "" {
			return
		}
		delta := map[string]interface{}{"content": text}
		if isFirstTextChunk && isFirstThinkChunk {
			delta["role"] = "assistant"
		}
		isFirstTextChunk = false
		sendChunk(delta)
	}

	processLine := func(line string) {
		chunk, err := yuanbao.ParseStreamLine(line)
		if err != nil || chunk == nil {
			return
		}

		if chunk.Type == "think" && chunk.Content != "" {
			thinkingText.WriteString(chunk.Content)
			delta := map[string]interface{}{"reasoning_content": chunk.Content}
			if isFirstThinkChunk {
				delta["role"] = "assistant"
				isFirstThinkChunk = false
			}
			sendChunk(delta)
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
					beforeTag := textBuffer[:startMatch.Index]
					textBuffer = textBuffer[startMatch.Index:]
					if beforeTag != "" {
						sendTextChunk(beforeTag)
					}
					inToolCall = true
					textBuffer = ""
				}

				if inToolCall {
					fullTextStr := fullText.String()
					endMatch := toolcall.DetectToolCallEndPublic(fullTextStr, 0)
					if endMatch.Index != -1 {
						inToolCall = false
						textBuffer = fullTextStr[endMatch.Index+len(endMatch.Tag):]
					}
				}

				if !inToolCall && !inNaturalToolCall {
					// Natural tool call detection
					if strings.Contains(textBuffer, `"name"`) && strings.Contains(textBuffer, `"arguments"`) {
						// Simple heuristic: check for natural pattern
						natIdx := findNaturalToolStart(textBuffer)
						if natIdx != -1 {
							beforeNat := textBuffer[:natIdx]
							textBuffer = textBuffer[natIdx:]
							if beforeNat != "" {
								sendTextChunk(beforeNat)
							}
							inNaturalToolCall = true
						}
					}
				}

				if inNaturalToolCall {
					fullTextStr := fullText.String()
					fromNatStart := len(fullTextStr) - len(textBuffer)
					subText := fullTextStr[fromNatStart:]
					if balanced := toolcall.ExtractBalancedJSONPublic(subText, 0); balanced != "" {
						inNaturalToolCall = false
						textBuffer = fullTextStr[fromNatStart+len(balanced):]
					}
				}

				if !inToolCall && !inNaturalToolCall {
					tagLookback := toolcall.ToolCallStartLength()
					natLookback := toolcall.NaturalToolPrefixLookback(textBuffer)
					lookback := max(tagLookback, natLookback)
					safeLen := len(textBuffer) - lookback
					if safeLen > 0 {
						safeText := textBuffer[:safeLen]
						textBuffer = textBuffer[safeLen:]
						sendTextChunk(safeText)
					}
				}
			} else {
				flushTextBuffer()
			}
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	done := make(chan bool)
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
		// Stream completed normally
	case <-streamTimeout.C:
		log.Printf("Stream timeout (OpenAI): no data for 120s, forcing end")
	}

	resp.Body.Close()

	// Parse tool calls from full text
	fullTextStr := fullText.String()
	toolCalls := toolcall.ParseToolCalls(fullTextStr)
	hasToolCalls := len(toolCalls) > 0

	if hasToolCalls {
		cleanText := toolcall.StripToolCalls(fullTextStr)
		formattedCalls := toolcall.FormatToolCalls(toolCalls, 0)
		// Send any remaining text before tool calls
		cleanText = strings.TrimSpace(cleanText)
		if cleanText != "" {
			delta := map[string]interface{}{"content": cleanText}
			if isFirstTextChunk && isFirstThinkChunk {
				delta["role"] = "assistant"
			}
			sendChunk(delta)
		}
		// Send tool calls
		for i, fc := range formattedCalls {
			tc := fc
			tc["index"] = i
			delta := map[string]interface{}{
				"tool_calls": []map[string]interface{}{tc},
			}
			sendChunk(delta)
		}
	} else if textBuffer != "" && !inNaturalToolCall {
		delta := map[string]interface{}{"content": textBuffer}
		if isFirstTextChunk && isFirstThinkChunk {
			delta["role"] = "assistant"
		}
		sendChunk(delta)
	}

	// Send finish
	finishReason := "stop"
	if hasToolCalls {
		finishReason = "tool_calls"
	}
	sendChunk(map[string]interface{}{
		"finish_reason": finishReason,
	})
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.(http.Flusher).Flush()
}

// handleOpenAINonStream handles non-streaming OpenAI response
func handleOpenAINonStream(c *gin.Context, resp *http.Response, model string, tools []models.Tool) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	var fullText strings.Builder
	var thinkingText strings.Builder
	hasTools := len(tools) > 0

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
	toolCalls := []toolcall.ToolCall{}
	if hasTools {
		toolCalls = toolcall.ParseToolCalls(fullTextStr)
	}
	hasToolCalls := len(toolCalls) > 0
	cleanText := fullTextStr
	if hasToolCalls {
		cleanText = toolcall.StripToolCalls(fullTextStr)
	}

	content := fullTextStr
	if hasToolCalls {
		if cleanText == "" {
			content = ""
		} else {
			content = cleanText
		}
	}

	// Build response message
	openaiMessage := models.ResponseMessage{
		Role: "assistant",
	}
	if hasToolCalls {
		openaiMessage.Content = nil
		if cleanText != "" {
			openaiMessage.Content = cleanText
		}
		formatted := toolcall.FormatToolCalls(toolCalls, 0)
		openaiToolCalls := make([]models.ToolCall, len(formatted))
		for i, fc := range formatted {
			fn := fc["function"].(map[string]interface{})
			openaiToolCalls[i] = models.ToolCall{
				ID:       fc["id"].(string),
				Type:     "function",
				Function: models.FunctionCall{
					Name:      fn["name"].(string),
					Arguments: fn["arguments"].(string),
				},
			}
		}
		openaiMessage.ToolCalls = openaiToolCalls
	} else {
		openaiMessage.Content = content
	}

	thinkingStr := thinkingText.String()
	if thinkingStr != "" {
		openaiMessage.ReasoningContent = thinkingStr
	}

	finishReason := "stop"
	if hasToolCalls {
		finishReason = "tool_calls"
	}

	response := models.OpenAIChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.Choice{
			{
				Index:        0,
				Message:      openaiMessage,
				FinishReason: finishReason,
			},
		},
		Usage: models.Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}

	c.JSON(http.StatusOK, response)
}

// findNaturalToolStart finds the start index of a natural tool call pattern in text
func findNaturalToolStart(text string) int {
	// Look for patterns like {"name": "...", "arguments": {
	// This is a simplified version
	nameIdx := strings.Index(text, `"name"`)
	if nameIdx == -1 {
		nameIdx = strings.Index(text, "name")
		if nameIdx == -1 {
			return -1
		}
	}
	// Check if there's an "arguments" after name
	argsIdx := strings.Index(text[nameIdx:], `"arguments"`)
	if argsIdx == -1 {
		argsIdx = strings.Index(text[nameIdx:], "arguments")
		if argsIdx == -1 {
			return -1
		}
	}
	// Find the opening brace before name
	braceIdx := strings.LastIndex(text[:nameIdx], "{")
	if braceIdx == -1 {
		return -1
	}
	return braceIdx
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
