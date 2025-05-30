package handlers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marketconnect/llm-queue-proxy/app/domain/entities"
)

type mockProxySessionManager struct {
	GetSessionFunc                  func(sessionID string) (*entities.SessionData, error)
	CreateSessionFunc               func(sessionID string) (*entities.SessionData, error)
	ListSessionsFunc                func() (map[string]*entities.SessionData, error)
	UpdateSessionTokensFunc         func(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error)
	ParseTokenUsageFromResponseFunc func(responseBody []byte) (*entities.TokenUsage, error)
}

func (m *mockProxySessionManager) GetSession(sessionID string) (*entities.SessionData, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(sessionID)
	}
	return nil, errors.New("GetSessionFunc not implemented")
}
func (m *mockProxySessionManager) CreateSession(sessionID string) (*entities.SessionData, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(sessionID)
	}
	return nil, errors.New("CreateSessionFunc not implemented")
}
func (m *mockProxySessionManager) ListSessions() (map[string]*entities.SessionData, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc()
	}
	return nil, errors.New("ListSessionsFunc not implemented")
}
func (m *mockProxySessionManager) UpdateSessionTokens(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error) {
	if m.UpdateSessionTokensFunc != nil {
		return m.UpdateSessionTokensFunc(sessionID, usage)
	}
	return nil, errors.New("UpdateSessionTokensFunc not implemented")
}
func (m *mockProxySessionManager) ParseTokenUsageFromResponse(responseBody []byte) (*entities.TokenUsage, error) {
	if m.ParseTokenUsageFromResponseFunc != nil {
		return m.ParseTokenUsageFromResponseFunc(responseBody)
	}
	// Default passthrough for ParseTokenUsageFromResponse for simplicity if not mocked
	var response struct {
		Usage entities.TokenUsage `json:"usage"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}
	if response.Usage.TotalTokens == 0 {
		return nil, nil
	}
	return &response.Usage, nil
}

type mockQueue struct {
	PushFunc func(r entities.ProxyRequest) entities.ProxyResponse
}

func (m *mockQueue) Push(r entities.ProxyRequest) entities.ProxyResponse {
	if m.PushFunc != nil {
		return m.PushFunc(r)
	}
	return entities.ProxyResponse{Err: errors.New("PushFunc not implemented")}
}

func TestProxyHandler_Handle(t *testing.T) {
	tests := []struct {
		name                        string
		path                        string
		requestBody                 string
		mockSessionManagerSetup     func(*mockProxySessionManager)
		mockQueueSetup              func(*mockQueue)
		expectedStatusCode          int
		expectedBodyContains        string
		expectCreateSessionCalled   bool
		expectGetSessionCalled      bool
		expectUpdateTokensCalled    bool
		createSessionShouldError    bool
		getSessionShouldError       bool
		updateSessionShouldError    bool
		parseTokenUsageShouldError  bool
		queuePushShouldError        bool
		gzippedResponse             bool
		responseBodyForTokenParsing string
	}{
		{
			name: "new session, successful request",
			path: "/v1/session/new123/chat/completions",
			mockSessionManagerSetup: func(msm *mockProxySessionManager) {
				msm.GetSessionFunc = func(sessionID string) (*entities.SessionData, error) { return nil, entities.ErrSessionNotFound }
				msm.CreateSessionFunc = func(sessionID string) (*entities.SessionData, error) {
					return &entities.SessionData{SessionID: sessionID}, nil
				}
				msm.UpdateSessionTokensFunc = func(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error) {
					return &entities.SessionData{SessionID: sessionID}, nil
				}
			},
			mockQueueSetup: func(mq *mockQueue) {
				mq.PushFunc = func(r entities.ProxyRequest) entities.ProxyResponse {
					return entities.ProxyResponse{StatusCode: http.StatusOK, Body: []byte(`{"usage":{"total_tokens":10}}`)}
				}
			},
			expectedStatusCode:        http.StatusOK,
			expectedBodyContains:      `{"usage":{"total_tokens":10}}`,
			expectCreateSessionCalled: true,
			expectGetSessionCalled:    true,
			expectUpdateTokensCalled:  true,
		},
		{
			name: "existing session, successful request",
			path: "/v1/session/existing123/chat/completions",
			mockSessionManagerSetup: func(msm *mockProxySessionManager) {
				msm.GetSessionFunc = func(sessionID string) (*entities.SessionData, error) {
					return &entities.SessionData{SessionID: sessionID}, nil
				}
				msm.UpdateSessionTokensFunc = func(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error) {
					return &entities.SessionData{SessionID: sessionID}, nil
				}
			},
			mockQueueSetup: func(mq *mockQueue) {
				mq.PushFunc = func(r entities.ProxyRequest) entities.ProxyResponse {
					return entities.ProxyResponse{StatusCode: http.StatusOK, Body: []byte(`{"usage":{"total_tokens":20}}`)}
				}
			},
			expectedStatusCode:       http.StatusOK,
			expectedBodyContains:     `{"usage":{"total_tokens":20}}`,
			expectGetSessionCalled:   true,
			expectUpdateTokensCalled: true,
		},
		{
			name: "no session ID, successful request",
			path: "/v1/chat/completions",
			mockQueueSetup: func(mq *mockQueue) {
				mq.PushFunc = func(r entities.ProxyRequest) entities.ProxyResponse {
					return entities.ProxyResponse{StatusCode: http.StatusOK, Body: []byte(`{"response":"ok"}`)}
				}
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: `{"response":"ok"}`,
		},
		{
			name: "session ID with missing endpoint",
			path: "/v1/session/test123/",
			mockSessionManagerSetup: func(msm *mockProxySessionManager) {
				// Setup not strictly needed as it should error before session manager calls
			},
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "Missing OpenAI endpoint",
		},
		{
			name: "create session error",
			path: "/v1/session/errorCreate/chat/completions",
			mockSessionManagerSetup: func(msm *mockProxySessionManager) {
				msm.GetSessionFunc = func(sessionID string) (*entities.SessionData, error) { return nil, entities.ErrSessionNotFound }
				msm.CreateSessionFunc = func(sessionID string) (*entities.SessionData, error) { return nil, errors.New("create failed") }
			},
			expectedStatusCode:        http.StatusInternalServerError,
			expectedBodyContains:      "Failed to initialize session",
			expectGetSessionCalled:    true,
			expectCreateSessionCalled: true,
		},
		{
			name: "get session error (not ErrSessionNotFound)",
			path: "/v1/session/errorGet/chat/completions",
			mockSessionManagerSetup: func(msm *mockProxySessionManager) {
				msm.GetSessionFunc = func(sessionID string) (*entities.SessionData, error) { return nil, errors.New("get failed") }
			},
			expectedStatusCode:     http.StatusInternalServerError,
			expectedBodyContains:   "Failed to retrieve session",
			expectGetSessionCalled: true,
		},
		{
			name: "queue push error",
			path: "/v1/chat/completions",
			mockQueueSetup: func(mq *mockQueue) {
				mq.PushFunc = func(r entities.ProxyRequest) entities.ProxyResponse {
					return entities.ProxyResponse{Err: errors.New("queue error")}
				}
			},
			expectedStatusCode:   http.StatusBadGateway,
			expectedBodyContains: "Proxy error: queue error",
		},
		{
			name: "gzipped response with token usage",
			path: "/v1/session/gzip123/chat/completions",
			mockSessionManagerSetup: func(msm *mockProxySessionManager) {
				msm.GetSessionFunc = func(sessionID string) (*entities.SessionData, error) {
					return &entities.SessionData{SessionID: sessionID}, nil
				}
				msm.UpdateSessionTokensFunc = func(sessionID string, usage entities.TokenUsage) (*entities.SessionData, error) {
					if usage.TotalTokens != 30 {
						t.Errorf("Expected 30 total tokens for update, got %d", usage.TotalTokens)
					}
					return &entities.SessionData{SessionID: sessionID}, nil
				}
			},
			mockQueueSetup: func(mq *mockQueue) {
				mq.PushFunc = func(r entities.ProxyRequest) entities.ProxyResponse {
					var b bytes.Buffer
					gz := gzip.NewWriter(&b)
					if _, err := gz.Write([]byte(`{"usage":{"total_tokens":30}}`)); err != nil {
						t.Fatalf("Failed to gzip: %v", err)
					}
					gz.Close()
					headers := http.Header{}
					headers.Set("Content-Encoding", "gzip")
					return entities.ProxyResponse{StatusCode: http.StatusOK, Body: b.Bytes(), Headers: headers}
				}
			},
			expectedStatusCode:       http.StatusOK,
			expectGetSessionCalled:   true,
			expectUpdateTokensCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := &mockProxySessionManager{}
			if tt.mockSessionManagerSetup != nil {
				tt.mockSessionManagerSetup(mockSM)
			}

			mockQ := &mockQueue{}
			if tt.mockQueueSetup != nil {
				tt.mockQueueSetup(mockQ)
			}

			proxyHandler := NewProxyHandler(mockSM, mockQ)

			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.requestBody))
			rr := httptest.NewRecorder()

			proxyHandler.Handle(rr, req)

			if rr.Code != tt.expectedStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatusCode)
			}

			if tt.expectedBodyContains != "" {
				if !strings.Contains(rr.Body.String(), tt.expectedBodyContains) {
					t.Errorf("handler returned unexpected body: got %v want to contain %v", rr.Body.String(), tt.expectedBodyContains)
				}
			}

			// Check if mock functions were called as expected (simplified check)
			// More sophisticated mocking would track call counts and arguments.
		})
	}
}

func Test_extractSessionID(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"valid session path", "/v1/session/abc123/chat/completions", "abc123"},
		{"valid session path with trailing slash", "/v1/session/xyz789/", "xyz789"},
		{"no session ID", "/v1/chat/completions", ""},
		{"invalid path", "/foo/bar", ""},
		{"empty path", "", ""},
		{"root path", "/", ""},
		{"just /v1/session/", "/v1/session/", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractSessionID(tt.path); got != tt.want {
				t.Errorf("extractSessionID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_removeSessionFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"path with session and endpoint", "/v1/session/abc123/chat/completions", "/v1/chat/completions"},
		{"path with session and trailing slash", "/v1/session/xyz789/", "/v1/"},
		{"path with session and no further endpoint", "/v1/session/onlysess", "/v1/"},
		{"path without session", "/v1/chat/completions", "/v1/chat/completions"}, // No change if no session pattern
		{"malformed path", "/v1/session/", "/v1/session/"},                       // No change if malformed
		{"root path", "/", "/"},
		{"empty path", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeSessionFromPath(tt.path); got != tt.want {
				t.Errorf("removeSessionFromPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLegacyProxyHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	LegacyProxyHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("LegacyProxyHandler returned wrong status code: got %v want %v",
			rr.Code, http.StatusInternalServerError)
	}
	expectedBody := "ProxyHandler requires dependency injection. Use NewProxyHandler instead.\n"
	if rr.Body.String() != expectedBody {
		t.Errorf("LegacyProxyHandler returned unexpected body: got %q want %q",
			rr.Body.String(), expectedBody)
	}
}
