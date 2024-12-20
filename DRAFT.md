# Multi-AI Proxy (formerly Azure OpenAI Proxy)

[![Go Report Card](https://goreportcard.com/badge/github.com/Gyarbij/azure-oai-proxy)](https://goreportcard.com/report/github.com/Gyarbij/azure-oai-proxy)
[![Main v Dev Commits](https://shields.git.vg/github/commits-difference/Gyarbij/azure-oai-proxy?base=main&head=dev)](https://github.com/gyarbij/azure-oai-proxy)
[![Taal](https://shields.git.vg/github/languages/top/Gyarbij/azure-oai-proxy)](https://github.com/gyarbij/azure-oai-proxy)
[![GHCR Build](https://shields.git.vg/github/actions/workflow/status/gyarbij/azure-oai-proxy/ghcr-docker-publish.yml)](https://github.com/gyarbij/azure-oai-proxy)
[![License](https://shields.git.vg/github/license/Gyarbij/azure-oai-proxy?style=for-the-badge&color=blue)](https://github.com/gyarbij/azure-oai-proxy/blob/main/LICENSE)

## Introduction

**Multi-AI Proxy** (formerly Azure OpenAI Proxy) is a lightweight, high-performance proxy server that provides a unified interface for multiple AI providers, including **Azure OpenAI Service**, **Google AI Studio** (Gemini), and **Vertex AI** (Gemini). It enables seamless integration between applications designed for the OpenAI API format and these diverse AI platforms. This project bridges the gap for tools and services that are built to work with OpenAI's API structure but need to utilize Azure's OpenAI, Google AI Studio, or Vertex AI.

## Key Features

- ✅ **Unified API**: Provides an OpenAI API-compatible interface for multiple AI providers.
- 🤖 **Support for Multiple Providers**:
    - **Azure OpenAI Service**
    - **Google AI Studio** (Gemini models)
    - **Vertex AI** (Gemini models)
- 🗺️ **Model Mapping**: Automatically maps OpenAI model names to Azure, Google AI Studio, and Vertex AI model names.
- 🔄 **Dynamic Model List**: Fetches available models directly from your Azure OpenAI, Google AI Studio, and Vertex AI deployments.
- 🌐 **Broad Endpoint Support**: Handles various API endpoints including chat completions, completions, embeddings, and more (support varies by provider).
- 🚦 **Error Handling**: Provides meaningful error messages and logging for easier debugging.
- ⚙️ **Configurable**: Easy to set up with environment variables for each AI provider's endpoints and API keys.
- 🔐 **Authentication**:
    - Supports Azure AI serverless deployments with custom authentication.
    - Uses Google Cloud service account credentials for Vertex AI.
    - Uses API keys for Google AI Studio.
- ⚡ **Streaming Support**: Supports streaming responses for chat completions with Google AI Studio and Vertex AI.

## Use Cases

This proxy is particularly useful for:

- Running applications like Open WebUI with Azure OpenAI Services, Google AI Studio, or Vertex AI in a simplified manner.
- Testing various AI models from different providers using a consistent API format.
- Transitioning projects from OpenAI to Azure OpenAI, Google AI Studio, or Vertex AI with minimal code changes.
- Building applications that need to leverage the strengths of multiple AI platforms.

## Important Note

While the Multi-AI Proxy serves as a convenient bridge, it's recommended to use the official SDKs or APIs of each provider directly in production environments or when building new services, especially for provider-specific features or optimizations.

Direct integration offers:

- Better performance
- More reliable and up-to-date feature support
- Simplified architecture with one less component to maintain
- Direct access to provider-specific features and optimizations

This proxy is ideal for testing, development, and scenarios where modifying the original application to use a specific provider's API directly is not feasible.

Also, I strongly recommend using TSL/SSL for secure communication between the proxy and the client, especially in a production environment.

## Supported APIs

The proxy currently supports the following OpenAI-compatible API endpoints, though support varies by the underlying AI provider:

| Path                           | Azure OpenAI | Google AI Studio | Vertex AI |
| ------------------------------ | :----------: | :--------------: | :-------: |
| /v1/chat/completions           |      ✅      |        ✅        |     ✅    |
| /v1/completions                |      ✅      |        ✅        |     ✅    |
| /v1/embeddings                 |      ✅      |        ✅        |     ✅    |
| /v1/images/generations         |      ✅      |        ❌        |     ❌    |
| /v1/fine_tunes                 |      ✅      |        ❌        |     ❌    |
| /v1/files                      |      ✅      |        ❌        |     ❌    |
| /v1/models                     |      ✅      |        ✅        |     ✅    |
| /deployments                   |      ✅      |        ❌        |     ❌    |
| /v1/audio/speech               |      ✅      |        ❌        |     ❌    |
| /v1/audio/transcriptions       |      ✅      |        ❌        |     ❌    |
| /v1/audio/translations         |      ✅      |        ❌        |     ❌    |
| /v1/models/:model_id/capabilities |      ✅      |        ❌        |     ❌    |

**Note:** Streaming is supported for chat completions with Google AI Studio and Vertex AI.

## Configuration

### Environment Variables

| Parameter                     | Description                                                                | Default Value        | Required                                  |
| ------------------------------- | -------------------------------------------------------------------------- | -------------------- | ----------------------------------------- |
| AZURE_OPENAI_ENDPOINT         | Azure OpenAI Endpoint                                                      |                      | Only for Azure OpenAI                     |
| AZURE_OPENAI_API_KEY          | Azure OpenAI API Key                                                       |                      | Only for Azure OpenAI                     |
| AZURE_OPENAI_PROXY_ADDRESS    | Service listening address                                                  | 0.0.0.0:11437        | No                                        |
| AZURE_OPENAI_PROXY_MODE       | Proxy mode: `azure`, `google`, `vertex`, or `openai`                         | azure                | No                                        |
| AZURE_OPENAI_APIVERSION       | Azure OpenAI API version                                                   | 2024-06-01           | No                                        |
| AZURE_OPENAI_MODEL_MAPPER     | Comma-separated list of model=deployment pairs (for Azure)                  |                      | No                                        |
| AZURE_AI_STUDIO_DEPLOYMENTS   | Comma-separated list of serverless deployments (for Azure)                  |                      | No                                        |
| AZURE_OPENAI_KEY_*            | API keys for serverless deployments (replace * with uppercase model name)  |                      | Only for Azure AI Studio serverless       |
| GOOGLE_AI_STUDIO_API_KEY      | Google AI Studio API key                                                   |                      | Only for Google AI Studio                 |
| VERTEX_AI_PROJECT_ID          | Google Cloud project ID for Vertex AI                                      |                      | Only for Vertex AI                        |
| GOOGLE_APPLICATION_CREDENTIALS | Path to the Google Cloud service account key file (JSON)                    |                      | Only for Vertex AI (see Important Note) |

**Important Note about `GOOGLE_APPLICATION_CREDENTIALS`:**

-   For Vertex AI, you need to set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the path of your service account key file. This is used for authentication with the Vertex AI API.
-   Make sure the service account associated with the key file has the necessary permissions to access Vertex AI.
-   When running in a Docker container, you'll need to mount the key file into the container using the `-v` option (see examples below).

## Usage

### Docker Compose

Here's an example `docker-compose.yml` file demonstrating various configurations:

```yaml
services:
  multi-ai-proxy:
    image: 'gyarbij/azure-oai-proxy:vertex' # or a different tag if available
    restart: always
    ports:
      - '11437:11437'
    environment:
      - AZURE_OPENAI_ENDPOINT=https://your-azure-endpoint.openai.azure.com/
      - AZURE_OPENAI_API_KEY=your-azure-openai-api-key
      - AZURE_OPENAI_PROXY_MODE=azure # Change to "google" or "vertex" as needed
      # - AZURE_OPENAI_APIVERSION=2024-06-01
      # - AZURE_OPENAI_MODEL_MAPPER=gpt-3.5-turbo=gpt-35-turbo,gpt-4=gpt-4-turbo
      # - AZURE_AI_STUDIO_DEPLOYMENTS=mistral-large-2407=Mistral-large2:swedencentral,llama-3.1-405B=Meta-Llama-3-1-405B-Instruct:northcentralus,llama-3.1-70B=Llama-31-70B:swedencentral
      # - AZURE_OPENAI_KEY_MISTRAL-LARGE-2407=your-api-key-1
      # - AZURE_OPENAI_KEY_LLAMA-3.1-8B=your-api-key-2
      # - AZURE_OPENAI_KEY_LLAMA-3.1-70B=your-api-key-3
      - GOOGLE_AI_STUDIO_API_KEY=your-google-ai-studio-api-key
      - VERTEX_AI_PROJECT_ID=your-vertex-ai-project-id
      - GOOGLE_APPLICATION_CREDENTIALS=/app/service-account-key.json
    volumes:
      - /path/to/your/service-account-key.json:/app/service-account-key.json
    # Uncomment the following line to use an .env file:
    # env_file: .env
```

To use this configuration:

1.  Save the above content in a file named `compose.yaml`.
2.  Replace the placeholder values (e.g., `your-azure-endpoint`, `your-google-ai-studio-api-key`, `your-vertex-ai-project-id`, etc.) with your actual configuration.
3.  If using Vertex AI, make sure to mount your service account key file into the container using the `volumes` section.
4.  Run the following command in the same directory as your `compose.yaml` file:

```bash
docker compose up -d
```

### Using an .env File

To use an `.env` file instead of environment variables directly in the Docker Compose file:

1.  Create a file named `.env` in the same directory as your `docker-compose.yml`.
2.  Add your environment variables to the `.env` file, one per line:

    ```
    AZURE_OPENAI_ENDPOINT=https://your-azure-endpoint.openai.azure.com/
    AZURE_OPENAI_API_KEY=your-azure-openai-api-key
    AZURE_OPENAI_APIVERSION=2024-06-01
    AZURE_AI_STUDIO_DEPLOYMENTS=mistral-large-2407=Mistral-large2:swedencentral,llama-3.1-405B=Meta-Llama-3-1-405B-Instruct:northcentralus
    AZURE_OPENAI_KEY_MISTRAL-LARGE-2407=your-api-key-1
    AZURE_OPENAI_KEY_LLAMA-3.1-405B=your-api-key-2
    GOOGLE_AI_STUDIO_API_KEY=your-google-ai-studio-api-key
    VERTEX_AI_PROJECT_ID=your-vertex-ai-project-id
    GOOGLE_APPLICATION_CREDENTIALS=/app/service-account-key.json
    ```

3.  Uncomment the `env_file: .env` line in your `docker-compose.yml`.
4.  Run `docker compose up -d` to start the container.

### Running Directly with Docker

**For Azure OpenAI:**

```bash
docker run -d -p 11437:11437 \
  -e AZURE_OPENAI_ENDPOINT=https://your-azure-endpoint.openai.azure.com/ \
  -e AZURE_OPENAI_API_KEY=your-azure-openai-api-key \
  -e AZURE_OPENAI_PROXY_MODE=azure \
  --name azure-oai-proxy \
  gyarbij/azure-oai-proxy:vertex
```

**For Google AI Studio:**

```bash
docker run -d -p 11437:11437 \
  -e GOOGLE_AI_STUDIO_API_KEY=your-google-ai-studio-api-key \
  -e AZURE_OPENAI_PROXY_MODE=google \
  --name azure-oai-proxy \
  gyarbij/azure-oai-proxy:vertex
```

**For Vertex AI:**

```bash
docker run -d -p 11437:11437 \
  -v /path/to/your/service-account-key.json:/app/service-account-key.json \
  -e GOOGLE_APPLICATION_CREDENTIALS=/app/service-account-key.json \
  -e VERTEX_AI_PROJECT_ID=your-vertex-ai-project-id \
  -e AZURE_OPENAI_PROXY_MODE=vertex \
  --name azure-oai-proxy \
  gyarbij/azure-oai-proxy:vertex
```

**Remember to replace placeholder values with your actual configuration.**

## Usage Examples

### Calling the API (Examples)

**Azure OpenAI (Chat Completion):**

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-azure-api-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Google AI Studio (Chat Completion):**

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-pro",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Vertex AI (Chat Completion):**

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "chat-bison",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Google AI Studio (Chat Completion with Streaming):**

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-pro",
    "messages": [{"role": "user", "content": "Write a short story about a robot learning to feel emotions."}],
    "stream": true
  }'
```

**Vertex AI (Chat Completion with Streaming):**

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "chat-bison",
    "messages": [{"role": "user", "content": "Write a short poem."}],
    "stream": true
  }'
```

**Google AI Studio (Chat Completion with Temperature):**

```bash
curl http://localhost:11437/v1/chat/completions \
-H "Content-Type: application/json" \
-d '{
  "model": "gemini-pro",
  "messages": [{"role": "user", "content": "Write a story about a magic backpack."}],
  "temperature": 0.8
}'
```

**Vertex AI (Chat Completion with Top-P):**

```bash
curl http://localhost:11437/v1/chat/completions \
-H "Content-Type: application/json" \
-d '{
  "model": "chat-bison",
  "messages": [{"role": "user", "content": "Explain quantum physics in simple terms."}],
  "top_p": 0.7
}'
```

**Google AI Studio (List Models):**

```bash
curl http://localhost:11437/v1/models
```

**Vertex AI (List Models):**

```bash
curl http://localhost:11437/v1/models
```

**Azure OpenAI (List Models):**

```bash
curl http://localhost:11437/v1/models \
  -H "Authorization: Bearer your-azure-api-key"
```

For serverless deployments with Azure, use the model name as defined in your `AZURE_AI_STUDIO_DEPLOYMENTS` configuration.

## Model Mapping

### Azure OpenAI Model Mapping

The proxy automatically maps OpenAI model names to Azure OpenAI deployment names. You can customize this mapping using the `AZURE_OPENAI_MODEL_MAPPER` environment variable.

**Default Mappings:**

| OpenAI Model                    | Azure OpenAI Model            |
| ------------------------------- | ----------------------------- |
| `"gpt-3.5-turbo"`               | `"gpt-35-turbo"`              |
| `"gpt-3.5-turbo-0125"`          | `"gpt-35-turbo-0125"`         |
| `"gpt-3.5-turbo-0613"`          | `"gpt-35-turbo-0613"`         |
| `"gpt-3.5-turbo-1106"`          | `"gpt-35-turbo-1106"`         |
| `"gpt-3.5-turbo-16k-0613"`      | `"gpt-35-turbo-16k-0613"`     |
| `"gpt-3.5-turbo-instruct-0914"` | `"gpt-35-turbo-instruct-0914"`|
| `"gpt-4"`                       | `"gpt-4-0613"`                |
| `"gpt-4-32k"`                   | `"gpt-4-32k"`                 |
| `"gpt-4-32k-0613"`              | `"gpt-4-32k-0613"`            |
| `"gpt-4o-mini"`                 | `"gpt-4o-mini-2024-07-18"`    |
| `"gpt-4o"`                      | `"gpt-4o"`                    |
| `"gpt-4o-2024-05-13"`           | `"gpt-4o-2024-05-13"`         |
| `"gpt-4-turbo"`                 | `"gpt-4-turbo"`               |
| `"gpt-4-vision-preview"`        | `"gpt-4-vision-preview"`      |
| `"gpt-4-turbo-2024-04-09"`      | `"gpt-4-turbo-2024-04-09"`    |
| `"gpt-4-1106-preview"`          | `"gpt-4-1106-preview"`        |
| `"text-embedding-ada-002"`      | `"text-embedding-ada-002"`    |
| `"dall-e-2"`                    | `"dall-e-2"`                  |
| `"dall-e-3"`                    | `"dall-e-3"`                  |
| `"babbage-002"`                 | `"babbage-002"`               |
| `"davinci-002"`                 | `"davinci-002"`               |
| `"whisper-1"`                   | `"whisper"`                   |
| `"tts-1"`                       | `"tts"`                       |
| `"tts-1-hd"`                    | `"tts-hd"`                    |
| `"text-embedding-3-small"`      | `"text-embedding-3-small-1"`  |
| `"text-embedding-3-large"`      | `"text-embedding-3-large-1"`  |

**Custom Mapping:**

You can define custom mappings using the `AZURE_OPENAI_MODEL_MAPPER` environment variable. For example:

```
AZURE_OPENAI_MODEL_MAPPER="gpt-3.5-turbo=my-gpt35-deployment,gpt-4=my-gpt4-deployment"
```

### Google AI Studio Model Mapping

The proxy uses the following default mappings for Google AI Studio models:

| OpenAI Model          | Google AI Studio Model |
| --------------------- | ---------------------- |
| `"gemini-pro"`          | `"gemini-pro"`          |
| `"gemini-pro-vision"`   | `"gemini-pro-vision"`   |
| `"embedding-gecko-001"` | `"embedding-001"`      |

### Vertex AI Model Mapping

The proxy uses the following default mappings for Vertex AI models:

| OpenAI Model                    | Vertex AI Model                     |
| ------------------------------- | ----------------------------------- |
| `"chat-bison"`                  | `"chat-bison@002"`                  |
| `"text-bison"`                  | `"text-bison@002"`                  |
| `"embedding-gecko"`             | `"textembedding-gecko@003"`         |
| `"embedding-gecko-multilingual"` | `"textembedding-gecko-multilingual@003"` |

## Important Notes

-   Always use HTTPS in production environments for secure communication.
-   Regularly update the proxy to ensure compatibility with the latest API changes from each provider.
-   Monitor your usage and costs for each AI provider, especially when using this proxy in high-traffic scenarios.

## Recently Updated

-   2024-12-20: Added support for Google AI Studio and Vertex AI, including streaming and advanced parameters. Updated `main.go`, `pkg/google/proxy.go`, and `pkg/vertex/proxy.go`.
-   2024-07-25: Implemented support for Azure AI Studio deployments with support for Meta LLama 3.1, Mistral-2407 (mistral large 2), and other open models including from Cohere AI.
-   2024-07-18: Added support for `gpt-4o-mini`.
-   2024-06-23: Implemented dynamic model fetching for `/v1/models` endpoint, replacing hardcoded model list.
-   2024-06-23: Unified token handling mechanism across the application, improving consistency and security.
-   2024-06-23: Added support for audio-related endpoints: `/v1/audio/speech`, `/v1/audio/transcriptions`, and `/v1/audio/translations`.
-   2024-06-23: Implemented flexible environment variable handling for configuration (AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_API_KEY, AZURE_OPENAI_TOKEN).
-   2024-06-23: Added support for model capabilities endpoint `/v1/models/:model_id/capabilities`.
-   2024-06-23: Improved cross-origin resource sharing (CORS) handling with OPTIONS requests.
-   2024-06-23: Enhanced proxy functionality to better handle various Azure OpenAI API endpoints.
-   2024-06-23: Implemented fallback model mapping for unsupported models.
-   2024-06-22: Added support for image generation `/v1/images/generations`, fine-tuning operations `/v1/fine_tunes`, and file management `/v1/files`.
-   2024-06-22: Implemented better error handling and logging for API requests.
-   2024-06-22: Improved handling of rate limiting and streaming responses.
-   2024-06-22: Updated model mappings to include the latest models (gpt-4-turbo, gpt-4-vision-preview, dall-e-3).
-   2024-06-23: Added support for deployments management (/deployments).

## Project Renaming

This project is undergoing a potential name change to reflect its expanded capabilities beyond Azure OpenAI. The new name is yet to be decided.

**To avoid disruption:**

1.  The current repository will likely be forked to a new location.
2.  This repository will be made into a public archive with a clear message pointing to the new repository.

This approach ensures that existing users can continue to access the original code while also allowing the project to evolve under a new name that better represents its broader scope.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.

## Disclaimer

This project is not officially associated with or endorsed by Microsoft Azure, OpenAI, or Google. Use at your own discretion and ensure compliance with all relevant terms of service.