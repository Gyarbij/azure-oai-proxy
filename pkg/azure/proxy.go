package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

var (
	AzureOpenAIAPIVersion          = "2024-08-01-preview" // API version for proxying requests - supports Azure Foundry features
	AzureOpenAIModelsAPIVersion    = "2024-10-21"         // API version for fetching models
	AzureOpenAIResponsesAPIVersion = "2024-08-01-preview" // API version for Responses API - supports O-series models
	AnthropicAPIVersion            = "2023-06-01"         // Anthropic API version for Claude models
	AzureOpenAIEndpoint            = ""
	ServerlessDeploymentInfo       = make(map[string]ServerlessDeployment)
	AzureOpenAIModelMapper         = make(map[string]string)
)

type ServerlessDeployment struct {
	Name   string
	Region string
	Key    string
}

func init() {
	if v := os.Getenv("AZURE_OPENAI_APIVERSION"); v != "" {
		AzureOpenAIAPIVersion = v
	}
	if v := os.Getenv("AZURE_OPENAI_MODELS_APIVERSION"); v != "" {
		AzureOpenAIModelsAPIVersion = v
	}
	if v := os.Getenv("AZURE_OPENAI_RESPONSES_APIVERSION"); v != "" {
		AzureOpenAIResponsesAPIVersion = v
	}
	if v := os.Getenv("ANTHROPIC_APIVERSION"); v != "" {
		AnthropicAPIVersion = v
	}
	if v := os.Getenv("AZURE_OPENAI_ENDPOINT"); v != "" {
		AzureOpenAIEndpoint = v
	}

	if v := os.Getenv("AZURE_AI_STUDIO_DEPLOYMENTS"); v != "" {
		for _, pair := range strings.Split(v, ",") {
			info := strings.Split(pair, "=")
			if len(info) == 2 {
				deploymentInfo := strings.Split(info[1], ":")
				if len(deploymentInfo) == 2 {
					ServerlessDeploymentInfo[strings.ToLower(info[0])] = ServerlessDeployment{
						Name:   deploymentInfo[0],
						Region: deploymentInfo[1],
						Key:    os.Getenv("AZURE_OPENAI_KEY_" + strings.ToUpper(info[0])),
					}
				}
			}
		}
	}

	// Initialize AzureOpenAIModelMapper with updated model list and hardcode as failsafe
	AzureOpenAIModelMapper = map[string]string{
		// GPT-5.2 series (NEW)
		"gpt-5.2":                 "gpt-5.2",
		"gpt-5.2-2025-12-11":      "gpt-5.2-2025-12-11",
		"gpt-5.2-chat":            "gpt-5.2-chat",
		"gpt-5.2-chat-2025-12-11": "gpt-5.2-chat-2025-12-11",
		// GPT-5.1 series (NEW)
		"gpt-5.1":                       "gpt-5.1",
		"gpt-5.1-2025-11-13":            "gpt-5.1-2025-11-13",
		"gpt-5.1-chat":                  "gpt-5.1-chat",
		"gpt-5.1-chat-2025-11-13":       "gpt-5.1-chat-2025-11-13",
		"gpt-5.1-codex":                 "gpt-5.1-codex",
		"gpt-5.1-codex-2025-11-13":      "gpt-5.1-codex-2025-11-13",
		"gpt-5.1-codex-mini":            "gpt-5.1-codex-mini",
		"gpt-5.1-codex-mini-2025-11-13": "gpt-5.1-codex-mini-2025-11-13",
		"gpt-5.1-codex-max":             "gpt-5.1-codex-max",
		"gpt-5.1-codex-max-2025-12-04":  "gpt-5.1-codex-max-2025-12-04",
		// GPT-5 series
		"gpt-5":                  "gpt-5",
		"gpt-5-2025-08-07":       "gpt-5-2025-08-07",
		"gpt-5-mini":             "gpt-5-mini",
		"gpt-5-mini-2025-08-07":  "gpt-5-mini-2025-08-07",
		"gpt-5-nano":             "gpt-5-nano",
		"gpt-5-nano-2025-08-07":  "gpt-5-nano-2025-08-07",
		"gpt-5-chat":             "gpt-5-chat",
		"gpt-5-chat-2025-08-07":  "gpt-5-chat-2025-08-07",
		"gpt-5-chat-2025-10-03":  "gpt-5-chat-2025-10-03",
		"gpt-5-codex":            "gpt-5-codex",
		"gpt-5-codex-2025-09-11": "gpt-5-codex-2025-09-11",
		"gpt-5-pro":              "gpt-5-pro",
		"gpt-5-pro-2025-10-06":   "gpt-5-pro-2025-10-06",
		// GPT-4.1 series
		"gpt-4.1":                 "gpt-4.1",
		"gpt-4.1-2025-04-14":      "gpt-4.1-2025-04-14",
		"gpt-4.1-mini":            "gpt-4.1-mini",
		"gpt-4.1-mini-2025-04-14": "gpt-4.1-mini-2025-04-14",
		"gpt-4.1-nano":            "gpt-4.1-nano",
		"gpt-4.1-nano-2025-04-14": "gpt-4.1-nano-2025-04-14",
		// O-series reasoning models
		"o1":                              "o1",
		"o1-2024-12-17":                   "o1-2024-12-17",
		"o1-preview":                      "o1-preview",
		"o1-preview-2024-09-12":           "o1-preview-2024-09-12",
		"o1-mini":                         "o1-mini",
		"o1-mini-2024-09-12":              "o1-mini-2024-09-12",
		"o3":                              "o3",
		"o3-2025-04-16":                   "o3-2025-04-16",
		"o3-mini":                         "o3-mini",
		"o3-mini-2025-01-31":              "o3-mini-2025-01-31",
		"o3-pro":                          "o3-pro",
		"o3-pro-2025-06-10":               "o3-pro-2025-06-10",
		"o3-deep-research":                "o3-deep-research",
		"o3-deep-research-2025-06-26":     "o3-deep-research-2025-06-26",
		"o4":                              "o4",
		"o4-mini":                         "o4-mini",
		"o4-mini-2025-04-16":              "o4-mini-2025-04-16",
		"codex-mini":                      "codex-mini",
		"codex-mini-2025-05-16":           "codex-mini-2025-05-16",
		"computer-use-preview":            "computer-use-preview",
		"computer-use-preview-2025-03-11": "computer-use-preview-2025-03-11",
		// gpt-oss (open-weight reasoning models)
		"gpt-oss-120b": "gpt-oss-120b",
		"gpt-oss-20b":  "gpt-oss-20b",
		// Claude models (Azure Foundry) - Claude 4.x series available via global standard deployment
		"claude-opus-4.5":   "claude-opus-4.5",
		"claude-opus-4-5":   "claude-opus-4.5",
		"claude-sonnet-4.5": "claude-sonnet-4.5",
		"claude-sonnet-4-5": "claude-sonnet-4.5",
		"claude-haiku-4.5":  "claude-haiku-4.5",
		"claude-haiku-4-5":  "claude-haiku-4.5",
		"claude-opus-4.1":   "claude-opus-4.1",
		"claude-opus-4-1":   "claude-opus-4.1",
		// GPT-4o models
		"gpt-4o":                 "gpt-4o",
		"gpt-4o-2024-05-13":      "gpt-4o-2024-05-13",
		"gpt-4o-2024-08-06":      "gpt-4o-2024-08-06",
		"gpt-4o-2024-11-20":      "gpt-4o-2024-11-20",
		"gpt-4o-mini":            "gpt-4o-mini",
		"gpt-4o-mini-2024-07-18": "gpt-4o-mini-2024-07-18",
		// GPT-4 models
		"gpt-4":                  "gpt-4-0613",
		"gpt-4-0613":             "gpt-4-0613",
		"gpt-4-1106-preview":     "gpt-4-1106-preview",
		"gpt-4-0125-preview":     "gpt-4-0125-preview",
		"gpt-4-vision-preview":   "gpt-4-vision-preview",
		"gpt-4-turbo":            "gpt-4-turbo",
		"gpt-4-turbo-2024-04-09": "gpt-4-turbo-2024-04-09",
		"gpt-4-32k":              "gpt-4-32k-0613",
		"gpt-4-32k-0613":         "gpt-4-32k-0613",
		// GPT-3.5 models
		"gpt-3.5-turbo":               "gpt-35-turbo-0613",
		"gpt-3.5-turbo-0301":          "gpt-35-turbo-0301",
		"gpt-3.5-turbo-0613":          "gpt-35-turbo-0613",
		"gpt-3.5-turbo-1106":          "gpt-35-turbo-1106",
		"gpt-3.5-turbo-0125":          "gpt-35-turbo-0125",
		"gpt-3.5-turbo-16k":           "gpt-35-turbo-16k-0613",
		"gpt-3.5-turbo-16k-0613":      "gpt-35-turbo-16k-0613",
		"gpt-3.5-turbo-instruct":      "gpt-35-turbo-instruct-0914",
		"gpt-3.5-turbo-instruct-0914": "gpt-35-turbo-instruct-0914",
		// Embedding models
		"text-embedding-3-small":   "text-embedding-3-small-1",
		"text-embedding-3-large":   "text-embedding-3-large-1",
		"text-embedding-ada-002":   "text-embedding-ada-002-2",
		"text-embedding-ada-002-1": "text-embedding-ada-002-1",
		"text-embedding-ada-002-2": "text-embedding-ada-002-2",
		// DALL-E models
		"dall-e-2":     "dall-e-2-2.0",
		"dall-e-2-2.0": "dall-e-2-2.0",
		"dall-e-3":     "dall-e-3-3.0",
		"dall-e-3-3.0": "dall-e-3-3.0",
		// Legacy models
		"babbage-002":   "babbage-002-1",
		"babbage-002-1": "babbage-002-1",
		"davinci-002":   "davinci-002-1",
		"davinci-002-1": "davinci-002-1",
		// Audio models - GPT-4o audio
		"gpt-4o-audio-preview":                    "gpt-4o-audio-preview",
		"gpt-4o-audio-preview-2024-12-17":         "gpt-4o-audio-preview-2024-12-17",
		"gpt-4o-mini-audio-preview":               "gpt-4o-mini-audio-preview",
		"gpt-4o-mini-audio-preview-2024-12-17":    "gpt-4o-mini-audio-preview-2024-12-17",
		"gpt-4o-realtime-preview":                 "gpt-4o-realtime-preview",
		"gpt-4o-realtime-preview-2024-12-17":      "gpt-4o-realtime-preview-2024-12-17",
		"gpt-4o-realtime-preview-2025-06-03":      "gpt-4o-realtime-preview-2025-06-03",
		"gpt-4o-mini-realtime-preview":            "gpt-4o-mini-realtime-preview",
		"gpt-4o-mini-realtime-preview-2024-12-17": "gpt-4o-mini-realtime-preview-2024-12-17",
		"gpt-realtime":                            "gpt-realtime",
		"gpt-realtime-2025-08-28":                 "gpt-realtime-2025-08-28",
		"gpt-realtime-mini":                       "gpt-realtime-mini",
		"gpt-realtime-mini-2025-10-06":            "gpt-realtime-mini-2025-10-06",
		"gpt-audio":                               "gpt-audio",
		"gpt-audio-2025-08-28":                    "gpt-audio-2025-08-28",
		"gpt-audio-mini":                          "gpt-audio-mini",
		"gpt-audio-mini-2025-10-06":               "gpt-audio-mini-2025-10-06",
		// Speech-to-text models
		"gpt-4o-transcribe":                    "gpt-4o-transcribe",
		"gpt-4o-transcribe-2025-03-20":         "gpt-4o-transcribe-2025-03-20",
		"gpt-4o-mini-transcribe":               "gpt-4o-mini-transcribe",
		"gpt-4o-mini-transcribe-2025-03-20":    "gpt-4o-mini-transcribe-2025-03-20",
		"gpt-4o-transcribe-diarize":            "gpt-4o-transcribe-diarize",
		"gpt-4o-transcribe-diarize-2025-10-15": "gpt-4o-transcribe-diarize-2025-10-15",
		// TTS models
		"tts":                        "tts-001",
		"tts-001":                    "tts-001",
		"tts-hd":                     "tts-hd-001",
		"tts-hd-001":                 "tts-hd-001",
		"gpt-4o-mini-tts":            "gpt-4o-mini-tts",
		"gpt-4o-mini-tts-2025-03-20": "gpt-4o-mini-tts-2025-03-20",
		// Whisper models
		"whisper":     "whisper-001",
		"whisper-001": "whisper-001",
		// Image generation models
		"gpt-image-1":                 "gpt-image-1",
		"gpt-image-1-2025-04-15":      "gpt-image-1-2025-04-15",
		"gpt-image-1-mini":            "gpt-image-1-mini",
		"gpt-image-1-mini-2025-10-06": "gpt-image-1-mini-2025-10-06",
		// Video generation models
		"sora":              "sora",
		"sora-2025-05-02":   "sora-2025-05-02",
		"sora-2":            "sora-2",
		"sora-2-2025-10-06": "sora-2-2025-10-06",
		// Phi models (Azure Foundry)
		"phi-3":        "phi-3",
		"phi-3-mini":   "phi-3-mini",
		"phi-3-small":  "phi-3-small",
		"phi-3-medium": "phi-3-medium",
		"phi-4":        "phi-4",
	}

	log.Printf("Loaded ServerlessDeploymentInfo: %+v", ServerlessDeploymentInfo)
	log.Printf("Azure OpenAI Endpoint: %s", AzureOpenAIEndpoint)
	log.Printf("Azure OpenAI API Version: %s", AzureOpenAIAPIVersion)
	log.Printf("Azure OpenAI Models API Version: %s", AzureOpenAIModelsAPIVersion)
	log.Printf("Azure OpenAI Responses API Version: %s", AzureOpenAIResponsesAPIVersion)
}

