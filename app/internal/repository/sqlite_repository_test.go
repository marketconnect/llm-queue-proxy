package repository_test

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
	"github.com/marketconnect/llm-queue-proxy/app/internal/repository"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

func setupTestDB(t *testing.T) (*repository.SQLiteRepository, func()) {
	t.Helper()
	// Using in-memory SQLite for tests
	// dsn := ":memory:"
	// Or, using a temporary file to better simulate file-based DB
	tempDir := t.TempDir()
	dsn := filepath.Join(tempDir, "test_sessions.db")

	repo, err := repository.NewSQLiteRepository(dsn)
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}

	if err := repo.Init(); err != nil {
		t.Fatalf("repo.Init() error = %v", err)
	}

	cleanup := func() {
		repo.Close()
		// os.Remove(dsn) // Not needed if using t.TempDir()
	}
	return repo, cleanup
}

func TestSQLiteRepository_InitClose(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	// Init is called in setupTestDB
	// Test closing
	if err := repo.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestSQLiteRepository_CreateGetSession(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	sessionID := "test-sqlite-session-1"

	// Create session
	createdSess, err := repo.CreateSession(sessionID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if createdSess.SessionID != sessionID {
		t.Errorf("CreateSession() SessionID = %v, want %v", createdSess.SessionID, sessionID)
	}

	// Get session
	retrievedSess, err := repo.GetSession(sessionID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if !reflect.DeepEqual(createdSess, retrievedSess) {
		t.Errorf("GetSession() retrieved = %+v, want %+v", retrievedSess, createdSess)
	}

	// Create existing session - should return existing
	existingSess, err := repo.CreateSession(sessionID)
	if err != nil {
		t.Fatalf("CreateSession() for existing ID error = %v", err)
	}
	if !reflect.DeepEqual(createdSess, existingSess) {
		t.Errorf("CreateSession() for existing ID = %+v, want %+v", existingSess, createdSess)
	}
}

func TestSQLiteRepository_GetNonExistentSession(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := repo.GetSession("non-existent-sqlite-session")
	if !errors.Is(err, entities.ErrSessionNotFound) {
		t.Errorf("GetSession() for non-existent ID error = %v, want %v", err, entities.ErrSessionNotFound)
	}
}

func TestSQLiteRepository_UpdateSessionTokens(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	sessionID := "test-sqlite-session-update"
	usage1 := entities.TokenUsage{PromptTokens: 100, CompletionTokens: 200, TotalTokens: 300}
	usage2 := entities.TokenUsage{PromptTokens: 50, CompletionTokens: 100, TotalTokens: 150}

	// Update non-existent session (should create it)
	updatedSess, err := repo.UpdateSessionTokens(sessionID, usage1)
	if err != nil {
		t.Fatalf("UpdateSessionTokens() error = %v", err)
	}
	expectedSess := &entities.SessionData{
		SessionID:             sessionID,
		TotalPromptTokens:     100,
		TotalCompletionTokens: 200,
		TotalTokens:           300,
		RequestCount:          1,
	}
	if !reflect.DeepEqual(updatedSess, expectedSess) {
		t.Errorf("UpdateSessionTokens() first update = %+v, want %+v", updatedSess, expectedSess)
	}

	// Update existing session
	updatedSess, err = repo.UpdateSessionTokens(sessionID, usage2)
	if err != nil {
		t.Fatalf("UpdateSessionTokens() second update error = %v", err)
	}
	expectedSess.TotalPromptTokens += 50
	expectedSess.TotalCompletionTokens += 100
	expectedSess.TotalTokens += 150
	expectedSess.RequestCount++
	if !reflect.DeepEqual(updatedSess, expectedSess) {
		t.Errorf("UpdateSessionTokens() second update = %+v, want %+v", updatedSess, expectedSess)
	}
}

func TestSQLiteRepository_ListSessions(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	repo.CreateSession("s1")
	repo.UpdateSessionTokens("s2", entities.TokenUsage{TotalTokens: 50})

	sessions, err := repo.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("ListSessions() len = %d, want 2", len(sessions))
	}
	if sessions["s2"].TotalTokens != 50 {
		t.Errorf("ListSessions() s2.TotalTokens = %d, want 50", sessions["s2"].TotalTokens)
	}
}
