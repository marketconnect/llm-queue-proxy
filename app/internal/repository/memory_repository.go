package repository

import (
	"sync"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

// MemoryRepository is an in-memory implementation of the Repository interface.
type MemoryRepository struct {
	sessions map[string]*entities.SessionData
	mu       sync.RWMutex
}

// NewMemoryRepository creates a new MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		sessions: make(map[string]*entities.SessionData),
	}
}

// Init initializes the memory repository (no-op for memory repository).
func (r *MemoryRepository) Init() error {
	return nil
}

// Close closes the memory repository (no-op for memory repository).
func (r *MemoryRepository) Close() error {
	return nil
}

// GetSession retrieves session data for a given session ID.
func (r *MemoryRepository) GetSession(sessionID string) (*entities.SessionData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sess, exists := r.sessions[sessionID]
	if !exists {
		return nil, entities.ErrSessionNotFound
	}
	// Return a copy to prevent modification outside of repository methods
	sessCopy := *sess
	return &sessCopy, nil
}

// CreateSession creates a new session with the given ID.
// If the session already exists, it returns the existing session.
func (r *MemoryRepository) CreateSession(sessionID string) (*entities.SessionData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sess, exists := r.sessions[sessionID]; exists {
		sessCopy := *sess
		return &sessCopy, nil // Session already exists
	}

	sess := &entities.SessionData{
		SessionID: sessionID,
	}
	r.sessions[sessionID] = sess
	sessCopy := *sess
	return &sessCopy, nil
}

// UpdateSessionTokens adds token usage to an existing session.
// If the session does not exist, it creates it.
func (r *MemoryRepository) UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sess, exists := r.sessions[sessionID]
	if !exists {
		sess = &entities.SessionData{SessionID: sessionID}
		r.sessions[sessionID] = sess
	}

	sess.TotalPromptTokens += usage.PromptTokens
	sess.TotalCompletionTokens += usage.CompletionTokens
	sess.TotalTokens += usage.TotalTokens
	sess.RequestCount++

	sessCopy := *sess
	return &sessCopy, nil
}

// ListSessions returns all session data.
func (r *MemoryRepository) ListSessions() (map[string]*entities.SessionData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*entities.SessionData, len(r.sessions))
	for k, v := range r.sessions {
		vCopy := *v
		result[k] = &vCopy
	}
	return result, nil
}
