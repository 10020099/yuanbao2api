package models

// Message represents a chat message
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// Tool represents a function tool definition
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function definition
type Function struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  interface{}     `json:"parameters,omitempty"`
}

// ToolCall represents a tool call from the model
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
	Index    int          `json:"index,omitempty"`
}

// FunctionCall represents the function call details
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIChatCompletionRequest represents the OpenAI API request format
type OpenAIChatCompletionRequest struct {
	Model            string      `json:"model"`
	Messages         []Message   `json:"messages"`
	Stream           bool        `json:"stream,omitempty"`
	Temperature      float32     `json:"temperature,omitempty"`
	TopP             float32     `json:"top_p,omitempty"`
	MaxTokens        int         `json:"max_tokens,omitempty"`
	PresencePenalty  float32     `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32     `json:"frequency_penalty,omitempty"`
	Tools            []Tool      `json:"tools,omitempty"`
	ToolChoice       interface{} `json:"tool_choice,omitempty"`
	DeepThinking     bool        `json:"deep_thinking,omitempty"`
	InternetSearch   bool        `json:"internet_search,omitempty"`
}

// OpenAIChatCompletionResponse represents the OpenAI API response format
type OpenAIChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a single choice in the response
type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message,omitempty"`
	Delta        *Delta          `json:"delta,omitempty"`
	FinishReason string          `json:"finish_reason"`
}

// ResponseMessage represents the message in the response
type ResponseMessage struct {
	Role             string     `json:"role"`
	Content          interface{} `json:"content"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
}

// Delta represents a streaming delta
type Delta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AnthropicMessageRequest represents the Anthropic Messages API request format
type AnthropicMessageRequest struct {
	Model       string      `json:"model"`
	Messages    []Message   `json:"messages"`
	MaxTokens   int         `json:"max_tokens"`
	Stream      bool        `json:"stream,omitempty"`
	System      interface{} `json:"system,omitempty"`
	Tools       []Tool      `json:"tools,omitempty"`
	ToolChoice  interface{} `json:"tool_choice,omitempty"`
	Temperature float32     `json:"temperature,omitempty"`
	TopP        float32     `json:"top_p,omitempty"`
	Thinking    interface{} `json:"thinking,omitempty"`
	DeepThinking bool       `json:"deep_thinking,omitempty"`
	InternetSearch bool     `json:"internet_search,omitempty"`
}

// AnthropicMessageResponse represents the Anthropic Messages API response format
type AnthropicMessageResponse struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                 `json:"model"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence interface{}            `json:"stop_sequence"`
	Usage        AnthropicUsage         `json:"usage"`
}

// AnthropicContentBlock represents a content block in Anthropic response
type AnthropicContentBlock struct {
	Type         string      `json:"type"`
	Text         string      `json:"text,omitempty"`
	Thinking     string      `json:"thinking,omitempty"`
	ToolUseID    string      `json:"id,omitempty"`
	Name         string      `json:"name,omitempty"`
	Input        interface{} `json:"input,omitempty"`
	ToolUseIDRef string      `json:"tool_use_id,omitempty"`
	Content      interface{} `json:"content,omitempty"`
}

// AnthropicUsage represents usage statistics for Anthropic API
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamEvent represents a streaming event from Anthropic API
type AnthropicStreamEvent struct {
	Type         string                  `json:"type"`
	Message      *AnthropicMessageResponse `json:"message,omitempty"`
	Index        int                     `json:"index,omitempty"`
	ContentBlock *AnthropicContentBlock  `json:"content_block,omitempty"`
	Delta        *AnthropicContentDelta  `json:"delta,omitempty"`
}

// AnthropicContentDelta represents a delta in Anthropic streaming
type AnthropicContentDelta struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	Thinking     string `json:"thinking,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence interface{} `json:"stop_sequence,omitempty"`
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID          string      `json:"id"`
	Object      string      `json:"object"`
	Created     int64       `json:"created"`
	OwnedBy     string      `json:"owned_by"`
	Permission  []interface{} `json:"permission"`
	Root        string      `json:"root"`
	Parent      interface{} `json:"parent"`
	Description string      `json:"description"`
}

// ModelsResponse represents the response for the models endpoint
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
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
