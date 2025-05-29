package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/marketconnect/llm-queue-proxy/app/internal/config"
	"github.com/marketconnect/llm-queue-proxy/app/internal/handlers"
	"github.com/marketconnect/llm-queue-proxy/app/internal/queue"
)

func Run() {
	cfg := config.GetConfig()

	queue.Init(cfg.OpenAI.RateLimitPerMin)

	// Unified proxy handler for both regular and session-based requests
	http.HandleFunc("/v1/", handlers.ProxyHandler)

	// Session status endpoints for token consumption stats
	http.HandleFunc("/sessions/status", handlers.SessionStatusPathHandler)

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Available endpoints:")
	log.Printf("  - Proxy (regular): /v1/...")
	log.Printf("  - Proxy (session): /v1/session/{sessionID}/...")
	log.Printf("  - Session stats: /sessions/status")
	log.Fatal(http.ListenAndServe(addr, nil))
}
