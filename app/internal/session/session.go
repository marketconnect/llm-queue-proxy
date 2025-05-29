package session

import (
	"encoding/json"
	"sync"
)

// TokenUsage represents the token usage from OpenAI API responses
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SessionData holds information about a session including accumulated token usage
type SessionData struct {
	SessionID             string `json:"session_id"`
	TotalPromptTokens     int    `json:"total_prompt_tokens"`
	TotalCompletionTokens int    `json:"total_completion_tokens"`
	TotalTokens           int    `json:"total_tokens"`
	RequestCount          int    `json:"request_count"`
}

type SessionManager struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
}

var manager *SessionManager
var once sync.Once

// GetManager returns the singleton session manager instance
func GetManager() *SessionManager {
	once.Do(func() {
		manager = &SessionManager{
			sessions: make(map[string]*SessionData),
		}
	})
	return manager
}

// GetSession retrieves session data for a given session ID
func (sm *SessionManager) GetSession(sessionID string) *SessionData {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil
	}
	return session
}

// CreateSession creates a new session with the given ID
func (sm *SessionManager) CreateSession(sessionID string) *SessionData {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &SessionData{
		SessionID:             sessionID,
		TotalPromptTokens:     0,
		TotalCompletionTokens: 0,
		TotalTokens:           0,
		RequestCount:          0,
	}

	sm.sessions[sessionID] = session
	return session
}

// UpdateSessionTokens adds token usage to an existing session
func (sm *SessionManager) UpdateSessionTokens(sessionID string, tokenUsage TokenUsage) *SessionData {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		// Create session if it doesn't exist
		session = &SessionData{
			SessionID:             sessionID,
			TotalPromptTokens:     0,
			TotalCompletionTokens: 0,
			TotalTokens:           0,
			RequestCount:          0,
		}
		sm.sessions[sessionID] = session
	}

	// Add new token usage to existing totals
	session.TotalPromptTokens += tokenUsage.PromptTokens
	session.TotalCompletionTokens += tokenUsage.CompletionTokens
	session.TotalTokens += tokenUsage.TotalTokens
	session.RequestCount++

	return session
}

// ParseTokenUsageFromResponse extracts token usage from OpenAI API response body
func ParseTokenUsageFromResponse(responseBody []byte) (*TokenUsage, error) {
	var response struct {
		Usage TokenUsage `json:"usage"`
	}

	err := json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, err
	}

	// Return nil if no usage data found (some endpoints might not include usage)
	if response.Usage.TotalTokens == 0 {
		return nil, nil
	}

	return &response.Usage, nil
}

// ListSessions returns all session data (for debugging/monitoring)
func (sm *SessionManager) ListSessions() map[string]*SessionData {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*SessionData)
	for k, v := range sm.sessions {
		result[k] = v
	}
	return result
}
