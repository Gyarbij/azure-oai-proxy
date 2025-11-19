package azure

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestIsClaudeModel(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"claude-sonnet-4.5", true},
		{"claude-sonnet-4-5", true},
		{"claude-haiku-4.5", true},
		{"claude-haiku-4-5", true},
		{"claude-opus-4.1", true},
		{"claude-opus-4-1", true},
		{"Claude-Sonnet-4.5", true}, // Test case insensitivity
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
		{"claude-opus-4.1", false},
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
		{"claude-opus-4.1", false},
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
		"claude-sonnet-4.5", "claude-sonnet-4-5", "claude-haiku-4.5",
		"claude-haiku-4-5", "claude-opus-4.1", "claude-opus-4-1",
		"gpt-4o", "gpt-4", "gpt-3.5-turbo",
	}

	for _, model := range expectedModels {
		if _, ok := AzureOpenAIModelMapper[model]; !ok {
			t.Errorf("Expected model %q to be in AzureOpenAIModelMapper", model)
		}
	}
}

func TestHandleGPT5Request(t *testing.T) {
	// Set up a test endpoint
	AzureOpenAIEndpoint = "https://test.openai.azure.com/"
	
	tests := []struct {
		name           string
		inputPath      string
		deployment     string
		expectedPath   string
	}{
		{
			name:         "chat completions",
			inputPath:    "/v1/chat/completions",
			deployment:   "gpt-5-pro",
			expectedPath: "/openai/deployments/gpt-5-pro/v1/chat/completions",
		},
		{
			name:         "completions",
			inputPath:    "/v1/completions",
			deployment:   "gpt-5",
			expectedPath: "/openai/deployments/gpt-5/v1/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "http://test.com"+tt.inputPath, nil)
			handleGPT5Request(req, tt.deployment)
			
			if req.URL.Path != tt.expectedPath {
				t.Errorf("handleGPT5Request() path = %q, want %q", req.URL.Path, tt.expectedPath)
			}
			
			// Check that api-version parameter was added
			if req.URL.Query().Get("api-version") == "" {
				t.Error("handleGPT5Request() did not add api-version query parameter")
			}
		})
	}
}

func TestHandleClaudeRequest(t *testing.T) {
	// Set up a test endpoint
	AzureOpenAIEndpoint = "https://test.openai.azure.com/"
	
	tests := []struct {
		name           string
		inputPath      string
		deployment     string
		expectedPath   string
	}{
		{
			name:         "chat completions",
			inputPath:    "/v1/chat/completions",
			deployment:   "claude-sonnet-4.5",
			expectedPath: "/models/claude-sonnet-4.5/chat/completions",
		},
		{
			name:         "completions",
			inputPath:    "/v1/completions",
			deployment:   "claude-opus-4.1",
			expectedPath: "/models/claude-opus-4.1/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "http://test.com"+tt.inputPath, nil)
			handleClaudeRequest(req, tt.deployment)
			
			if req.URL.Path != tt.expectedPath {
				t.Errorf("handleClaudeRequest() path = %q, want %q", req.URL.Path, tt.expectedPath)
			}
			
			// Check that api-version parameter was added
			if req.URL.Query().Get("api-version") == "" {
				t.Error("handleClaudeRequest() did not add api-version query parameter")
			}
		})
	}
}

func TestHandleRegularRequest(t *testing.T) {
	// Set up a test endpoint
	originalEndpoint := AzureOpenAIEndpoint
	AzureOpenAIEndpoint = "https://test.openai.azure.com/"
	defer func() { AzureOpenAIEndpoint = originalEndpoint }()
	
	tests := []struct {
		name           string
		inputPath      string
		deployment     string
		expectGPT5     bool
		expectClaude   bool
		expectedPrefix string
	}{
		{
			name:           "GPT-5 model",
			inputPath:      "/v1/chat/completions",
			deployment:     "gpt-5-pro",
			expectGPT5:     true,
			expectedPrefix: "/openai/deployments/gpt-5-pro/v1/",
		},
		{
			name:           "Claude model",
			inputPath:      "/v1/chat/completions",
			deployment:     "claude-opus-4.1",
			expectClaude:   true,
			expectedPrefix: "/models/claude-opus-4.1/",
		},
		{
			name:           "Regular GPT-4 model",
			inputPath:      "/v1/chat/completions",
			deployment:     "gpt-4",
			expectedPrefix: "/openai/deployments/gpt-4/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedURL, _ := url.Parse("https://test.openai.azure.com/")
			req := &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "http",
					Host:   "test.com",
					Path:   tt.inputPath,
				},
			}
			
			// Call the function
			handleRegularRequest(req, tt.deployment)
			
			// Verify the URL was modified correctly
			if req.URL.Scheme != parsedURL.Scheme {
				t.Errorf("URL scheme = %q, want %q", req.URL.Scheme, parsedURL.Scheme)
			}
			
			if req.URL.Host != parsedURL.Host {
				t.Errorf("URL host = %q, want %q", req.URL.Host, parsedURL.Host)
			}
			
			// For GPT-5 and Claude, the paths should have been set by their handlers
			if tt.expectGPT5 || tt.expectClaude {
				if !strings.HasPrefix(req.URL.Path, tt.expectedPrefix) {
					t.Errorf("Path = %q, want prefix %q", req.URL.Path, tt.expectedPrefix)
				}
			}
		})
	}
}
