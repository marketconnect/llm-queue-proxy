package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/marketconnect/llm-queue-proxy/app/internal/session"
)

// SessionStatusHandler handles GET requests to view session token usage
func SessionStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionManager := session.GetManager()

	// Check if specific session ID is requested: /v1/session/{sessionID}/status
	sessionID := extractSessionID(r.URL.Path)

	w.Header().Set("Content-Type", "application/json")

	if sessionID != "" {
		// Return specific session data
		sessionData := sessionManager.GetSession(sessionID)
		if sessionData == nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(sessionData); err != nil {
			log.Printf("Error encoding session data: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		// Return all sessions
		allSessions := sessionManager.ListSessions()
		if err := json.NewEncoder(w).Encode(allSessions); err != nil {
			log.Printf("Error encoding sessions data: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// SessionStatusPathHandler handles the /sessions/status endpoint to list all sessions
func SessionStatusPathHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionManager := session.GetManager()
	allSessions := sessionManager.ListSessions()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(allSessions); err != nil {
		log.Printf("Error encoding sessions data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
