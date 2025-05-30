#!/bin/bash
set -e

echo "=== GIT PULL ==="
git pull

echo "=== BUILD ==="
go build -o /usr/local/bin/llm-queue-proxy ./app/cmd/main.go

echo "=== SYSTEMD RESTART ==="
sudo systemctl restart llm-queue-proxy.service

echo "=== SHOW JOURNAL ==="
sudo journalctl -u llm-queue-proxy.service -n 50 --no-pager