// stripModelVersion removes date/version suffixes from model names
// Examples: gpt-5.2-2025-12-11 -> gpt-5.2, claude-haiku-4-5-20251001 -> claude-haiku-4-5
func stripModelVersion(model string) string {
	// Pattern matches: -YYYY-MM-DD or -YYYYMMDD at the end of the string
	re := regexp.MustCompile(`-\d{4}-\d{2}-\d{2}$|-\d{8}$`)
	stripped := re.ReplaceAllString(model, "")
	if stripped != model {
		log.Printf("Stripped version suffix from model: %s -> %s", model, stripped)
	}
	return stripped
}

// resolveModelDeployment resolves a model name to its deployment name
// It handles versioned model names automatically and falls back to the model mapper
func resolveModelDeployment(model string) string {
	modelLower := strings.ToLower(model)

	// First, try exact match in the mapper
	if azureModel, ok := AzureOpenAIModelMapper[modelLower]; ok {
		log.Printf("Model %s found in mapper as %s", model, azureModel)
		return azureModel
	}

	// Try stripping version suffix and matching again
	strippedModel := stripModelVersion(modelLower)
	if strippedModel != modelLower {
		if azureModel, ok := AzureOpenAIModelMapper[strippedModel]; ok {
			log.Printf("Model %s matched stripped version %s in mapper as %s", model, strippedModel, azureModel)
			return azureModel
		}
	}

	// If not found, use the original model name (works for custom deployments)
	log.Printf("Model %s not found in mapper, using as-is for deployment", model)
	return model
}

func NewOpenAIReverseProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director:       makeDirector(),
		ModifyResponse: modifyResponse,
		FlushInterval:  -1, // Flush immediately for SSE streaming - critical for OpenWebUI compatibility
	}
}

func HandleToken(req *http.Request) {
	model := getModelFromRequest(req)
	modelLower := strings.ToLower(model)
	// Check if it's a serverless deployment
	if info, ok := ServerlessDeploymentInfo[modelLower]; ok {
		// Set the correct authorization header for serverless
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", info.Key))
		req.Header.Del("api-key")
		log.Printf("Using serverless deployment authentication for %s", model)
	} else {
		// For regular Azure OpenAI deployments, use the api-key
		apiKey := req.Header.Get("api-key")
		if apiKey == "" {
			apiKey = req.Header.Get("Authorization")
			if strings.HasPrefix(apiKey, "Bearer ") {
				apiKey = strings.TrimPrefix(apiKey, "Bearer ")
			}
		}
		if apiKey == "" {
			log.Printf("Warning: No api-key or Authorization header found for deployment: %s", model)
		} else {
			req.Header.Set("api-key", apiKey)
			req.Header.Del("Authorization")
			log.Printf("Using regular Azure OpenAI authentication for %s", model)
		}
	}
}

func makeDirector() func(*http.Request) {
	return func(req *http.Request) {
		model := getModelFromRequest(req)
		originURL := req.URL.String()
		log.Printf("========== NEW REQUEST ==========")
		log.Printf("Original request URL: %s", originURL)
		log.Printf("Request method: %s", req.Method)
		log.Printf("Request path: %s", req.URL.Path)
		log.Printf("Model from request: %s", model)

		// Check if this is a Claude model - use Anthropic Messages API
		if isClaudeModel(model) && strings.HasPrefix(req.URL.Path, "/v1/chat/completions") {
			log.Printf("Model %s is a Claude model - converting to Anthropic Messages API format", model)
			convertChatToAnthropicMessages(req, model)
		}

		// Check if this is a chat completion request for a model that should use Responses API
		if strings.HasPrefix(req.URL.Path, "/v1/chat/completions") && shouldUseResponsesAPI(model) {
			log.Printf("Model %s requires Responses API - converting from chat/completions", model)
			// Convert the chat completion request to a responses request
			convertChatToResponses(req)
		}

		// Handle the token
		HandleToken(req)

		// Convert model to lowercase for case-insensitive matching
		modelLower := strings.ToLower(model)

		// Check if it's a serverless deployment
		if info, ok := ServerlessDeploymentInfo[modelLower]; ok {
			log.Printf("Model %s matched serverless deployment: %s in region %s", model, info.Name, info.Region)
			handleServerlessRequest(req, info, model)
		} else {
			// Resolve the model deployment (handles versioned names automatically)
			deployment := resolveModelDeployment(model)
			log.Printf("Using deployment name: %s for model: %s", deployment, model)
			handleRegularRequest(req, deployment)
		}

		log.Printf("Final proxied URL: %s", req.URL.String())
		log.Printf("=================================")
	}
}

