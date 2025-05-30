package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

type mockSessionManager struct {
	GetSessionFunc          func(sessionID string) (*entities.SessionData, error)
	ListSessionsFunc        func() (map[string]*entities.SessionData, error)
	UpdateSessionTokensFunc func(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error)
	ParseTokenUsageFunc     func(responseBody []byte) (*entities.TokenUsage, error)
}

func (m *mockSessionManager) GetSession(sessionID string) (*entities.SessionData, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(sessionID)
	}
	return nil, errors.New("GetSession not implemented")
}

func (m *mockSessionManager) ListSessions() (map[string]*entities.SessionData, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc()
	}
	return nil, errors.New("ListSessions not implemented")
}

func (m *mockSessionManager) UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error) {
	return nil, errors.New("UpdateSessionTokens not implemented")
}

func (m *mockSessionManager) ParseTokenUsageFromResponse(responseBody []byte) (*entities.TokenUsage, error) {
	return nil, errors.New("ParseTokenUsageFromResponse not implemented")
}

func TestSessionStatusHandler_HandleList(t *testing.T) {
	tests := []struct {
		name               string
		mockSetup          func(*mockSessionManager)
		expectedStatusCode int
		expectedBody       string // Expected JSON string or part of it
	}{
		{
			name: "successful list",
			mockSetup: func(msm *mockSessionManager) {
				msm.ListSessionsFunc = func() (map[string]*entities.SessionData, error) {
					return map[string]*entities.SessionData{
						"sess1": {SessionID: "sess1", TotalTokens: 100},
						"sess2": {SessionID: "sess2", TotalTokens: 200},
					}, nil
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"sess1":{"session_id":"sess1","total_prompt_tokens":0,"total_completion_tokens":0,"total_tokens":100,"request_count":0},"sess2":{"session_id":"sess2","total_prompt_tokens":0,"total_completion_tokens":0,"total_tokens":200,"request_count":0}}`,
		},
		{
			name: "empty list",
			mockSetup: func(msm *mockSessionManager) {
				msm.ListSessionsFunc = func() (map[string]*entities.SessionData, error) {
					return map[string]*entities.SessionData{}, nil
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{}`,
		},
		{
			name: "error listing sessions",
			mockSetup: func(msm *mockSessionManager) {
				msm.ListSessionsFunc = func() (map[string]*entities.SessionData, error) {
					return nil, errors.New("db error")
				}
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "Internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msm := &mockSessionManager{}
			tt.mockSetup(msm)

			handler := NewSessionStatusHandler(msm)
			req := httptest.NewRequest(http.MethodGet, "/sessions/status", nil)
			rr := httptest.NewRecorder()

			handler.HandleList(rr, req)

			if rr.Code != tt.expectedStatusCode {
				t.Errorf("HandleList status code = %v, want %v", rr.Code, tt.expectedStatusCode)
			}
			// For JSON, it's better to unmarshal and compare, but string comparison is simpler here.
			if !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("HandleList body = %q, want to contain %q", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestSessionStatusHandler_HandleSingle(t *testing.T) {
	// Note: HandleSingle uses extractSessionID, which expects /v1/session/{id}/...
	// So, for testing HandleSingle for a specific session, path should be like /v1/session/sess1/status
	// If path is /status, extractSessionID returns "", and it lists all sessions.

	tests := []struct {
		name               string
		path               string // Path to simulate request URL
		mockSetup          func(*mockSessionManager)
		expectedStatusCode int
		expectedBody       string
	}{
		{
			name: "get specific session successfully",
			path: "/v1/session/sess1/status",
			mockSetup: func(msm *mockSessionManager) {
				msm.GetSessionFunc = func(sessionID string) (*entities.SessionData, error) {
					if sessionID == "sess1" {
						return &entities.SessionData{SessionID: "sess1", TotalTokens: 150}, nil
					}
					return nil, entities.ErrSessionNotFound
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"session_id":"sess1","total_prompt_tokens":0,"total_completion_tokens":0,"total_tokens":150,"request_count":0}`,
		},
		// Add more tests for HandleSingle: session not found, error getting session, path without session ID (lists all)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msm := &mockSessionManager{}
			tt.mockSetup(msm)

			handler := NewSessionStatusHandler(msm)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			handler.HandleSingle(rr, req)

			if rr.Code != tt.expectedStatusCode {
				t.Errorf("HandleSingle status code = %v, want %v", rr.Code, tt.expectedStatusCode)
			}

			// Unmarshal and compare for more robust JSON checking if needed
			var expectedJSON, actualJSON interface{}
			json.Unmarshal([]byte(tt.expectedBody), &expectedJSON)
			json.Unmarshal(rr.Body.Bytes(), &actualJSON)

			// This is a simplified check. For complex JSON, reflect.DeepEqual is better.
			if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(tt.expectedBody) {
				t.Errorf("HandleSingle body = %q, want %q", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}
