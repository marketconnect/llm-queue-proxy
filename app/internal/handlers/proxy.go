package handlers

import (
	"io"
	"net/http"

	"github.com/marketconnect/llm-queue-proxy/app/internal/queue"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(io.NopCloser(nil)) // prevent reuse

	queue.Push(queue.ProxyRequest{W: w, R: r, Body: body})
}
