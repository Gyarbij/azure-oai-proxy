package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gyarbij/azure-oai-proxy/pkg/azure"
	"github.com/gyarbij/azure-oai-proxy/pkg/google"
	"github.com/gyarbij/azure-oai-proxy/pkg/openai"
	"github.com/gyarbij/azure-oai-proxy/pkg/vertex"
	"github.com/joho/godotenv"
)

var (
	Address   = "0.0.0.0:11437"
	ProxyMode = "azure"
)

// Define the ModelList and Model types based on the API documentation
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
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
	Name                       string       `json:"name"`                  // Add Name field
	Version                    string       `json:"version,omitempty"`     // Add Version field
	Description                string       `json:"description,omitempty"` // Add Description field
	InputTokenLimit            int          `json:"input_token_limit,omitempty"`
	OutputTokenLimit           int          `json:"output_token_limit,omitempty"`
	SupportedGenerationMethods []string     `json:"supported_generation_methods,omitempty"`
	Temperature                float64      `json:"temperature,omitempty"`
	TopP                       float64      `json:"top_p,omitempty"`
	TopK                       int          `json:"top_k,omitempty"`
}

type Capabilities struct {
	FineTune       bool `json:"fine_tune"`
	Inference      bool `json:"inference"`
	Completion     bool `json:"completion"`
	ChatCompletion bool `json:"chat_completion"`
	Embeddings     bool `json:"embeddings"`
}

type Deprecation struct {
	FineTune  int64 `json:"fine_tune,omitempty"`
	Inference int64 `json:"inference"`
}

func init() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	gin.SetMode(gin.ReleaseMode)
	if v := os.Getenv("AZURE_OPENAI_PROXY_ADDRESS"); v != "" {
		Address = v
	}
	if v := os.Getenv("AZURE_OPENAI_PROXY_MODE"); v != "" {
		ProxyMode = v
	}
	log.Printf("loading azure openai proxy address: %s", Address)
	log.Printf("loading azure openai proxy mode: %s", ProxyMode)

	// Load Azure OpenAI Model Mapper
	if v := os.Getenv("AZURE_OPENAI_MODEL_MAPPER"); v != "" {
		for _, pair := range strings.Split(v, ",") {
			info := strings.Split(pair, "=")
			if len(info) == 2 {
				azure.AzureOpenAIModelMapper[info[0]] = info[1]
			}
		}
	}

	// Initialize Google AI Studio
	if v := os.Getenv("GOOGLE_AI_STUDIO_API_KEY"); v != "" {
		google.Init(v)
	}

	// Initialize Vertex AI
	if v := os.Getenv("VERTEX_AI_PROJECT_ID"); v != "" {
		vertex.Init(v)
	}
}

func main() {
	router := gin.Default()

	// Health check endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// Proxy routes
	router.OPTIONS("/v1/*path", handleOptions)
	router.GET("/v1/models", handleGetModels)
	router.POST("/v1/chat/completions", handleProxy)
	router.POST("/v1/completions", handleProxy)
	router.POST("/v1/embeddings", handleProxy)
	router.POST("/v1/images/generations", handleProxy)
	router.POST("/v1/audio/speech", handleProxy)
	router.GET("/v1/audio/voices", handleProxy)
	router.POST("/v1/audio/transcriptions", handleProxy)
	router.POST("/v1/audio/translations", handleProxy)
	router.POST("/v1/fine_tunes", handleProxy)
	router.GET("/v1/fine_tunes", handleProxy)
	router.GET("/v1/fine_tunes/:fine_tune_id", handleProxy)
	router.POST("/v1/fine_tunes/:fine_tune_id/cancel", handleProxy)
	router.GET("/v1/fine_tunes/:fine_tune_id/events", handleProxy)
	router.POST("/v1/files", handleProxy)
	router.GET("/v1/files", handleProxy)
	router.DELETE("/v1/files/:file_id", handleProxy)
	router.GET("/v1/files/:file_id", handleProxy)
	router.GET("/v1/files/:file_id/content", handleProxy)
	router.GET("/deployments", handleProxy)                      // Azure-specific
	router.GET("/deployments/:deployment_id", handleProxy)       // Azure-specific
	router.GET("/v1/models/:model_id/capabilities", handleProxy) // Azure-specific

	router.Run(Address)
}

