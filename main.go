package main

import (
    "github.com/gyarbij/azure-oai-proxy/pkg/azure"
    "github.com/gyarbij/azure-oai-proxy/pkg/openai"
    "github.com/gin-gonic/gin"
    "log"
    "net/http"
    "os"
)

var (
    Address   = "0.0.0.0:11437"
    ProxyMode = "azure"
)

func init() {
    gin.SetMode(gin.ReleaseMode)
    if v := os.Getenv("AZURE_OPENAI_PROXY_ADDRESS"); v != "" {
        Address = v
    }
    if v := os.Getenv("AZURE_OPENAI_PROXY_MODE"); v != "" {
        ProxyMode = v
    }
    log.Printf("loading azure openai proxy address: %s", Address)
    log.Printf("loading azure openai proxy mode: %s", ProxyMode)
}

func main() {
    router := gin.Default()
    if ProxyMode == "azure" {
        router.GET("/v1/models", handleGetModels)
        router.OPTIONS("/v1/*path", handleOptions)
        // Existing routes
        router.POST("/v1/chat/completions", handleAzureProxy)
        router.POST("/v1/completions", handleAzureProxy)
        router.POST("/v1/embeddings", handleAzureProxy)
        // New DALL-E routes
        router.POST("/v1/images/generations", handleAzureProxy)
		// Whisper speech-to-text
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
    } else {
        router.Any("*path", handleOpenAIProxy)
    }

    router.Run(Address)
}

func handleGetModels(c *gin.Context) {
	models := []string{
		"gpt-4o", "gpt-4-turbo", "gpt-4", "gpt-4o-2024-05-13", "gpt-4-turbo-2024-04-09", "gpt-4-0613", "gpt-4-1106-preview", "gpt-4-0125-preview", "gpt-4-vision-preview", "gpt-4-32k-0613",
		"gpt-35-turbo-0301", "gpt-35-turbo-0613", "gpt-35-turbo-1106", "gpt-35-turbo-0125", "gpt-35-turbo-16k",
		"text-embedding-3-large", "text-embedding-3-small", "text-embedding-ada-002",
		"dall-e-2", "dall-e-3",
		"babbage-002", "davinci-002", "whisper-001",
	}
	result := azure.ListModelResponse{
		Object: "list",
	}
	for _, model := range models {
		result.Data = append(result.Data, azure.Model{
			ID:      model,
			Object:  "model",
			Created: 1677649963,
			OwnedBy: "openai",
			Permission: []azure.ModelPermission{
				{
					ID:                 "",
					Object:             "model",
					Created:            1679602087,
					AllowCreateEngine:  true,
					AllowSampling:      true,
					AllowLogprobs:      true,
					AllowSearchIndices: true,
					AllowView:          true,
					AllowFineTuning:    true,
					Organization:       "*",
					Group:              nil,
					IsBlocking:         false,
				},
			},
			Root:   model,
			Parent: nil,
		})
	}
	c.JSON(200, result)
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
