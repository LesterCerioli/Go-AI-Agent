package models

import "time"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

type Message struct {
	ID        string    `json:"id"`
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Tokens    int       `json:"tokens,omitempty"`
}

type Session struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Messages     []Message `json:"messages"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	TotalTokens  int       `json:"total_tokens"`
}

type ChatRequest struct {
	SessionID    string  `json:"session_id,omitempty"`
	Message      string  `json:"message" validate:"required,min=1,max=32000"`
	Model        string  `json:"model,omitempty"`
	SystemPrompt string  `json:"system_prompt,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
}

type ChatResponse struct {
	SessionID string  `json:"session_id"`
	MessageID string  `json:"message_id"`
	Content   string  `json:"content"`
	Model     string  `json:"model"`
	Tokens    int     `json:"tokens"`
	Duration  float64 `json:"duration_ms"`
}

type StreamChunk struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Done    bool   `json:"done"`
	Error   string `json:"error,omitempty"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Category    string                 `json:"category"`
}

type ToolExecuteRequest struct {
	Tool       string                 `json:"tool" validate:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}

type ToolExecuteResponse struct {
	Tool    string      `json:"tool"`
	Result  interface{} `json:"result"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MaxTokens   int    `json:"max_tokens"`
	Default     bool   `json:"default"`
}
