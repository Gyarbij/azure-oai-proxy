package azure

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/http/httputil"
    "net/url"
    "os"
    "path"
    "regexp"
    "strings"

    "github.com/tidwall/gjson"
)

var (
    AzureOpenAIToken       = ""
    AzureOpenAIAPIVersion  = "2024-05-01-preview"
    AzureOpenAIEndpoint    = ""
    AzureOpenAIModelMapper = map[string]string{
        "gpt-3.5-turbo":      "gpt-35-turbo",
        "gpt-3.5-turbo-0125": "gpt-35-turbo-0125",
        "gpt-4o":             "gpt-4o",
        "gpt-4":              "gpt-4",
        "gpt-4-32k":          "gpt-4-32k",
        "gpt-4-vision-preview": "gpt-4-vision",
        "gpt-4-turbo":        "gpt-4-turbo",
        "text-embedding-ada-002": "text-embedding-ada-002",
        "dall-e-3":           "dall-e-3",
		"whisper-1":          "whisper",
		"tts-1":           	  "tts",
		"tts-1-hd":           "tts-hd",
    }
    fallbackModelMapper = regexp.MustCompile(`[.:]`)
)

func init() {
	if v := os.Getenv("AZURE_OPENAI_APIVERSION"); v != "" {
		AzureOpenAIAPIVersion = v
	}
	if v := os.Getenv("AZURE_OPENAI_ENDPOINT"); v != "" {
		AzureOpenAIEndpoint = v
	}
	if v := os.Getenv("AZURE_OPENAI_MODEL_MAPPER"); v != "" {
		for _, pair := range strings.Split(v, ",") {
			info := strings.Split(pair, "=")
			if len(info) != 2 {
				log.Printf("error parsing AZURE_OPENAI_MODEL_MAPPER, invalid value %s", pair)
				os.Exit(1)
			}
			AzureOpenAIModelMapper[info[0]] = info[1]
		}
	}
	if v := os.Getenv("AZURE_OPENAI_TOKEN"); v != "" {
		AzureOpenAIToken = v
		log.Printf("loading azure api token from env")
	}

	log.Printf("loading azure api endpoint: %s", AzureOpenAIEndpoint)
	log.Printf("loading azure api version: %s", AzureOpenAIAPIVersion)
	for k, v := range AzureOpenAIModelMapper {
		log.Printf("loading azure model mapper: %s -> %s", k, v)
	}
}

func NewOpenAIReverseProxy() *httputil.ReverseProxy {
    remote, err := url.Parse(AzureOpenAIEndpoint)
    if err != nil {
        log.Printf("error parse endpoint: %s\n", AzureOpenAIEndpoint)
        os.Exit(1)
    }
    
    return &httputil.ReverseProxy{
        Director:       makeDirector(remote),
        ModifyResponse: modifyResponse,
    }
}

func makeDirector(remote *url.URL) func(*http.Request) {
    return func(req *http.Request) {
        // Get model and map it to deployment
        model := getModelFromRequest(req)
        deployment := GetDeploymentByModel(model)

        // Handle token
        handleToken(req)

        // Set the Host, Scheme, Path, and RawPath of the request
        originURL := req.URL.String()
        req.Host = remote.Host
        req.URL.Scheme = remote.Scheme
        req.URL.Host = remote.Host

        // Handle different endpoints
        switch {
        case strings.HasPrefix(req.URL.Path, "/v1/chat/completions"):
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), "chat/completions")
        case strings.HasPrefix(req.URL.Path, "/v1/completions"):
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), "completions")
        case strings.HasPrefix(req.URL.Path, "/v1/embeddings"):
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), "embeddings")
        case strings.HasPrefix(req.URL.Path, "/v1/images/generations"):
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), "images/generations")
        case strings.HasPrefix(req.URL.Path, "/v1/fine_tunes"):
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), "fine-tunes")
        case strings.HasPrefix(req.URL.Path, "/v1/files"):
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), "files")
        case strings.HasPrefix(req.URL.Path, "/v1/audio/transcriptions"):
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), "transcriptions")
        case strings.HasPrefix(req.URL.Path, "/v1/audio/translations"):
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), "translations")
        default:
            req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), strings.TrimPrefix(req.URL.Path, "/v1/"))
        }

        req.URL.RawPath = req.URL.EscapedPath()

        // Add the api-version query parameter
        query := req.URL.Query()
        query.Add("api-version", AzureOpenAIAPIVersion)
        req.URL.RawQuery = query.Encode()

        log.Printf("proxying request [%s] %s -> %s", model, originURL, req.URL.String())
    }
}

func getModelFromRequest(req *http.Request) string {
    if req.Body == nil {
        return ""
    }
    body, _ := ioutil.ReadAll(req.Body)
    req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
    return gjson.GetBytes(body, "model").String()
}

func handleToken(req *http.Request) {
    token := ""
    if AzureOpenAIToken != "" {
        token = AzureOpenAIToken
    } else {
        token = strings.ReplaceAll(req.Header.Get("Authorization"), "Bearer ", "")
    }
    req.Header.Set("api-key", token)
    req.Header.Del("Authorization")
}

func modifyResponse(res *http.Response) error {
    // Handle rate limiting headers
    if res.StatusCode == http.StatusTooManyRequests {
        log.Printf("Rate limit exceeded: %s", res.Header.Get("Retry-After"))
    }

    // Handle streaming responses
    if res.Header.Get("Content-Type") == "text/event-stream" {
        res.Header.Set("X-Accel-Buffering", "no")
    }

    return nil
}

func GetDeploymentByModel(model string) string {
    if v, ok := AzureOpenAIModelMapper[model]; ok {
        return v
    }
    return fallbackModelMapper.ReplaceAllString(model, "")
}