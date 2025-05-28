package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/marketconnect/llm-queue-proxy/app/internal/queue"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request for: %s", r.URL.String())
	// ...

	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(io.NopCloser(nil)) // prevent reuse

	log.Printf("Got response: %s", string(body))
	queue.Push(queue.ProxyRequest{W: w, R: r, Body: body})
}
