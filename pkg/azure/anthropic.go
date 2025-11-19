package azure

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const defaultAnthropicMaxTokens = 1024

func deriveAnthropicEndpoint(openAIEndpoint string) string {
	remote, err := url.Parse(openAIEndpoint)
	if err != nil {
		return ""
	}

	host := remote.Hostname()
	if host == "" {
		host = remote.Host
	}

	const suffix = ".openai.azure.com"
	if strings.HasSuffix(host, suffix) {
		resource := strings.TrimSuffix(host, suffix)
		if resource == "" {
			return ""
		}
		return fmt.Sprintf("%s://%s.services.ai.azure.com/anthropic", remote.Scheme, resource)
	}
	return ""
}

func isClaudeModel(model string) bool {
	modelLower := strings.ToLower(model)
	return strings.Contains(modelLower, "claude")
}

func handleClaudeProxyRequest(req *http.Request, model string) error {
	if AzureAnthropicEndpoint == "" {
		return fmt.Errorf("anthropic endpoint is not configured")
	}

	stream, err := convertOpenAIRequestToAnthropic(req)
	if err != nil {
		return err
	}

	base, err := url.Parse(AzureAnthropicEndpoint)
	if err != nil {
		return err
	}

	req.URL.Scheme = base.Scheme
	req.URL.Host = base.Host
	req.Host = base.Host
	targetPath := path.Join(base.Path, "v1", "messages")
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	req.URL.Path = targetPath
	req.URL.RawQuery = ""

	req.Header.Set("X-Original-Path", "/v1/chat/completions")
	req.Header.Set("X-Proxy-Provider", "anthropic")
	req.Header.Set("X-Model", model)

	apiKey := extractAPIKey(req)
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	req.Header.Del("api-key")
	req.Header.Del("Authorization")

	req.Header.Set("anthropic-version", AzureAnthropicAPIVersion)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
	}

	return nil
}

func convertOpenAIRequestToAnthropic(req *http.Request) (bool, error) {
	if req.Body == nil {
		return false, fmt.Errorf("request body is required for Claude models")
	}

	originalBody, err := io.ReadAll(req.Body)
	if err != nil {
		return false, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(originalBody, &payload); err != nil {
		return false, err
	}

	messagesRaw, ok := payload["messages"].([]interface{})
	if !ok || len(messagesRaw) == 0 {
		return false, fmt.Errorf("Claude requests require at least one message")
	}

	var (
		systemPrompts     []string
		anthropicMessages []map[string]interface{}
	)

	for _, rawMsg := range messagesRaw {
		msg, ok := rawMsg.(map[string]interface{})
		if !ok {
			continue
		}

		role := strings.ToLower(getStringValue(msg["role"]))
		switch role {
		case "system":
			if text := flattenContentToString(msg["content"]); text != "" {
				systemPrompts = append(systemPrompts, text)
			}
		case "user", "assistant":
			blocks := convertContentToBlocks(msg["content"])
			if len(blocks) == 0 {
				continue
			}
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    role,
				"content": blocks,
			})
		case "tool":
			blocks := convertContentToBlocks(msg["content"])
			if len(blocks) == 0 {
				continue
			}
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    "assistant",
				"content": blocks,
			})
		}
	}

	if len(anthropicMessages) == 0 {
		return false, fmt.Errorf("no user or assistant messages were provided")
	}

	if role, _ := anthropicMessages[0]["role"].(string); role != "user" {
		for i := 1; i < len(anthropicMessages); i++ {
			if anthropicMessages[i]["role"] == "user" {
				userMsg := anthropicMessages[i]
				copy(anthropicMessages[1:i+1], anthropicMessages[0:i])
				anthropicMessages[0] = userMsg
				break
			}
		}
	}

	maxTokens := getIntFromInterface(payload["max_tokens"])
	if maxTokens <= 0 {
		maxTokens = defaultAnthropicMaxTokens
	}

	temperature := getFloatFromInterface(payload["temperature"])
	topP := getFloatFromInterface(payload["top_p"])
	stopSequences := extractStopSequences(payload["stop"])
	if len(stopSequences) == 0 {
		stopSequences = extractStopSequences(payload["stop_sequences"])
	}

	stream := false
	if streamVal, ok := payload["stream"].(bool); ok && streamVal {
		stream = true
	}

	newBody := map[string]interface{}{
		"model":      payload["model"],
		"messages":   anthropicMessages,
		"max_tokens": maxTokens,
	}

	if len(systemPrompts) > 0 {
		newBody["system"] = strings.Join(systemPrompts, "\n")
	}
	if temperature > 0 {
		newBody["temperature"] = temperature
	}
	if topP > 0 {
		newBody["top_p"] = topP
	}
	if len(stopSequences) > 0 {
		newBody["stop_sequences"] = stopSequences
	}
	if stream {
		newBody["stream"] = true
	}
	if metadata, ok := payload["metadata"].(map[string]interface{}); ok && len(metadata) > 0 {
		newBody["metadata"] = metadata
	}

	newBodyBytes, err := json.Marshal(newBody)
	if err != nil {
		return stream, err
	}

	req.Body = io.NopCloser(bytes.NewBuffer(newBodyBytes))
	req.ContentLength = int64(len(newBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	return stream, nil
}

func getStringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", v)
}