func handleServerlessRequest(req *http.Request, info ServerlessDeployment, model string) {
	req.URL.Scheme = "https"
	req.URL.Host = fmt.Sprintf("%s.%s.models.ai.azure.com", info.Name, info.Region)
	req.Host = req.URL.Host // Preserve query parameters from the original request
	originalQuery := req.URL.Query()
	for key, values := range originalQuery {
		for _, value := range values {
			req.URL.Query().Add(key, value)
		}
	}

	// Set the correct authorization header for serverless
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", info.Key))
	req.Header.Del("api-key")
	log.Printf("Using serverless deployment for %s", model)
}

func handleRegularRequest(req *http.Request, deployment string) {
	remote, _ := url.Parse(AzureOpenAIEndpoint)
	req.URL.Scheme = remote.Scheme
	req.URL.Host = remote.Host
	req.Host = remote.Host

	log.Printf("Setting up regular Azure OpenAI request for deployment: %s", deployment)
	log.Printf("Azure endpoint: %s", AzureOpenAIEndpoint)

	// Handle Responses API endpoints
	if strings.Contains(req.URL.Path, "/v1/responses") {
		// For Responses API, we need to handle the paths differently
		if strings.HasPrefix(req.URL.Path, "/v1/responses") && !strings.Contains(req.URL.Path, "/responses/") {
			// POST /v1/responses - Create response
			req.URL.Path = "/openai/v1/responses"
			log.Printf("Responses API: Using path /openai/v1/responses")
		} else {
			// Other responses endpoints (GET, DELETE, etc.)
			// Convert /v1/responses/{id} to /openai/v1/responses/{id}
			req.URL.Path = strings.Replace(req.URL.Path, "/v1/", "/openai/v1/", 1)
			log.Printf("Responses API: Converted path to %s", req.URL.Path)
		}

		// Use the preview API version for Responses API
		query := req.URL.Query()
		query.Set("api-version", AzureOpenAIResponsesAPIVersion)
		req.URL.RawQuery = query.Encode()
		log.Printf("Responses API: Using API version %s", AzureOpenAIResponsesAPIVersion)
	} else {
		// Existing logic for other endpoints
		var endpointType string
		switch {
		case strings.HasPrefix(req.URL.Path, "/v1/anthropic/messages"):
			// Claude models use Anthropic Messages API
			req.URL.Path = "/anthropic/v1/messages"
			endpointType = "anthropic/messages"
			log.Printf("Claude model detected - using Anthropic Messages API endpoint: %s", req.URL.Path)
		case strings.HasPrefix(req.URL.Path, "/v1/chat/completions"):
			req.URL.Path = path.Join("/openai/deployments", deployment, "chat/completions")
			endpointType = "chat/completions"
		case strings.HasPrefix(req.URL.Path, "/v1/completions"):
			req.URL.Path = path.Join("/openai/deployments", deployment, "completions")
			endpointType = "completions"
		case strings.HasPrefix(req.URL.Path, "/v1/embeddings"):
			req.URL.Path = path.Join("/openai/deployments", deployment, "embeddings")
			endpointType = "embeddings"
		case strings.HasPrefix(req.URL.Path, "/v1/images/generations"):
			req.URL.Path = path.Join("/openai/deployments", deployment, "images/generations")
			endpointType = "images/generations"
		case strings.HasPrefix(req.URL.Path, "/v1/audio/"):
			// Handle audio endpoints
			audioPath := strings.TrimPrefix(req.URL.Path, "/v1/")
			req.URL.Path = path.Join("/openai/deployments", deployment, audioPath)
			endpointType = "audio"
		case strings.HasPrefix(req.URL.Path, "/v1/files"):
			// Files API doesn't use deployment in path
			req.URL.Path = strings.Replace(req.URL.Path, "/v1/", "/openai/", 1)
			endpointType = "files"
		default:
			req.URL.Path = path.Join("/openai/deployments", deployment, strings.TrimPrefix(req.URL.Path, "/v1/"))
			endpointType = "other"
		}
		log.Printf("Endpoint type: %s, Path set to: %s", endpointType, req.URL.Path)

		// Add api-version query parameter for non-Responses API (but not for Anthropic API)
		if endpointType != "anthropic/messages" {
			query := req.URL.Query()
			query.Add("api-version", AzureOpenAIAPIVersion)
			req.URL.RawQuery = query.Encode()
			log.Printf("Using API version: %s", AzureOpenAIAPIVersion)
		} else {
			// For Anthropic Messages API, set the anthropic-version header
			req.Header.Set("anthropic-version", AnthropicAPIVersion)
			log.Printf("Anthropic Messages API: Set anthropic-version header to %s, skipping Azure api-version query parameter", AnthropicAPIVersion)
		}
	}

	// Use the api-key from the original request for regular deployments
	apiKey := req.Header.Get("api-key")
	if apiKey == "" {
		log.Printf("Warning: No api-key found in request headers for deployment: %s", deployment)
	} else {
		// For Anthropic Messages API, convert to Authorization Bearer header
		if strings.Contains(req.URL.Path, "/anthropic/v1/messages") {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			req.Header.Del("api-key")
			log.Printf("Anthropic API: Using Authorization Bearer header for deployment: %s", deployment)
		} else {
			log.Printf("API key found for deployment: %s", deployment)
		}
	}
}

