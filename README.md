# llm-queue-proxy

**llm-queue-proxy** is a smart, self-hosted proxy for LLM APIs (like OpenAI) with session-based token tracking and automatic request queueing to avoid rate limits.

Ideal for multi-agent systems, microservices, or corporate environments where multiple apps share the same API key and need token usage tracking per session.

---

## ✨ Features

- ⏳ Smart request queueing (no 429 errors)
- 🧠 Configurable rate limiting per minute
- 📊 **Session-based token tracking** - track usage across multiple requests
- 🗄️ **Pluggable storage** (Memory or SQLite) for session persistence
- 🏗️ **Dependency injection** architecture for easy testing and customization
- 🔁 Automatic retry with delay
- 🧵 Minimal threading (no Redis required)
- 🪪 Works with `systemd` as a Linux service
- 🔐 Secrets via environment variables

---

## 📦 Use Cases

- **Multi-agent systems** where you need to track token usage per session/conversation
- **Microservices** that share the same OpenAI API key
- **Rate limit management** - automatically queue requests to avoid 429 errors
- **Token usage monitoring** - track costs per session or application

---

## 🚀 Installation

```bash
git clone https://github.com/yourname/llm-queue-proxy.git
cd llm-queue-proxy

# Build the application
go build -o llm-queue-proxy ./app/cmd/main.go

# For system-wide installation
sudo cp llm-queue-proxy /usr/local/bin/
```

---

## 🛠 Configuration

### Environment Variables

```bash
# Required
OPENAI_API_KEY=sk-your-openai-api-key-here

# Optional - OpenAI API settings
OPENAI_BASE_URL=https://api.openai.com/v1  # Default
RATE_LIMIT_PER_MIN=60                       # Default

# Optional - Server settings  
PORT=8080                                   # Default

# Optional - Repository settings
REPOSITORY_TYPE=memory                      # Default: "memory" or "sqlite"
SQLITE_DSN=sessions.db                      # Default (only used if REPOSITORY_TYPE=sqlite)
```

### Configuration Examples

#### Memory Repository (Default)
```bash
export OPENAI_API_KEY=sk-your-key-here
./llm-queue-proxy
```

#### SQLite Repository (Persistent)
```bash
export OPENAI_API_KEY=sk-your-key-here
export REPOSITORY_TYPE=sqlite
export SQLITE_DSN=/var/lib/llm-queue-proxy/sessions.db
./llm-queue-proxy
```

---

## 📥 How It Works

### Session-Based Architecture
```text
[Your App] → [llm-queue-proxy] → [api.openai.com]
                ↓
    [Session Token Tracking]
    [Memory/SQLite Repository]
```

### Request Flow
1. **Session requests**: `/v1/session/{sessionID}/chat/completions`
2. **Token tracking**: Automatic extraction and accumulation per session
3. **Rate limiting**: Intelligent queueing based on configured limits
4. **Persistence**: Session data stored in memory or SQLite

---

## 🔌 API Endpoints

### Session-Based Requests (with token tracking)
```bash
# Chat completions with session tracking
curl -X POST http://localhost:8080/v1/session/my-session-123/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Session Statistics
```bash
# Get all session statistics
curl http://localhost:8080/sessions/status

# Response example:
{
  "my-session-123": {
    "session_id": "my-session-123",
    "total_prompt_tokens": 150,
    "total_completion_tokens": 200,
    "total_tokens": 350,
    "request_count": 5
  }
}
```

### Regular Requests (no session tracking)
```bash
# Direct proxy without session tracking
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{...}'
```

---

## 🏗️ Architecture

### Dependency Injection Design
```text
Dependencies
├── Config (environment variables)
├── Repository (Memory/SQLite)
├── SessionManager (tracks token usage)
├── Queue (rate limiting)
└── Handlers (HTTP endpoints)
```

### Key Components
- **Repository Interface**: Pluggable storage (Memory/SQLite)
- **Session Manager**: Token tracking and session lifecycle
- **Queue**: Rate-limited request processing
- **Handlers**: HTTP request processing with dependency injection

---

## 🧪 Development & Testing

### Running Tests
```bash
go test ./...
```

### Local Development
```bash
# Set environment variables
export OPENAI_API_KEY=sk-your-key-here
export REPOSITORY_TYPE=memory
export PORT=8080

# Run the application
go run ./app/cmd/main.go
```

### Adding SQLite Support
To use SQLite persistence, add the driver import to your main.go:
```go
import _ "github.com/mattn/go-sqlite3"
```

And install the dependency:
```bash
go get github.com/mattn/go-sqlite3
```

---

## 🐳 Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o llm-queue-proxy ./app/cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/llm-queue-proxy .
CMD ["./llm-queue-proxy"]
```

---

## 🎯 Use Case Examples

### Multi-Agent System
```bash
# Agent 1
curl -X POST http://localhost:8080/v1/session/agent-1/chat/completions ...

# Agent 2  
curl -X POST http://localhost:8080/v1/session/agent-2/chat/completions ...

# Check token usage per agent
curl http://localhost:8080/sessions/status
```

### Cost Tracking per Customer
```bash
# Customer A's requests
curl -X POST http://localhost:8080/v1/session/customer-a/chat/completions ...

# Customer B's requests
curl -X POST http://localhost:8080/v1/session/customer-b/chat/completions ...

# Get usage breakdown
curl http://localhost:8080/sessions/status
```

---

## 📝 License
MIT — free to use and modify.