func flattenContentToString(content interface{}) string {
	blocks := convertContentToBlocks(content)
	if len(blocks) == 0 {
		return ""
	}

	var parts []string
	for _, block := range blocks {
		if text, ok := block["text"].(string); ok {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func convertContentToBlocks(content interface{}) []map[string]interface{} {
	switch v := content.(type) {
	case nil:
		return nil
	case string:
		if v == "" {
			return nil
		}
		return []map[string]interface{}{{"type": "text", "text": v}}
	case []interface{}:
		var blocks []map[string]interface{}
		for _, item := range v {
			switch block := item.(type) {
			case string:
				if block != "" {
					blocks = append(blocks, map[string]interface{}{"type": "text", "text": block})
				}
			case map[string]interface{}:
				if normalized := normalizeContentBlock(block); normalized != nil {
					blocks = append(blocks, normalized)
				}
			}
		}
		return blocks
	case map[string]interface{}:
		if normalized := normalizeContentBlock(v); normalized != nil {
			return []map[string]interface{}{normalized}
		}
		return nil
	default:
		return []map[string]interface{}{{"type": "text", "text": fmt.Sprintf("%v", v)}}
	}
}

func normalizeContentBlock(block map[string]interface{}) map[string]interface{} {
	blockType, _ := block["type"].(string)
	switch blockType {
	case "text", "input_text":
		if text, ok := block["text"].(string); ok && text != "" {
			return map[string]interface{}{"type": "text", "text": text}
		}
	}
	return nil
}

func getIntFromInterface(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case float32:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case json.Number:
		i, _ := val.Int64()
		return int(i)
	default:
		return 0
	}
}

func getFloatFromInterface(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	default:
		return 0
	}
}

func extractStopSequences(v interface{}) []string {
	switch val := v.(type) {
	case string:
		if val == "" {
			return nil
		}
		return []string{val}
	case []interface{}:
		var stops []string
		for _, item := range val {
			if s, ok := item.(string); ok && s != "" {
				stops = append(stops, s)
			}
		}
		return stops
	case []string:
		return val
	default:
		return nil
	}
}

func convertAnthropicResponse(res *http.Response) {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading Anthropic response: %v", err)
		res.Body = io.NopCloser(bytes.NewBuffer(nil))
		return
	}

	res.Body.Close()

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("Error unmarshaling Anthropic response: %v", err)
		res.Body = io.NopCloser(bytes.NewBuffer(body))
		return
	}

	contentText := extractAnthropicText(response)
	finishReason := mapAnthropicFinishReason(getStringValue(response["stop_reason"]))

	usage := map[string]int{
		"prompt_tokens":     getIntFromInterface(getNestedValue(response, "usage", "input_tokens")),
		"completion_tokens": getIntFromInterface(getNestedValue(response, "usage", "output_tokens")),
		"total_tokens":      0,
	}
	usage["total_tokens"] = usage["prompt_tokens"] + usage["completion_tokens"]

	created := getIntFromInterface(response["created_at"])
	chatResponse := map[string]interface{}{
		"id":      response["id"],
		"object":  "chat.completion",
		"created": created,
		"model":   response["model"],
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": contentText,
				},
				"finish_reason": finishReason,
			},
		},
		"usage": usage,
	}

	newBody, err := json.Marshal(chatResponse)
	if err != nil {
		log.Printf("Error marshaling converted Anthropic response: %v", err)
		res.Body = io.NopCloser(bytes.NewBuffer(body))
		return
	}

	res.Body = io.NopCloser(bytes.NewBuffer(newBody))
	res.ContentLength = int64(len(newBody))
	res.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
	res.Header.Set("Content-Type", "application/json")
}

func extractAnthropicText(response map[string]interface{}) string {
	contentRaw, ok := response["content"].([]interface{})
	if !ok {
		return ""
	}

	var parts []string
	for _, item := range contentRaw {
		block, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if block["type"] == "text" {
			if text, ok := block["text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func getNestedValue(data map[string]interface{}, keys ...string) interface{} {
	current := interface{}(data)
	for _, key := range keys {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = obj[key]
	}
	return current
}

func mapAnthropicFinishReason(reason string) string {
	switch reason {
	case "max_tokens":
		return "length"
	case "stop_sequence", "end_turn":
		return "stop"
	default:
		if reason == "" {
			return "stop"
		}
		return reason
	}
}

// Anthropic streaming conversion

type AnthropicStreamingConverter struct {
	reader io.Reader
	writer io.Writer
	model  string
}

func NewAnthropicStreamingConverter(reader io.Reader, writer io.Writer, model string) *AnthropicStreamingConverter {
	return &AnthropicStreamingConverter{reader: reader, writer: writer, model: model}
}

func (c *AnthropicStreamingConverter) Convert() error {
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
			case "content_block_delta":
				c.handleDelta(data)
			case "message_stop":
				c.handleStop()
			}
		}
	}

	return scanner.Err()
}

func (c *AnthropicStreamingConverter) handleDelta(data string) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		log.Printf("Error parsing Anthropic delta event: %v", err)
		return
	}

	delta, _ := getNestedValue(payload, "delta").(map[string]interface{})
	text, _ := delta["text"].(string)
	if text == "" {
		return
	}

	chunk := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   c.model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"content": text,
				},
				"finish_reason": nil,
			},
		},
	}

	c.writeChunk(chunk)
}

func (c *AnthropicStreamingConverter) handleStop() {
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
	c.writer.Write([]byte("data: [DONE]\n\n"))
	if flusher, ok := c.writer.(flushWriter); ok {
		flusher.Flush()
	}
}

func (c *AnthropicStreamingConverter) writeChunk(chunk map[string]interface{}) {
	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		log.Printf("Error marshaling Anthropic chunk: %v", err)
		return
	}

	c.writer.Write([]byte("data: "))
	c.writer.Write(chunkJSON)
	c.writer.Write([]byte("\n\n"))

	if flusher, ok := c.writer.(flushWriter); ok {
		flusher.Flush()
	}
}
