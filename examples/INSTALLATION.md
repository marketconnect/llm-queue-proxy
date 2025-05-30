# Installation Guide

This guide shows how to install and configure llm-queue-proxy as a systemd service on Linux.

## Prerequisites

- Go 1.21+ (for building)
- Linux with systemd
- OpenAI API key

## 1. Build and Install

```bash
# Clone the repository
git clone https://github.com/yourname/llm-queue-proxy.git
cd llm-queue-proxy

# Build the application
go build -o llm-queue-proxy ./app/cmd/main.go

# Install binary
sudo cp llm-queue-proxy /usr/local/bin/
sudo chmod +x /usr/local/bin/llm-queue-proxy
```

## 2. Create User and Directories

```bash
# Create dedicated user for security
sudo useradd --system --no-create-home --shell /usr/sbin/nologin llm-proxy

# Create directories
sudo mkdir -p /etc/llm-queue-proxy
sudo mkdir -p /var/lib/llm-queue-proxy
sudo mkdir -p /var/log/llm-queue-proxy

# Set ownership
sudo chown llm-proxy:llm-proxy /var/lib/llm-queue-proxy
sudo chown llm-proxy:llm-proxy /var/log/llm-queue-proxy
```

## 3. Configuration

```bash
# Copy configuration template
sudo cp examples/llm-queue-proxy.env /etc/llm-queue-proxy/proxy.env

# Edit configuration
sudo nano /etc/llm-queue-proxy/proxy.env
```

### Configuration Examples

#### Memory Repository (Default)
```bash
# Required Configuration
OPENAI_API_KEY=sk-your-actual-openai-key-here

# OpenAI API Configuration
OPENAI_BASE_URL=https://api.openai.com/v1
RATE_LIMIT_PER_MIN=60

# Server Configuration
PORT=8080

# Repository Configuration (Memory - non-persistent)
REPOSITORY_TYPE=memory
```

#### SQLite Repository (Persistent)
```bash
# Required Configuration
OPENAI_API_KEY=sk-your-actual-openai-key-here

# OpenAI API Configuration
OPENAI_BASE_URL=https://api.openai.com/v1
RATE_LIMIT_PER_MIN=60

# Server Configuration
PORT=8080

# Repository Configuration (SQLite - persistent)
REPOSITORY_TYPE=sqlite
SQLITE_DSN=/var/lib/llm-queue-proxy/sessions.db
```

### Secure the Configuration
```bash
sudo chown root:llm-proxy /etc/llm-queue-proxy/proxy.env
sudo chmod 640 /etc/llm-queue-proxy/proxy.env
```

## 4. Install systemd Service

```bash
# Copy service file
sudo cp examples/llm-queue-proxy.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable and start service
sudo systemctl enable llm-queue-proxy
sudo systemctl start llm-queue-proxy
```

## 5. Verify Installation

```bash
# Check service status
sudo systemctl status llm-queue-proxy

# Check logs
sudo journalctl -u llm-queue-proxy -f

# Test the service
curl http://localhost:8080/sessions/status
```

## 6. Testing with Session Tracking

```bash
# Test session-based request (replace with your API key)
curl -X POST http://localhost:8080/v1/session/test-session/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-openai-key-here" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello, world!"}],
    "max_tokens": 50
  }'

# Check session statistics
curl http://localhost:8080/sessions/status
```

## 7. Monitoring and Maintenance

### View Logs
```bash
# Real-time logs
sudo journalctl -u llm-queue-proxy -f

# Recent logs
sudo journalctl -u llm-queue-proxy -n 100

# Logs for specific date
sudo journalctl -u llm-queue-proxy --since "2024-01-01"
```

### Service Management
```bash
# Start service
sudo systemctl start llm-queue-proxy

# Stop service
sudo systemctl stop llm-queue-proxy

# Restart service
sudo systemctl restart llm-queue-proxy

# Reload configuration (restart required)
sudo systemctl restart llm-queue-proxy

# Check status
sudo systemctl status llm-queue-proxy
```

### Database Maintenance (SQLite only)
```bash
# Check database size
sudo du -h /var/lib/llm-queue-proxy/sessions.db

# Backup database
sudo cp /var/lib/llm-queue-proxy/sessions.db /var/lib/llm-queue-proxy/sessions.db.backup.$(date +%Y%m%d)

# View database contents (optional)
sudo sqlite3 /var/lib/llm-queue-proxy/sessions.db "SELECT * FROM sessions;"
```

## 8. Troubleshooting

### Common Issues

#### Service fails to start
```bash
# Check detailed logs
sudo journalctl -u llm-queue-proxy -e

# Check configuration syntax
sudo -u llm-proxy /usr/local/bin/llm-queue-proxy
```

#### Permission denied errors
```bash
# Fix ownership
sudo chown -R llm-proxy:llm-proxy /var/lib/llm-queue-proxy
sudo chmod 755 /var/lib/llm-queue-proxy
```

#### SQLite database issues
```bash
# Check if SQLite file is writable
sudo -u llm-proxy touch /var/lib/llm-queue-proxy/sessions.db
sudo -u llm-proxy sqlite3 /var/lib/llm-queue-proxy/sessions.db "SELECT 1;"
```

### Performance Tuning

#### High Traffic Environments
```bash
# Increase rate limits in proxy.env
RATE_LIMIT_PER_MIN=180

# Increase file descriptor limits in service file
LimitNOFILE=131072
```

#### Memory Usage Optimization
```bash
# For memory repository - sessions are lost on restart but use less disk
REPOSITORY_TYPE=memory

# For persistent sessions with automatic cleanup
REPOSITORY_TYPE=sqlite
SQLITE_DSN=/var/lib/llm-queue-proxy/sessions.db
```

## 9. Security Considerations

- ✅ Dedicated user account (`llm-proxy`)
- ✅ Restricted file permissions
- ✅ systemd security features enabled
- ✅ No root privileges required
- ✅ Environment variables for secrets
- ⚠️ Consider firewall rules for port 8080
- ⚠️ Consider reverse proxy (nginx) for TLS

## 10. Uninstallation

```bash
# Stop and disable service
sudo systemctl stop llm-queue-proxy
sudo systemctl disable llm-queue-proxy

# Remove files
sudo rm /etc/systemd/system/llm-queue-proxy.service
sudo rm -rf /etc/llm-queue-proxy
sudo rm -rf /var/lib/llm-queue-proxy
sudo rm -rf /var/log/llm-queue-proxy
sudo rm /usr/local/bin/llm-queue-proxy

# Remove user
sudo userdel llm-proxy

# Reload systemd
sudo systemctl daemon-reload
``` 