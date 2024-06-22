# Azure OpenAI Proxy

[![Go Report Card](https://goreportcard.com/badge/github.com/Gyarbij/azure-oai-proxy)](https://goreportcard.com/report/github.com/Gyarbij/azure-oai-proxy)
[![License](https://badgen.net/badge/license/MIT/cyan)](https://github.com/gyarbij/azure-oai-proxy/blob/main/LICENSE)
[![Release](https://badgen.net/github/release/gyarbij/azure-oai-proxy/latest)](https://github.com/gyarbij/azure-oai-proxy)
[![Azure](https://badgen.net/badge/icon/Azure?icon=azure&label)](https://github.com/gyarbij/azure-oai-proxy)
[![Azure](https://badgen.net/badge/icon/OpenAI?icon=azure&label)](https://github.com/gyarbij/azure-oai-proxy)
[![Azure](https://badgen.net/badge/icon/docker?icon=docker&label)](https://github.com/gyarbij/azure-oai-proxy)

## Introduction

Azure OAI Proxy is a lightweight, high-performance proxy server that enables seamless integration between Azure OpenAI Services and applications designed for only OpenAI API compatible endpoints. This project bridges the gap for tools and services that are built to work with OpenAI's API structure but need to utilize Azure's OpenAI.

## Key Features

- ✅ **API Compatibility**: Translates requests from OpenAI API format to Azure OpenAI Services format on-the-fly.
- 🗺️ **Model Mapping**: Automatically maps OpenAI model names to Azure scheme.
- 🔄 **Dynamic Model List**: Fetches available models directly from your Azure OpenAI deployment to have feature parity with normal OpenAI, in projects such as Open WebUI.
- 🌐 **Support for Multiple Endpoints**: Handles various API endpoints including image, speech, completions, chat completions, embeddings, and more.
- 🚦 **Error Handling**: Provides meaningful error messages and logging for easier debugging.
- ⚙️ **Configurable**: Easy to set up with environment variables for Azure OpenAI endpoint and API key.

## Use Cases

This proxy is particularly useful for:

- Running applications like Open WebUI with Azure OpenAI Services in a simplfied manner vs LiteLLM (which has additional features such as cost tracking).
- Testing Azure OpenAI capabilities using tools built for the OpenAI API.
- Transitioning projects from OpenAI to Azure OpenAI with minimal code changes.

## Important Note

While this proxy serves as a convenient bridge, it's recommended to use the official Azure OpenAI SDK or API directly in production environments or when building new services. Direct integration offers:

- Better performance
- More reliable and up-to-date feature support
- Simplified architecture with one less component to maintain
- Direct access to Azure-specific features and optimizations

This proxy is ideal for testing, development, and scenarios where modifying the original application to use Azure OpenAI directly is not feasible.

Also, I strongly recommend using TSL/SSL for secure communication between the proxy and the client. This is especially important when using the proxy in a production environment (even though you shouldn't but well, here you are anyway). TBD: Add docker compose including nginx proxy manager.

## Supported APIs

The latest version of the Azure OpenAI service now supports the following APIs:

| Path                  | Status |
| --------------------- | ------ |
| /v1/chat/completions  |  ✅   |
| /v1/completions       | ✅    |
| /v1/embeddings        | ✅    |
| /v1/images/generations | ✅   |
| /v1/fine_tunes        | ✅    |
| /v1/files             | ✅    |
| /v1/models            | ✅    |
| /deployments          | ✅    |
| /v1/audio             | ✅    |

> Other APIs not supported by Azure will be returned in a mock format (such as OPTIONS requests initiated by browsers). If you find your project need additional OpenAI-supported APIs, feel free to submit a PR.

## Getting Started

It's easy to get started with Azure OAI Proxy. You can either deploy it as a reverse proxy or use it as a forward proxy as detailed below. However if you're ready to jump right in and start using the proxy, you can use the following Docker command:

```docker
docker pull gyarbij/azure-oai-proxy:latest

docker run -d -p 11437:11437 --name=azure-oai-proxy \
  --env AZURE_OPENAI_ENDPOINT=https://{YOURENDPOINT}.openai.azure.com \
  gyarbij/azure-oai-proxy:latest
```

## Configuration

### 1. Used as reverse proxy (i.e. an OpenAI API gateway)

Environment Variables

| Parameters                 | Description                                                                                                                                                                                                                                                                                                    | Default Value                                                           |
| :------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :---------------------------------------------------------------------- |
| AZURE_OPENAI_PROXY_ADDRESS | Service listening address                                                                                                                                                                                                                                                                                      | 0.0.0.0:11437                                                            |
| AZURE_OPENAI_PROXY_MODE    | Proxy mode, can be either "azure" or "openai".                                                                                                                                                                                                                                                                 | azure                                                                   |
| AZURE_OPENAI_ENDPOINT      | Azure OpenAI Endpoint, usually looks like https://{custom}.openai.azure.com. Required.                                                                                                                                                                                                                         |                                                                         |
| AZURE_OPENAI_APIVERSION    | Azure OpenAI API version. Default is 2024-05-01-preview.                                                                                                                                                                                                                                                       | 2024-05-01-preview                                                      |
| AZURE_OPENAI_MODEL_MAPPER (DEPRECATED)  | A comma-separated list of model=deployment pairs. Maps model names to deployment names. For example, `gpt-3.5-turbo=gpt-35-turbo`, `gpt-3.5-turbo-0301=gpt-35-turbo-0301`. If there is no match, the proxy will pass model as deployment name directly (in fact, most Azure model names are same with OpenAI). | `gpt-3.5-turbo=gpt-35-turbo`<br/>`gpt-3.5-turbo-0301=gpt-35-turbo-0301` |
| AZURE_OPENAI_TOKEN         | Azure OpenAI API Token. If this environment variable is set, the token in the request header will be ignored.                                                                                                                                                                                                  | ""                                                                      |

Use in command line

```shell
curl https://{your-custom-domain}/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {your azure api key}" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### 2. Used as forward proxy (i.e. an HTTP proxy)

When accessing Azure OpenAI API through HTTP, it can be used directly as a proxy, but this tool does not have built-in HTTPS support, so you need an HTTPS proxy such as Nginx to support accessing HTTPS version of OpenAI API.

Assuming that the proxy domain you configured is `https://{your-domain}.com`, you can execute the following commands in the terminal to use the https proxy:

```shell
export https_proxy=https://{your-domain}.com

curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {your azure api key}" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

Or configure it as an HTTP proxy in other open source Web ChatGPT projects:

```
export HTTPS_PROXY=https://{your-domain}.com
```

## Deploy

Deploying through Docker

```shell
docker pull gyarbij/azure-oai-proxy:latest
docker run -p 11437:11437 --name=azure-oai-proxy \
  --env AZURE_OPENAI_ENDPOINT=https://{YOURENDPOINT}.openai.azure.com/ \
  gyarbij/azure-oai-proxy:latest
```

Calling

```shell
curl https://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {your azure api key}" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Recently Updated

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

## Model Mapping Mechanism (DEPRECATED)

There are a series of rules for model mapping pre-defined in `AZURE_OPENAI_MODEL_MAPPER`, and the default configuration basically satisfies the mapping of all Azure models. The rules include:

- `gpt-3.5-turbo` -> `gpt-35-turbo`
- `gpt-3.5-turbo-0301` -> `gpt-35-turbo-0301`
- A mapping mechanism that pass model name directly as fallback.

For custom fine-tuned models, the model name can be passed directly. For models with deployment names different from the model names, custom mapping relationships can be defined, such as:

| Model Name         | Deployment Name              |
| :----------------- | :--------------------------- |
| gpt-3.5-turbo      | gpt-35-turbo-upgrade         |
| gpt-3.5-turbo-0301 | gpt-35-turbo-0301-fine-tuned |

## Contributing

We welcome contributions! Rest TBD.

## License

MIT License

## Disclaimer

This project is not officially associated with or endorsed by Microsoft Azure or OpenAI. Use at your own discretion and ensure compliance with all relevant terms of service.