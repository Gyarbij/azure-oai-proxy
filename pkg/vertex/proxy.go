package vertex

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"strings"
)

var (
	VertexAIProjectID   = ""
	VertexAIEndpoint    = "us-central1-aiplatform.googleapis.com"
	VertexAIAPIVersion  = "v1"
	VertexAILocation    = "us-central1"
	VertexAIModelMapper = map[string]string{
		"chat-bison":                   "chat-bison@001",
		"text-bison":                   "text-bison@001",
		"embedding-gecko":              "textembedding-gecko@001",
		"embedding-gecko-multilingual": "textembedding-gecko-multilingual@001",
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

func NewVertexAIReverseProxy() *httputil.ReverseProxy {
	config := &VertexAIConfig{
		ProjectID:   VertexAIProjectID,
		Endpoint:    VertexAIEndpoint,
		APIVersion:  VertexAIAPIVersion,
		Location:    VertexAILocation,
		ModelMapper: VertexAIModelMapper,
	}

	return newVertexAIReverseProxy(config)
}

func newVertexAIReverseProxy(config *VertexAIConfig) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		originalURL := req.URL.String()
		model := getModelFromRequest(req)

		// Map the model name if necessary
		if mappedModel, ok := config.ModelMapper[strings.ToLower(model)]; ok {
			model = mappedModel
		}

		// Construct the new URL
		targetURL := fmt.Sprintf("https://%s/%s/projects/%s/locations/%s/publishers/google/models/%s:predict", config.Endpoint, config.APIVersion, config.ProjectID, config.Location, model)
		target, err := url.Parse(targetURL)
		if err != nil {
			log.Printf("Error parsing target URL: %v", err)
			return
		}

		// Set the target
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path

		// Set Authorization header using Google Application Default Credentials (ADC)
		token, err := getAccessToken()
		if err != nil {
			log.Printf("Error getting access token: %v", err)
			return
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

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

func getAccessToken() (string, error) {
	// Use Application Default Credentials (ADC) to get an access token
	// Ensure that your environment is set up with ADC, e.g., by running:
	// gcloud auth application-default login
	// Or by setting the GOOGLE_APPLICATION_CREDENTIALS environment variable
	output, err := exec.Command("gcloud", "auth", "print-access-token").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
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

	token, err := getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	url := fmt.Sprintf("https://%s/%s/projects/%s/locations/%s/publishers/google/models", VertexAIEndpoint, VertexAIAPIVersion, VertexAIProjectID, VertexAILocation)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
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
			// Add other relevant fields if needed
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vertexModels); err != nil {
		return nil, err
	}

	var models []Model
	for _, m := range vertexModels.Models {
		// Extract model ID from the name field (e.g., "publishers/google/models/chat-bison")
		parts := strings.Split(m.Name, "/")
		modelID := parts[len(parts)-1]

		models = append(models, Model{
			ID:     modelID,
			Object: "model",
			Capabilities: Capabilities{
				Completion:     true,
				ChatCompletion: strings.Contains(modelID, "chat"),
				Embeddings:     strings.Contains(modelID, "embedding"),
			},
			LifecycleStatus: "active",
			Status:          "ready",
			Name:            m.Name,
			Description:     m.Description,
		})
	}

	return models, nil
}
