# Azure OpenAI Proxy - Anthropic Integration

## Architecture Overview

This proxy provides OpenAI-compatible API endpoints for Azure OpenAI services, including **native Anthropic Claude model support** through Azure AI Foundry.

### Key Components

```
pkg/
├── anthropic/          # Native Anthropic API integration
│   ├── types.go        # Anthropic Messages API types
│   └── converter.go    # OpenAI ↔ Anthropic conversion
├── azure/              # Azure OpenAI proxy logic
│   ├── proxy.go        # Main reverse proxy
│   ├── streaming.go    # Responses API streaming (O-series)
│   └── types.go        # Azure-specific types
└── openai/             # Direct OpenAI proxy
    └── proxy.go        # OpenAI API passthrough
```

## Claude Model Support

### How It Works

1. **Detection**: Automatically detects Claude models (e.g., `claude-sonnet-4-5-20250929`)
2. **Conversion**: Converts OpenAI chat completion format to Anthropic Messages API format
3. **Routing**: Routes requests to Azure's `/anthropic/v1/messages` endpoint
4. **Response**: Converts Anthropic responses back to OpenAI format

### Request Flow

```
OpenWebUI/Client (OpenAI format)
    ↓
Proxy detects Claude model
    ↓
anthropic.ConvertOpenAIToAnthropic()
    ↓
Azure AI Foundry (/anthropic/v1/messages)
    ↓
anthropic.StreamConverter (streaming)
    OR
anthropic.ReadNonStreamingResponse() (non-streaming)
    ↓
Client receives OpenAI-formatted response
```

### Streaming Support

The proxy properly handles **Anthropic SSE (Server-Sent Events)** streaming:

**Anthropic Events** → **OpenAI Chunks**
- `message_start` → Initial chunk with role
- `content_block_delta` → Text delta chunks
- `message_stop` → Final chunk with `[DONE]`

## API Endpoints

### Chat Completions
```http
POST /v1/chat/completions
Content-Type: application/json

{
  "model": "claude-sonnet-4-5-20250929",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": true
}
```

**Response (Streaming):**
```
data: {"id":"msg_xxx","object":"chat.completion.chunk","created":0,"model":"claude-sonnet-4-5-20250929","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"msg_xxx","object":"chat.completion.chunk","created":0,"model":"claude-sonnet-4-5-20250929","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: [DONE]
```

**Response (Non-Streaming):**
```json
{
  "id": "msg_xxx",
  "object": "chat.completion",
  "created": 1734355200,
  "model": "claude-sonnet-4-5-20250929",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you today?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

## Configuration

### Environment Variables

```bash
# Azure OpenAI Configuration
AZURE_OPENAI_ENDPOINT=https://your-foundry.services.ai.azure.com/
AZURE_OPENAI_KEY=your-api-key

# API Versions
AZURE_OPENAI_APIVERSION=2025-04-01-preview
AZURE_OPENAI_RESPONSES_APIVERSION=preview
ANTHROPIC_APIVERSION=2023-06-01

# Model Mapping
AZURE_OPENAI_MODEL_MAPPER=claude-sonnet-4-5=claude-sonnet-4.5,gpt-5-pro=gpt-5-pro-2025-10-06

# Server Configuration
AZURE_OPENAI_PROXY_ADDRESS=0.0.0.0:11437
AZURE_OPENAI_PROXY_MODE=azure
```

### Model Mapping

The proxy automatically strips version suffixes and maps models:

```
claude-sonnet-4-5-20250929  →  claude-sonnet-4.5 (deployment)
gpt-5-pro-2025-10-06        →  gpt-5-pro (deployment)
```

## Deployment

### Docker Compose

```yaml
services:
  azure-oai-proxy:
    image: ghcr.io/gyarbij/azure-oai-proxy:latest
    ports:
      - "11437:11437"
    environment:
      - AZURE_OPENAI_ENDPOINT=${AZURE_OPENAI_ENDPOINT}
      - AZURE_OPENAI_KEY=${AZURE_OPENAI_KEY}
      - AZURE_OPENAI_MODEL_MAPPER=${AZURE_OPENAI_MODEL_MAPPER}
    restart: unless-stopped
```

```bash
# Build and deploy
docker compose build
docker compose up -d

# View logs
docker compose logs -f
```

## Integration with OpenWebUI

### Configuration

1. **Add Connection**:
   - URL: `http://your-proxy:11437/v1`
   - API Key: Your Azure OpenAI key

2. **Model Selection**:
   - Use full model names: `claude-sonnet-4-5-20250929`
   - Proxy handles version stripping automatically

3. **Streaming**:
   - Enable streaming in OpenWebUI settings
   - Proxy automatically handles SSE conversion

### Known Compatibility

✅ **Working:**
- OpenWebUI streaming responses
- Hoppscotch (streaming and non-streaming)
- curl with `-N` flag
- Any OpenAI-compatible client

⚠️ **Notes:**
- System messages converted to Anthropic `system` parameter
- Temperature, max_tokens properly mapped
- Stop sequences supported

## Troubleshooting

### Check Logs

```bash
docker compose logs -f azure-oai-proxy
```

**Key Log Messages:**
- `"Model X is a Claude model - converting to Anthropic Messages API format"` - Claude detected
- `"Using new Anthropic streaming converter"` - Streaming enabled
- `"Converted to Anthropic Messages API request"` - Request converted
- `"Converted Anthropic response to OpenAI format"` - Response converted

### Common Issues

**1. Streaming Not Working**
- Verify `FlushInterval: -1` in reverse proxy
- Check Content-Type is `text/event-stream`
- Ensure no buffering proxy between client and server

**2. Model Not Found**
- Check model mapping in environment variables
- Verify deployment name in Azure AI Foundry
- Review stripped version in logs

**3. Authentication Errors**
- Verify `AZURE_OPENAI_KEY` is set
- Check Azure API permissions
- Confirm endpoint URL is correct

### Debug Mode

Enable verbose logging:
```bash
# Set in environment or .env file
LOG_LEVEL=debug
```

## Performance Tuning

### Streaming Optimization

The proxy uses:
- `FlushInterval: -1` for immediate SSE flushing
- Buffered scanners with 64KB/1MB limits for large responses
- Goroutines for concurrent stream processing

### Connection Settings

```go
// Reverse proxy configured with:
FlushInterval: -1  // Immediate flush for SSE
```

## Development

### Project Structure

```
azure-oai-proxy/
├── main.go                 # Entry point
├── pkg/
│   ├── anthropic/         # Anthropic API module
│   ├── azure/             # Azure proxy logic
│   └── openai/            # OpenAI proxy logic
├── Dockerfile
├── docker-compose.yaml
└── README.md
```

### Adding New Models

1. **Update Model Mapper**:
   ```bash
   AZURE_OPENAI_MODEL_MAPPER=new-model=deployment-name
   ```

2. **Check Detection**:
   - Claude models detected by name containing "claude"
   - O-series models use Responses API
   - Others use standard OpenAI format

### Testing

```bash
# Test streaming
curl -N http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AZURE_OPENAI_KEY" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "messages": [{"role": "user", "content": "test"}],
    "stream": true
  }'

# Test non-streaming
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AZURE_OPENAI_KEY" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "messages": [{"role": "user", "content": "test"}],
    "stream": false
  }'
```

## References

- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Azure AI Foundry Documentation](https://learn.microsoft.com/azure/ai-studio/)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)

## License

See [LICENSE](LICENSE) file for details.
