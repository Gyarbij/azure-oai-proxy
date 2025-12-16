# Quick Reference: Claude Models Integration

## TL;DR

Claude models on Azure AI Foundry now use **native Anthropic Messages API**. The proxy handles all format conversions automatically - you keep using OpenAI format.

## Quick Start

```bash
# 1. Set environment variable
ANTHROPIC_APIVERSION=2023-06-01

# 2. Rebuild and deploy
docker compose build
docker compose up -d --force-recreate

# 3. Test with OpenAI format (no changes needed in your code!)
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "model": "claude-sonnet-4.5",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

## Key Differences

| Aspect | OpenAI Models | Claude Models (Anthropic API) |
|--------|--------------|-------------------------------|
| **Endpoint** | `/openai/deployments/{name}/chat/completions` | `/anthropic/v1/messages` |
| **API Version** | `?api-version=2024-08-01-preview` | No query param |
| **Auth Header** | `api-key: xxx` | `x-api-key: xxx` |
| **Extra Header** | None | `anthropic-version: 2023-06-01` |
| **System Message** | In messages array | Top-level `system` parameter |
| **Streaming** | OpenAI SSE format | Anthropic SSE format (auto-converted) |
| **Required Field** | None | `max_tokens` (default: 1000) |

## Request Conversion

### What You Send (OpenAI Format)
```json
{
  "model": "claude-sonnet-4.5",
  "messages": [
    {"role": "system", "content": "You are helpful"},
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1000,
  "temperature": 0.7,
  "stream": true
}
```

### What the Proxy Sends to Azure (Anthropic Format)
```json
{
  "model": "claude-sonnet-4.5",
  "system": "You are helpful",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1000,
  "temperature": 0.7,
  "stream": true
}
```

### What You Receive (OpenAI Format)
```json
{
  "id": "msg_xxx",
  "object": "chat.completion",
  "model": "claude-sonnet-4.5",
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you?"
    },
    "finish_reason": "stop"
  }],
  "usage": {...}
}
```

## Streaming Event Conversion

| Anthropic SSE Event | Converted to OpenAI Chunk |
|---------------------|---------------------------|
| `event: message_start` | `{"delta":{"role":"assistant"}}` |
| `event: content_block_delta`<br>`data: {"delta":{"text":"Hi"}}` | `{"delta":{"content":"Hi"}}` |
| `event: message_stop` | `data: [DONE]` |

## Environment Variables

```bash
# Azure endpoint (use .services.ai.azure.com for Foundry)
AZURE_OPENAI_ENDPOINT=https://your-foundry.services.ai.azure.com/

# Anthropic API version
ANTHROPIC_APIVERSION=2023-06-01

# Optional: Model mapping if deployment name differs
AZURE_OPENAI_MODEL_MAPPER=claude-sonnet-4.5=Your-Deployment-Name
```

## Log Messages to Look For

### ✅ Success
```
Model claude-sonnet-4.5 is a Claude model - converting to Anthropic Messages API format
Using new Anthropic streaming converter for model: claude-sonnet-4.5
Converted to Anthropic Messages API request
```

### ❌ Issues
```
Error: Resource not found (404)
→ Check: Endpoint, deployment exists, model mapping

Error: Invalid API key
→ Check: API key is correct, has Foundry access

Timeout or hanging
→ Check: Old image? Run: docker compose build --no-cache
```

## Code Examples

### JavaScript/TypeScript
```typescript
const response = await fetch('http://localhost:11437/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${AZURE_API_KEY}`
  },
  body: JSON.stringify({
    model: 'claude-sonnet-4.5',
    messages: [{role: 'user', content: 'Hello'}],
    stream: true
  })
});
```

### Python
```python
from openai import OpenAI

client = OpenAI(
    api_key=AZURE_API_KEY,
    base_url="http://localhost:11437/v1"
)

response = client.chat.completions.create(
    model="claude-sonnet-4.5",
    messages=[{"role": "user", "content": "Hello"}],
    stream=True
)
```

### cURL
```bash
curl -N http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AZURE_API_KEY" \
  -d '{
    "model": "claude-sonnet-4.5",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

## Supported Claude Models

- `claude-opus-4.5` / `claude-opus-4-5`
- `claude-sonnet-4.5` / `claude-sonnet-4-5` 
- `claude-haiku-4.5` / `claude-haiku-4-5`
- `claude-opus-4.1` / `claude-opus-4-1`

## Common Issues

### "Resource not found" (404)
```bash
# Wrong endpoint type?
echo $AZURE_OPENAI_ENDPOINT
# Should be: .services.ai.azure.com (Foundry)
# Not: .openai.azure.com (standard Azure OpenAI)

# Deployment name mismatch?
AZURE_OPENAI_MODEL_MAPPER=claude-sonnet-4.5=Your-Actual-Deployment
```

### Streaming Cuts Off
```bash
# Old Docker image?
docker compose down
docker compose build --no-cache
docker compose up -d
```

### Authentication Failed
```bash
# Test direct access
curl https://your-foundry.services.ai.azure.com/anthropic/v1/messages \
  -H "x-api-key: $AZURE_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":"test"}],"max_tokens":100}'
```

## Architecture Overview

```
Client (OpenAI format)
    ↓
Proxy: anthropic.IsClaudeModel()
    ↓
Proxy: anthropic.ConvertOpenAIToAnthropic()
    ↓
Azure AI Foundry: /anthropic/v1/messages
    ↓
Anthropic SSE Response
    ↓
Proxy: anthropic.StreamConverter
    ↓
Client (OpenAI format)
```

## Files to Check

- [ARCHITECTURE.md](ARCHITECTURE.md) - Full technical details
- [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) - Upgrade instructions
- [pkg/anthropic/types.go](pkg/anthropic/types.go) - Type definitions
- [pkg/anthropic/converter.go](pkg/anthropic/converter.go) - Conversion logic
- [pkg/azure/proxy.go](pkg/azure/proxy.go) - Main proxy logic

## Testing Checklist

- [ ] Docker image rebuilt: `docker compose build`
- [ ] Deployed: `docker compose up -d --force-recreate`
- [ ] Logs checked: `docker compose logs -f`
- [ ] Non-streaming works: `"stream": false`
- [ ] Streaming works: `"stream": true`
- [ ] Complete responses (no cutoff)
- [ ] OpenWebUI integration works

## Need Help?

1. Check [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
2. Review [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) for troubleshooting
3. Check logs: `docker compose logs -f`
4. Open GitHub issue with logs and configuration (redact API keys)

---

**Remember:** You don't need to change your client code! The proxy handles all the complexity. Just rebuild, redeploy, and it works. 🎉
