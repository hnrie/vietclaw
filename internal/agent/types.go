package agent

import "database/sql"

const (
	DefaultUserID  = "local"
	DefaultChannel = "web"

	ProviderLocal = "local"
	ModelRule     = "rule"

	RoleUser      = "user"
	RoleAssistant = "assistant"

	RunStatusRunning       = "running"
	RunStatusCompleted     = "completed"
	RunStatusFailed        = "failed"
	RunStatusBlocked       = "blocked"
	RunStatusNeedsApproval = "needs_approval"
)

type ChatRequest struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Channel   string `json:"channel"`
	Message   string `json:"message"`
	Mode      string `json:"mode"`
}

type ChatResponse struct {
	OK        bool    `json:"ok"`
	SessionID string  `json:"session_id"`
	Intent    string  `json:"intent"`
	Reply     string  `json:"reply"`
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	CostUSD   float64 `json:"cost_usd"`
	Error     string  `json:"error,omitempty"`
}

type Session struct {
	ID        string         `json:"id"`
	Channel   string         `json:"channel"`
	UserID    string         `json:"user_id"`
	Title     sql.NullString `json:"title"`
	Summary   sql.NullString `json:"summary"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

type Message struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type SessionDetail struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
}
