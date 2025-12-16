package anthropic

import "testing"

func TestConvertOpenAIToAnthropicMessagesArray(t *testing.T) {
	payload := map[string]interface{}{
		"model": "claude-sonnet-4-5",
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "stay helpful"},
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": "hello"},
				},
			},
		},
		"max_tokens": float64(128),
	}

	req, err := ConvertOpenAIToAnthropic(payload)
	if err != nil {
		t.Fatalf("ConvertOpenAIToAnthropic returned error: %v", err)
	}

	if req.Model != "claude-sonnet-4-5" {
		t.Fatalf("unexpected model: %s", req.Model)
	}

	if req.System != "stay helpful" {
		t.Fatalf("unexpected system message: %q", req.System)
	}

	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(req.Messages))
	}

	if req.Messages[0].Role != "user" {
		t.Fatalf("unexpected role: %s", req.Messages[0].Role)
	}

	if req.Messages[0].Content != "hello" {
		t.Fatalf("unexpected content: %q", req.Messages[0].Content)
	}

	if req.MaxTokens != 128 {
		t.Fatalf("unexpected max tokens: %d", req.MaxTokens)
	}
}

func TestConvertOpenAIToAnthropicInputOnly(t *testing.T) {
	payload := map[string]interface{}{
		"model": "claude-sonnet-4-5",
		"input": "show me code",
		"stop":  "STOP",
	}

	req, err := ConvertOpenAIToAnthropic(payload)
	if err != nil {
		t.Fatalf("ConvertOpenAIToAnthropic returned error: %v", err)
	}

	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 normalized message, got %d", len(req.Messages))
	}

	if req.Messages[0].Content != "show me code" {
		t.Fatalf("unexpected normalized content: %q", req.Messages[0].Content)
	}

	if req.Messages[0].Role != "user" {
		t.Fatalf("expected role user, got %s", req.Messages[0].Role)
	}

	if len(req.StopSequences) != 1 || req.StopSequences[0] != "STOP" {
		t.Fatalf("unexpected stop sequences: %#v", req.StopSequences)
	}
}
