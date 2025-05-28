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
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
	Reply   chan ProxyResponse
}

type ProxyResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Err        error
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

func Push(r ProxyRequest) ProxyResponse {
	r.Reply = make(chan ProxyResponse, 1)
	ch <- r
	return <-r.Reply
}

func handle(p ProxyRequest) {
	ctx := context.Background()
	targetURL := cfg.OpenAI.BASE_URL + p.Path

	req, err := http.NewRequestWithContext(ctx, p.Method, targetURL, bytes.NewReader(p.Body))
	if err != nil {
		p.Reply <- ProxyResponse{Err: err}
		return
	}

	req.Header = p.Headers.Clone()
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAI.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		p.Reply <- ProxyResponse{Err: err}
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	p.Reply <- ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       respBody,
	}
}
