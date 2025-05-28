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

	http.HandleFunc("/v1/", handlers.ProxyHandler)

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
