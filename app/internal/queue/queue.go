package queue

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

// Queue handles request queueing and rate limiting
type Queue struct {
	ch           chan entities.ProxyRequest
	baseURL      string
	openAIAPIKey string
	closed       bool
	mu           sync.Mutex
}

// NewQueue creates a new queue with injected config
func NewQueue(limitPerMin int, baseURL string, openAIAPIKey string) *Queue {
	q := &Queue{
		ch:           make(chan entities.ProxyRequest, 1000),
		baseURL:      baseURL,
		openAIAPIKey: openAIAPIKey,
		closed:       false,
	}

	if limitPerMin <= 0 {
		log.Printf("Warning: RateLimitPerMin is %d, which is invalid. Defaulting to 60.", limitPerMin)
		limitPerMin = 60 // Default to a sensible value
	}

	interval := time.Minute / time.Duration(limitPerMin)
	go func() {
		for req := range q.ch {
			time.Sleep(interval)
			go q.handle(req)
		}
	}()

	return q
}

// Push adds a request to the queue and returns the response
func (q *Queue) Push(r entities.ProxyRequest) entities.ProxyResponse {
	r.Reply = make(chan entities.ProxyResponse, 1)
	q.ch <- r
	return <-r.Reply
}

// Close gracefully shuts down the queue
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		close(q.ch)
		q.closed = true
	}
}

func (q *Queue) handle(p entities.ProxyRequest) {
	ctx := context.Background()
	targetURL := q.baseURL + p.Path

	log.Printf("Forwarding request to upstream URL: %s", targetURL)
	log.Printf("Request method: %s", p.Method)
	log.Printf("Request body length: %d bytes", len(p.Body))

	req, err := http.NewRequestWithContext(ctx, p.Method, targetURL, bytes.NewReader(p.Body))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		p.Reply <- entities.ProxyResponse{Err: err}
		return
	}

	// Initialize headers if nil
	if p.Headers == nil {
		p.Headers = make(http.Header)
	}
	req.Header = p.Headers.Clone()
	req.Header.Set("Authorization", "Bearer "+q.openAIAPIKey)

	log.Printf("Making request to %s", targetURL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		p.Reply <- entities.ProxyResponse{Err: err}
		return
	}
	defer resp.Body.Close()

	log.Printf("Received response with status: %d", resp.StatusCode)
	log.Printf("Response headers: %v", resp.Header)

	respBody, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		log.Printf("Error reading response body: %v", errRead)
		p.Reply <- entities.ProxyResponse{
			StatusCode: http.StatusBadGateway, // Or resp.StatusCode if headers are still relevant
			Headers:    resp.Header.Clone(),
			Body:       nil,
			Err:        fmt.Errorf("failed to read upstream response body: %w", errRead),
		}
		return
	}

	p.Reply <- entities.ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       respBody,
	}
}
