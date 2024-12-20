package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
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

func HandleGoogleAIProxy(c *gin.Context) {
	if GoogleAIAPIKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google AI Studio API key not set"})
		return
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(GoogleAIAPIKey))
	if err != nil {
		log.Printf("Error creating Google AI client: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Google AI client"})
		return
	}
	defer client.Close()

	modelName := getModelFromRequestBody(c.Request)
	if mappedModel, ok := GoogleAIModelMap[strings.ToLower(modelName)]; ok {
		modelName = mappedModel
	}

	model := client.GenerativeModel(modelName)

	// Handle chat/completions
	if strings.HasSuffix(c.Request.URL.Path, "/chat/completions") {
		handleChatCompletion(c, model)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid endpoint for Google AI Studio"})
	}
}

func getModelFromRequestBody(req *http.Request) string {
	body, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(strings.NewReader(string(body))) // Restore the body
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		if model, ok := data["model"].(string); ok {
			return model
		}
	}
	return ""
}

func handleChatCompletion(c *gin.Context, model *genai.GenerativeModel) {
	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	cs := model.StartChat()
	cs.History = []*genai.Content{}

	for _, msg := range req.Messages {
		cs.History = append(cs.History, &genai.Content{
			Parts: []genai.Part{
				genai.Text(msg.Content),
			},
			Role: msg.Role,
		})
	}

	// Use SendMessage for a single response, or SendMessageStream for streaming responses
	resp, err := cs.SendMessage(context.Background(), genai.Text(req.Messages[len(req.Messages)-1].Content))
	if err != nil {
		log.Printf("Error generating content: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate content"})
		return
	}

	// Convert the response to OpenAI format
	openaiResp := convertToOpenAIResponse(resp)
	c.JSON(http.StatusOK, openaiResp)
}

// Helper function to convert Google AI response to OpenAI format
func convertToOpenAIResponse(resp *genai.GenerateContentResponse) map[string]interface{} {
	// This is a simplified conversion. You'll need to adjust it based on your needs.
	var choices []map[string]interface{}
	for _, candidate := range resp.Candidates {
		choices = append(choices, map[string]interface{}{
			"index": candidate.Index,
			"message": map[string]interface{}{
				"role":    "model",
				"content": fmt.Sprintf("%v", candidate.Content.Parts),
			},
		})
	}

	return map[string]interface{}{
		"object":  "chat.completion",
		"choices": choices,
		// Add other fields like usage, model, etc. if needed
	}
}

type Model struct {
	ID                         string       `json:"id"`
	Object                     string       `json:"object"`
	CreatedAt                  int64        `json:"created_at"`
	Capabilities               Capabilities `json:"capabilities"`
	LifecycleStatus            string       `json:"lifecycle_status"`
	Status                     string       `json:"status"`
	Deprecation                Deprecation  `json:"deprecation"`
	FineTune                   string       `json:"fine_tune,omitempty"`
	Name                       string       `json:"name"`
	Version                    string       `json:"version"`
	Description                string       `json:"description"`
	InputTokenLimit            int          `json:"inputTokenLimit"`
	OutputTokenLimit           int          `json:"outputTokenLimit"`
	SupportedGenerationMethods []string     `json:"supportedGenerationMethods"`
	Temperature                float64      `json:"temperature,omitempty"`
	TopP                       float64      `json:"topP,omitempty"`
	TopK                       int          `json:"topK,omitempty"`
}

// Capabilities represents the capabilities of a Google AI model.
type Capabilities struct {
	FineTune       bool `json:"fine_tune"`
	Inference      bool `json:"inference"`
	Completion     bool `json:"completion"`
	ChatCompletion bool `json:"chat_completion"`
	Embeddings     bool `json:"embeddings"`
}

// Deprecation represents the deprecation status of a Google AI model.
type Deprecation struct {
	FineTune  int64 `json:"fine_tune,omitempty"`
	Inference int64 `json:"inference,omitempty"`
}

func FetchGoogleAIModels() ([]Model, error) {
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

	var models []Model
	for _, m := range googleModels.Models {
		// Extract model ID from the name field (e.g., "models/gemini-pro")
		modelID := strings.TrimPrefix(m.Name, "models/")

		models = append(models, Model{
			ID:                         modelID,
			Object:                     "model",
			Name:                       m.Name,
			Version:                    m.Version,
			Description:                m.Description,
			InputTokenLimit:            m.InputTokenLimit,
			OutputTokenLimit:           m.OutputTokenLimit,
			SupportedGenerationMethods: m.SupportedGenerationMethods,
			Temperature:                m.Temperature,
			TopP:                       m.TopP,
			TopK:                       m.TopK,
			Capabilities: Capabilities{
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
