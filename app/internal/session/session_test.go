package session_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
	"github.com/marketconnect/llm-queue-proxy/app/internal/session"
)

type mockRepository struct {
	GetSessionFunc          func(sessionID string) (*entities.SessionData, error)
	CreateSessionFunc       func(sessionID string) (*entities.SessionData, error)
	UpdateSessionTokensFunc func(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error)
	ListSessionsFunc        func() (map[string]*entities.SessionData, error)
	InitFunc                func() error
	CloseFunc               func() error
}

func (m *mockRepository) Init() error {
	if m.InitFunc != nil {
		return m.InitFunc()
	}
	return nil
}
func (m *mockRepository) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
func (m *mockRepository) GetSession(sessionID string) (*entities.SessionData, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(sessionID)
	}
	return nil, errors.New("GetSessionFunc not implemented")
}
func (m *mockRepository) CreateSession(sessionID string) (*entities.SessionData, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(sessionID)
	}
	return nil, errors.New("CreateSessionFunc not implemented")
}
func (m *mockRepository) UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error) {
	if m.UpdateSessionTokensFunc != nil {
		return m.UpdateSessionTokensFunc(sessionID, usage)
	}
	return nil, errors.New("UpdateSessionTokensFunc not implemented")
}
func (m *mockRepository) ListSessions() (map[string]*entities.SessionData, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc()
	}
	return nil, errors.New("ListSessionsFunc not implemented")
}

func TestSessionManager_PassthroughMethods(t *testing.T) {
	mockRepo := &mockRepository{}
	sm := session.NewSessionManager(mockRepo)

	// Test GetSession
	expectedSession := &entities.SessionData{SessionID: "s1"}
	mockRepo.GetSessionFunc = func(sessionID string) (*entities.SessionData, error) {
		if sessionID == "s1" {
			return expectedSession, nil
		}
		return nil, errors.New("not found")
	}
	sess, err := sm.GetSession("s1")
	if err != nil || sess != expectedSession {
		t.Errorf("GetSession: got (%v, %v), want (%v, nil)", sess, err, expectedSession)
	}

	// Test CreateSession
	mockRepo.CreateSessionFunc = func(sessionID string) (*entities.SessionData, error) {
		if sessionID == "s2" {
			return expectedSession, nil // Re-use expectedSession for simplicity
		}
		return nil, errors.New("create error")
	}
	sess, err = sm.CreateSession("s2")
	if err != nil || sess != expectedSession {
		t.Errorf("CreateSession: got (%v, %v), want (%v, nil)", sess, err, expectedSession)
	}

	// Test UpdateSessionTokens
	usage := entities.TokenUsage{TotalTokens: 10}
	mockRepo.UpdateSessionTokensFunc = func(sessionID string, u entities.TokenUsage) (*entities.SessionData, error) {
		if sessionID == "s3" && u.TotalTokens == 10 {
			return expectedSession, nil
		}
		return nil, errors.New("update error")
	}
	sess, err = sm.UpdateSessionTokens("s3", usage)
	if err != nil || sess != expectedSession {
		t.Errorf("UpdateSessionTokens: got (%v, %v), want (%v, nil)", sess, err, expectedSession)
	}

	// Test ListSessions
	expectedMap := map[string]*entities.SessionData{"s4": expectedSession}
	mockRepo.ListSessionsFunc = func() (map[string]*entities.SessionData, error) {
		return expectedMap, nil
	}
	mapSess, err := sm.ListSessions()
	if err != nil || !reflect.DeepEqual(mapSess, expectedMap) {
		t.Errorf("ListSessions: got (%v, %v), want (%v, nil)", mapSess, err, expectedMap)
	}

	// Test Close
	var closeCalled bool
	mockRepo.CloseFunc = func() error {
		closeCalled = true
		return nil
	}
	sm.Close()
	if !closeCalled {
		t.Error("Close: repository.Close was not called")
	}
}

func TestSessionManager_ParseTokenUsageFromResponse(t *testing.T) {
	sm := session.NewSessionManager(nil) // Repository not needed for this method

	validBody := []byte(`{"usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}}`)
	expectedUsage := &entities.TokenUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30}
	usage, err := sm.ParseTokenUsageFromResponse(validBody)
	if err != nil || !reflect.DeepEqual(usage, expectedUsage) {
		t.Errorf("ParseTokenUsageFromResponse(valid): got (%+v, %v), want (%+v, nil)", usage, err, expectedUsage)
	}

	noUsageBody := []byte(`{"model": "gpt-4"}`) // No usage field
	usage, err = sm.ParseTokenUsageFromResponse(noUsageBody)
	if err != nil || usage != nil { // Expect nil usage, nil error if "usage" is missing or empty
		t.Errorf("ParseTokenUsageFromResponse(no usage): got (%+v, %v), want (nil, nil)", usage, err)
	}

	invalidJsonBody := []byte(`{invalid`)
	usage, err = sm.ParseTokenUsageFromResponse(invalidJsonBody)
	if err == nil {
		t.Errorf("ParseTokenUsageFromResponse(invalid json): got err nil, want error. Usage: %+v", usage)
	}
}
