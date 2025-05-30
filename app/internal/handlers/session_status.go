package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

type SessionManager interface {
	GetSession(sessionID string) (*entities.SessionData, error)
	ListSessions() (map[string]*entities.SessionData, error)

	UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error)
	ParseTokenUsageFromResponse(responseBody []byte) (*entities.TokenUsage, error)
}

// SessionStatusHandler handles requests to get session statistics
type SessionStatusHandler struct {
	sessionManager SessionManager
}

// NewSessionStatusHandler creates a new SessionStatusHandler with injected dependencies
func NewSessionStatusHandler(sessionManager SessionManager) *SessionStatusHandler {
	return &SessionStatusHandler{
		sessionManager: sessionManager,
	}
}

// HandleSingle handles requests to get specific session statistics
func (ssh *SessionStatusHandler) HandleSingle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if specific session ID is requested: /v1/session/{sessionID}/status
	sessionID := extractSessionID(r.URL.Path)

	w.Header().Set("Content-Type", "application/json")

	if sessionID != "" {
		// Return specific session data
		sessionData, errGet := ssh.sessionManager.GetSession(sessionID)
		if errGet != nil {
			if errors.Is(errGet, entities.ErrSessionNotFound) {
				http.Error(w, "Session not found", http.StatusNotFound)
			} else {
				log.Printf("Error retrieving session %s: %v", sessionID, errGet)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		if err := json.NewEncoder(w).Encode(sessionData); err != nil {
			log.Printf("Error encoding session data: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		// Return all sessions
		allSessions, errList := ssh.sessionManager.ListSessions()
		if errList != nil {
			log.Printf("Error listing sessions: %v", errList)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(allSessions); err != nil {
			log.Printf("Error encoding sessions data: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// HandleList handles the /sessions/status endpoint to list all sessions
func (ssh *SessionStatusHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allSessions, errList := ssh.sessionManager.ListSessions()
	if errList != nil {
		log.Printf("Error listing sessions: %v", errList)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(allSessions); err != nil {
		log.Printf("Error encoding all sessions data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// Legacy functions for backward compatibility
func SessionStatusHandler_Legacy(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "SessionStatusHandler requires dependency injection. Use NewSessionStatusHandler instead.", http.StatusInternalServerError)
}

func SessionStatusPathHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "SessionStatusPathHandler requires dependency injection. Use NewSessionStatusHandler instead.", http.StatusInternalServerError)
}
