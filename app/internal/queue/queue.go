package queue

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"time"

	"github.com/marketconnect/llm-queue-proxy/app/internal/config"
)

type ProxyRequest struct {
	W    http.ResponseWriter
	R    *http.Request
	Body []byte
}

var ch chan ProxyRequest
var cfg *config.Config

func Init(limitPerMin int) {
	cfg = config.GetConfig()
	ch = make(chan ProxyRequest, 1000)

	interval := time.Minute / time.Duration(limitPerMin)
	go func() {
		for req := range ch {
			time.Sleep(interval)
			go handle(req)
		}
	}()
}

func Push(r ProxyRequest) {
	ch <- r
}

func handle(p ProxyRequest) {
	ctx := context.Background()
	targetURL := cfg.OpenAI.BASE_URL + p.R.URL.Path

	req, err := http.NewRequestWithContext(ctx, p.R.Method, targetURL, bytes.NewReader(p.Body))
	if err != nil {
		http.Error(p.W, "failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header = p.R.Header.Clone()
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAI.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(p.W, "proxy request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	p.W.WriteHeader(resp.StatusCode)
	io.Copy(p.W, resp.Body)
}
