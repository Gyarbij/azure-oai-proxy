package azure

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// RealTimeModels defines which models should use the realtime endpoint
	RealTimeModels = map[string]bool{
		"gpt-4o-realtime":         true,
		"gpt-4o-realtime-preview": true,
	}

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // TODO: Implement proper origin checks for production
		},
	}
)

// Session represents a realtime connection session
type Session struct {
	ID         string
	ClientConn *websocket.Conn
	AzureConn  *websocket.Conn
	Config     *SessionConfig
	closed     bool
	closeMutex sync.RWMutex
	closeOnce  sync.Once
}

// SessionConfig holds the session configuration
type SessionConfig struct {
	Voice            string         `json:"voice,omitempty"`
	Instructions     string         `json:"instructions,omitempty"`
	InputAudioFormat string         `json:"input_audio_format,omitempty"`
	TurnDetection    *TurnDetection `json:"turn_detection,omitempty"`
	Tools            []Tool         `json:"tools,omitempty"`
}

type TurnDetection struct {
	Type              string  `json:"type"`
	Threshold         float64 `json:"threshold,omitempty"`
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"`
}

type Tool struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// IsRealtimeModel checks if a model should use the realtime endpoint
func IsRealtimeModel(model string) bool {
	modelLower := strings.ToLower(model)
	return RealTimeModels[modelLower]
}

// HandleRealtime handles WebSocket connections for the realtime endpoint
func HandleRealtime(w http.ResponseWriter, r *http.Request) {
	// Extract and validate parameters
	deployment := r.URL.Query().Get("deployment")
	apiVersion := r.URL.Query().Get("api-version")
	if deployment == "" || apiVersion == "" {
		http.Error(w, "missing required query parameters: deployment or api-version", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Connect to Azure's WebSocket endpoint
	azureConn, err := connectToAzureWebSocket(r, deployment)
	if err != nil {
		log.Printf("Azure WebSocket connection failed: %v", err)
		closeWithError(clientConn, "Failed to connect to Azure OpenAI")
		return
	}

	// Create and initialize session
	session := &Session{
		ID:         fmt.Sprintf("session-%d", generateSessionID()),
		ClientConn: clientConn,
		AzureConn:  azureConn,
	}

	// Send session.created message
	if err := session.sendSessionCreated(); err != nil {
		log.Printf("Failed to send session.created: %v", err)
		session.Close()
		return
	}

	// Start bidirectional message relay
	session.relayMessages()
}

func connectToAzureWebSocket(r *http.Request, deployment string) (*websocket.Conn, error) {
	endpoint := AzureOpenAIEndpoint
	if strings.HasPrefix(endpoint, "https://") {
		endpoint = "wss://" + endpoint[8:]
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	// Construct Azure WebSocket URL
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %v", err)
	}

	u.Path = "/openai/realtime"
	q := u.Query()
	q.Set("api-version", "2024-10-01-preview")
	q.Set("deployment", deployment)
	u.RawQuery = q.Encode()

	// Setup headers for authentication
	headers := http.Header{}
	if apiKey := r.Header.Get("api-key"); apiKey != "" {
		headers.Set("api-key", apiKey)
	} else if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		headers.Set("Authorization", authHeader)
	}

	// Connect to Azure WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Azure: %v", err)
	}

	return conn, nil
}

func (s *Session) sendSessionCreated() error {
	message := map[string]interface{}{
		"type":       "session.created",
		"session_id": s.ID,
	}
	return s.ClientConn.WriteJSON(message)
}

func (s *Session) relayMessages() {
	// Handle client to Azure messages
	go func() {
		for {
			messageType, message, err := s.ClientConn.ReadMessage()
			if err != nil {
				s.handleError("Error reading from client", err)
				return
			}

			// Handle session.update commands
			if messageType == websocket.TextMessage {
				if err := s.handlePossibleSessionUpdate(message); err != nil {
					s.handleError("Error handling session update", err)
					continue
				}
			}

			// Forward message to Azure
			if err := s.AzureConn.WriteMessage(messageType, message); err != nil {
				s.handleError("Error writing to Azure", err)
				return
			}
		}
	}()

	// Handle Azure to client messages
	go func() {
		for {
			messageType, message, err := s.AzureConn.ReadMessage()
			if err != nil {
				s.handleError("Error reading from Azure", err)
				return
			}

			if err := s.ClientConn.WriteMessage(messageType, message); err != nil {
				s.handleError("Error writing to client", err)
				return
			}
		}
	}()
}

func (s *Session) handlePossibleSessionUpdate(message []byte) error {
	var cmd map[string]interface{}
	if err := json.Unmarshal(message, &cmd); err != nil {
		return fmt.Errorf("error parsing message: %v", err)
	}

	if cmdType, ok := cmd["type"].(string); ok && cmdType == "session.update" {
		if sessionData, ok := cmd["session"].(map[string]interface{}); ok {
			configBytes, _ := json.Marshal(sessionData)
			var config SessionConfig
			if err := json.Unmarshal(configBytes, &config); err != nil {
				return fmt.Errorf("error parsing session config: %v", err)
			}
			s.Config = &config

			// Send session.updated response
			response := map[string]interface{}{
				"type":    "session.updated",
				"session": s.Config,
			}
			if err := s.ClientConn.WriteJSON(response); err != nil {
				return fmt.Errorf("error sending session.updated: %v", err)
			}
		}
	}
	return nil
}

func (s *Session) handleError(context string, err error) {
	log.Printf("%s: %v", context, err)
	s.Close()
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		s.closeMutex.Lock()
		s.closed = true
		s.closeMutex.Unlock()

		if s.ClientConn != nil {
			s.ClientConn.Close()
		}
		if s.AzureConn != nil {
			s.AzureConn.Close()
		}
	})
}

func closeWithError(conn *websocket.Conn, message string) {
	conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseInternalServerErr, message),
	)
	conn.Close()
}

func generateSessionID() int64 {
	return time.Now().UnixNano()
}
