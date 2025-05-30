package queue_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
	"github.com/marketconnect/llm-queue-proxy/app/internal/queue"
)

func TestQueue_PushAndHandle(t *testing.T) {
	var upstreamCalled bool
	var requestPath string
	var requestMethod string
	var requestBody string
	var authHeader string

	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		requestPath = r.URL.Path
		requestMethod = r.Method
		bodyBytes, _ := io.ReadAll(r.Body)
		requestBody = string(bodyBytes)
		authHeader = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response":"ok"}`))
	}))
	defer mockUpstream.Close()

	q := queue.NewQueue(60, mockUpstream.URL, "test-api-key") // 60 requests per minute
	defer q.Close()

	proxyReq := entities.ProxyRequest{
		Method:  http.MethodPost,
		Path:    "/v1/test",
		Headers: http.Header{"X-Custom-Header": []string{"custom-value"}},
		Body:    []byte(`{"input":"hello"}`),
	}

	resp := q.Push(proxyReq)

	if resp.Err != nil {
		t.Fatalf("Push returned an error: %v", resp.Err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if string(resp.Body) != `{"response":"ok"}` {
		t.Errorf("Expected body %s, got %s", `{"response":"ok"}`, string(resp.Body))
	}
	if resp.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got '%s'", resp.Headers.Get("Content-Type"))
	}

	// Wait a bit for the handler goroutine to process
	time.Sleep(100 * time.Millisecond) // Adjust if needed, though ideally Push should block until response

	if !upstreamCalled {
		t.Error("Upstream server was not called")
	}
	if requestPath != "/v1/test" {
		t.Errorf("Expected request path '/v1/test', got '%s'", requestPath)
	}
	if requestMethod != http.MethodPost {
		t.Errorf("Expected request method 'POST', got '%s'", requestMethod)
	}
	if requestBody != `{"input":"hello"}` {
		t.Errorf("Expected request body '%s', got '%s'", `{"input":"hello"}`, requestBody)
	}
	if authHeader != "Bearer test-api-key" {
		t.Errorf("Expected Authorization header 'Bearer test-api-key', got '%s'", authHeader)
	}
	// Check if custom header was passed (Note: http.DefaultClient might strip some headers)
	// This test setup doesn't explicitly check for X-Custom-Header on the server side,
	// but req.Header = p.Headers.Clone() in queue.go should preserve it.
}

func TestQueue_RateLimitingConcept(t *testing.T) {
	// This test demonstrates sequential processing due to rate limiting,
	// not precise timing of the rate limit itself.
	var callCount int
	var mu sync.Mutex

	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond) // Simulate work
		w.WriteHeader(http.StatusOK)
	}))
	defer mockUpstream.Close()

	// High rate limit for test speed, but interval will still enforce some delay
	q := queue.NewQueue(1200, mockUpstream.URL, "test-api-key") // 20 reqs/sec
	defer q.Close()

	numRequests := 3
	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Push(entities.ProxyRequest{Path: "/test"})
		}()
	}
	wg.Wait() // Wait for all Push calls to complete (meaning their responses are received)

	if callCount != numRequests {
		t.Errorf("Expected %d calls to upstream, got %d", numRequests, callCount)
	}
}

func TestNewQueue_InvalidRateLimit(t *testing.T) {
	// Test that NewQueue defaults RateLimitPerMin if 0 or negative.
	// This is hard to verify without inspecting internal state or observing behavior.
	// The log "Warning: RateLimitPerMin is %d..." indicates it.
	// For this test, we'll just ensure it doesn't panic.
	q := queue.NewQueue(0, "http://localhost:1234", "test-key")
	if q == nil {
		t.Fatal("NewQueue returned nil for 0 rate limit")
	}
	q.Close()

	q = queue.NewQueue(-10, "http://localhost:1234", "test-key")
	if q == nil {
		t.Fatal("NewQueue returned nil for negative rate limit")
	}
	q.Close()
}
