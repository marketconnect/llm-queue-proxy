package handlers

import (
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/marketconnect/llm-queue-proxy/app/internal/queue"
	"github.com/marketconnect/llm-queue-proxy/app/internal/session"
)

// ProxyHandler handles both regular and session-based requests
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request for: %s", r.URL.String())

	// Check if this is a session-based request
	sessionID := extractSessionID(r.URL.Path)
	var sessionManager *session.SessionManager
	var sessionData *session.SessionData

	if sessionID != "" {
		log.Printf("Extracted session ID: %s", sessionID)

		// Validate that there's an endpoint after the session ID
		upstreamPath := removeSessionFromPath(r.URL.Path)
		if upstreamPath == "/v1/" {
			http.Error(w, "Missing OpenAI endpoint. Use format: /v1/session/{sessionID}/chat/completions", http.StatusBadRequest)
			return
		}

		// Get or create session
		sessionManager = session.GetManager()
		sessionData = sessionManager.GetSession(sessionID)
		if sessionData == nil {
			_ = sessionManager.CreateSession(sessionID)
			log.Printf("Created new session: %s", sessionID)
		}
	}

	for k, v := range r.Header {
		log.Printf("Header: %s: %s", k, v)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Request body: %s", string(body))

	// Determine the upstream path
	var upstreamPath string
	if sessionID != "" {
		// Remove session ID from path for upstream request
		upstreamPath = removeSessionFromPath(r.URL.Path)
	} else {
		// Use original path for regular requests
		upstreamPath = r.URL.Path
	}

	req := queue.ProxyRequest{
		Method:  r.Method,
		Path:    upstreamPath,
		Headers: r.Header.Clone(),
		Body:    body,
	}

	resp := queue.Push(req)
	if resp.Err != nil {
		http.Error(w, "Proxy error: "+resp.Err.Error(), http.StatusBadGateway)
		return
	}

	// Log the response body before parsing (for debugging)
	log.Printf("Response body from upstream: %s", string(resp.Body))

	// Parse token usage from response if this is a session-based request
	if sessionID != "" && sessionManager != nil {
		if tokenUsage, err := session.ParseTokenUsageFromResponse(resp.Body); err == nil && tokenUsage != nil {
			updatedSession := sessionManager.UpdateSessionTokens(sessionID, *tokenUsage)
			log.Printf("Updated session %s token usage - Prompt: %d, Completion: %d, Total: %d, Requests: %d",
				sessionID, updatedSession.TotalPromptTokens, updatedSession.TotalCompletionTokens,
				updatedSession.TotalTokens, updatedSession.RequestCount)
		} else if err != nil {
			log.Printf("Error parsing token usage: %v", err)
		}
	}

	for k, v := range resp.Headers {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(resp.Body)
}

// extractSessionID extracts session ID from URL path like /v1/session/{sessionID}/chat/completions
func extractSessionID(path string) string {
	// Pattern: /v1/session/{sessionID}/...
	re := regexp.MustCompile(`^/v1/session/([^/]+)`)
	matches := re.FindStringSubmatch(path)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// removeSessionFromPath removes the session part from the path for upstream request
// e.g., /v1/session/abc123/chat/completions -> /v1/chat/completions
func removeSessionFromPath(path string) string {
	// Pattern: /v1/session/{sessionID}/... -> /v1/...
	re := regexp.MustCompile(`^/v1/session/[^/]+(/.*)?$`)
	matches := re.FindStringSubmatch(path)
	if len(matches) < 2 {
		// If no match, return original path (fallback)
		return path
	}

	// If there's a remaining path after session ID, use it; otherwise use /v1/
	if matches[1] != "" {
		return "/v1" + matches[1]
	}
	return "/v1/"
}
