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
	"google.golang.org/api/iterator"
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
	} else if c.Request.URL.Path == "/v1/models" {
		// Handle model listing
		models, err := FetchGoogleAIModels()
		if err != nil {
			log.Printf("Error fetching Google AI models: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Google AI models"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": models})
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
		Stream      *bool    `json:"stream,omitempty"`
		Temperature *float64 `json:"temperature,omitempty"`
		TopP        *float64 `json:"top_p,omitempty"`
		TopK        *int     `json:"top_k,omitempty"`
		// Add other parameters as needed
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

	// Set advanced parameters if provided
	if req.Temperature != nil {
		model.SetTemperature(float32(*req.Temperature))
	}
	if req.TopP != nil {
		model.SetTopP(float32(*req.TopP))
	}
	if req.TopK != nil {
		model.SetTopK(int32(*req.TopK))
	}
	// Set other parameters as needed

	// Handle streaming if requested
	if req.Stream != nil && *req.Stream {
		iter := cs.SendMessageStream(context.Background(), genai.Text(req.Messages[len(req.Messages)-1].Content))
		c.Stream(func(w io.Writer) bool {
			resp, err := iter.Next()
			if err == iterator.Done {
				return false
			}
			if err != nil {
				log.Printf("Error generating content: %v", err)
				c.SSEvent("error", "Failed to generate content")
				return false
			}

			// Convert each response to OpenAI format and send as SSE
			openaiResp := convertToOpenAIResponseStream(resp)
			c.SSEvent("message", openaiResp)
			return true
		})
	} else {
		// Use SendMessage for a single response
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
}

// Helper function to convert a single response to OpenAI format (for streaming)
func convertToOpenAIResponseStream(resp *genai.GenerateContentResponse) map[string]interface{} {
	var parts []string
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			parts = append(parts, fmt.Sprintf("%v", part))
		}
	}

	return map[string]interface{}{
		"object": "chat.completion.chunk",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"role":    "assistant",
					"content": strings.Join(parts, ""),
				},
				"finish_reason": "stop",
			},
		},
	}
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
	CreatedAt                  int64        `json:"created_at,omitempty"`
	Capabilities               Capabilities `json:"capabilities,omitempty"`
	LifecycleStatus            string       `json:"lifecycle_status,omitempty"`
	Status                     string       `json:"status,omitempty"`
	Deprecation                Deprecation  `json:"deprecation,omitempty"`
	FineTune                   string       `json:"fine_tune,omitempty"`
	Name                       string       `json:"name,omitempty"`
	Version                    string       `json:"version,omitempty"`
	Description                string       `json:"description,omitempty"`
	InputTokenLimit            int          `json:"input_token_limit,omitempty"`
	OutputTokenLimit           int          `json:"output_token_limit,omitempty"`
	SupportedGenerationMethods []string     `json:"supported_generation_methods,omitempty"`
	Temperature                float64      `json:"temperature,omitempty"`
	TopP                       float64      `json:"top_p,omitempty"`
	TopK                       int          `json:"top_k,omitempty"`
	IsExperimental             bool         `json:"is_experimental,omitempty"`
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

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(GoogleAIAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google AI client: %v", err)
	}
	defer client.Close()

	var models []Model
	// Fetch regular models
	regIter := client.ListModels(ctx)
	for {
		m, err := regIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list regular Google AI models: %v", err)
		}

		modelID := strings.TrimPrefix(m.Name, "models/")

		models = append(models, Model{
			ID:                         modelID,
			Object:                     "model",
			Name:                       m.DisplayName,
			Version:                    m.Version,
			Description:                m.Description,
			InputTokenLimit:            int(m.InputTokenLimit),
			OutputTokenLimit:           int(m.OutputTokenLimit),
			SupportedGenerationMethods: m.SupportedGenerationMethods,
			Temperature:                0.0, // Default or set based on your needs
			TopP:                       0.0, // Default or set based on your needs
			TopK:                       0,   // Default or set based on your needs
			Capabilities: Capabilities{
				Completion:     true,
				ChatCompletion: true,
				Embeddings:     strings.Contains(modelID, "embedding"),
			},
			LifecycleStatus: "active", // You may need to adjust this based on the actual model status
			Status:          "ready",  // You may need to adjust this based on the actual model status
			IsExperimental:  false,
		})
	}

	// Fetch experimental models
	expIter := client.ListModels(ctx)
	for {
		m, err := expIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list experimental Google AI models: %v", err)
		}

		modelID := strings.TrimPrefix(m.Name, "models/")

		models = append(models, Model{
			ID:                         modelID,
			Object:                     "model",
			Name:                       m.DisplayName,
			Version:                    m.Version,
			Description:                m.Description,
			InputTokenLimit:            int(m.InputTokenLimit),
			OutputTokenLimit:           int(m.OutputTokenLimit),
			SupportedGenerationMethods: m.SupportedGenerationMethods,
			Temperature:                0.0, // Default or set based on your needs
			TopP:                       0.0, // Default or set based on your needs
			TopK:                       0,   // Default or set based on your needs
			Capabilities: Capabilities{
				Completion:     true,
				ChatCompletion: true,
				Embeddings:     strings.Contains(modelID, "embedding"),
			},
			LifecycleStatus: "experimental", // You may need to adjust this based on the actual model status
			Status:          "ready",        // You may need to adjust this based on the actual model status
			IsExperimental:  true,
		})
	}

	return models, nil
}
