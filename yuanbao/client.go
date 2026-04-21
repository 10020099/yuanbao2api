package yuanbao

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Config holds the Yuanbao API configuration
type Config struct {
	BaseURL      string
	ChatEndpoint string
	Headers      http.Header
	Cookies      string
}

// Client provides methods to communicate with the Yuanbao API
type Client struct {
	Config *Config
}

// NewClient creates a new Yuanbao API client
func NewClient() *Client {
	cookie := os.Getenv("YUANBAO_COOKIE")

	headers := http.Header{}
	headers.Set("x-device-id", "")
	headers.Set("x-language", "zh-CN")
	headers.Set("x-requested-with", "XMLHttpRequest")
	headers.Set("content-type", "text/plain;charset=UTF-8")
	headers.Set("x-platform", "win")
	headers.Set("x-source", "web")
	headers.Set("x-webversion", "2.63.0")
	headers.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	headers.Set("origin", "https://yuanbao.tencent.com")
	headers.Set("referer", "https://yuanbao.tencent.com/chat")

	return &Client{
		Config: &Config{
			BaseURL:      "https://yuanbao.tencent.com",
			ChatEndpoint: "/api/chat",
			Headers:      headers,
			Cookies:      cookie,
		},
	}
}

// YuanbaoRequest represents the request structure for Yuanbao API
type YuanbaoRequest struct {
	Model             string                 `json:"model"`
	Prompt            string                 `json:"prompt"`
	Plugin            string                 `json:"plugin,omitempty"`
	DisplayPrompt     string                 `json:"displayPrompt,omitempty"`
	DisplayPromptType int                    `json:"displayPromptType,omitempty"`
	AgentID           string                 `json:"agentId,omitempty"`
	ProjectID         string                 `json:"projectId,omitempty"`
	IsTemporary       bool                   `json:"isTemporary,omitempty"`
	ChatModelID       string                 `json:"chatModelId,omitempty"`
	SupportFunctions  []string               `json:"supportFunctions,omitempty"`
	DocOpenID         string                 `json:"docOpenid,omitempty"`
	Options           map[string]interface{} `json:"options,omitempty"`
	Multimedia        []interface{}          `json:"multimedia,omitempty"`
	SupportHint       int                    `json:"supportHint,omitempty"`
	ChatModelExtInfo  string                 `json:"chatModelExtInfo,omitempty"`
	ApplicationIDList []string               `json:"applicationIdList,omitempty"`
	Version           string                 `json:"version,omitempty"`
	ExtReportParams   interface{}            `json:"extReportParams,omitempty"`
	IsAtomInput       bool                   `json:"isAtomInput,omitempty"`
	OffsetOfHour      int                    `json:"offsetOfHour,omitempty"`
	OffsetOfMinute    int                    `json:"offsetOfMinute,omitempty"`
}

// YuanbaoResponseChunk represents a chunk from Yuanbao streaming response
type YuanbaoResponseChunk struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Msg     string `json:"msg,omitempty"`
}

// SendRequestWithID sends a request to the Yuanbao API with explicit agentID and conversationID
func (c *Client) SendRequestWithID(request YuanbaoRequest, agentID string, conversationID string) (*http.Response, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	yuanbaoURL := fmt.Sprintf("%s%s/%s", c.Config.BaseURL, c.Config.ChatEndpoint, conversationID)

	req, err := http.NewRequest("POST", yuanbaoURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for key, values := range c.Config.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Set cookie and other required headers
	if c.Config.Cookies != "" {
		req.Header.Set("cookie", c.Config.Cookies)
	} else {
		req.Header.Set("cookie", os.Getenv("YUANBAO_COOKIE"))
	}
	req.Header.Set("x-agentid", fmt.Sprintf("%s/%s", agentID, conversationID))
	req.Header.Set("x-timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	req.Header.Set("referer", fmt.Sprintf("https://yuanbao.tencent.com/chat/%s/%s", agentID, conversationID))

	log.Printf("=== 发送给元宝的请求 ===")
	log.Printf("URL: %s", yuanbaoURL)
	log.Printf("Model: %s", request.Model)
	log.Printf("ChatModelID: %s", request.ChatModelID)
	log.Printf("Prompt length: %d chars", len(request.Prompt))
	log.Printf("========================")

	client := &http.Client{Timeout: 120 * time.Second}
	return client.Do(req)
}

// ParseResponse reads the full response body
func (c *Client) ParseResponse(resp *http.Response) (string, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("yuanbao API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// ParseStreamLine parses a single line from Yuanbao streaming response
func ParseStreamLine(line string) (*YuanbaoResponseChunk, error) {
	trimmed := bytes.TrimSpace([]byte(line))
	if len(trimmed) == 0 {
		return nil, nil
	}

	if !bytes.HasPrefix(trimmed, []byte("data:")) {
		return nil, nil
	}

	var payload []byte
	if bytes.HasPrefix(trimmed, []byte("data: ")) {
		payload = trimmed[6:]
	} else {
		payload = trimmed[5:]
	}

	if string(payload) == "[DONE]" {
		return nil, nil
	}

	var chunk YuanbaoResponseChunk
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return nil, err
	}

	return &chunk, nil
}

// GetEnv helper to get environment variable
func GetEnv(key string) string {
	return os.Getenv(key)
}
