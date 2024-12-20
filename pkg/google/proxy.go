package google

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gyarbij/azure-oai-proxy/pkg/azure"
)

var (
	GoogleAIAPIKey     = ""
	GoogleAIEndpoint   = "https://generativelanguage.googleapis.com"
	GoogleAIAPIVersion = "v1"
	GoogleAIModelMap   = map[string]string{
		"gemini-pro":          "gemini-pro",
		"gemini-pro-vision":   "gemini-pro-vision",
		"embedding-gecko-001": "embedding-001",
	}
)

type GoogleAIConfig struct {
	APIKey     string
	Endpoint   string
	APIVersion string
	ModelMap   map[string]string
}

func Init(apiKey string) {
	GoogleAIAPIKey = apiKey
	log.Printf("Google AI Studio initialized with API key: %s", apiKey)
}

func NewGoogleAIReverseProxy() *httputil.ReverseProxy {
	config := &GoogleAIConfig{
		APIKey:     GoogleAIAPIKey,
		Endpoint:   GoogleAIEndpoint,
		APIVersion: GoogleAIAPIVersion,
		ModelMap:   GoogleAIModelMap,
	}
	return newGoogleAIReverseProxy(config)
}

func newGoogleAIReverseProxy(config *GoogleAIConfig) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		originalURL := req.URL.String()
		model := getModelFromRequest(req)

		// Map the model name if necessary
		if mappedModel, ok := config.ModelMap[strings.ToLower(model)]; ok {
			model = mappedModel
		}

		// Construct the new URL
		targetURL := fmt.Sprintf("%s/%s/models/%s:generateContent", config.Endpoint, config.APIVersion, model)
		target, err := url.Parse(targetURL)
		if err != nil {
			log.Printf("Error parsing target URL: %v", err)
			return
		}

		// Set the target
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path

		// Add API key as a query parameter
		query := req.URL.Query()
		query.Add("key", config.APIKey)
		req.URL.RawQuery = query.Encode()

		// Remove Authorization header if present
		req.Header.Del("Authorization")

		log.Printf("proxying request %s -> %s", originalURL, req.URL.String())
	}

	return &httputil.ReverseProxy{Director: director}
}

func getModelFromRequest(req *http.Request) string {
	// Check the URL path for the model
	parts := strings.Split(req.URL.Path, "/")
	for i, part := range parts {
		if part == "models" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	// If not found in the path, try to get it from the request body
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(strings.NewReader(string(body))) // Restore the body
		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			if model, ok := data["model"].(string); ok {
				return model
			}
		}
	}

	return ""
}

func FetchGoogleAIModels() ([]azure.Model, error) {
	if GoogleAIAPIKey == "" {
		return nil, fmt.Errorf("Google AI Studio API key not set")
	}

	url := fmt.Sprintf("%s/%s/models?key=%s", GoogleAIEndpoint, GoogleAIAPIVersion, GoogleAIAPIKey)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch Google AI Studio models: %s", string(body))
	}

	var googleModels struct {
		Models []struct {
			Name                       string   `json:"name"`
			Version                    string   `json:"version"`
			DisplayName                string   `json:"displayName"`
			Description                string   `json:"description"`
			InputTokenLimit            int      `json:"inputTokenLimit"`
			OutputTokenLimit           int      `json:"outputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
			Temperature                float64  `json:"temperature,omitempty"`
			TopP                       float64  `json:"topP,omitempty"`
			TopK                       int      `json:"topK,omitempty"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleModels); err != nil {
		return nil, err
	}

	var models []azure.Model
	for _, m := range googleModels.Models {
		// Extract model ID from the name field (e.g., "models/gemini-pro")
		modelID := strings.TrimPrefix(m.Name, "models/")

		models = append(models, azure.Model{
			ID:     modelID,
			Object: "model",
			Capabilities: azure.Capabilities{
				Completion:     true,
				ChatCompletion: true,
				Embeddings:     strings.Contains(modelID, "embedding"),
			},
			LifecycleStatus: "active",
			Status:          "ready",
		})
	}

	return models, nil
}
