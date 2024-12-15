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
	"strings"

	"github.com/tidwall/gjson"
)

var (
	AzureOpenAIAPIVersion    = "2024-06-01"
	AzureOpenAIEndpoint      = ""
	ServerlessDeploymentInfo = make(map[string]ServerlessDeployment)
	AzureOpenAIModelMapper   = make(map[string]string)
	RealTimeModels           = make(map[string]bool)
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
	if v := os.Getenv("AZURE_OPENAI_ENDPOINT"); v != "" {
		AzureOpenAIEndpoint = v
	}

	// Initialize serverless deployments
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

	// Initialize AzureOpenAIModelMapper (you might want to load this from an environment variable or config file)
	AzureOpenAIModelMapper = map[string]string{
		"gpt-3.5-turbo":               "gpt-35-turbo",
		"gpt-3.5-turbo-0125":          "gpt-35-turbo-0125",
		"gpt-3.5-turbo-0613":          "gpt-35-turbo-0613",
		"gpt-3.5-turbo-1106":          "gpt-35-turbo-1106",
		"gpt-3.5-turbo-16k-0613":      "gpt-35-turbo-16k-0613",
		"gpt-3.5-turbo-instruct-0914": "gpt-35-turbo-instruct-0914",
		"gpt-4":                       "gpt-4-0613",
		"gpt-4-32k":                   "gpt-4-32k",
		"gpt-4-32k-0613":              "gpt-4-32k-0613",
		"gpt-4o":                      "gpt-4o",
		"gpt-4o-mini":                 "gpt-4o-mini",
		"gpt-4o-2024-05-13":           "gpt-4o-2024-05-13",
		"gpt-4-turbo":                 "gpt-4-turbo",
		"gpt-4-vision-preview":        "gpt-4-vision-preview",
		"gpt-4-turbo-2024-04-09":      "gpt-4-turbo-2024-04-09",
		"gpt-4-1106-preview":          "gpt-4-1106-preview",
		"text-embedding-ada-002":      "text-embedding-ada-002",
		"dall-e-2":                    "dall-e-2",
		"dall-e-3":                    "dall-e-3",
		"babbage-002":                 "babbage-002",
		"davinci-002":                 "davinci-002",
		"whisper-1":                   "whisper",
		"tts-1":                       "tts",
		"tts-1-hd":                    "tts-hd",
		"text-embedding-3-small":      "text-embedding-3-small-1",
		"text-embedding-3-large":      "text-embedding-3-large-1",
		"gpt-4o-realtime":             "gpt-4o-realtime-preview",
		"gpt-4o-realtime-preview":     "gpt-4o-realtime-preview",
	}

	// Initialize realtime models set
	RealTimeModels = map[string]bool{
		"gpt-4o-realtime":         true,
		"gpt-4o-realtime-preview": true,
	}

	log.Printf("Loaded ServerlessDeploymentInfo: %+v", ServerlessDeploymentInfo)
	log.Printf("Azure OpenAI Endpoint: %s", AzureOpenAIEndpoint)
	log.Printf("Azure OpenAI API Version: %s", AzureOpenAIAPIVersion)
	log.Printf("Initialized realtime models: %v", RealTimeModels)
}

func NewOpenAIReverseProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director:       makeDirector(),
		ModifyResponse: modifyResponse,
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
		log.Printf("Original request URL: %s for model: %s", originURL, model)

		// Handle the token
		HandleToken(req)

		// Convert model to lowercase for case-insensitive matching
		modelLower := strings.ToLower(model)

		// Check if it's a serverless deployment
		if info, ok := ServerlessDeploymentInfo[modelLower]; ok {
			handleServerlessRequest(req, info, model)
		} else if azureModel, ok := AzureOpenAIModelMapper[modelLower]; ok {
			handleRegularRequest(req, azureModel)
		} else {
			log.Printf("Warning: Unknown model %s, treating as regular Azure OpenAI deployment", model)
			handleRegularRequest(req, model)
		}

		log.Printf("Proxying request [%s] %s -> %s", model, originURL, req.URL.String())
		// log.Printf("Final request headers: %v", sanitizeHeaders(req.Header))
	}
}

func handleRegularRequest(req *http.Request, deployment string) {
	// Check for realtime models first
	modelLower := strings.ToLower(deployment)
	if _, isRealtime := RealTimeModels[modelLower]; isRealtime {
		if strings.HasPrefix(req.URL.Path, "/v1/chat/completions") {
			// Return a helpful error message for chat completions with realtime models
			w := req.Response.Writer
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			errorResp := map[string]interface{}{
				"error": map[string]interface{}{
					"message": fmt.Sprintf("Model %s requires using the /realtime endpoint. For regular chat, please use a standard model like gpt-4-turbo or gpt-4-1106-preview.", deployment),
					"type":    "invalid_request_error",
					"param":   "model",
					"code":    "model_not_supported_for_completion",
				},
			}
			json.NewEncoder(w).Encode(errorResp)
			return
		}

		// Check if this is a voice/audio request that should use realtime
		if isVoiceRequest(req) {
			handleRealtimeRedirect(req)
			return
		}
	}

	// Regular request handling
	remote, _ := url.Parse(AzureOpenAIEndpoint)
	req.URL.Scheme = remote.Scheme
	req.URL.Host = remote.Host
	req.Host = remote.Host

	// Handle different endpoints
	switch {
	case strings.HasPrefix(req.URL.Path, "/v1/chat/completions"):
		req.URL.Path = path.Join("/openai/deployments", deployment, "chat/completions")
	case strings.HasPrefix(req.URL.Path, "/v1/completions"):
		req.URL.Path = path.Join("/openai/deployments", deployment, "completions")
	case strings.HasPrefix(req.URL.Path, "/v1/audio/speech"):
		req.URL.Path = path.Join("/openai/deployments", deployment, "audio/speech")
	case strings.HasPrefix(req.URL.Path, "/v1/audio/transcriptions"):
		req.URL.Path = path.Join("/openai/deployments", deployment, "audio/transcriptions")
	default:
		req.URL.Path = path.Join("/openai/deployments", deployment, strings.TrimPrefix(req.URL.Path, "/v1/"))
	}

	// Add api-version query parameter
	query := req.URL.Query()
	query.Add("api-version", AzureOpenAIAPIVersion)
	req.URL.RawQuery = query.Encode()

	log.Printf("Modified request URL: %s", req.URL.String())
}

func isVoiceRequest(req *http.Request) bool {
	return strings.Contains(req.URL.Path, "/audio/speech") ||
		strings.Contains(req.URL.Path, "/audio/transcriptions")
}

// handleRealtimeRedirect modifies the request to use the realtime endpoint
func handleRealtimeRedirect(req *http.Request) {
	req.URL.Path = "/openai/realtime"
	query := req.URL.Query()
	query.Add("api-version", "2024-10-01-preview")
	req.URL.RawQuery = query.Encode()
}

func getModelFromRequest(req *http.Request) string {
	// First, try to get the model from the URL path
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
	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		log.Printf("Azure API Error Response: Status: %d, Body: %s", res.StatusCode, string(body))
		res.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	if res.Header.Get("Content-Type") == "text/event-stream" {
		res.Header.Set("X-Accel-Buffering", "no")
	}

	return nil
}