func getModelFromRequest(req *http.Request) string {
	// For Responses API, always check the body first
	if strings.Contains(req.URL.Path, "/responses") && req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(body))

		// The Responses API uses "model" field in the request body
		model := gjson.GetBytes(body, "model").String()
		if model != "" {
			return model
		}
	}

	// Existing logic for path-based model detection
	parts := strings.Split(req.URL.Path, "/")
	for i, part := range parts {
		if part == "deployments" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	// If not found in the path, try to get it from the request body
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		model := gjson.GetBytes(body, "model").String()
		if model != "" {
			return model
		}
	}

	// If still not found, return an empty string
	return ""
}

func sanitizeHeaders(headers http.Header) http.Header {
	sanitized := make(http.Header)
	for key, values := range headers {
		if key == "Authorization" || key == "api-key" {
			sanitized[key] = []string{"[REDACTED]"}
		} else {
			sanitized[key] = values
		}
	}
	return sanitized
}

func modifyResponse(res *http.Response) error {
	// Check if this is a streaming response that needs conversion
	if strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		res.Header.Set("X-Accel-Buffering", "no")
		res.Header.Set("Cache-Control", "no-cache")
		res.Header.Set("Connection", "keep-alive")

		// Check if this needs streaming conversion
		if origPath := res.Request.Header.Get("X-Original-Path"); origPath == "/v1/chat/completions" {
			// Get the model from the request
			model := res.Request.Header.Get("X-Model")
			if model == "" {
				model = "unknown"
			}

			// Create a pipe for the conversion
			pr, pw := io.Pipe()

			// Determine which converter to use based on the endpoint
			if strings.Contains(res.Request.URL.Path, "/anthropic/v1/messages") {
				// Use Anthropic streaming converter
				log.Printf("Using Anthropic streaming converter for model: %s", model)
				go func() {
					defer pw.Close()
					defer res.Body.Close()

					converter := NewAnthropicStreamingConverter(res.Body, pw, model)
					if err := converter.Convert(); err != nil {
						log.Printf("Anthropic streaming conversion error: %v", err)
					}
				}()
			} else {
				// Use Responses API streaming converter
				log.Printf("Using Responses API streaming converter for model: %s", model)
				go func() {
					defer pw.Close()
					defer res.Body.Close()

					converter := NewStreamingResponseConverter(res.Body, pw, model)
					if err := converter.Convert(); err != nil {
						log.Printf("Streaming conversion error: %v", err)
					}
				}()
			}

			// Replace the response body
			res.Body = pr
		}

		return nil
	}

	// Handle non-streaming responses
	if strings.Contains(res.Request.URL.Path, "/openai/v1/responses") && res.StatusCode == 200 {
		// Check if the original request was for chat completions
		if origPath := res.Request.Header.Get("X-Original-Path"); origPath == "/v1/chat/completions" {
			convertResponsesToChatCompletion(res)
		}
	}

	// Handle Anthropic Messages API responses
	if strings.Contains(res.Request.URL.Path, "/anthropic/v1/messages") && res.StatusCode == 200 {
		// Check if the original request was for chat completions
		if origPath := res.Request.Header.Get("X-Original-Path"); origPath == "/v1/chat/completions" {
			convertAnthropicToChatCompletion(res)
		}
	}

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		log.Printf("========== API ERROR ==========")
		log.Printf("Azure API Error Response")
		log.Printf("Status Code: %d", res.StatusCode)
		log.Printf("Request URL: %s", res.Request.URL.String())
		log.Printf("Request Method: %s", res.Request.Method)
		log.Printf("Response Body: %s", string(body))
		log.Printf("Response Headers: %v", res.Header)
		log.Printf("===============================")
		res.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	return nil
}