func handleGetModels(c *gin.Context) {
	var allModels []Model

	// Always fetch Azure models if AZURE_OPENAI_ENDPOINT is set
	if os.Getenv("AZURE_OPENAI_ENDPOINT") != "" {
		azureModels, err := fetchAzureModels(c.Request)
		if err != nil {
			log.Printf("error fetching Azure models: %v", err)
		} else {
			for _, am := range azureModels {
				allModels = append(allModels, Model{
					ID:              am.ID,
					Object:          am.Object,
					CreatedAt:       am.CreatedAt,
					Capabilities:    mapAzureCapabilities(am.Capabilities),
					LifecycleStatus: am.LifecycleStatus,
					Status:          am.Status,
					Deprecation:     mapAzureDeprecation(am.Deprecation),
					FineTune:        am.FineTune,
					Name:            am.ID, // Use ID as Name for Azure models
				})
			}
		}
	}

	// Fetch models from other services based on ProxyMode or specific environment variables
	switch ProxyMode {
	case "google":
		if googleModels, err := google.FetchGoogleAIModels(); err != nil {
			log.Printf("error fetching Google AI Studio models: %v", err)
		} else {
			for _, gm := range googleModels {
				allModels = append(allModels, Model{
					ID:                         gm.ID,
					Object:                     gm.Object,
					CreatedAt:                  gm.CreatedAt,
					Capabilities:               mapGoogleCapabilities(gm.Capabilities),
					LifecycleStatus:            gm.LifecycleStatus,
					Status:                     gm.Status,
					Deprecation:                mapGoogleDeprecation(gm.Deprecation),
					FineTune:                   gm.FineTune,
					Name:                       gm.Name,
					Version:                    gm.Version,
					Description:                gm.Description,
					InputTokenLimit:            gm.InputTokenLimit,
					OutputTokenLimit:           gm.OutputTokenLimit,
					SupportedGenerationMethods: gm.SupportedGenerationMethods,
					Temperature:                gm.Temperature,
					TopP:                       gm.TopP,
					TopK:                       gm.TopK,
				})
			}
		}
	case "vertex":
		if vertexModels, err := vertex.FetchVertexAIModels(); err != nil {
			log.Printf("error fetching Vertex AI models: %v", err)
		} else {
			for _, vm := range vertexModels {
				allModels = append(allModels, Model{
					ID:              vm.ID,
					Object:          vm.Object,
					CreatedAt:       vm.CreatedAt,
					Capabilities:    mapVertexCapabilities(vm.Capabilities),
					LifecycleStatus: vm.LifecycleStatus,
					Status:          vm.Status,
					Deprecation:     mapVertexDeprecation(vm.Deprecation),
					FineTune:        vm.FineTune,
					Name:            vm.Name,
					Description:     vm.Description,
				})
			}
		}
	default: // If ProxyMode is "azure" or not set, we've already fetched Azure models
		if os.Getenv("GOOGLE_AI_STUDIO_API_KEY") != "" {
			if googleModels, err := google.FetchGoogleAIModels(); err != nil {
				log.Printf("error fetching Google AI Studio models: %v", err)
			} else {
				for _, gm := range googleModels {
					allModels = append(allModels, Model{
						ID:                         gm.ID,
						Object:                     gm.Object,
						CreatedAt:                  gm.CreatedAt,
						Capabilities:               mapGoogleCapabilities(gm.Capabilities),
						LifecycleStatus:            gm.LifecycleStatus,
						Status:                     gm.Status,
						Deprecation:                mapGoogleDeprecation(gm.Deprecation),
						FineTune:                   gm.FineTune,
						Name:                       gm.Name,
						Version:                    gm.Version,
						Description:                gm.Description,
						InputTokenLimit:            gm.InputTokenLimit,
						OutputTokenLimit:           gm.OutputTokenLimit,
						SupportedGenerationMethods: gm.SupportedGenerationMethods,
						Temperature:                gm.Temperature,
						TopP:                       gm.TopP,
						TopK:                       gm.TopK,
					})
				}
			}
		}
		if os.Getenv("VERTEX_AI_PROJECT_ID") != "" {
			if vertexModels, err := vertex.FetchVertexAIModels(); err != nil {
				log.Printf("error fetching Vertex AI models: %v", err)
			} else {
				for _, vm := range vertexModels {
					allModels = append(allModels, Model{
						ID:              vm.ID,
						Object:          vm.Object,
						CreatedAt:       vm.CreatedAt,
						Capabilities:    mapVertexCapabilities(vm.Capabilities),
						LifecycleStatus: vm.LifecycleStatus,
						Status:          vm.Status,
						Deprecation:     mapVertexDeprecation(vm.Deprecation),
						FineTune:        vm.FineTune,
						Name:            vm.Name,
						Description:     vm.Description,
					})
				}
			}
		}
	}

	result := ModelList{
		Object: "list",
		Data:   allModels,
	}
	c.JSON(http.StatusOK, result)
}

