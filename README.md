# Azure OpenAI Proxy

[![Go Report Card](https://goreportcard.com/badge/github.com/Gyarbij/azure-oai-proxy)](https://goreportcard.com/report/github.com/Gyarbij/azure-oai-proxy)
[![Main v Dev Commits](https://shields.git.vg/github/commits-difference/Gyarbij/azure-oai-proxy?base=main&head=dev)](https://github.com/gyarbij/azure-oai-proxy)
[![Taal](https://shields.git.vg/github/languages/top/Gyarbij/azure-oai-proxy)](https://github.com/gyarbij/azure-oai-proxy)
[![GHCR Build](https://shields.git.vg/github/actions/workflow/status/gyarbij/azure-oai-proxy/ghcr-docker-publish.yml)](https://github.com/gyarbij/azure-oai-proxy)
[![License](https://shields.git.vg/github/license/Gyarbij/azure-oai-proxy?style=for-the-badge&color=blue)](https://github.com/gyarbij/azure-oai-proxy/blob/main/LICENSE)

## Introduction

Azure OAI Proxy is a lightweight, high-performance proxy server that enables seamless integration between Azure OpenAI Services and applications designed for OpenAI API only compatible endpoints. This project bridges the gap for tools and services that are built to work with OpenAI's API structure but need to utilize Azure's OpenAI services, including support for the latest reasoning models through Azure's Responses API.

## Key Features

-   ✅ **API Compatibility**: Translates requests from OpenAI API format to Azure OpenAI Services format on-the-fly.
-   🧠 **Advanced Reasoning Model Support**: Full support for Azure's advanced reasoning models (O1, O3, O4 series) through automatic Responses API integration.
-   📡 **Streaming Support**: Real-time streaming for both traditional chat models and reasoning models with proper format conversion.
-   🗺️ **Model Mapping**: Automatically maps OpenAI model names to Azure scheme, with a comprehensive failsafe list.
-   🔄 **Dynamic Model List**: Fetches available models directly from your Azure OpenAI deployment using a dedicated API version.
-   🌐 **Support for Multiple Endpoints**: Handles various API endpoints including image, speech, completions, chat completions, embeddings, responses API, and more.
-   🚦 **Error Handling**: Provides meaningful error messages and logging for easier debugging.
-   ⚙️ **Configurable**: Easy to set up with environment variables for Azure AI/Azure OAI endpoint, API keys, and API versions.
-   🔐 **Serverless Deployment Support**: Supports Azure AI serverless deployments with custom authentication.
-   🔀 **Automatic API Selection**: Intelligently routes requests to Chat Completions API or Responses API based on model capabilities.

## Use Cases

This proxy is particularly useful for:

-   Running applications like Open WebUI with Azure OpenAI Services, including advanced reasoning models like O3 and O1.
-   Seamlessly using Azure's latest reasoning models in tools built for OpenAI API.
-   Testing Azure OpenAI capabilities using tools built for the OpenAI API.
-   Transitioning projects from OpenAI to Azure OpenAI with minimal code changes.
-   Accessing Azure-exclusive models and features through familiar OpenAI interfaces.

## Important Note

While azure oai proxy serves as a convenient bridge, it's recommended to use the official Azure OpenAI SDK or API directly in production environments or when building new services.

Direct integration offers:

-   Better performance
-   More reliable and up-to-date feature support
-   Simplified architecture with one less component to maintain
-   Direct access to Azure-specific features and optimizations

This proxy is ideal for testing, development, and scenarios where modifying the original application to use Azure OpenAI directly is not feasible.

Also, I strongly recommend using TLS/SSL for secure communication between the proxy and the client. This is especially important when using the proxy in a production environment (even though you shouldn't but well, here you are anyway). TBD: Add docker compose including nginx proxy manager.

## Supported APIs

The latest version of the Azure OpenAI service supports the following APIs:

| Path                               | Status | Notes |
| :--------------------------------- | :----- | :---- |
| /v1/chat/completions               | ✅     | Auto-routes to Responses API for reasoning models |
| /v1/completions                    | ✅     |       |
| /v1/embeddings                     | ✅     |       |
| /v1/images/generations             | ✅     |       |
| /v1/fine_tunes                     | ✅     |       |
| /v1/files                          | ✅     |       |
| /v1/models                         | ✅     |       |
| /v1/responses                      | ✅     | **New** - Azure Responses API support |
| /v1/responses/:response_id         | ✅     | **New** - Retrieve, delete, cancel operations |
| /v1/responses/:response_id/input_items | ✅ | **New** - List input items |
| /deployments                       | ✅     |       |
| /v1/audio/speech                   | ✅     |       |
| /v1/audio/transcriptions            | ✅     |       |
| /v1/audio/translations             | ✅     |       |
| /v1/models/:model_id/capabilities | ✅     |       |

## Model Support & API Routing

The proxy automatically detects model capabilities and routes requests appropriately:

### Traditional Models (Chat Completions API)
- GPT-3.5 series (gpt-3.5-turbo, etc.)
- GPT-4 series (gpt-4, gpt-4-turbo, etc.)
- GPT-4o series (gpt-4o, gpt-4o-mini, etc.)

### Reasoning Models (Responses API)
- **O1 Series**: o1, o1-preview, o1-mini
- **O3 Series**: o3, o3-pro, o3-mini
- **O4 Series**: o4, o4-mini

*Reasoning models automatically use Azure's Responses API while maintaining OpenAI chat completion interface compatibility.*

## Configuration

### Environment Variables

| Parameter                       | Description                                                    | Default Value    | Required |
| :------------------------------ | :------------------------------------------------------------- | :--------------- | :------- |
| AZURE_OPENAI_ENDPOINT           | Azure OpenAI Endpoint                                          |                  | Yes      |
| AZURE_OPENAI_PROXY_ADDRESS      | Service listening address                                      | 0.0.0.0:11437    | No       |
| AZURE_OPENAI_PROXY_MODE         | Proxy mode, can be either "azure" or "openai"                 | azure            | No       |
| AZURE_OPENAI_APIVERSION         | Azure OpenAI API version (for general operations)             | 2024-12-01-preview      | No       |
| AZURE_OPENAI_MODELS_APIVERSION  | Azure OpenAI API version (for fetching models)                | 2024-10-21       | No       |
| AZURE_OPENAI_RESPONSES_APIVERSION | Azure OpenAI API version (for Responses API)                | preview          | No       |
| AZURE_OPENAI_MODEL_MAPPER       | Comma-separated list of model=deployment pairs                 |                  | No       |
| AZURE_AI_STUDIO_DEPLOYMENTS     | Comma-separated list of serverless deployments                 |                  | No       |
| AZURE_OPENAI_KEY_\*             | API keys for serverless deployments (replace \* with uppercase model name) |                  | No       |

## Usage

### Docker Compose

Here's an example `docker-compose.yml` file with all possible environment variable options:

```yaml
services:
  azure-oai-proxy:
    image: 'gyarbij/azure-oai-proxy:latest'
    # container_name: azure-oai-proxy
    # Alternatively, use GitHub Container Registry:
    # image: 'ghcr.io/gyarbij/azure-oai-proxy:latest'
    restart: always
    environment:
      - AZURE_OPENAI_ENDPOINT=https://your-endpoint.openai.azure.com/
      - AZURE_OPENAI_MODELS_APIVERSION=2024-10-21
      # - AZURE_OPENAI_PROXY_ADDRESS=0.0.0.0:11437
      # - AZURE_OPENAI_PROXY_MODE=azure
      # - AZURE_OPENAI_APIVERSION=2024-12-01-preview
      # - AZURE_OPENAI_RESPONSES_APIVERSION=preview
      # - AZURE_OPENAI_MODEL_MAPPER=gpt-3.5-turbo=gpt-35-turbo,gpt-4=gpt-4-turbo
      # - AZURE_AI_STUDIO_DEPLOYMENTS=mistral-large-2407=Mistral-large2:swedencentral,llama-3.1-405B=Meta-Llama-3-1-405B-Instruct:northcentralus,llama-3.1-70B=Llama-31-70B:swedencentral
      # - AZURE_OPENAI_KEY_MISTRAL-LARGE-2407=your-api-key-1
      # - AZURE_OPENAI_KEY_LLAMA-3.1-8B=your-api-key-2
      # - AZURE_OPENAI_KEY_LLAMA-3.1-70B=your-api-key-3
    ports:
      - '11437:11437'
    # Uncomment the following line to use an .env file:
    # env_file: .env
```

To use this configuration:

1.  Save the above content in a file named `compose.yaml`.
2.  Replace the placeholder values (e.g., `your-endpoint`, `your-api-key-1`, etc.) with your actual Azure OpenAI configuration.
3.  Run the following command in the same directory as your `compose.yaml` file:

```sh
docker compose up -d
```

### Using an .env File

To use an .env file instead of environment variables in the Docker Compose file:

1.  Create a file named `.env` in the same directory as your `docker-compose.yml`.
2.  Add your environment variables to the `.env` file, one per line:

```
AZURE_OPENAI_ENDPOINT=https://your-endpoint.openai.azure.com/
AZURE_OPENAI_APIVERSION=2024-12-01-preview
AZURE_OPENAI_MODELS_APIVERSION=2024-10-21
AZURE_OPENAI_RESPONSES_APIVERSION=preview
AZURE_AI_STUDIO_DEPLOYMENTS=mistral-large-2407=Mistral-large2:swedencentral,llama-3.1-405B=Meta-Llama-3-1-405B-Instruct:northcentralus
AZURE_OPENAI_KEY_MISTRAL-LARGE-2407=your-api-key-1
AZURE_OPENAI_KEY_LLAMA-3.1-405B=your-api-key-2
```

3.  Uncomment the `env_file: .env` line in your `docker-compose.yml`.
4.  Run `docker-compose up -d` to start the container with the environment variables from the .env file.

### Running from GitHub Container Registry

To run the Azure OAI Proxy using the image from GitHub Container Registry:

```sh
docker run -d -p 11437:11437 \
 -e AZURE_OPENAI_ENDPOINT=https://your-endpoint.openai.azure.com/ \
 -e AZURE_OPENAI_MODELS_APIVERSION=2024-10-21 \
 -e AZURE_AI_STUDIO_DEPLOYMENTS=mistral-large-2407=Mistral-large2:swedencentral \
 -e AZURE_OPENAI_KEY_MISTRAL-LARGE-2407=your-api-key \
 ghcr.io/gyarbij/azure-oai-proxy:latest
```

Replace the placeholder values with your actual Azure OpenAI configuration.

## Usage Examples

### Calling the API

Once the proxy is running, you can call it using the OpenAI API format:

#### Traditional Chat Models
```sh
curl http://localhost:11437/v1/chat/completions \
 -H "Content-Type: application/json" \
 -H "Authorization: Bearer your-azure-api-key" \
 -d '{
  "model": "gpt-4o",
  "messages": [{"role": "user", "content": "Hello!"}]
 }'
```

#### Reasoning Models (Automatically routed to Responses API)
```sh
curl http://localhost:11437/v1/chat/completions \
 -H "Content-Type: application/json" \
 -H "Authorization: Bearer your-azure-api-key" \
 -d '{
  "model": "o3-pro",
  "messages": [{"role": "user", "content": "Solve this complex reasoning problem..."}],
  "stream": true
 }'
```

#### Direct Responses API Access
```sh
curl http://localhost:11437/v1/responses \
 -H "Content-Type: application/json" \
 -H "Authorization: Bearer your-azure-api-key" \
 -d '{
  "model": "o3-pro",
  "input": "What are the implications of quantum computing?",
  "stream": false
 }'
```

For serverless deployments, use the model name as defined in your `AZURE_AI_STUDIO_DEPLOYMENTS` configuration.

## Model Mapping Mechanism (Used for Custom deployment names)

These are the default mappings for the most common models, if your Azure OpenAI deployment uses different names, you can set the `AZURE_OPENAI_MODEL_MAPPER` environment variable to define custom mappings. The proxy also includes a comprehensive **failsafe list** to handle a wide variety of model names:

| OpenAI Model                 | Azure OpenAI Model           |
| :--------------------------- | :--------------------------- |
| `"o1"`                       | `"o1"`                       |
| `"o1-preview"`               | `"o1-preview"`               |
| `"o1-mini-2024-09-12"`       | `"o1-mini-2024-09-12"`       |
| `"gpt-4o"`                   | `"gpt-4o"`                   |
| `"gpt-4o-2024-05-13"`        | `"gpt-4o-2024-05-13"`        |
| `"gpt-4o-2024-08-06"`        | `"gpt-4o-2024-08-06"`        |
| `"gpt-4o-mini"`              | `"gpt-4o-mini"`              |
| `"gpt-4o-mini-2024-07-18"`   | `"gpt-4o-mini-2024-07-18"`   |
| `"gpt-4"`                    | `"gpt-4-0613"`               |
| `"gpt-4-0613"`               | `"gpt-4-0613"`               |
| `"gpt-4-1106-preview"`       | `"gpt-4-1106-preview"`       |
| `"gpt-4-0125-preview"`       | `"gpt-4-0125-preview"`       |
| `"gpt-4-vision-preview"`     | `"gpt-4-vision-preview"`     |
| `"gpt-4-turbo-2024-04-09"`   | `"gpt-4-turbo-2024-04-09"`   |
| `"gpt-4-32k"`                | `"gpt-4-32k-0613"`           |
| `"gpt-4-32k-0613"`           | `"gpt-4-32k-0613"`           |
| `"gpt-3.5-turbo"`            | `"gpt-35-turbo-0613"`        |
| `"gpt-3.5-turbo-0301"`       | `"gpt-35-turbo-0301"`       |
| `"gpt-3.5-turbo-0613"`       | `"gpt-35-turbo-0613"`       |
| `"gpt-3.5-turbo-1106"`       | `"gpt-35-turbo-1106"`       |
| `"gpt-3.5-turbo-0125"`       | `"gpt-35-turbo-0125"`       |
| `"gpt-3.5-turbo-16k"`        | `"gpt-35-turbo-16k-0613"`   |
| `"gpt-3.5-turbo-16k-0613"`   | `"gpt-35-turbo-16k-0613"`   |
| `"gpt-3.5-turbo-instruct"`   | `"gpt-35-turbo-instruct-0914"` |
| `"gpt-3.5-turbo-instruct-0914"` | `"gpt-35-turbo-instruct-0914"` |
| `"text-embedding-3-small"`   | `"text-embedding-3-small-1"` |
| `"text-embedding-3-large"`   | `"text-embedding-3-large-1"` |
| `"text-embedding-ada-002"`   | `"text-embedding-ada-002-2"` |
| `"text-embedding-ada-002-1"` | `"text-embedding-ada-002-1"` |
| `"text-embedding-ada-002-2"` | `"text-embedding-ada-002-2"` |
| `"dall-e-2"`                | `"dall-e-2-2.0"`             |
| `"dall-e-2-2.0"`            | `"dall-e-2-2.0"`             |
| `"dall-e-3"`                | `"dall-e-3-3.0"`             |
| `"dall-e-3-3.0"`            | `"dall-e-3-3.0"`             |
| `"babbage-002"`              | `"babbage-002-1"`           |
| `"babbage-002-1"`            | `"babbage-002-1"`           |
| `"davinci-002"`              | `"davinci-002-1"`           |
| `"davinci-002-1"`            | `"davinci-002-1"`           |
| `"tts"`                      | `"tts-001"`                  |
| `"tts-001"`                  | `"tts-001"`                  |
| `"tts-hd"`                   | `"tts-hd-001"`               |
| `"tts-hd-001"`               | `"tts-hd-001"`               |
| `"whisper"`                  | `"whisper-001"`              |
| `"whisper-001"`              | `"whisper-001"`              |

For custom fine-tuned models, the model name can be passed directly. For models with deployment names different from the model names, custom mapping relationships can be defined, such as:

| Model Name        | Deployment Name          |
| :---------------- | :----------------------- |
| gpt-3.5-turbo     | gpt-35-turbo-upgrade     |
| gpt-3.5-turbo-0301 | gpt-35-turbo-0301-fine-tuned |

## Reasoning Models & Responses API

### Automatic Detection
The proxy automatically detects when you're using reasoning models (O1, O3, O4 series) and:

1. **Routes to Responses API**: Automatically converts `/v1/chat/completions` requests to use Azure's `/openai/v1/responses` endpoint
2. **Converts Request Format**: Transforms OpenAI chat messages to Responses API input format
3. **Handles Streaming**: Converts Responses API SSE events to OpenAI-compatible streaming format
4. **Maintains Compatibility**: Your client code doesn't need to change - use standard OpenAI format

### Supported Reasoning Models
- **O1 Family**: `o1`, `o1-preview`, `o1-mini`, `o1-mini-2024-09-12`
- **O3 Family**: `o3`, `o3-pro`, `o3-mini`, `o3-pro-2025-06-10` 
- **O4 Family**: `o4`, `o4-mini`

### Response API Features
When using reasoning models, you get access to:
- **Advanced Reasoning**: Enhanced problem-solving capabilities
- **Reasoning Traces**: Detailed reasoning process (when available)
- **Background Processing**: Support for long-running reasoning tasks
- **Chain of Thought**: Structured reasoning outputs

## Important Notes

-   Always use HTTPS in production environments for secure communication.
-   Regularly update the proxy to ensure compatibility with the latest Azure OpenAI API changes.
-   Monitor your Azure OpenAI usage and costs, especially when using this proxy in high-traffic scenarios.
-   Reasoning models may have higher latency due to their advanced processing capabilities.
-   Some reasoning models may have usage limits or require special access permissions.

## Recently Updated
-   **2025-08-03 (v1.0.8)** Added comprehensive support for Azure OpenAI Responses API with automatic reasoning model detection and streaming conversion.
-   2025-01-24 Added support for Azure OpenAI API version 2024-12-01-preview and new model fetching mechanism.
-   2024-07-25 Implemented support for Azure AI Studio deployments with support for Meta LLama 3.1, Mistral-2407 (mistral large 2), and other open models including from Cohere AI.
-   2024-07-18 Added support for `gpt-4o-mini`.
-   2024-06-23 Implemented dynamic model fetching for `/v1/models` endpoint, replacing hardcoded model list.
-   2024-06-23 Unified token handling mechanism across the application, improving consistency and security.
-   2024-06-23 Added support for audio-related endpoints: `/v1/audio/speech`, `/v1/audio/transcriptions`, and `/v1/audio/translations`.
-   2024-06-23 Implemented flexible environment variable handling for configuration (AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_API_KEY, AZURE_OPENAI_TOKEN).
-   2024-06-23 Added support for model capabilities endpoint `/v1/models/:model_id/capabilities`.
-   2024-06-23 Improved cross-origin resource sharing (CORS) handling with OPTIONS requests.
-   2024-06-23 Enhanced proxy functionality to better handle various Azure OpenAI API endpoints.
-   2024-06-23 Implemented fallback model mapping for unsupported models.
-   2024-06-22 Added support for image generation `/v1/images/generations`, fine-tuning operations `/v1/fine_tunes`, and file management `/v1/files`.
-   2024-06-22 Implemented better error handling and logging for API requests.
-   2024-06-22 Improved handling of rate limiting and streaming responses.
-   2024-06-22 Updated model mappings to include the latest models (gpt-4-turbo, gpt-4-vision-preview, dall-e-3).
-   2024-06-23 Added support for deployments management (/deployments).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.

## Disclaimer

This project is not officially associated with or endorsed by Microsoft Azure or OpenAI. Use at your own discretion and ensure compliance with all relevant terms of service.