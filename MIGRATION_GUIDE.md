# Migration Guide: Upgrading to Native Anthropic API

## Overview

If you're upgrading from an older version of azure-oai-proxy that supported Claude models, this guide will help you understand the changes and ensure a smooth transition.

## What Changed?

### Previous Architecture (Before)
- Claude models were treated like OpenAI models
- Routed to `/openai/deployments/{deployment-name}/chat/completions`
- Used Azure OpenAI API version query parameters
- Expected OpenAI-formatted responses
- Streaming often failed or got cut off

### New Architecture (Now)
- Claude models use **native Anthropic Messages API**
- Route to `/anthropic/v1/messages` (no deployment in path, no api-version query param)
- Use Anthropic-specific authentication (`x-api-key`, `anthropic-version`)
- Automatic bidirectional format conversion (OpenAI ↔ Anthropic)
- Streaming works reliably with proper SSE event handling

## Breaking Changes

### 1. Endpoint Routing
**Before:**
```
POST https://your-foundry.services.ai.azure.com/openai/deployments/claude-sonnet-45/chat/completions?api-version=2024-08-01-preview
```

**After:**
```
POST https://your-foundry.services.ai.azure.com/anthropic/v1/messages
```

### 2. Authentication Headers
**Before:**
```
api-key: your-azure-api-key
```

**After:**
```
x-api-key: your-azure-api-key
anthropic-version: 2023-06-01
```

### 3. System Messages
**Before:**
```json
{
  "messages": [
    {"role": "system", "content": "You are helpful"},
    {"role": "user", "content": "Hello"}
  ]
}
```

**After (internal conversion):**
```json
{
  "system": "You are helpful",
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

### 4. Streaming Events
**Before (expected but didn't work):**
```
data: {"choices":[{"delta":{"content":"Hello"}}]}
```

**After (Anthropic SSE → converted to OpenAI):**
```
event: message_start
data: {"type":"message_start","message":{...}}

event: content_block_delta
data: {"type":"content_block_delta","delta":{"text":"Hello"}}

[Converted to OpenAI format automatically]
```

## What You Need to Do

### ✅ Required Changes

1. **Update Environment Variables** (if not already set):
   ```bash
   ANTHROPIC_APIVERSION=2023-06-01
   ```

2. **Rebuild and Deploy**:
   ```bash
   docker compose build
   docker compose up -d --force-recreate
   ```

3. **Verify Endpoint** (if using Azure AI Foundry):
   ```bash
   AZURE_OPENAI_ENDPOINT=https://your-foundry.services.ai.azure.com/
   # NOT .openai.azure.com (that's for standard Azure OpenAI)
   ```

### ✅ Optional Changes

1. **Review Model Mappings** (if using custom deployment names):
   ```bash
   # Example: If your deployment is named "Claude-Sonnet-45-20251001"
   AZURE_OPENAI_MODEL_MAPPER=claude-sonnet-4.5=Claude-Sonnet-45-20251001
   ```

2. **Update Client Code** (if calling proxy directly):
   - No changes needed! Continue using OpenAI format
   - The proxy handles all conversions automatically

## What Stays the Same

### ✅ No Changes Needed For:

1. **Client Applications**:
   - OpenWebUI - no changes needed
   - Hoppscotch - no changes needed
   - Custom clients using OpenAI SDK - no changes needed

2. **API Request Format**:
   ```bash
   # You still send OpenAI format
   curl http://localhost:11437/v1/chat/completions \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $API_KEY" \
     -d '{
       "model": "claude-sonnet-4.5",
       "messages": [{"role": "user", "content": "Hello"}]
     }'
   ```

3. **API Response Format**:
   ```json
   {
     "id": "msg_xxx",
     "object": "chat.completion",
     "choices": [{
       "message": {
         "role": "assistant",
         "content": "Hello! How can I help?"
       }
     }]
   }
   ```

## Testing Your Migration

### 1. Check Logs
After deploying, look for these log messages:

✅ **Success indicators:**
```
Model claude-sonnet-4.5 is a Claude model - converting to Anthropic Messages API format
Using new Anthropic streaming converter for model: claude-sonnet-4.5
Converted to Anthropic Messages API request: {system:..., messages:...}
```

❌ **Problem indicators:**
```
Error: Resource not found
Error: 404 Not Found
Timeout waiting for response
```

### 2. Test Streaming
Send a streaming request and verify:
- ✅ Response starts immediately (not after 60+ seconds)
- ✅ Content streams in real-time
- ✅ Complete response received (no cutoff)
- ✅ Ends with `data: [DONE]`

### 3. Test Non-Streaming
Send a non-streaming request and verify:
- ✅ Response received quickly
- ✅ Complete content in response
- ✅ Usage statistics included

## Troubleshooting

### Issue: "Resource not found" (404)

**Possible Causes:**
1. Using wrong endpoint (`.openai.azure.com` instead of `.services.ai.azure.com`)
2. Claude model not deployed in your Azure AI Foundry account
3. Incorrect deployment name

**Solutions:**
```bash
# 1. Verify endpoint
echo $AZURE_OPENAI_ENDPOINT
# Should be: https://xxx.services.ai.azure.com/

# 2. Check deployment exists in Azure portal

# 3. Use model mapper if deployment name differs
AZURE_OPENAI_MODEL_MAPPER=claude-sonnet-4.5=Your-Deployment-Name
```

### Issue: Streaming Still Cuts Off

**Possible Causes:**
1. Old Docker image still running
2. Environment variables not updated
3. Client-side timeout

**Solutions:**
```bash
# 1. Force rebuild and recreate
docker compose down
docker compose build --no-cache
docker compose up -d

# 2. Check environment variables
docker compose exec azure-oai-proxy env | grep ANTHROPIC

# 3. Check client timeout settings
# In OpenWebUI: Settings → Advanced → Request Timeout
```

### Issue: "Invalid API key"

**Possible Causes:**
1. API key format changed (now uses `x-api-key` header internally)
2. Wrong API key for Azure AI Foundry

**Solutions:**
```bash
# Verify your API key works with curl
curl https://your-foundry.services.ai.azure.com/anthropic/v1/messages \
  -H "x-api-key: $AZURE_OPENAI_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4.5",
    "messages": [{"role": "user", "content": "test"}],
    "max_tokens": 100
  }'
```

## Rollback Plan

If you need to rollback to the previous version:

```bash
# 1. Stop current version
docker compose down

# 2. Use previous image tag
# In compose.yaml, change:
image: 'gyarbij/azure-oai-proxy:latest'
# To:
image: 'gyarbij/azure-oai-proxy:v1.0.8'  # or whatever version you were using

# 3. Start old version
docker compose up -d

# Note: Claude streaming may not work in older versions
```

## Additional Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) - Complete technical documentation
- [README.md](README.md) - User guide and examples
- [CHANGELOG.md](CHANGELOG.md) - Detailed version history
- [Anthropic API Docs](https://docs.anthropic.com/en/api/messages) - Official Anthropic documentation

## Support

If you encounter issues:

1. Check logs: `docker compose logs -f`
2. Review [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
3. Verify configuration in [example.env](example.env)
4. Open an issue on GitHub with:
   - Your configuration (redact API keys)
   - Log output
   - Steps to reproduce

## Summary

**The good news:** Your client code doesn't need to change! The proxy handles all the complexity of:
- Converting request formats
- Routing to correct endpoints
- Handling authentication headers
- Converting streaming events
- Formatting responses

**The only thing you need to do:** Rebuild and redeploy the proxy with the latest code.

That's it! 🎉
