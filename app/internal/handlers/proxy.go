package handlers

import (
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

	req := queue.ProxyRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: r.Header.Clone(),
		Body:    body,
	}

	resp := queue.Push(req)
	if resp.Err != nil {
		http.Error(w, "Proxy error: "+resp.Err.Error(), http.StatusBadGateway)
		return
	}

	for k, v := range resp.Headers {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(resp.Body)
}
