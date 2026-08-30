package ai

import "time"

// Message is a chat message (plain Content and/or multimodal Parts).
type Message struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	Parts      []ContentPart `json:"-"` // marshaled as content array when set
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"` // role=tool
	Name       string        `json:"name,omitempty"`         // optional tool/function name
}

// ChatRequest is a chat completion request.
type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Tools          []Tool          `json:"tools,omitempty"`
	ToolChoice     *ToolChoice     `json:"tool_choice,omitempty"`
}

// ChatResponse is a chat completion response.
type ChatResponse struct {
	ID           string    `json:"id"`
	Model        string    `json:"model"`
	Message      Message   `json:"message"`
	FinishReason string    `json:"finish_reason,omitempty"` // stop | tool_calls | length | …
	Usage        Usage     `json:"usage"`
	Created      time.Time `json:"created_at"`
}

// Usage tracks token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbedRequest is an embedding request.
type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbedResponse is an embedding response.
type EmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
	Usage      Usage       `json:"usage"`
}

// Defaults are applied by Manager when a request omits values.
type Defaults struct {
	Model             string
	Temperature       *float64
	MaxTokens         int
	Timeout           time.Duration
	Retry             RetryPolicy
	FallbackOnTimeout bool // when true, profile chain continues after DeadlineExceeded
}
