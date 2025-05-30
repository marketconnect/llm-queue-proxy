package session

import (
	"encoding/json"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

type Repository interface {
	Init() error
	Close() error
	GetSession(sessionID string) (*entities.SessionData, error)
	CreateSession(sessionID string) (*entities.SessionData, error)
	UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error)
	ListSessions() (map[string]*entities.SessionData, error)
}

type SessionManager struct {
	repository Repository
}

// NewSessionManager creates a new SessionManager with the provided repository
func NewSessionManager(repo Repository) *SessionManager {
	return &SessionManager{
		repository: repo,
	}
}

// Close closes the underlying repository connection if applicable.
func (sm *SessionManager) Close() error {
	if sm.repository != nil {
		return sm.repository.Close()
	}
	return nil
}

// GetSession retrieves session data for a given session ID
func (sm *SessionManager) GetSession(sessionID string) (*entities.SessionData, error) {
	return sm.repository.GetSession(sessionID)
}

// CreateSession creates a new session with the given ID
func (sm *SessionManager) CreateSession(sessionID string) (*entities.SessionData, error) {
	return sm.repository.CreateSession(sessionID)
}

// UpdateSessionTokens adds token usage to an existing session
func (sm *SessionManager) UpdateSessionTokens(sessionID string, tokenUsage entities.TokenUsage) (*entities.SessionData, error) {
	return sm.repository.UpdateSessionTokens(sessionID, tokenUsage)
}

// ParseTokenUsageFromResponse extracts token usage from OpenAI API response body
func (sm *SessionManager) ParseTokenUsageFromResponse(responseBody []byte) (*entities.TokenUsage, error) {
	var response struct {
		Usage entities.TokenUsage `json:"usage"`
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
func (sm *SessionManager) ListSessions() (map[string]*entities.SessionData, error) {
	return sm.repository.ListSessions()
}
