package handlers

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/marketconnect/llm-queue-proxy/app/internal/queue"
	"github.com/marketconnect/llm-queue-proxy/app/internal/session"
)

// ProxyHandler handles both regular and session-based requests
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request for: %s", r.URL.String())

	// Check if this is a session-based request
	sessionID := extractSessionID(r.URL.Path)
	log.Printf("Path: %s", r.URL.Path)
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

	// Decompress response body if it's gzipped for token parsing
	var responseBodyForParsing []byte
	if sessionID != "" && sessionManager != nil {
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
		if tokenUsage, err := session.ParseTokenUsageFromResponse(responseBodyForParsing); err == nil && tokenUsage != nil {
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
	log.Printf("Removing session from path: %s", path)

	// Pattern: /v1/session/{sessionID}/... -> /v1/...
	re := regexp.MustCompile(`^/v1/session/[^/]+(/.*)?$`)
	matches := re.FindStringSubmatch(path)

	log.Printf("Regex matches: %v", matches)

	if len(matches) < 1 {
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
