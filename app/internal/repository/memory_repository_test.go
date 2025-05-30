package repository_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
	"github.com/marketconnect/llm-queue-proxy/app/internal/repository"
)

func TestMemoryRepository_InitClose(t *testing.T) {
	repo := repository.NewMemoryRepository()
	if err := repo.Init(); err != nil {
		t.Errorf("Init() error = %v, wantErr nil", err)
	}
	if err := repo.Close(); err != nil {
		t.Errorf("Close() error = %v, wantErr nil", err)
	}
}

func TestMemoryRepository_CreateGetSession(t *testing.T) {
	repo := repository.NewMemoryRepository()
	sessionID := "test-session-1"

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
		t.Errorf("GetSession() retrieved = %v, want %v", retrievedSess, createdSess)
	}

	// Create existing session - should return existing
	existingSess, err := repo.CreateSession(sessionID)
	if err != nil {
		t.Fatalf("CreateSession() for existing ID error = %v", err)
	}
	if !reflect.DeepEqual(createdSess, existingSess) {
		t.Errorf("CreateSession() for existing ID = %v, want %v", existingSess, createdSess)
	}
}

func TestMemoryRepository_GetNonExistentSession(t *testing.T) {
	repo := repository.NewMemoryRepository()
	_, err := repo.GetSession("non-existent-session")
	if !errors.Is(err, entities.ErrSessionNotFound) {
		t.Errorf("GetSession() for non-existent ID error = %v, want %v", err, entities.ErrSessionNotFound)
	}
}

func TestMemoryRepository_UpdateSessionTokens(t *testing.T) {
	repo := repository.NewMemoryRepository()
	sessionID := "test-session-update"
	usage1 := entities.TokenUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30}
	usage2 := entities.TokenUsage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15}

	// Update non-existent session (should create it)
	updatedSess, err := repo.UpdateSessionTokens(sessionID, usage1)
	if err != nil {
		t.Fatalf("UpdateSessionTokens() error = %v", err)
	}
	expectedSess := &entities.SessionData{
		SessionID:             sessionID,
		TotalPromptTokens:     10,
		TotalCompletionTokens: 20,
		TotalTokens:           30,
		RequestCount:          1,
	}
	if !reflect.DeepEqual(updatedSess, expectedSess) {
		t.Errorf("UpdateSessionTokens() first update = %v, want %v", updatedSess, expectedSess)
	}

	// Update existing session
	updatedSess, err = repo.UpdateSessionTokens(sessionID, usage2)
	if err != nil {
		t.Fatalf("UpdateSessionTokens() second update error = %v", err)
	}
	expectedSess.TotalPromptTokens += 5
	expectedSess.TotalCompletionTokens += 10
	expectedSess.TotalTokens += 15
	expectedSess.RequestCount++
	if !reflect.DeepEqual(updatedSess, expectedSess) {
		t.Errorf("UpdateSessionTokens() second update = %v, want %v", updatedSess, expectedSess)
	}
}

func TestMemoryRepository_ListSessions(t *testing.T) {
	repo := repository.NewMemoryRepository()

	// List empty
	sessions, err := repo.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() empty error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("ListSessions() empty len = %d, want 0", len(sessions))
	}

	// Add some sessions
	repo.CreateSession("sess1")
	repo.UpdateSessionTokens("sess2", entities.TokenUsage{TotalTokens: 100})

	sessions, err = repo.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() with items error = %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("ListSessions() with items len = %d, want 2", len(sessions))
	}
	if _, ok := sessions["sess1"]; !ok {
		t.Error("ListSessions() missing 'sess1'")
	}
	if sessions["sess2"].TotalTokens != 100 {
		t.Errorf("ListSessions() 'sess2' TotalTokens = %d, want 100", sessions["sess2"].TotalTokens)
	}
}
