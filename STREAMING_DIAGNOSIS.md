# Azure Anthropic Streaming Issue - Diagnosis

## Current Problem

OpenWebUI requests with `stream: true` are **hanging for 66+ seconds** and then failing, while Hoppscotch non-streaming requests work fine.

## Root Cause Analysis

Looking at the logs:

```
2025/12/16 10:56:15 Using Anthropic streaming converter for model: claude-sonnet-4-5-20250929
[66 SECOND HANG - NO OUTPUT]
```

This indicates:
1. Azure **IS** returning `Content-Type: text/event-stream` (otherwise converter wouldn't start)
2. The streaming converter **starts** but produces **no output**
3. The converter **hangs waiting for SSE events** that either:
   - Never arrive
   - Arrive in wrong format
   - Are not being read correctly

## Most Likely Causes

### 1. Azure Not Sending SSE Events
Azure might be returning `Content-Type: text/event-stream` but sending a **complete JSON response** instead of SSE chunks. The scanner waits forever for events that never come in SSE format.

### 2. SSE Format Mismatch
Azure might be sending SSE but with:
- Different event names than expected
- Missing `event:` prefixes
- Different line ending format (\\r\\n vs \\n)

### 3. Network/Buffering Issue
The SSE events are being buffered somewhere and not flushed, so the scanner never receives them.

## Added Diagnostic Logging

I've added extensive logging to see exactly what's happening:

1. **In modifyResponse** - logs Content-Type, status, and path
2. **In streaming converter** - logs when it starts and first 5 lines + every 100th line
3. **In proxy handler** - logs when goroutine completes

## Next Steps

### Build and Deploy
```bash
# On VM
cd ~/aoip
docker compose build
docker compose up -d --force-recreate
docker compose logs -f
```

### Test with OpenWebUI
Send the same request that hung before. Look for these new log messages:

**What to look for:**
- `"modifyResponse called - Content-Type: ..."` - Is it text/event-stream?
- `"Anthropic converter started for model: ..."` - Does converter start?
- `"Anthropic converter line 1: ..."` - What are the first few lines?
- `"Anthropic streaming converter goroutine completed"` - Does it complete?

### Expected Outputs

**If Azure returns proper SSE:**
```
Anthropic converter line 1: "event: message_start"
Anthropic converter line 2: "data: {...}"
Anthropic converter line 3: ""
```

**If Azure returns JSON instead:**
```
Anthropic converter line 1: "{\"model\":\"claude-sonnet-4-5-20250929\",\"id\":\"msg_...}"
[HANG - scanner waits for SSE format]
```

**If Content-Type is wrong:**
```
modifyResponse called - Content-Type: application/json, Status: 200
[No converter started - falls through to non-streaming path]
```

## Potential Solutions

### Solution 1: Force Non-Streaming for Anthropic
If Azure doesn't actually support streaming for Anthropic Messages API:

```go
// In convertChatToAnthropicMessages
// Remove stream parameter
// if stream {
//     newBody["stream"] = true
// }
```

### Solution 2: Add Timeout and Fallback
Add timeout to streaming converter so it doesn't hang forever:

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Use context-aware scanner
```

### Solution 3: Peek at Response
Before starting converter, peek at first bytes to see if it's SSE or JSON:

```go
// Read first 100 bytes
peekBytes := make([]byte, 100)
n, _ := res.Body.Read(peekBytes)

if bytes.HasPrefix(peekBytes, []byte("{")) {
    // It's JSON, not SSE - handle as non-streaming
}
```

## Test Commands

### From OpenWebUI (streaming request):
Will send `stream: true` automatically

### From Hoppscotch (force streaming):
```json
POST https://your-proxy/v1/chat/completions
{
  "model": "claude-sonnet-4-5-20250929",
  "messages": [{"role": "user", "content": "test"}],
  "stream": true
}
```

### From curl (streaming):
```bash
curl -N -X POST https://your-proxy/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "messages": [{"role": "user", "content": "test"}],
    "stream": true
  }'
```

The `-N` flag disables buffering in curl so you can see SSE events as they arrive.

## What I Suspect

Based on the second (non-streaming) request working fine, I believe **Azure's Anthropic Messages API does NOT support streaming**, despite accepting the `stream: true` parameter. It returns:
- `Content-Type: text/event-stream` (triggering our converter)
- But sends **complete JSON response** (not SSE chunks)
- Our scanner waits for SSE format that never comes
- After 66 seconds, something times out

The fix would be to **NOT send `stream: true`** to Azure's Anthropic endpoint, and instead **simulate streaming** by chunking the complete response on the proxy side.
