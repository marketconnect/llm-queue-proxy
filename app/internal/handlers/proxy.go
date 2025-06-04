package handlers

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

type Queue interface {
	Push(r entities.ProxyRequest) entities.ProxyResponse
}

type ProxySessionManager interface {
	GetSession(sessionID string) (*entities.SessionData, error)
	CreateSession(sessionID string) (*entities.SessionData, error)
	ListSessions() (map[string]*entities.SessionData, error)
	UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error)
	ParseTokenUsageFromResponse(responseBody []byte) (*entities.TokenUsage, error)
}

// ProxyHandler handles both regular and session-based requests
type ProxyHandler struct {
	sessionManager ProxySessionManager
	queue          Queue
}

// NewProxyHandler creates a new ProxyHandler with injected dependencies
func NewProxyHandler(sessionManager ProxySessionManager, queue Queue) *ProxyHandler {
	return &ProxyHandler{
		sessionManager: sessionManager,
		queue:          queue,
	}
}

// Handle processes the HTTP request
func (ph *ProxyHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request for: %s", r.URL.String())
	for k, v := range r.Header {
		for _, val := range v {
			log.Printf("Header: %s: %s", k, val)
		}
	}

	// Check if this is a session-based request
	sessionID := extractSessionID(r.URL.Path)
	log.Printf("Path: %s", r.URL.Path)

	if sessionID != "" {
		log.Printf("Extracted session ID: %s", sessionID)

		// Validate that there's an endpoint after the session ID
		upstreamPath := removeSessionFromPath(r.URL.Path)
		if upstreamPath == "/v1/" {
			http.Error(w, "Missing OpenAI endpoint. Use format: /v1/session/{sessionID}/chat/completions", http.StatusBadRequest)
			return
		}

		// Get or create session
		_, errSess := ph.sessionManager.GetSession(sessionID)
		if errSess != nil {
			if errors.Is(errSess, entities.ErrSessionNotFound) {
				_, errSess = ph.sessionManager.CreateSession(sessionID)
				if errSess != nil {
					log.Printf("Error creating session %s: %v", sessionID, errSess)
					http.Error(w, "Failed to initialize session", http.StatusInternalServerError)
					return
				}
				log.Printf("Created new session: %s", sessionID)
			} else {
				log.Printf("Error retrieving session %s: %v", sessionID, errSess)
				http.Error(w, "Failed to retrieve session", http.StatusInternalServerError)
				return
			}
		}
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

	req := entities.ProxyRequest{
		Reply:   make(chan entities.ProxyResponse, 1),
		Method:  r.Method,
		Path:    upstreamPath,
		Headers: r.Header.Clone(),
		Body:    body,
	}

	resp := ph.queue.Push(req)
	if resp.Err != nil {
		http.Error(w, "Proxy error: "+resp.Err.Error(), http.StatusBadGateway)
		return
	}

	// Decompress response body if it's gzipped for token parsing
	var responseBodyForParsing []byte
	if sessionID != "" && ph.sessionManager != nil {
		// Check if response is gzipped
		contentEncoding := resp.Headers.Get("Content-Encoding")
		if strings.Contains(strings.ToLower(contentEncoding), "gzip") {
			// Decompress for token parsing
			reader, err := gzip.NewReader(bytes.NewReader(resp.Body))
			if err != nil {
				log.Printf("Error creating gzip reader: %v", err)
				responseBodyForParsing = resp.Body
			} else {
				decompressed, err := io.ReadAll(reader)
				reader.Close()
				if err != nil {
					log.Printf("Error decompressing response: %v", err)
					responseBodyForParsing = resp.Body
				} else {
					responseBodyForParsing = decompressed
					log.Printf("Decompressed response body: %s", string(responseBodyForParsing))
				}
			}
		} else {
			responseBodyForParsing = resp.Body
			log.Printf("Response body from upstream: %s", string(responseBodyForParsing))
		}

		// Parse token usage from decompressed response
		if tokenUsage, err := ph.sessionManager.ParseTokenUsageFromResponse(responseBodyForParsing); err == nil && tokenUsage != nil {
			updatedSession, errUpdate := ph.sessionManager.UpdateSessionTokens(sessionID, *tokenUsage)
			if errUpdate != nil {
				log.Printf("Error updating session tokens for %s: %v", sessionID, errUpdate)
				// Potentially return an error to client, or just log and continue
			} else {
				log.Printf("Updated session %s token usage - Prompt: %d, Completion: %d, Total: %d, Requests: %d",
					sessionID, updatedSession.TotalPromptTokens, updatedSession.TotalCompletionTokens,
					updatedSession.TotalTokens, updatedSession.RequestCount)
			}
		} else if err != nil {
			log.Printf("Error parsing token usage for session %s: %v", sessionID, err)
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

// Legacy function for backward compatibility - renamed to avoid conflict
func LegacyProxyHandler(w http.ResponseWriter, r *http.Request) {
	// This would need a global session manager, but we're moving away from this pattern
	// For now, return an error indicating the new pattern should be used
	http.Error(w, "ProxyHandler requires dependency injection. Use NewProxyHandler instead.", http.StatusInternalServerError)
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
	log.Printf("Removing session from path: %s", path)

	// Pattern: /v1/session/{sessionID}/... -> /v1/...
	re := regexp.MustCompile(`^/v1/session/[^/]+(/.*)?$`)
	matches := re.FindStringSubmatch(path)

	log.Printf("Regex matches: %v", matches)

	if matches == nil {
		// If no match, return original path (fallback)
		log.Printf("No regex match, returning original path: %s", path)
		return path
	}

	// If there's a remaining path after session ID, use it; otherwise use /v1/
	if len(matches) > 1 && matches[1] != "" {
		result := "/v1" + matches[1]
		log.Printf("Transformed path: %s -> %s", path, result)
		return result
	} else {
		log.Printf("No path after session ID, returning /v1/")
		return "/v1/"
	}
}
