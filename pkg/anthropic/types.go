// Package anthropic provides types and functions for interacting with the Anthropic Messages API.
// This package enables the Azure OpenAI proxy to handle Claude models through Azure AI Foundry's
// native Anthropic API integration.
//
// The Anthropic Messages API uses a different format than OpenAI's chat completions:
// - System messages are passed as a top-level "system" parameter (not in messages array)
// - Streaming uses SSE with events: message_start, content_block_start, content_block_delta, etc.
// - Authentication uses "x-api-key" header and "anthropic-version" header
// - Endpoint path is /anthropic/v1/messages (not /openai/deployments/...)
//
// References:
// - Official API docs: https://docs.anthropic.com/en/api/messages
// - Azure AI Foundry integration: https://learn.microsoft.com/azure/ai-studio/
package anthropic

// MessagesRequest represents a request to the Anthropic Messages API.
// This is the format expected by Azure AI Foundry's /anthropic/v1/messages endpoint.
type MessagesRequest struct {
	Model         string    `json:"model"`
	Messages      []Message `json:"messages"`
	MaxTokens     int       `json:"max_tokens"`
	System        string    `json:"system,omitempty"`
	Temperature   float64   `json:"temperature,omitempty"`
	TopP          float64   `json:"top_p,omitempty"`
	TopK          int       `json:"top_k,omitempty"`
	Stream        bool      `json:"stream,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
}

type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// Non-streaming response
type MessagesResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"` // "message"
	Role         string        `json:"role"` // "assistant"
	Content      []ContentItem `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"`
	StopSequence string        `json:"stop_sequence,omitempty"`
	Usage        Usage         `json:"usage"`
}

type ContentItem struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Streaming events
type StreamEvent struct {
	Type string `json:"type"`
}

type MessageStartEvent struct {
	Type    string           `json:"type"` // "message_start"
	Message MessagesResponse `json:"message"`
}

type ContentBlockStartEvent struct {
	Type         string      `json:"type"` // "content_block_start"
	Index        int         `json:"index"`
	ContentBlock ContentItem `json:"content_block"`
}

type ContentBlockDeltaEvent struct {
	Type  string       `json:"type"` // "content_block_delta"
	Index int          `json:"index"`
	Delta ContentDelta `json:"delta"`
}

type ContentDelta struct {
	Type string `json:"type"` // "text_delta"
	Text string `json:"text"`
}

type ContentBlockStopEvent struct {
	Type  string `json:"type"` // "content_block_stop"
	Index int    `json:"index"`
}

type MessageDeltaEvent struct {
	Type  string       `json:"type"` // "message_delta"
	Delta MessageDelta `json:"delta"`
	Usage UsageDelta   `json:"usage"`
}

type MessageDelta struct {
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

type UsageDelta struct {
	OutputTokens int `json:"output_tokens"`
}

type MessageStopEvent struct {
	Type string `json:"type"` // "message_stop"
}

type PingEvent struct {
	Type string `json:"type"` // "ping"
}
