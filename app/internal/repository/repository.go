package repository

import (
	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

// Repository defines the interface for session storage.
// This allows for different storage backends (e.g., in-memory, SQLite).
type Repository interface {
	// Init performs any necessary initialization for the repository (e.g., DB connection, table creation).
	Init() error
	// Close performs cleanup tasks (e.g., closing DB connection).
	Close() error

	GetSession(sessionID string) (*entities.SessionData, error)
	CreateSession(sessionID string) (*entities.SessionData, error)
	UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error)
	ListSessions() (map[string]*entities.SessionData, error)
}
