package azure

import (
	"testing"
)

func TestIsClaudeModel(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"claude-3-5-sonnet", true},
		{"claude-3.5-sonnet", true},
		{"claude-3-opus", true},
		{"claude-3-sonnet", true},
		{"claude-3-haiku", true},
		{"Claude-3-Opus", true}, // Test case insensitivity
		{"gpt-4", false},
		{"gpt-5-pro", false},
		{"o1-preview", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := isClaudeModel(tt.model)
			if result != tt.expected {
				t.Errorf("isClaudeModel(%q) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestIsGPT5Model(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"gpt-5", true},
		{"gpt-5-pro", true},
		{"gpt-5-mini", true},
		{"GPT-5-Pro", true}, // Test case insensitivity
		{"gpt-4", false},
		{"gpt-4o", false},
		{"claude-3-opus", false},
		{"o1-preview", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := isGPT5Model(tt.model)
			if result != tt.expected {
				t.Errorf("isGPT5Model(%q) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestShouldUseResponsesAPI(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"o3-pro", true},
		{"codex-mini", true},
		{"codex-mini-2025-05-16", true},
		{"O3-Pro", true}, // Test case insensitivity
		{"gpt-5", false},
		{"gpt-4", false},
		{"claude-3-opus", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := shouldUseResponsesAPI(tt.model)
			if result != tt.expected {
				t.Errorf("shouldUseResponsesAPI(%q) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestModelMapper(t *testing.T) {
	// Test that all expected models are in the mapper
	expectedModels := []string{
		"gpt-5", "gpt-5-pro", "gpt-5-mini",
		"claude-3-5-sonnet", "claude-3.5-sonnet", "claude-3-opus",
		"claude-3-sonnet", "claude-3-haiku",
		"gpt-4o", "gpt-4", "gpt-3.5-turbo",
	}

	for _, model := range expectedModels {
		if _, ok := AzureOpenAIModelMapper[model]; !ok {
			t.Errorf("Expected model %q to be in AzureOpenAIModelMapper", model)
		}
	}
}
