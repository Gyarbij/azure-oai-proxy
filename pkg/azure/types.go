package azure

type ListModelResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// AzureConfig represents the configuration for Azure OpenAI
type AzureConfig struct {
	APIVersion             string `json:"api_version"`
	Endpoint               string `json:"endpoint"`
	Token                  string `json:"token"`
	ModelMapperMode        string `json:"model_mapper_mode"`
	AIStudioDeploymentsRaw string `json:"ai_studio_deployments_raw"`
}

// AzureAIStudioDeployment represents a deployment in Azure AI Studio
type AzureAIStudioDeployment struct {
	ModelName      string `json:"model_name"`
	DeploymentName string `json:"deployment_name"`
	Region         string `json:"region"`
}

// Update Model struct to include new fields if necessary
type Model struct {
	ID              string            `json:"id"`
	Object          string            `json:"object"`
	CreatedAt       int64             `json:"created_at"`
	Capabilities    Capabilities      `json:"capabilities"`
	LifecycleStatus string            `json:"lifecycle_status"`
	Status          string            `json:"status"`
	Deprecation     Deprecation       `json:"deprecation,omitempty"`
	FineTune        string            `json:"fine_tune,omitempty"`
	Created         int               `json:"created"`
	OwnedBy         string            `json:"owned_by"`
	Permission      []ModelPermission `json:"permission"`
	Root            string            `json:"root"`
	Parent          any               `json:"parent"`
	// Add any new fields from the latest Azure OpenAI API here
}

type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type ModelPermission struct {
	ID                 string `json:"id"`
	Object             string `json:"object"`
	Created            int    `json:"created"`
	AllowCreateEngine  bool   `json:"allow_create_engine"`
	AllowSampling      bool   `json:"allow_sampling"`
	AllowLogprobs      bool   `json:"allow_logprobs"`
	AllowSearchIndices bool   `json:"allow_search_indices"`
	AllowView          bool   `json:"allow_view"`
	AllowFineTuning    bool   `json:"allow_fine_tuning"`
	Organization       string `json:"organization"`
	Group              any    `json:"group"`
	IsBlocking         bool   `json:"is_blocking"`
}

// DeployedModel represents a model deployed in Azure
type DeployedModel struct {
	ID                 string   `json:"id"`
	ModelID            string   `json:"model"`
	DeploymentID       string   `json:"deployment_id"`
	Status             string   `json:"status"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
	CapabilityVersions []string `json:"capability_versions,omitempty"`
}

// ListDeployedModelsResponse represents the response for listing deployed models
type ListDeployedModelsResponse struct {
	Data []DeployedModel `json:"data"`
}

// JSONModeRequest represents a request with JSON mode enabled
type JSONModeRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ResponseFormat specifies the desired format for the model's output
type ResponseFormat struct {
	Type string `json:"type"` // Can be "text" or "json_object"
}

// ChatMessage represents a message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// JSONModeResponse represents a response when JSON mode is enabled
type JSONModeResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []JSONChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

// JSONChoice represents a choice in the JSON mode response
type JSONChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// DeploymentCapability represents the capabilities of a deployment
type DeploymentCapability struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// DeploymentDetails represents detailed information about a deployment
type DeploymentDetails struct {
	ID            string                 `json:"id"`
	ModelID       string                 `json:"model"`
	OwnerID       string                 `json:"owner"`
	Status        string                 `json:"status"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
	Capabilities  []DeploymentCapability `json:"capabilities"`
	ScaleSettings ScaleSettings          `json:"scale_settings"`
	RaiPolicy     string                 `json:"rai_policy"`
}

// ScaleSettings represents the scale settings for a deployment
type ScaleSettings struct {
	ScaleType string `json:"scale_type"`
	Capacity  int    `json:"capacity"`
}

// SpeechRequest represents the request structure for text-to-speech
type SpeechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

// SpeechResponse represents the response structure from text-to-speech
type SpeechResponse struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

// VoiceInfo represents information about available voices
type VoiceInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Gender   string `json:"gender"`
}

// ListVoicesResponse represents the response structure for listing available voices
type ListVoicesResponse struct {
	Voices []VoiceInfo `json:"voices"`
}

