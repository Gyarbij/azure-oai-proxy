package anthropic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
)

// ConvertOpenAIToAnthropic converts an OpenAI chat completion request to Anthropic Messages API format.
// This function handles:
// - Extracting system messages and setting them as the "system" parameter
// - Converting message roles and content
// - Mapping OpenAI parameters (temperature, max_tokens, stop) to Anthropic equivalents
// - Preserving streaming settings
//
// Parameters are mapped as follows:
// - messages[role=system] → system parameter (extracted from messages array)
// - messages[role=user/assistant] → messages array
// - max_tokens → max_tokens (required by Anthropic)
// - temperature → temperature (0.0-1.0)
// - stop → stop_sequences
// - stream → stream
func ConvertOpenAIToAnthropic(openAIRequest map[string]interface{}) (*MessagesRequest, error) {
	req := &MessagesRequest{
		MaxTokens: 1000, // Default
		Messages:  make([]Message, 0),
	}

	if model, ok := openAIRequest["model"].(string); ok {
		req.Model = model
	}

	if rawMessages, exists := openAIRequest["messages"]; exists {
		req.Messages = append(req.Messages, normalizeMessages(rawMessages, &req.System)...)
	}

	if len(req.Messages) == 0 {
		if input, exists := openAIRequest["input"]; exists {
			converted := normalizeMessages(input, &req.System)
			if len(converted) == 0 {
				if text := extractMessageContent(input); text != "" {
					converted = append(converted, Message{Role: "user", Content: text})
				}
			}
			req.Messages = append(req.Messages, converted...)
		}
	}

	if len(req.Messages) == 0 {
		if prompt, ok := openAIRequest["prompt"].(string); ok && strings.TrimSpace(prompt) != "" {
			req.Messages = append(req.Messages, Message{Role: "user", Content: prompt})
		}
	}

	if temp, ok := openAIRequest["temperature"].(float64); ok {
		req.Temperature = temp
	}

	if topP, ok := openAIRequest["top_p"].(float64); ok {
		req.TopP = topP
	}

	if topK, ok := openAIRequest["top_k"].(float64); ok {
		req.TopK = int(topK)
	}

	if maxTokens, ok := openAIRequest["max_tokens"].(float64); ok {
		req.MaxTokens = int(maxTokens)
	}

	if stream, ok := openAIRequest["stream"].(bool); ok {
		req.Stream = stream
	}

	if stop, exists := openAIRequest["stop"]; exists {
		req.StopSequences = append(req.StopSequences, normalizeStopSequences(stop)...)
	}

	if len(req.Messages) == 0 {
		req.Messages = make([]Message, 0)
	}

	return req, nil
}

func normalizeMessages(source interface{}, system *string) []Message {
	messages := make([]Message, 0)

	switch typed := source.(type) {
	case []interface{}:
		for _, entry := range typed {
			messages = append(messages, normalizeMessages(entry, system)...)
		}
	case map[string]interface{}:
		role, _ := typed["role"].(string)
		content := extractMessageContent(typed["content"])

		if strings.EqualFold(role, "system") {
			appendSystem(system, content)
			return messages
		}

		if role == "" && content == "" {
			return messages
		}

		if role == "" {
			role = "user"
		}

		if trimmed := strings.TrimSpace(content); trimmed != "" {
			messages = append(messages, Message{Role: role, Content: trimmed})
		}
	case string:
		if trimmed := strings.TrimSpace(typed); trimmed != "" {
			messages = append(messages, Message{Role: "user", Content: trimmed})
		}
	default:
		if text := extractMessageContent(typed); text != "" {
			messages = append(messages, Message{Role: "user", Content: text})
		}
	}

	return messages
}

func extractMessageContent(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		return value
	case []interface{}:
		var builder strings.Builder
		for _, part := range value {
			piece := strings.TrimSpace(extractMessageContent(part))
			if piece == "" {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(piece)
		}
		return builder.String()
	case map[string]interface{}:
		if text, ok := value["text"].(string); ok {
			return text
		}
		if text, ok := value["input_text"].(string); ok {
			return text
		}
		if nested, exists := value["content"]; exists {
			return extractMessageContent(nested)
		}
		if nested, ok := value["value"].(string); ok {
			return nested
		}
		if nested, ok := value["message"].(map[string]interface{}); ok {
			return extractMessageContent(nested)
		}
	case nil:
		return ""
	}

	if raw == nil {
		return ""
	}

	if marshaled, err := json.Marshal(raw); err == nil {
		return string(marshaled)
	}

	return fmt.Sprintf("%v", raw)
}

func appendSystem(system *string, fragment string) {
	trimmed := strings.TrimSpace(fragment)
	if trimmed == "" {
		return
	}
	if *system == "" {
		*system = trimmed
		return
	}
	*system = *system + "\n" + trimmed
}

func normalizeStopSequences(source interface{}) []string {
	stops := make([]string, 0)

	switch typed := source.(type) {
	case string:
		if trimmed := strings.TrimSpace(typed); trimmed != "" {
			stops = append(stops, trimmed)
		}
	case []interface{}:
		for _, entry := range typed {
			if text, ok := entry.(string); ok {
				if trimmed := strings.TrimSpace(text); trimmed != "" {
					stops = append(stops, trimmed)
				}
			}
		}
	}

	return stops
}

