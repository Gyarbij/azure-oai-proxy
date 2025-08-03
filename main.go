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
	"github.com/gyarbij/azure-oai-proxy/pkg/openai"
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
	ID              string       `json:"id"`
	Object          string       `json:"object"`
	CreatedAt       int64        `json:"created_at"`
	Capabilities    Capabilities `json:"capabilities"`
	LifecycleStatus string       `json:"lifecycle_status"`
	Status          string       `json:"status"`
	Deprecation     Deprecation  `json:"deprecation"`
	FineTune        string       `json:"fine_tune,omitempty"`
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
}

func main() {
	router := gin.Default()

	// Proxy routes
	if ProxyMode == "azure" {
		router.GET("/v1/models", handleGetModels)
		router.OPTIONS("/v1/*path", handleOptions)
		// Existing routes
		router.POST("/v1/chat/completions", handleAzureProxy)
		router.POST("/v1/completions", handleAzureProxy)
		router.POST("/v1/embeddings", handleAzureProxy)
		// DALL-E routes
		router.POST("/v1/images/generations", handleAzureProxy)
		// speech- routes
		router.POST("/v1/audio/speech", handleAzureProxy)
		router.GET("/v1/audio/voices", handleAzureProxy)
		router.POST("/v1/audio/transcriptions", handleAzureProxy)
		router.POST("/v1/audio/translations", handleAzureProxy)
		// Fine-tuning routes
		router.POST("/v1/fine_tunes", handleAzureProxy)
		router.GET("/v1/fine_tunes", handleAzureProxy)
		router.GET("/v1/fine_tunes/:fine_tune_id", handleAzureProxy)
		router.POST("/v1/fine_tunes/:fine_tune_id/cancel", handleAzureProxy)
		router.GET("/v1/fine_tunes/:fine_tune_id/events", handleAzureProxy)
		// Files management routes
		router.POST("/v1/files", handleAzureProxy)
		router.GET("/v1/files", handleAzureProxy)
		router.DELETE("/v1/files/:file_id", handleAzureProxy)
		router.GET("/v1/files/:file_id", handleAzureProxy)
		router.GET("/v1/files/:file_id/content", handleAzureProxy)
		// Deployments management routes
		router.GET("/deployments", handleAzureProxy)
		router.GET("/deployments/:deployment_id", handleAzureProxy)
		router.GET("/v1/models/:model_id/capabilities", handleAzureProxy)

		// Responses API routes
		router.POST("/v1/responses", handleAzureProxy)
		router.GET("/v1/responses/:response_id", handleAzureProxy)
		router.DELETE("/v1/responses/:response_id", handleAzureProxy)
		router.POST("/v1/responses/:response_id/cancel", handleAzureProxy)
		router.GET("/v1/responses/:response_id/input_items", handleAzureProxy)
	} else {
		router.Any("*path", handleOpenAIProxy)
	}

	// Health check endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	router.Run(Address)
}

func handleGetModels(c *gin.Context) {
	req, _ := http.NewRequest("GET", c.Request.URL.String(), nil)
	req.Header.Set("Authorization", c.GetHeader("Authorization"))

	models, err := fetchDeployedModels(req)
	if err != nil {
		log.Printf("error fetching deployed models: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch deployed models"})
		return
	}

	// Add serverless deployments to the models list
	for deploymentName := range azure.ServerlessDeploymentInfo {
		models = append(models, Model{
			ID:     deploymentName,
			Object: "model",
			Capabilities: Capabilities{
				Completion:     true,
				ChatCompletion: true,
				Inference:      true,
			},
			LifecycleStatus: "active",
			Status:          "ready",
		})
	}

	result := ModelList{
		Object: "list",
		Data:   models,
	}
	c.JSON(http.StatusOK, result)
}

func fetchDeployedModels(originalReq *http.Request) ([]Model, error) {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	if endpoint == "" {
		endpoint = azure.AzureOpenAIEndpoint
	}

	// Use the separate models API version
	modelsAPIVersion := azure.AzureOpenAIModelsAPIVersion
	url := fmt.Sprintf("%s/openai/models?api-version=%s", endpoint, modelsAPIVersion)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("failed to fetch deployed models: %s", string(body))
	}

	var deployedModelsResponse ModelList
	if err := json.NewDecoder(resp.Body).Decode(&deployedModelsResponse); err != nil {
		return nil, err
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

func handleAzureProxy(c *gin.Context) {
	if c.Request.Method == http.MethodOptions {
		handleOptions(c)
		return
	}
	server := azure.NewOpenAIReverseProxy()
	server.ServeHTTP(c.Writer, c.Request)
	if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
		if _, err := c.Writer.Write([]byte("\n")); err != nil {
			log.Printf("rewrite azure response error: %v", err)
		}
	}
	// Enhanced error logging
	if c.Writer.Status() >= 400 {
		log.Printf("Azure API request failed: %s %s, Status: %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}

func handleOpenAIProxy(c *gin.Context) {
	server := openai.NewOpenAIReverseProxy()
	server.ServeHTTP(c.Writer, c.Request)
}