// Add a function to check if a model is Claude model
func isClaudeModel(model string) bool {
	modelLower := strings.ToLower(model)
	claudePrefixes := []string{
		"claude-opus", "claude-sonnet", "claude-haiku",
		"claude-3", "claude-4",
	}

	for _, prefix := range claudePrefixes {
		if strings.HasPrefix(modelLower, prefix) {
			return true
		}
	}
	return false
}

// Add a function to check if a model should use Responses API
func shouldUseResponsesAPI(model string) bool {
	modelLower := strings.ToLower(model)
	// Models that should use Responses API instead of chat completions
	// These are primarily reasoning models and codex models
	responsesModels := []string{
		// O-series reasoning models
		"o1", "o1-preview", "o1-mini",
		"o3", "o3-mini", "o3-pro", "o3-deep-research",
		"o4", "o4-mini",
		// Codex models (Responses API only)
		"codex-mini",
		"gpt-5.1-codex", "gpt-5-codex",
		// GPT-5 Pro (Responses API only)
		"gpt-5-pro",
		// Computer use preview (Responses API only)
		"computer-use-preview",
	}

	for _, m := range responsesModels {
		if strings.HasPrefix(modelLower, m) {
			return true
		}
	}
	return false
}

// Function to convert chat completion request to responses format
func convertChatToResponses(req *http.Request) {
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)

		log.Printf("Original chat completion request: %s", string(body))

		// Parse the chat completion request
		model := gjson.GetBytes(body, "model").String()
		messages := gjson.GetBytes(body, "messages").Array()
		temperature := gjson.GetBytes(body, "temperature").Float()
		maxTokens := gjson.GetBytes(body, "max_tokens").Int()
		stream := gjson.GetBytes(body, "stream").Bool()

		// Create new request body for Responses API
		newBody := map[string]interface{}{
			"model": model,
		}

		// For simple requests, we can use a string input
		if len(messages) == 1 && messages[0].Get("role").String() == "user" {
			// Use simple string input for single user message
			newBody["input"] = messages[0].Get("content").String()
		} else {
			// Convert messages to input format for Responses API
			var input []map[string]interface{}
			for _, msg := range messages {
				role := msg.Get("role").String()
				content := msg.Get("content").String()

				inputMsg := map[string]interface{}{
					"role": role,
					"content": []map[string]interface{}{
						{
							"type": "input_text",
							"text": content,
						},
					},
				}
				input = append(input, inputMsg)
			}
			newBody["input"] = input
		}

		if temperature > 0 {
			newBody["temperature"] = temperature
		}
		if maxTokens > 0 {
			newBody["max_output_tokens"] = maxTokens
		}
		if stream {
			newBody["stream"] = true
		}

		// Marshal the new body
		newBodyBytes, _ := json.Marshal(newBody)

		log.Printf("Converted to Responses API request: %s", string(newBodyBytes))

		req.Body = io.NopCloser(bytes.NewBuffer(newBodyBytes))
		req.ContentLength = int64(len(newBodyBytes))

		// Update the path to use responses endpoint
		req.URL.Path = "/v1/responses"
		req.Header.Set("X-Original-Path", "/v1/chat/completions")
		req.Header.Set("X-Model", model) // Store model for streaming response
	}
}