func fetchAzureModels(originalReq *http.Request) ([]azure.Model, error) {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	if endpoint == "" {
		endpoint = azure.AzureOpenAIEndpoint
	}

	url := fmt.Sprintf("%s/openai/models?api-version=%s", endpoint, azure.AzureOpenAIAPIVersion)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Preserve original Authorization header for Azure
	req.Header.Set("Authorization", originalReq.Header.Get("Authorization"))

	azure.HandleToken(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch deployed models from Azure: %s", string(body))
	}

	var deployedModelsResponse azure.ModelList
	if err := json.NewDecoder(resp.Body).Decode(&deployedModelsResponse); err != nil {
		return nil, err
	}

	// Add serverless deployments to the models list for Azure
	for deploymentName := range azure.ServerlessDeploymentInfo {
		deployedModelsResponse.Data = append(deployedModelsResponse.Data, azure.Model{
			ID:     deploymentName,
			Object: "model",
			Capabilities: azure.Capabilities{
				Completion:     true,
				ChatCompletion: true,
				Inference:      true,
			},
			LifecycleStatus: "active",
			Status:          "ready",
		})
	}

	return deployedModelsResponse.Data, nil
}

func handleOptions(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
	c.Status(200)
	return
}

func handleProxy(c *gin.Context) {
	if c.Request.Method == http.MethodOptions {
		handleOptions(c)
		return
	}

	var server http.Handler

	// Choose the proxy based on ProxyMode or specific environment variables
	switch ProxyMode {
	case "azure":
		server = azure.NewOpenAIReverseProxy()
	case "google":
		google.HandleGoogleAIProxy(c)
		return // Add this return statement
	case "vertex":
		server = vertex.NewVertexAIReverseProxy()
	default:
		// Default to Azure if not specified, but only if the endpoint is set
		if os.Getenv("AZURE_OPENAI_ENDPOINT") != "" {
			server = azure.NewOpenAIReverseProxy()
		} else {
			// If no endpoint is configured, default to OpenAI
			server = openai.NewOpenAIReverseProxy()
		}
	}

	if ProxyMode != "google" {
		server.ServeHTTP(c.Writer, c.Request)
	}

	if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
		if _, err := c.Writer.Write([]byte("\n")); err != nil {
			log.Printf("rewrite response error: %v", err)
		}
	}

	// Enhanced error logging
	if c.Writer.Status() >= 400 {
		log.Printf("API request failed: %s %s, Status: %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}

// Helper functions to map capabilities and deprecation
func mapAzureCapabilities(caps azure.Capabilities) Capabilities {
	return Capabilities{
		FineTune:       caps.FineTune,
		Inference:      caps.Inference,
		Completion:     caps.Completion,
		ChatCompletion: caps.ChatCompletion,
		Embeddings:     caps.Embeddings,
	}
}

func mapAzureDeprecation(dep azure.Deprecation) Deprecation {
	return Deprecation{
		FineTune:  int64(dep.FineTune),  // Cast to int64
		Inference: int64(dep.Inference), // Cast to int64
	}
}

func mapGoogleCapabilities(caps google.Capabilities) Capabilities {
	return Capabilities{
		FineTune:       caps.FineTune,
		Inference:      caps.Inference,
		Completion:     caps.Completion,
		ChatCompletion: caps.ChatCompletion,
		Embeddings:     caps.Embeddings,
	}
}

func mapGoogleDeprecation(dep google.Deprecation) Deprecation {
	return Deprecation{
		FineTune:  int64(dep.FineTune),  // Cast to int64
		Inference: int64(dep.Inference), // Cast to int64
	}
}

func mapVertexCapabilities(caps vertex.Capabilities) Capabilities {
	return Capabilities{
		FineTune:       caps.FineTune,
		Inference:      caps.Inference,
		Completion:     caps.Completion,
		ChatCompletion: caps.ChatCompletion,
		Embeddings:     caps.Embeddings,
	}
}

func mapVertexDeprecation(dep vertex.Deprecation) Deprecation {
	return Deprecation{
		FineTune:  int64(dep.FineTune),  // Cast to int64
		Inference: int64(dep.Inference), // Cast to int64
	}
}
