package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gyarbij/azure-oai-proxy/pkg/azure"
	"github.com/gyarbij/azure-oai-proxy/pkg/openai"
	"github.com/joho/godotenv"
)

var (
	AddressNetHTTP   = "0.0.0.0:11437"
	ProxyModeNetHTTP = "azure"
)

// Define the ModelList and Model types based on the API documentation
type ModelListNetHTTP struct {
	Object string         `json:"object"`
	Data   []ModelNetHTTP `json:"data"`
}

type ModelNetHTTP struct {
	ID              string              `json:"id"`
	Object          string              `json:"object"`
	CreatedAt       int64               `json:"created_at"`
	Capabilities    CapabilitiesNetHTTP `json:"capabilities"`
	LifecycleStatus string              `json:"lifecycle_status"`
	Status          string              `json:"status"`
	Deprecation     DeprecationNetHTTP  `json:"deprecation"`
	FineTune        string              `json:"fine_tune,omitempty"`
}

type CapabilitiesNetHTTP struct {
	FineTune       bool `json:"fine_tune"`
	Inference      bool `json:"inference"`
	Completion     bool `json:"completion"`
	ChatCompletion bool `json:"chat_completion"`
	Embeddings     bool `json:"embeddings"`
}

type DeprecationNetHTTP struct {
	FineTune  int64 `json:"fine_tune,omitempty"`
	Inference int64 `json:"inference"`
}

func initNetHTTP() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	if v := os.Getenv("AZURE_OPENAI_PROXY_ADDRESS"); v != "" {
		AddressNetHTTP = v
	}
	if v := os.Getenv("AZURE_OPENAI_PROXY_MODE"); v != "" {
		ProxyModeNetHTTP = v
	}
	log.Printf("loading azure openai proxy address: %s", AddressNetHTTP)
	log.Printf("loading azure openai proxy mode: %s", ProxyModeNetHTTP)

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

func mainNetHTTP() {
	initNetHTTP()

	mux := http.NewServeMux()

	// Proxy routes
	if ProxyModeNetHTTP == "azure" {
		mux.HandleFunc("/v1/models", handleGetModelsNetHTTP)
		mux.HandleFunc("/v1/chat/completions", handleAzureProxyNetHTTP)
		mux.HandleFunc("/v1/completions", handleAzureProxyNetHTTP)
		mux.HandleFunc("/v1/embeddings", handleAzureProxyNetHTTP)
		mux.HandleFunc("/v1/audio/speech", handleAzureProxyNetHTTP)
		mux.HandleFunc("/v1/audio/transcriptions", handleAzureProxyNetHTTP)
		mux.HandleFunc("/v1/images/generations", handleAzureProxyNetHTTP)
	} else if ProxyModeNetHTTP == "openai" {
		mux.HandleFunc("/v1/", handleOpenAIProxyNetHTTP)
	}

	// Health check endpoint
	mux.HandleFunc("/healthz", handleHealthCheckNetHTTP)

	// Add CORS middleware
	handler := corsMiddleware(mux)

	log.Printf("Starting server on %s", AddressNetHTTP)
	if err := http.ListenAndServe(AddressNetHTTP, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleHealthCheckNetHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"mode":   ProxyModeNetHTTP,
	})
}

func handleGetModelsNetHTTP(w http.ResponseWriter, r *http.Request) {
	models, err := fetchDeployedModelsNetHTTP(r)
	if err != nil {
		log.Printf("Error fetching deployed models: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch deployed models"})
		return
	}

	// Convert Azure models to OpenAI format
	var openAIModels []map[string]interface{}
	for _, model := range models {
		openAIModels = append(openAIModels, map[string]interface{}{
			"id":       model.ID,
			"object":   "model",
			"owned_by": "azure",
			"created":  model.CreatedAt,
		})
	}

	response := map[string]interface{}{
		"object": "list",
		"data":   openAIModels,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func fetchDeployedModelsNetHTTP(originalReq *http.Request) ([]ModelNetHTTP, error) {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	apiKey := os.Getenv("AZURE_OPENAI_KEY")

	if endpoint == "" || apiKey == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_ENDPOINT or AZURE_OPENAI_KEY not set")
	}

	url := fmt.Sprintf("%s/openai/models?api-version=%s", endpoint, azure.AzureOpenAIModelsAPIVersion)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("api-key", apiKey)
	req.Header.Set("Authorization", originalReq.Header.Get("Authorization"))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var modelList ModelListNetHTTP
	if err := json.Unmarshal(body, &modelList); err != nil {
		return nil, err
	}

	return modelList.Data, nil
}

func handleAzureProxyNetHTTP(w http.ResponseWriter, r *http.Request) {
	// Create the reverse proxy
	server := azure.NewOpenAIReverseProxy()

	// Serve the request
	server.ServeHTTP(w, r)

	// For SSE responses, ensure proper flushing
	if w.Header().Get("Content-Type") == "text/event-stream" ||
		strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream") {
		// Ensure the response is flushed
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

func handleOpenAIProxyNetHTTP(w http.ResponseWriter, r *http.Request) {
	server := openai.NewOpenAIReverseProxy()
	server.ServeHTTP(w, r)
}