// Function to convert chat completion request to Anthropic Messages API format
func convertChatToAnthropicMessages(req *http.Request, model string) {
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)

		log.Printf("Original chat completion request for Claude: %s", string(body))

		// Parse the chat completion request
		messages := gjson.GetBytes(body, "messages").Array()
		temperature := gjson.GetBytes(body, "temperature").Float()
		maxTokens := gjson.GetBytes(body, "max_tokens").Int()
		stream := gjson.GetBytes(body, "stream").Bool()

		// Check if this is a Responses API format (has "input" field instead of "messages")
		input := gjson.GetBytes(body, "input").String()

		// Extract system message if present
		var systemMessage string
		var anthropicMessages []map[string]interface{}

		if input != "" {
			// This is a Responses API format - convert to Anthropic Messages format
			log.Printf("Detected Responses API format with input field, converting to Anthropic Messages format")
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    "user",
				"content": input,
			})
		} else {
			// Standard chat completion format with messages array
			for _, msg := range messages {
				role := msg.Get("role").String()
				content := msg.Get("content").String()

				if role == "system" {
					// Anthropic uses separate system parameter
					systemMessage = content
				} else {
					// Convert user/assistant messages
					anthropicMsg := map[string]interface{}{
						"role":    role,
						"content": content,
					}
					anthropicMessages = append(anthropicMessages, anthropicMsg)
				}
			}
		}

		// Create new request body for Anthropic Messages API
		newBody := map[string]interface{}{
			"model":      model,
			"messages":   anthropicMessages,
			"max_tokens": maxTokens,
		}

		if systemMessage != "" {
			newBody["system"] = systemMessage
		}

		if temperature > 0 {
			newBody["temperature"] = temperature
		}

		if stream {
			newBody["stream"] = true
		}

		// Default max_tokens if not specified
		if maxTokens == 0 {
			newBody["max_tokens"] = 1000
		}

		// Marshal the new body
		newBodyBytes, _ := json.Marshal(newBody)

		log.Printf("Converted to Anthropic Messages API request: %s", string(newBodyBytes))

		req.Body = io.NopCloser(bytes.NewBuffer(newBodyBytes))
		req.ContentLength = int64(len(newBodyBytes))

		// Update the path to use Anthropic Messages API endpoint
		req.URL.Path = "/v1/anthropic/messages"
		req.Header.Set("X-Original-Path", "/v1/chat/completions")
		req.Header.Set("X-Model", model) // Store model for response conversion

		// Set Anthropic-specific headers
		req.Header.Set("anthropic-version", AnthropicAPIVersion)
		log.Printf("Set anthropic-version header: %s", AnthropicAPIVersion)
	}
}

