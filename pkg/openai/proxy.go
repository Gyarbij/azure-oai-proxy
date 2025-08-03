package openai

import (
    "os"
	"bytes"
    "io"
    "log"
    "net/http"
    "net/http/httputil"
    "net/url"
    "strings"
)

var (
    OpenAIEndpoint = "https://api.openai.com"
)

func init() {
    // Allow overriding the OpenAI endpoint if needed (e.g., for testing or proxies)
    if v := os.Getenv("OPENAI_API_ENDPOINT"); v != "" {
        OpenAIEndpoint = v
    }
}

func NewOpenAIReverseProxy() *httputil.ReverseProxy {
    return &httputil.ReverseProxy{
        Director:       makeDirector(),
        ModifyResponse: modifyResponse,
        ErrorHandler:   errorHandler,
    }
}

func makeDirector() func(*http.Request) {
    remote, err := url.Parse(OpenAIEndpoint)
    if err != nil {
        log.Printf("Error parsing OpenAI endpoint: %v", err)
        // Fallback to default
        remote, _ = url.Parse("https://api.openai.com")
    }

    return func(req *http.Request) {
        originURL := req.URL.String()
        
        // Preserve the original path and query
        originalPath := req.URL.Path
        originalRawQuery := req.URL.RawQuery
        
        // Set the scheme and host
        req.URL.Scheme = remote.Scheme
        req.URL.Host = remote.Host
        req.Host = remote.Host
        
        // Preserve the path - OpenAI uses the same paths as the proxy exposes
        req.URL.Path = originalPath
        req.URL.RawQuery = originalRawQuery
        
        // Handle Authorization header
        handleAuthorization(req)
        
        // Add OpenAI-specific headers if needed
        req.Header.Set("User-Agent", "Azure-OAI-Proxy/1.0")
        
        log.Printf("Proxying request [OpenAI] %s -> %s", originURL, req.URL.String())
    }
}

func handleAuthorization(req *http.Request) {
    // Ensure the Authorization header is properly formatted
    auth := req.Header.Get("Authorization")
    if auth != "" && !strings.HasPrefix(auth, "Bearer ") {
        // If it's just the API key, add the Bearer prefix
        req.Header.Set("Authorization", "Bearer "+auth)
    }
    
    // Remove any Azure-specific headers that might have been passed
    req.Header.Del("api-key")
}

func modifyResponse(res *http.Response) error {
    // Log errors for debugging
    if res.StatusCode >= 400 {
        body, _ := io.ReadAll(res.Body)
        log.Printf("OpenAI API Error Response: Status: %d, Body: %s", res.StatusCode, string(body))
        res.Body = io.NopCloser(bytes.NewBuffer(body))
    }
    
    // Handle streaming responses
    if res.Header.Get("Content-Type") == "text/event-stream" {
        res.Header.Set("X-Accel-Buffering", "no")
        res.Header.Set("Cache-Control", "no-cache")
        res.Header.Set("Connection", "keep-alive")
    }
    
    return nil
}

func errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
    log.Printf("OpenAI proxy error: %v", err)
    
    // Return a proper error response
    rw.Header().Set("Content-Type", "application/json")
    rw.WriteHeader(http.StatusBadGateway)
    
    errorResponse := `{"error": {"message": "Failed to connect to OpenAI API", "type": "proxy_error", "code": "bad_gateway"}}`
    rw.Write([]byte(errorResponse))
}