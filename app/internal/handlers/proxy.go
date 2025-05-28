package handlers

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/marketconnect/llm-queue-proxy/app/internal/queue"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request for: %s", r.URL.String())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Request body: %s", string(body))

	// Восстанавливаем тело для повторного использования
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	queue.Push(queue.ProxyRequest{W: w, R: r, Body: body})
}
