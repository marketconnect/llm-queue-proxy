package entities

// SessionData holds information about a session including accumulated token usage
type SessionData struct {
	SessionID             string `json:"session_id"`
	TotalPromptTokens     int    `json:"total_prompt_tokens"`
	TotalCompletionTokens int    `json:"total_completion_tokens"`
	TotalTokens           int    `json:"total_tokens"`
	RequestCount          int    `json:"request_count"`
}
