# Azure OpenAI Proxy

[![Go Report Card](https://goreportcard.com/badge/github.com/Gyarbij/azure-oai-proxy)](https://goreportcard.com/report/github.com/Gyarbij/azure-oai-proxy)
[![Main v Dev Commits](https://shields.git.vg/github/commits-difference/Gyarbij/azure-oai-proxy?base=main&head=dev)](https://github.com/gyarbij/azure-oai-proxy)
[![Taal](https://shields.git.vg/github/languages/top/Gyarbij/azure-oai-proxy)](https://github.com/gyarbij/azure-oai-proxy)
[![GHCR Build](https://shields.git.vg/github/actions/workflow/status/gyarbij/azure-oai-proxy/ghcr-docker-publish.yml)](https://github.com/gyarbij/azure-oai-proxy)
[![License](https://shields.git.vg/github/license/Gyarbij/azure-oai-proxy?style=for-the-badge&color=blue)](https://github.com/gyarbij/azure-oai-proxy/blob/main/LICENSE)

## Introduction

Azure OAI Proxy is a lightweight, high-performance proxy server that enables seamless integration between Azure OpenAI Services and applications designed for OpenAI API only compatible endpoints. This project bridges the gap for tools and services that are built to work with OpenAI's API structure but need to utilize Azure's OpenAI.

## Key Features

- ✅ **API Compatibility**: Translates requests from OpenAI API format to Azure OpenAI Services format on-the-fly.
- 🗺️ **Model Mapping**: Automatically maps OpenAI model names to Azure scheme.
- 🔄 **Dynamic Model List**: Fetches available models directly from your Azure OpenAI deployment.
- 🌐 **Support for Multiple Endpoints**: Handles various API endpoints including image, speech, completions, chat completions, embeddings, and more.
- 🚦 **Error Handling**: Provides meaningful error messages and logging for easier debugging.
- ⚙️ **Configurable**: Easy to set up with environment variables for Azure AI/Azure OAI endpoint and API keys.
- 🔐 **Serverless Deployment Support**: Supports Azure AI serverless deployments with custom authentication.

## Use Cases

This proxy is particularly useful for:

- Running applications like Open WebUI with Azure OpenAI Services in a simplfied manner vs LiteLLM (which has additional features such as cost tracking).
- Testing Azure OpenAI capabilities using tools built for the OpenAI API.
- Transitioning projects from OpenAI to Azure OpenAI with minimal code changes.

## Important Note

While azure oai proxy serves as a convenient bridge, it's recommended to use the official Azure OpenAI SDK or API directly in production environments or when building new services. 

Direct integration offers:

- Better performance
- More reliable and up-to-date feature support
- Simplified architecture with one less component to maintain
- Direct access to Azure-specific features and optimizations

This proxy is ideal for testing, development, and scenarios where modifying the original application to use Azure OpenAI directly is not feasible.

Also, I strongly recommend using TSL/SSL for secure communication between the proxy and the client. This is especially important when using the proxy in a production environment (even though you shouldn't but well, here you are anyway). TBD: Add docker compose including nginx proxy manager.

## Supported APIs

The latest version of the Azure OpenAI service supports the following APIs:

| Path                           | Status |
|--------------------------------|--------|
| /v1/chat/completions           | ✅     |
| /v1/completions                | ✅     |
| /v1/embeddings                 | ✅     |
| /v1/images/generations         | ✅     |
| /v1/fine_tunes                 | ✅     |
| /v1/files                      | ✅     |
| /v1/models                     | ✅     |
| /deployments                   | ✅     |
| /v1/audio/speech               | ✅     |
| /v1/audio/transcriptions       | ✅     |
| /v1/audio/translations         | ✅     |
| /v1/models/:model_id/capabilities | ✅     |

## Configuration

### Environment Variables

| Parameter                     | Description                                                                | Default Value        | Required |
|-------------------------------|----------------------------------------------------------------------------|----------------------|----------|
| AZURE_OPENAI_ENDPOINT         | Azure OpenAI Endpoint                                                      |                      | Yes      |
| AZURE_OPENAI_PROXY_ADDRESS    | Service listening address                                                  | 0.0.0.0:11437        | No       |
| AZURE_OPENAI_PROXY_MODE       | Proxy mode, can be either "azure" or "openai"                              | azure                | No       |
| AZURE_OPENAI_APIVERSION       | Azure OpenAI API version                                                   | 2024-06-01           | No       |
| AZURE_OPENAI_MODEL_MAPPER     | Comma-separated list of model=deployment pairs                             |                      | No       |
| AZURE_AI_STUDIO_DEPLOYMENTS   | Comma-separated list of serverless deployments                             |                      | No       |
| AZURE_OPENAI_KEY_*            | API keys for serverless deployments (replace * with uppercase model name)  |                      | No       |

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
    # - AZURE_OPENAI_PROXY_ADDRESS=0.0.0.0:11437
    # - AZURE_OPENAI_PROXY_MODE=azure
    # - AZURE_OPENAI_APIVERSION=2024-06-01
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

1. Save the above content in a file named `compose.yaml`.
2. Replace the placeholder values (e.g., `your-endpoint`, `your-api-key-1`, etc.) with your actual Azure OpenAI configuration.
3. Run the following command in the same directory as your `compose.yaml` file:

```sh
docker compose up -d
```

### Using an .env File

To use an .env file instead of environment variables in the Docker Compose file:

1. Create a file named `.env` in the same directory as your `docker-compose.yml`.
2. Add your environment variables to the `.env` file, one per line:

```
AZURE_OPENAI_ENDPOINT=https://your-endpoint.openai.azure.com/
AZURE_OPENAI_APIVERSION=2024-06-01
AZURE_AI_STUDIO_DEPLOYMENTS=mistral-large-2407=Mistral-large2:swedencentral,llama-3.1-405B=Meta-Llama-3-1-405B-Instruct:northcentralus
AZURE_OPENAI_KEY_MISTRAL-LARGE-2407=your-api-key-1
AZURE_OPENAI_KEY_LLAMA-3.1-405B=your-api-key-2
```

3. Uncomment the `env_file: .env` line in your `docker-compose.yml`.
4. Run `docker-compose up -d` to start the container with the environment variables from the .env file.

### Running from GitHub Container Registry

To run the Azure OAI Proxy using the image from GitHub Container Registry:

```sh
docker run -d -p 11437:11437 \
  -e AZURE_OPENAI_ENDPOINT=https://your-endpoint.openai.azure.com/ \
  -e AZURE_AI_STUDIO_DEPLOYMENTS=mistral-large-2407=Mistral-large2:swedencentral \
  -e AZURE_OPENAI_KEY_MISTRAL-LARGE-2407=your-api-key \
  ghcr.io/gyarbij/azure-oai-proxy:latest
```

Replace the placeholder values with your actual Azure OpenAI configuration.

## Usage Examples

### Calling the API

Once the proxy is running, you can call it using the OpenAI API format:

```sh
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-azure-api-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

For serverless deployments, use the model name as defined in your `AZURE_AI_STUDIO_DEPLOYMENTS` configuration.

## Model Mapping Mechanism (Used for Custom deployment names)

These are the default mappings for the most common models, if your Azure OpenAI deployment uses different names, you can set the `AZURE_OPENAI_MODEL_MAPPER` environment variable to define custom mappings.:

| OpenAI Model                    | Azure OpenAI Model            |
|---------------------------------|-------------------------------|
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

For custom fine-tuned models, the model name can be passed directly. For models with deployment names different from the model names, custom mapping relationships can be defined, such as:

| Model Name         | Deployment Name              |
| :----------------- | :--------------------------- |
| gpt-3.5-turbo      | gpt-35-turbo-upgrade         |
| gpt-3.5-turbo-0301 | gpt-35-turbo-0301-fine-tuned |

## Important Notes

- Always use HTTPS in production environments for secure communication.
- Regularly update the proxy to ensure compatibility with the latest Azure OpenAI API changes.
- Monitor your Azure OpenAI usage and costs, especially when using this proxy in high-traffic scenarios.

## Recently Updated

+ 2024-07-25 Implemented support for Azure AI Studio deployments with support for Meta LLama 3.1, Mistral-2407 (mistral large 2), and other open models including from Cohere AI.
+ 2024-07-18 Added support for `gpt-4o-mini`.
+ 2024-06-23 Implemented dynamic model fetching for `/v1/models endpoint`, replacing hardcoded model list.
+ 2024-06-23 Unified token handling mechanism across the application, improving consistency and security.
+ 2024-06-23 Added support for audio-related endpoints: `/v1/audio/speech`, `/v1/audio/transcriptions`, and `/v1/audio/translations`.
+ 2024-06-23 Implemented flexible environment variable handling for configuration (AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_API_KEY, AZURE_OPENAI_TOKEN).
+ 2024-06-23 Added support for model capabilities endpoint `/v1/models/:model_id/capabilities`.
+ 2024-06-23 Improved cross-origin resource sharing (CORS) handling with OPTIONS requests.
+ 2024-06-23 Enhanced proxy functionality to better handle various Azure OpenAI API endpoints.
+ 2024-06-23 Implemented fallback model mapping for unsupported models.
+ 2024-06-22 Added support for image generation `/v1/images/generations`, fine-tuning operations `/v1/fine_tunes`, and file management `/v1/files`.
+ 2024-06-22 Implemented better error handling and logging for API requests.
+ 2024-06-22 Improved handling of rate limiting and streaming responses.
+ 2024-06-22 Updated model mappings to include the latest models (gpt-4-turbo, gpt-4-vision-preview, dall-e-3).
+ 2024-06-23 Added support for deployments management (/deployments).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.

## Disclaimer

This project is not officially associated with or endorsed by Microsoft Azure or OpenAI. Use at your own discretion and ensure compliance with all relevant terms of service.