// ImageGenerationRequest represents the request structure for image generation
type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	NumImages      int    `json:"num_images,omitempty"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// ImageGenerationResponse represents the response structure from image generation
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData represents data of a generated image
type ImageData struct {
	URL string `json:"url"`
}

// AudioTranscriptionRequest represents the request structure for audio transcription
type AudioTranscriptionRequest struct {
	Model    string `json:"model"`
	Audio    []byte `json:"audio"`
	Language string `json:"language,omitempty"`
}

// AudioTranscriptionResponse represents the response structure from audio transcription
type AudioTranscriptionResponse struct {
	Text string `json:"text"`
}

// Update if there are any changes to the capabilities
type Capabilities struct {
	ChatCompletion bool `json:"chat_completion"`
	Completion     bool `json:"completion"`
	Embeddings     bool `json:"embeddings"`
	FineTune       bool `json:"fine_tune"`
	Inference      bool `json:"inference"`
	// Add any new capabilities here
}

// Update if there are any changes to the deprecation structure
type Deprecation struct {
	FineTune  int `json:"fine_tune,omitempty"`
	Inference int `json:"inference,omitempty"`
	// Add any new deprecation fields here
}

// ResponsesCreateRequest represents the request to create a response
type ResponsesCreateRequest struct {
    Model               string                 `json:"model"`
    Input               interface{}            `json:"input"`
    Instructions        string                 `json:"instructions,omitempty"`
    Tools               []ResponseTool         `json:"tools,omitempty"`
    ToolChoice          interface{}            `json:"tool_choice,omitempty"`
    ParallelToolCalls   bool                   `json:"parallel_tool_calls,omitempty"`
    Metadata            map[string]interface{} `json:"metadata,omitempty"`
    Temperature         float64                `json:"temperature,omitempty"`
    TopP                float64                `json:"top_p,omitempty"`
    MaxOutputTokens     int                    `json:"max_output_tokens,omitempty"`
    Store               bool                   `json:"store,omitempty"`
    Stream              bool                   `json:"stream,omitempty"`
    Background          bool                   `json:"background,omitempty"`
    PreviousResponseID  string                 `json:"previous_response_id,omitempty"`
    ReasoningEffort     string                 `json:"reasoning_effort,omitempty"`
    Include             []string               `json:"include,omitempty"`
}

// ResponseTool represents a tool in the Responses API
type ResponseTool struct {
    Type         string                 `json:"type"`
    Name         string                 `json:"name,omitempty"`
    Description  string                 `json:"description,omitempty"`
    Parameters   map[string]interface{} `json:"parameters,omitempty"`
    Container    *ToolContainer         `json:"container,omitempty"`
    ServerLabel  string                 `json:"server_label,omitempty"`
    ServerURL    string                 `json:"server_url,omitempty"`
    Headers      map[string]string      `json:"headers,omitempty"`
    RequireApproval string              `json:"require_approval,omitempty"`
}

// ToolContainer for code interpreter
type ToolContainer struct {
    Type  string   `json:"type"`
    Files []string `json:"files,omitempty"`
}

// Response represents a response from the Responses API
type Response struct {
    ID                  string                 `json:"id"`
    Object              string                 `json:"object"`
    CreatedAt           float64                `json:"created_at"`
    Error               *ResponseError         `json:"error,omitempty"`
    IncompleteDetails   *IncompleteDetails     `json:"incomplete_details,omitempty"`
    Instructions        string                 `json:"instructions,omitempty"`
    Metadata            map[string]interface{} `json:"metadata"`
    Model               string                 `json:"model"`
    Output              []ResponseOutput       `json:"output"`
    ParallelToolCalls   bool                   `json:"parallel_tool_calls,omitempty"`
    Temperature         float64                `json:"temperature"`
    ToolChoice          interface{}            `json:"tool_choice,omitempty"`
    Tools               []ResponseTool         `json:"tools"`
    TopP                float64                `json:"top_p"`
    MaxOutputTokens     int                    `json:"max_output_tokens,omitempty"`
    PreviousResponseID  string                 `json:"previous_response_id,omitempty"`
    Reasoning           interface{}            `json:"reasoning,omitempty"`
    Status              string                 `json:"status"`
    Text                string                 `json:"text,omitempty"`
    Truncation          interface{}            `json:"truncation,omitempty"`
    Usage               *ResponseUsage         `json:"usage,omitempty"`
    User                string                 `json:"user,omitempty"`
    ReasoningEffort     string                 `json:"reasoning_effort,omitempty"`
}

// ResponseError represents an error in a response
type ResponseError struct {
    Type    string `json:"type"`
    Message string `json:"message"`
}

// IncompleteDetails for incomplete responses
type IncompleteDetails struct {
    Reason string `json:"reason"`
}

// ResponseOutput represents output from a response
type ResponseOutput struct {
    ID          string                 `json:"id"`
    Content     []ResponseContent      `json:"content,omitempty"`
    Role        string                 `json:"role,omitempty"`
    Status      string                 `json:"status,omitempty"`
    Type        string                 `json:"type"`
    Name        string                 `json:"name,omitempty"`
    CallID      string                 `json:"call_id,omitempty"`
    Arguments   map[string]interface{} `json:"arguments,omitempty"`
    Result      string                 `json:"result,omitempty"`
    ServerLabel string                 `json:"server_label,omitempty"`
}

// ResponseContent represents content in a response output
type ResponseContent struct {
    Annotations []interface{} `json:"annotations,omitempty"`
    Text        string        `json:"text,omitempty"`
    Type        string        `json:"type"`
}

// ResponseUsage represents usage statistics
type ResponseUsage struct {
    InputTokens        int                        `json:"input_tokens"`
    OutputTokens       int                        `json:"output_tokens"`
    TotalTokens        int                        `json:"total_tokens"`
    OutputTokenDetails *ResponseOutputTokenDetail `json:"output_tokens_details,omitempty"`
}

// ResponseOutputTokenDetail represents detailed output token usage
type ResponseOutputTokenDetail struct {
    ReasoningTokens int `json:"reasoning_tokens"`
}

// InputItemsList represents a list of input items
type InputItemsList struct {
    Data    []ResponseOutput `json:"data"`
    HasMore bool             `json:"has_more"`
    Object  string           `json:"object"`
    FirstID string           `json:"first_id"`
    LastID  string           `json:"last_id"`
}