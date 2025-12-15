package azure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

// StreamingResponseConverter handles the conversion of Responses API SSE to Chat Completions SSE
type StreamingResponseConverter struct {
	reader io.Reader
	writer io.Writer
	model  string
}

// NewStreamingResponseConverter creates a new streaming converter
func NewStreamingResponseConverter(reader io.Reader, writer io.Writer, model string) *StreamingResponseConverter {
	return &StreamingResponseConverter{
		reader: reader,
		writer: writer,
		model:  model,
	}
}

// Convert performs the streaming conversion
func (c *StreamingResponseConverter) Convert() error {
	scanner := bufio.NewScanner(c.reader)
	var eventType string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			switch eventType {
			case "response.output_text.delta":
				c.handleTextDelta(data)
			case "response.completed":
				log.Printf("Responses stream completed for model: %s", c.model)
				c.handleCompleted(data)
				return nil
			case "response.created", "response.in_progress", "response.output_item.added",
				"response.output_item.done", "response.content_part.added",
				"response.content_part.done", "response.output_text.done":
				// These events don't need to be converted for chat completion streaming
				continue
			}
		}

		// Empty line (event separator)
		if line == "" {
			eventType = ""
			continue
		}
	}

	return scanner.Err()
}

func (c *StreamingResponseConverter) handleTextDelta(data string) {
	var deltaEvent map[string]interface{}
	if err := json.Unmarshal([]byte(data), &deltaEvent); err != nil {
		log.Printf("Error parsing delta event: %v", err)
		return
	}

	delta, ok := deltaEvent["delta"].(string)
	if !ok {
		return
	}

	// Create chat completion chunk
	chunk := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   c.model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"content": delta,
				},
				"finish_reason": nil,
			},
		},
	}

	c.writeChunk(chunk)
}

func (c *StreamingResponseConverter) handleCompleted(data string) {
	// First send an empty delta to indicate the end of content
	chunk := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   c.model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{},
				"finish_reason": "stop",
			},
		},
	}

	c.writeChunk(chunk)

	// Then send the [DONE] marker
	c.writer.Write([]byte("data: [DONE]\n\n"))
	if flusher, ok := c.writer.(flushWriter); ok {
		flusher.Flush()
	}
}

func (c *StreamingResponseConverter) writeChunk(chunk map[string]interface{}) {
	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		log.Printf("Error marshaling chunk: %v", err)
		return
	}

	c.writer.Write([]byte("data: "))
	c.writer.Write(chunkJSON)
	c.writer.Write([]byte("\n\n"))

	if flusher, ok := c.writer.(flushWriter); ok {
		flusher.Flush()
	}
}

type flushWriter interface {
	io.Writer
	Flush()
}

// AnthropicStreamingConverter handles the conversion of Anthropic Messages API SSE to OpenAI Chat Completions SSE
type AnthropicStreamingConverter struct {
	reader io.Reader
	writer io.Writer
	model  string
}

// NewAnthropicStreamingConverter creates a new Anthropic streaming converter
func NewAnthropicStreamingConverter(reader io.Reader, writer io.Writer, model string) *AnthropicStreamingConverter {
	return &AnthropicStreamingConverter{
		reader: reader,
		writer: writer,
		model:  model,
	}
}

// Convert performs the Anthropic streaming conversion
func (c *AnthropicStreamingConverter) Convert() error {
	scanner := bufio.NewScanner(c.reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // Increase buffer size for large events

	var eventType string
	var messageID string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse event type
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		// Parse data
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			// Skip empty data or ping events
			if data == "" || data == "{\"type\": \"ping\"}" {
				continue
			}

			switch eventType {
			case "message_start":
				c.handleMessageStart(data, &messageID)
			case "content_block_delta":
				c.handleContentDelta(data, messageID)
			case "message_delta":
				c.handleMessageDelta(data, messageID)
			case "message_stop":
				log.Printf("Anthropic stream completed for model: %s", c.model)
				c.handleMessageStop(messageID)
				return nil
			case "content_block_start", "content_block_stop", "ping":
				// These events don't need conversion
				continue
			default:
				log.Printf("Unhandled Anthropic event type: %s", eventType)
			}
		}

		// Empty line (event separator)
		if line == "" {
			eventType = ""
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
		return err
	}

	return nil
}

func (c *AnthropicStreamingConverter) handleMessageStart(data string, messageID *string) {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		log.Printf("Error parsing message_start event: %v", err)
		return
	}

	// Extract message ID
	if message, ok := event["message"].(map[string]interface{}); ok {
		if id, ok := message["id"].(string); ok {
			*messageID = id
		}
	}

	// Send initial chunk with role
	chunk := map[string]interface{}{
		"id":      *messageID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
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

	c.writeChunk(chunk)
}

func (c *AnthropicStreamingConverter) handleContentDelta(data string, messageID string) {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		log.Printf("Error parsing content_block_delta event: %v", err)
		return
	}

	// Extract text delta
	var textDelta string
	if delta, ok := event["delta"].(map[string]interface{}); ok {
		if text, ok := delta["text"].(string); ok {
			textDelta = text
		}
	}

	if textDelta == "" {
		return
	}

	// Create chat completion chunk
	chunk := map[string]interface{}{
		"id":      messageID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   c.model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"content": textDelta,
				},
				"finish_reason": nil,
			},
		},
	}

	c.writeChunk(chunk)
}

func (c *AnthropicStreamingConverter) handleMessageDelta(data string, messageID string) {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		log.Printf("Error parsing message_delta event: %v", err)
		return
	}

	// Extract stop reason
	var stopReason string
	if delta, ok := event["delta"].(map[string]interface{}); ok {
		if reason, ok := delta["stop_reason"].(string); ok {
			stopReason = reason
		}
	}

	// Map Anthropic stop reason to OpenAI finish_reason
	finishReason := "stop"
	switch stopReason {
	case "end_turn":
		finishReason = "stop"
	case "max_tokens":
		finishReason = "length"
	case "stop_sequence":
		finishReason = "stop"
	}

	// Send final chunk with finish_reason
	chunk := map[string]interface{}{
		"id":      messageID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   c.model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{},
				"finish_reason": finishReason,
			},
		},
	}

	c.writeChunk(chunk)
}

func (c *AnthropicStreamingConverter) handleMessageStop(messageID string) {
	// Send [DONE] marker
	c.writer.Write([]byte("data: [DONE]\n\n"))
	if flusher, ok := c.writer.(flushWriter); ok {
		flusher.Flush()
	}
}

func (c *AnthropicStreamingConverter) writeChunk(chunk map[string]interface{}) {
	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		log.Printf("Error marshaling chunk: %v", err)
		return
	}

	c.writer.Write([]byte("data: "))
	c.writer.Write(chunkJSON)
	c.writer.Write([]byte("\n\n"))

	if flusher, ok := c.writer.(flushWriter); ok {
		flusher.Flush()
	}
}
