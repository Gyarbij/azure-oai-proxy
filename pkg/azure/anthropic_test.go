package azure

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestAnthropicStreamingConverterHandleDeltaStringContent(t *testing.T) {
	originalSetting := OpenAIStreamingDeltaContentArray
	OpenAIStreamingDeltaContentArray = false
	defer func() { OpenAIStreamingDeltaContentArray = originalSetting }()

	buf := &bytes.Buffer{}
	converter := &AnthropicStreamingConverter{writer: buf, model: "claude-sonnet"}

	converter.handleDelta(`{"delta":{"text":"Hello"}}`)

	chunk := decodeStreamingChunk(t, buf.String())
	choices, ok := chunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Fatalf("expected choices slice in chunk, got %v", chunk["choices"])
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		t.Fatalf("choice payload is not a map: %T", choices[0])
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		t.Fatalf("delta is not a map: %T", choice["delta"])
	}

	content, ok := delta["content"].(string)
	if !ok {
		t.Fatalf("expected string content, got %T", delta["content"])
	}

	if content != "Hello" {
		t.Fatalf("expected content \"Hello\", got %q", content)
	}
}

func TestAnthropicStreamingConverterHandleDeltaArrayContent(t *testing.T) {
	originalSetting := OpenAIStreamingDeltaContentArray
	OpenAIStreamingDeltaContentArray = true
	defer func() { OpenAIStreamingDeltaContentArray = originalSetting }()

	buf := &bytes.Buffer{}
	converter := &AnthropicStreamingConverter{writer: buf, model: "claude-opus"}

	converter.handleDelta(`{"delta":{"text":"Chunk"}}`)

	chunk := decodeStreamingChunk(t, buf.String())
	choices, ok := chunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Fatalf("expected choices slice in chunk, got %v", chunk["choices"])
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		t.Fatalf("choice payload is not a map: %T", choices[0])
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		t.Fatalf("delta is not a map: %T", choice["delta"])
	}

	contentBlocks, ok := delta["content"].([]interface{})
	if !ok || len(contentBlocks) != 1 {
		t.Fatalf("expected single content block, got %v", delta["content"])
	}

	block, ok := contentBlocks[0].(map[string]interface{})
	if !ok {
		t.Fatalf("content block is not a map: %T", contentBlocks[0])
	}

	if block["type"] != "output_text" {
		t.Fatalf("unexpected content block type: %v", block["type"])
	}

	if block["text"] != "Chunk" {
		t.Fatalf("unexpected content block text: %v", block["text"])
	}
}

func decodeStreamingChunk(t *testing.T, raw string) map[string]interface{} {
	t.Helper()

	raw = strings.TrimSpace(raw)
	if raw == "" {
		t.Fatalf("stream contained no data")
	}

	line := raw
	if idx := strings.Index(raw, "\n"); idx >= 0 {
		line = raw[:idx]
	}

	line = strings.TrimPrefix(line, "data: ")
	if line == raw {
		t.Fatalf("chunk did not start with data prefix: %q", raw)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("failed to unmarshal chunk: %v", err)
	}

	return payload
}
