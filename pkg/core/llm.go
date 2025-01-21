package core

import (
	"context"
)

// Role represents the role of a message participant
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleFunction  Role = "function"
	RoleTool      Role = "tool"
)

// LLMProvider represents different LLM providers
type LLMProvider string

const (
	OpenAI          LLMProvider = "OPEN_AI"
	Azure           LLMProvider = "AZURE"
	AzureAD         LLMProvider = "AZURE_AD"
	CloudflareAzure LLMProvider = "CLOUDFLARE_AZURE"
	Gemini          LLMProvider = "GEMINI"
	Claude          LLMProvider = "CLAUDE"
	Ollama          LLMProvider = "OLLAMA"
	DeepSeek        LLMProvider = "DEEPSEEK"
)

// Message represents a single message in a chat conversation
type Message struct {
	Role      Role       `json:"role"`
	Content   string     `json:"content"`
	Name      string     `json:"name,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatCompletionRequest represents a generic request for chat completion
type ChatCompletionRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	Temperature      float32   `json:"temperature,omitempty"`
	TopP             float32   `json:"top_p,omitempty"`
	N                int       `json:"n,omitempty"`
	Stop             []string  `json:"stop,omitempty"`
	MaxTokens        int       `json:"max_tokens,omitempty"`
	PresencePenalty  float32   `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32   `json:"frequency_penalty,omitempty"`
	User             string    `json:"user,omitempty"`
	Tools            []Tool    `json:"tools,omitempty"`
	Stream           bool      `json:"stream,omitempty"`
}

// ChatCompletionResponse represents a generic response from chat completion
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLM defines the interface that all LLM providers must implement
type LLM interface {
	CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error)
	CreateChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (ChatCompletionStream, error)
}

// ChatCompletionStream represents a streaming response
type ChatCompletionStream interface {
	Recv() (ChatCompletionResponse, error)
	Close() error
}

// StreamDelta represents a delta in a streaming response
type StreamDelta struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// StreamChoice represents a choice in a streaming response
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

// ToolCall represents a tool/function call from the LLM
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