// ConvertAnthropicToOpenAI converts Anthropic Messages API response to OpenAI chat completion format
func ConvertAnthropicToOpenAI(anthropicResp *MessagesResponse) map[string]interface{} {
	// Extract text content
	var content string
	for _, item := range anthropicResp.Content {
		if item.Type == "text" {
			content += item.Text
		}
	}

	return map[string]interface{}{
		"id":      anthropicResp.ID,
		"object":  "chat.completion",
		"created": 0, // Anthropic doesn't provide timestamp
		"model":   anthropicResp.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": mapStopReason(anthropicResp.StopReason),
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     anthropicResp.Usage.InputTokens,
			"completion_tokens": anthropicResp.Usage.OutputTokens,
			"total_tokens":      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}
}

func mapStopReason(anthropicReason string) string {
	switch anthropicReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

// StreamConverter converts Anthropic SSE stream to OpenAI SSE stream
type StreamConverter struct {
	reader    io.Reader
	writer    io.Writer
	messageID string
	model     string
}

func NewStreamConverter(reader io.Reader, writer io.Writer, model string) *StreamConverter {
	return &StreamConverter{
		reader: reader,
		writer: writer,
		model:  model,
	}
}

func (c *StreamConverter) Convert() error {
	scanner := bufio.NewScanner(c.reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var eventType string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			eventType = ""
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			if data == "" || data == "{\"type\": \"ping\"}" {
				continue
			}

			switch eventType {
			case "message_start":
				if err := c.handleMessageStart(data); err != nil {
					log.Printf("Error handling message_start: %v", err)
				}
			case "content_block_delta":
				if err := c.handleContentDelta(data); err != nil {
					log.Printf("Error handling content_block_delta: %v", err)
				}
			case "message_stop":
				if err := c.handleMessageStop(); err != nil {
					log.Printf("Error handling message_stop: %v", err)
				}
				return nil
			case "content_block_start", "content_block_stop", "message_delta", "ping":
				// Skip these events
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %v", err)
	}

	return nil
}

func (c *StreamConverter) handleMessageStart(data string) error {
	var event MessageStartEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return err
	}

	c.messageID = event.Message.ID

	// Send OpenAI format chunk
	chunk := map[string]interface{}{
		"id":      c.messageID,
		"object":  "chat.completion.chunk",
		"created": 0,
		"model":   c.model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"role": "assistant",
				},
				"finish_reason": nil,
			},
		},
	}

	return c.writeChunk(chunk)
}

func (c *StreamConverter) handleContentDelta(data string) error {
	var event ContentBlockDeltaEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return err
	}

	// Send OpenAI format chunk with text delta
	chunk := map[string]interface{}{
		"id":      c.messageID,
		"object":  "chat.completion.chunk",
		"created": 0,
		"model":   c.model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"content": event.Delta.Text,
				},
				"finish_reason": nil,
			},
		},
	}

	return c.writeChunk(chunk)
}

func (c *StreamConverter) handleMessageStop() error {
	// Send final chunk with finish_reason
	chunk := map[string]interface{}{
		"id":      c.messageID,
		"object":  "chat.completion.chunk",
		"created": 0,
		"model":   c.model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{},
				"finish_reason": "stop",
			},
		},
	}

	if err := c.writeChunk(chunk); err != nil {
		return err
	}

	// Send [DONE] message
	_, err := c.writer.Write([]byte("data: [DONE]\n\n"))
	return err
}

func (c *StreamConverter) writeChunk(chunk map[string]interface{}) error {
	jsonData, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	// Write in SSE format
	if _, err := c.writer.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := c.writer.Write(jsonData); err != nil {
		return err
	}
	if _, err := c.writer.Write([]byte("\n\n")); err != nil {
		return err
	}

	// Flush if possible
	if flusher, ok := c.writer.(interface{ Flush() }); ok {
		flusher.Flush()
	}

	return nil
}

// ReadNonStreamingResponse reads and converts a non-streaming Anthropic response
func ReadNonStreamingResponse(body io.Reader) (map[string]interface{}, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var anthropicResp MessagesResponse
	if err := json.Unmarshal(data, &anthropicResp); err != nil {
		return nil, err
	}

	return ConvertAnthropicToOpenAI(&anthropicResp), nil
}

// CreateRequestBody creates an Anthropic Messages API request body from OpenAI format
func CreateRequestBody(openAIBody []byte) ([]byte, error) {
	var openAIReq map[string]interface{}
	if err := json.Unmarshal(openAIBody, &openAIReq); err != nil {
		return nil, err
	}

	anthropicReq, err := ConvertOpenAIToAnthropic(openAIReq)
	if err != nil {
		return nil, err
	}

	return json.Marshal(anthropicReq)
}

// IsClaudeModel checks if a model name is a Claude model
func IsClaudeModel(model string) bool {
	model = strings.ToLower(model)
	return strings.Contains(model, "claude")
}
