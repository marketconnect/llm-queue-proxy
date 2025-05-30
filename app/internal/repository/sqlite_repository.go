package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

// SQLiteRepository implements the Repository interface using an SQLite database.
type SQLiteRepository struct {
	db  *sql.DB
	dsn string
}

// NewSQLiteRepository creates a new SQLiteRepository.
// The DSN is the data source name for the SQLite database.
func NewSQLiteRepository(dsn string) (*SQLiteRepository, error) {
	// The driver "sqlite3" must be registered by the application importing this package,
	// typically by a blank import like `_ "github.com/mattn/go-sqlite3"`.
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	return &SQLiteRepository{db: db, dsn: dsn}, nil
}

// Init initializes the SQLite repository, creating the necessary tables if they don't exist.
func (r *SQLiteRepository) Init() error {
	query := `
    CREATE TABLE IF NOT EXISTS sessions (
        session_id TEXT PRIMARY KEY,
        total_prompt_tokens INTEGER DEFAULT 0,
        total_completion_tokens INTEGER DEFAULT 0,
        total_tokens INTEGER DEFAULT 0,
        request_count INTEGER DEFAULT 0
    );`

	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}
	log.Println("SQLite sessions table initialized successfully.")
	return nil
}

// Close closes the database connection.
func (r *SQLiteRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// GetSession retrieves session data for a given session ID.
func (r *SQLiteRepository) GetSession(sessionID string) (*entities.SessionData, error) {
	query := `SELECT session_id, total_prompt_tokens, total_completion_tokens, total_tokens, request_count
              FROM sessions WHERE session_id = ?;`
	row := r.db.QueryRow(query, sessionID)

	var sess entities.SessionData
	err := row.Scan(
		&sess.SessionID,
		&sess.TotalPromptTokens,
		&sess.TotalCompletionTokens,
		&sess.TotalTokens,
		&sess.RequestCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &sess, nil
}

// CreateSession creates a new session with the given ID.
// If the session already exists, it returns the existing session data.
func (r *SQLiteRepository) CreateSession(sessionID string) (*entities.SessionData, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Insert with default zero values, or do nothing if it already exists.
	queryInsert := `
    INSERT INTO sessions (session_id, total_prompt_tokens, total_completion_tokens, total_tokens, request_count)
    VALUES (?, 0, 0, 0, 0)
    ON CONFLICT(session_id) DO NOTHING;`

	_, err = tx.ExecContext(ctx, queryInsert, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert or ignore session: %w", err)
	}

	// Select the session (either existing or newly created with zeros).
	querySelect := `SELECT session_id, total_prompt_tokens, total_completion_tokens, total_tokens, request_count
                     FROM sessions WHERE session_id = ?;`
	row := tx.QueryRowContext(ctx, querySelect, sessionID)

	var sess entities.SessionData
	err = row.Scan(&sess.SessionID, &sess.TotalPromptTokens, &sess.TotalCompletionTokens, &sess.TotalTokens, &sess.RequestCount)
	if err != nil {
		// This should not happen if INSERT OR IGNORE worked, unless DB is corrupted.
		return nil, fmt.Errorf("failed to select session after create: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &sess, nil
}

// UpdateSessionTokens adds token usage to an existing session.
// If the session does not exist, it creates it with the given token usage.
func (r *SQLiteRepository) UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	queryUpsert := `
    INSERT INTO sessions (session_id, total_prompt_tokens, total_completion_tokens, total_tokens, request_count)
    VALUES (?, ?, ?, ?, 1)
    ON CONFLICT(session_id) DO UPDATE SET
        total_prompt_tokens = sessions.total_prompt_tokens + excluded.total_prompt_tokens,
        total_completion_tokens = sessions.total_completion_tokens + excluded.total_completion_tokens,
        total_tokens = sessions.total_tokens + excluded.total_tokens,
        request_count = sessions.request_count + 1;`

	_, err = tx.ExecContext(ctx, queryUpsert, sessionID, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert session tokens: %w", err)
	}

	// After upserting, retrieve the updated session data
	// This is similar to GetSession but within the same transaction
	querySelect := `SELECT session_id, total_prompt_tokens, total_completion_tokens, total_tokens, request_count
                     FROM sessions WHERE session_id = ?;`
	row := tx.QueryRowContext(ctx, querySelect, sessionID)
	var sess entities.SessionData
	if errScan := row.Scan(&sess.SessionID, &sess.TotalPromptTokens, &sess.TotalCompletionTokens, &sess.TotalTokens, &sess.RequestCount); errScan != nil {
		return nil, fmt.Errorf("failed to select session after update: %w", errScan)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return &sess, nil
}

// ListSessions returns all session data.
func (r *SQLiteRepository) ListSessions() (map[string]*entities.SessionData, error) {
	query := `SELECT session_id, total_prompt_tokens, total_completion_tokens, total_tokens, request_count FROM sessions;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	sessionsMap := make(map[string]*entities.SessionData)
	for rows.Next() {
		var sess entities.SessionData
		if err := rows.Scan(&sess.SessionID, &sess.TotalPromptTokens, &sess.TotalCompletionTokens, &sess.TotalTokens, &sess.RequestCount); err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessionsMap[sess.SessionID] = &sess
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}
	return sessionsMap, nil
}