// convert Responses API response to chat completion format
func convertResponsesToChatCompletion(res *http.Response) {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return
	}

	// Log the raw response for debugging
	log.Printf("Raw Responses API response: %s", string(body))

	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		log.Printf("Error unmarshaling response: %v", err)
		res.Body = io.NopCloser(bytes.NewBuffer(body))
		return
	}

	// Check if it's a streaming response
	if strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		// For streaming, we need to handle it differently
		res.Body = io.NopCloser(bytes.NewBuffer(body))
		return
	}

	// Check if there's an error
	if errorData, ok := responseData["error"]; ok && errorData != nil {
		// Return the error as-is
		res.Body = io.NopCloser(bytes.NewBuffer(body))
		return
	}

	// Extract the content - the Responses API has output_text at the root level
	content := ""
	if outputText, ok := responseData["output_text"].(string); ok {
		content = outputText
	} else {
		// Fallback to extracting from output array if output_text is not present
		if outputsRaw, ok := responseData["output"]; ok && outputsRaw != nil {
			outputs, ok := outputsRaw.([]interface{})
			if ok {
				for _, output := range outputs {
					outputMap, ok := output.(map[string]interface{})
					if !ok {
						continue
					}

					if outputMap["type"] == "message" && outputMap["role"] == "assistant" {
						if contentsRaw, ok := outputMap["content"]; ok && contentsRaw != nil {
							contents, ok := contentsRaw.([]interface{})
							if ok {
								for _, c := range contents {
									contentMap, ok := c.(map[string]interface{})
									if !ok {
										continue
									}
									if contentMap["type"] == "output_text" {
										if text, ok := contentMap["text"].(string); ok {
											content = text
											break
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Determine finish reason
	finishReason := "stop"
	if status, ok := responseData["status"].(string); ok && status != "completed" {
		finishReason = status
	}

	// Extract usage data safely
	usage := map[string]interface{}{
		"prompt_tokens":     0,
		"completion_tokens": 0,
		"total_tokens":      0,
	}

	if usageRaw, ok := responseData["usage"]; ok && usageRaw != nil {
		if usageMap, ok := usageRaw.(map[string]interface{}); ok {
			if inputTokens, ok := usageMap["input_tokens"].(float64); ok {
				usage["prompt_tokens"] = int(inputTokens)
			}
			if outputTokens, ok := usageMap["output_tokens"].(float64); ok {
				usage["completion_tokens"] = int(outputTokens)
			}
			if totalTokens, ok := usageMap["total_tokens"].(float64); ok {
				usage["total_tokens"] = int(totalTokens)
			}
		}
	}

	// Get created timestamp, use current time if not present
	created := int64(getFloat64(responseData["created_at"]))
	if created == 0 {
		created = time.Now().Unix()
	}

	// Create chat completion response
	chatResponse := map[string]interface{}{
		"id":      responseData["id"],
		"object":  "chat.completion",
		"created": created,
		"model":   responseData["model"],
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": finishReason,
				"logprobs":      nil,
			},
		},
		"usage":              usage,
		"system_fingerprint": nil,
	}

	// Marshal and set as new body
	newBody, _ := json.Marshal(chatResponse)
	res.Body = io.NopCloser(bytes.NewBuffer(newBody))
	res.ContentLength = int64(len(newBody))
	res.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
}

// convert Anthropic Messages API response to chat completion format
func convertAnthropicToChatCompletion(res *http.Response) {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading Anthropic response body: %v", err)
		return
	}

	// Log the raw response for debugging
	log.Printf("Raw Anthropic Messages API response: %s", string(body))

	var anthropicResponse map[string]interface{}
	if err := json.Unmarshal(body, &anthropicResponse); err != nil {
		log.Printf("Error unmarshaling Anthropic response: %v", err)
		res.Body = io.NopCloser(bytes.NewBuffer(body))
		return
	}

	// Check if there's an error
	if errorData, ok := anthropicResponse["error"]; ok && errorData != nil {
		log.Printf("Error in Anthropic response, passing through: %v", errorData)
		res.Body = io.NopCloser(bytes.NewBuffer(body))
		return
	}

	// Get model from request header
	model := res.Request.Header.Get("X-Model")
	if model == "" {
		model = "claude-unknown"
	}

	// Extract content from Anthropic response
	var content string
	if contentArray, ok := anthropicResponse["content"].([]interface{}); ok && len(contentArray) > 0 {
		if contentBlock, ok := contentArray[0].(map[string]interface{}); ok {
			if text, ok := contentBlock["text"].(string); ok {
				content = text
			}
		}
	}

	// Extract usage information
	usage := map[string]interface{}{
		"prompt_tokens":     0,
		"completion_tokens": 0,
		"total_tokens":      0,
	}
	if usageData, ok := anthropicResponse["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := usageData["input_tokens"]; ok {
			usage["prompt_tokens"] = promptTokens
		}
		if completionTokens, ok := usageData["output_tokens"]; ok {
			usage["completion_tokens"] = completionTokens
		}
		// Calculate total
		promptInt := getInt64(usage["prompt_tokens"])
		completionInt := getInt64(usage["completion_tokens"])
		usage["total_tokens"] = promptInt + completionInt
	}

	// Get stop reason and map to OpenAI finish_reason
	finishReason := "stop"
	if stopReason, ok := anthropicResponse["stop_reason"].(string); ok {
		switch stopReason {
		case "end_turn":
			finishReason = "stop"
		case "max_tokens":
			finishReason = "length"
		case "stop_sequence":
			finishReason = "stop"
		default:
			finishReason = "stop"
		}
	}

	// Get current Unix timestamp for created field
	created := time.Now().Unix()

	// Create OpenAI chat completion format response
	chatResponse := map[string]interface{}{
		"id":      anthropicResponse["id"],
		"object":  "chat.completion",
		"created": created,
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": finishReason,
				"logprobs":      nil,
			},
		},
		"usage":              usage,
		"system_fingerprint": nil,
	}

	// Marshal and set as new body
	newBody, _ := json.Marshal(chatResponse)
	log.Printf("Converted Anthropic response to OpenAI format: %s", string(newBody))

	res.Body = io.NopCloser(bytes.NewBuffer(newBody))
	res.ContentLength = int64(len(newBody))
	res.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
}

// Helper function to safely get int64
func getInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}

// Helper function to safely get float64
func getFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}
