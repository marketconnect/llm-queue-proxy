# Session Tracking Features

This LLM Queue Proxy now supports session-based request tracking and token usage monitoring with a unified proxy handler.

## New Features

### 1. Unified Proxy Handler

**Single endpoint handles both:**
- Regular requests: `http://localhost:8080/v1/...`
- Session-based requests: `http://localhost:8080/v1/session/{sessionID}/...`

The handler automatically detects session-based requests and:
- Extracts session ID from the URL path
- Automatically creates or retrieves existing sessions
- Tracks token usage per session across multiple requests
- Forwards requests to upstream OpenAI API by removing session information

**Example Usage:**
```bash
# Regular request (no session tracking)
curl -X POST "http://localhost:8080/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ]
  }'

# Session-based request (with token tracking)
curl -X POST "http://localhost:8080/v1/session/user123/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ]
  }'
```

### 2. Token Usage Tracking

For session-based requests (`/v1/session/{sessionID}/...`), the proxy automatically:
- Parses OpenAI API response bodies to extract token usage
- Accumulates token counts per session:
  - `prompt_tokens`: Total prompt tokens used in session
  - `completion_tokens`: Total completion tokens generated in session  
  - `total_tokens`: Total tokens (prompt + completion) used in session
- Tracks the number of requests made per session
- Logs token usage updates for monitoring

### 3. Session Status Endpoint

#### Get Token Consumption Stats
**Endpoint:** `GET /sessions/status`

Returns all active sessions and their token usage:
```json
{
  "user123": {
    "session_id": "user123",
    "total_prompt_tokens": 150,
    "total_completion_tokens": 85,
    "total_tokens": 235,
    "request_count": 3
  },
  "session456": {
    "session_id": "session456", 
    "total_prompt_tokens": 75,
    "total_completion_tokens": 42,
    "total_tokens": 117,
    "request_count": 1
  }
}
```

## Implementation Details

### Unified Handler Logic
- Single `ProxyHandler` function handles all `/v1/...` requests
- Automatically detects session-based URLs using regex pattern `/v1/session/([^/]+)`
- Session tracking is enabled only for URLs matching the session pattern
- Regular requests bypass session tracking entirely

### Session Management
- Thread-safe session storage using `sync.RWMutex`
- Automatic session creation on first request
- In-memory storage (sessions are lost on server restart)
- Unique session identification via URL path extraction

### Token Parsing
- Parses JSON response bodies from OpenAI API
- Extracts `usage.prompt_tokens`, `usage.completion_tokens`, and `usage.total_tokens`
- Handles responses without usage data gracefully
- Accumulates token counts across multiple requests
- Only processes token usage for session-based requests

### Path Processing
- Extracts session ID using regex: `/v1/session/([^/]+)`
- Removes session information before forwarding to OpenAI API
- Example: `/v1/session/user123/chat/completions` → `/v1/chat/completions`
- Regular requests: `/v1/chat/completions` → `/v1/chat/completions` (unchanged)

### Backwards Compatibility
- All existing `/v1/...` endpoints work exactly as before
- No breaking changes to existing API usage
- Session tracking is opt-in via URL structure

## Configuration

No additional configuration required. The session tracking features work with existing configuration.

## Monitoring and Logging

- Session creation events are logged
- Token usage updates are logged with detailed information for session-based requests
- Parsing errors are logged for debugging
- Request processing is logged for all requests

## Example Flows

### Regular Request (No Session)
```
POST /v1/chat/completions
→ Forwards to OpenAI: POST /v1/chat/completions
→ Returns response (no token tracking)
```

### Session-Based Request Flow
1. **First Request:** 
   ```
   POST /v1/session/user123/chat/completions
   → Creates session "user123"
   → Forwards to OpenAI: POST /v1/chat/completions
   → Tracks tokens: prompt=10, completion=15, total=25
   ```

2. **Second Request:**
   ```
   POST /v1/session/user123/chat/completions  
   → Uses existing session "user123"
   → Forwards to OpenAI: POST /v1/chat/completions
   → Updates totals: prompt=25, completion=35, total=60
   ```

3. **Status Check:**
   ```
   GET /sessions/status
   → Returns: {"user123": {"total_tokens": 60, "request_count": 2, ...}}
   ``` 