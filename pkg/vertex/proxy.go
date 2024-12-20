package vertex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	VertexAIProjectID   = ""
	VertexAIEndpoint    = "us-central1-aiplatform.googleapis.com"
	VertexAIAPIVersion  = "v1"
	VertexAILocation    = "us-central1"
	VertexAIModelMapper = map[string]string{
		"chat-bison":                   "chat-bison@002",
		"text-bison":                   "text-bison@002",
		"embedding-gecko":              "textembedding-gecko@003",
		"embedding-gecko-multilingual": "textembedding-gecko-multilingual@003",
	}
)

type VertexAIConfig struct {
	ProjectID   string
	Endpoint    string
	APIVersion  string
	Location    string
	ModelMapper map[string]string
}

func Init(projectID string) {
	VertexAIProjectID = projectID
	log.Printf("Vertex AI initialized with Project ID: %s", projectID)
}

func HandleVertexAIProxy(c *gin.Context) {
	if VertexAIProjectID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Vertex AI Project ID not set"})
		return
	}

	ctx := context.Background()

	// Use the GOOGLE_APPLICATION_CREDENTIALS environment variable to set the credentials
	creds := option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	client, err := genai.NewClient(ctx, creds)
	if err != nil {
		log.Printf("Error creating Vertex AI client: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Vertex AI client"})
		return
	}
	defer client.Close()

	modelName := getModelFromRequestBody(c.Request)
	if mappedModel, ok := VertexAIModelMapper[strings.ToLower(modelName)]; ok {
		modelName = mappedModel
	}

	model := client.GenerativeModel(modelName)

	// Handle chat/completions
	if strings.HasSuffix(c.Request.URL.Path, "/chat/completions") {
		handleChatCompletion(c, model)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid endpoint for Vertex AI"})
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

// Helper function to convert a single response to OpenAI format (for non-streaming)
func convertToOpenAIResponse(resp *genai.GenerateContentResponse) map[string]interface{} {
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
	}
}

type Model struct {
	ID              string       `json:"id"`
	Object          string       `json:"object"`
	CreatedAt       int64        `json:"created_at"`
	Capabilities    Capabilities `json:"capabilities"`
	LifecycleStatus string       `json:"lifecycle_status"`
	Status          string       `json:"status"`
	Deprecation     Deprecation  `json:"deprecation"`
	FineTune        string       `json:"fine_tune,omitempty"`
	Name            string       `json:"name"`
	Description     string       `json:"description"`
}

// Capabilities represents the capabilities of a Vertex AI model.
type Capabilities struct {
	FineTune       bool `json:"fine_tune"`
	Inference      bool `json:"inference"`
	Completion     bool `json:"completion"`
	ChatCompletion bool `json:"chat_completion"`
	Embeddings     bool `json:"embeddings"`
}

// Deprecation represents the deprecation status of a Vertex AI model.
type Deprecation struct {
	FineTune  int64 `json:"fine_tune,omitempty"`
	Inference int64 `json:"inference,omitempty"`
}

func FetchVertexAIModels() ([]Model, error) {
	if VertexAIProjectID == "" {
		return nil, fmt.Errorf("Vertex AI Project ID not set")
	}

	ctx := context.Background()
	creds := option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	client, err := genai.NewClient(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %v", err)
	}
	defer client.Close()

	url := fmt.Sprintf("https://%s/%s/projects/%s/locations/%s/publishers/google/models", VertexAIEndpoint, VertexAIAPIVersion, VertexAIProjectID, VertexAILocation)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch Vertex AI models: %s", string(body))
	}

	var vertexModels struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vertexModels); err != nil {
		return nil, err
	}

	var models []Model
	for _, m := range vertexModels.Models {
		parts := strings.Split(m.Name, "/")
		modelID := parts[len(parts)-1]

		models = append(models, Model{
			ID:              modelID,
			Object:          "model",
			Name:            m.Name,
			Description:     m.Description,
			LifecycleStatus: "active", // You might need to adjust this based on actual Vertex AI model data
			Status:          "ready",  // You might need to adjust this based on actual Vertex AI model data
			Capabilities: Capabilities{
				Completion:     true,
				ChatCompletion: strings.Contains(modelID, "chat"),
				Embeddings:     strings.Contains(modelID, "embedding"),
			},
		})
	}

	return models, nil
}
