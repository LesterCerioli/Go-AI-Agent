package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-ai-agent/internal/config"
	"go-ai-agent/internal/models"

	"github.com/google/uuid"
)

// Service handles all agent logic
type Service struct {
	cfg      *config.Config
	sessions map[string]*models.Session
	mu       sync.RWMutex
	client   *http.Client
}

// New creates a new agent service
func New(cfg *config.Config) *Service {
	return &Service{
		cfg:      cfg,
		sessions: make(map[string]*models.Session),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GetOrCreateSession returns existing session or creates a new one
func (s *Service) GetOrCreateSession(sessionID, model, systemPrompt string) *models.Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID != "" {
		if sess, ok := s.sessions[sessionID]; ok {
			return sess
		}
	}

	if model == "" {
		model = s.cfg.DefaultModel
	}

	sess := &models.Session{
		ID:           uuid.NewString(),
		Title:        "New Conversation",
		Messages:     []models.Message{},
		Model:        model,
		SystemPrompt: systemPrompt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.sessions[sess.ID] = sess
	return sess
}

// GetSession retrieves a session by ID
func (s *Service) GetSession(id string) (*models.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// ListSessions returns all sessions
func (s *Service) ListSessions() []*models.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*models.Session, 0, len(s.sessions))
	for _, s := range s.sessions {
		list = append(list, s)
	}
	return list
}

// DeleteSession removes a session
func (s *Service) DeleteSession(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[id]; ok {
		delete(s.sessions, id)
		return true
	}
	return false
}

// Chat sends a message and returns the full response
func (s *Service) Chat(req *models.ChatRequest) (*models.ChatResponse, error) {
	start := time.Now()

	sess := s.GetOrCreateSession(req.SessionID, req.Model, req.SystemPrompt)

	// Add user message
	userMsg := models.Message{
		ID:        uuid.NewString(),
		Role:      models.RoleUser,
		Content:   req.Message,
		Timestamp: time.Now(),
	}

	s.mu.Lock()
	sess.Messages = append(sess.Messages, userMsg)
	if len(sess.Messages) == 1 {
		// Auto-title from first message
		title := req.Message
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		sess.Title = title
	}
	s.mu.Unlock()

	// Build Anthropic API request
	apiReq, err := s.buildAPIRequest(sess, req)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	// Call Anthropic API
	resp, err := s.callAnthropicAPI(apiReq)
	if err != nil {
		return nil, fmt.Errorf("calling API: %w", err)
	}

	// Extract text from response
	content := extractTextContent(resp)
	tokens := resp.Usage.OutputTokens + resp.Usage.InputTokens

	// Store assistant message
	assistantMsg := models.Message{
		ID:        uuid.NewString(),
		Role:      models.RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
		Tokens:    tokens,
	}

	s.mu.Lock()
	sess.Messages = append(sess.Messages, assistantMsg)
	sess.TotalTokens += tokens
	sess.UpdatedAt = time.Now()
	s.mu.Unlock()

	duration := float64(time.Since(start).Milliseconds())

	return &models.ChatResponse{
		SessionID: sess.ID,
		MessageID: assistantMsg.ID,
		Content:   content,
		Model:     sess.Model,
		Tokens:    tokens,
		Duration:  duration,
	}, nil
}

// StreamChat sends a message and streams the response as SSE
func (s *Service) StreamChat(req *models.ChatRequest, writeFn func(chunk models.StreamChunk) error) error {
	sess := s.GetOrCreateSession(req.SessionID, req.Model, req.SystemPrompt)

	userMsg := models.Message{
		ID:        uuid.NewString(),
		Role:      models.RoleUser,
		Content:   req.Message,
		Timestamp: time.Now(),
	}

	s.mu.Lock()
	sess.Messages = append(sess.Messages, userMsg)
	if len(sess.Messages) == 1 {
		title := req.Message
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		sess.Title = title
	}
	s.mu.Unlock()

	apiReq, err := s.buildAPIRequest(sess, req)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	apiReq["stream"] = true

	// Build HTTP request for streaming
	body, _ := json.Marshal(apiReq)
	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", s.cfg.AnthropicAPIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var fullContent strings.Builder
	decoder := newSSEDecoder(resp.Body)

	for {
		event, err := decoder.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch event.Type {
		case "content_block_delta":
			var delta struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(event.Data), &delta); err == nil {
				if delta.Delta.Type == "text_delta" {
					fullContent.WriteString(delta.Delta.Text)
					if err := writeFn(models.StreamChunk{
						Type:    "delta",
						Content: delta.Delta.Text,
					}); err != nil {
						return err
					}
				}
			}
		case "message_stop":
			content := fullContent.String()
			assistantMsg := models.Message{
				ID:        uuid.NewString(),
				Role:      models.RoleAssistant,
				Content:   content,
				Timestamp: time.Now(),
			}
			s.mu.Lock()
			sess.Messages = append(sess.Messages, assistantMsg)
			sess.UpdatedAt = time.Now()
			s.mu.Unlock()

			return writeFn(models.StreamChunk{Done: true})
		}
	}

	return nil
}

// ---- Internal helpers ----

type anthropicRequest struct {
	Model     string                   `json:"model"`
	MaxTokens int                      `json:"max_tokens"`
	System    string                   `json:"system,omitempty"`
	Messages  []map[string]interface{} `json:"messages"`
	Stream    bool                     `json:"stream,omitempty"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *Service) buildAPIRequest(sess *models.Session, req *models.ChatRequest) (map[string]interface{}, error) {
	maxTokens := s.cfg.MaxTokens
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}

	messages := make([]map[string]interface{}, 0, len(sess.Messages))
	for _, m := range sess.Messages {
		if m.Role == models.RoleSystem {
			continue
		}
		messages = append(messages, map[string]interface{}{
			"role":    string(m.Role),
			"content": m.Content,
		})
	}

	system := sess.SystemPrompt
	if system == "" {
		system = "You are a helpful AI assistant for backend Go development. You provide clear, idiomatic Go code examples and concise explanations."
	}

	result := map[string]interface{}{
		"model":      sess.Model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   messages,
	}

	if req.Temperature > 0 {
		result["temperature"] = req.Temperature
	}

	return result, nil
}

func (s *Service) callAnthropicAPI(reqBody map[string]interface{}) (*anthropicResponse, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", s.cfg.AnthropicAPIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("anthropic API error: %s", result.Error.Message)
	}

	return &result, nil
}

func extractTextContent(resp *anthropicResponse) string {
	var sb strings.Builder
	for _, c := range resp.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String()
}

// sseDecoder decodes server-sent events from a stream
type sseDecoder struct {
	reader *bufio.Reader
}

type sseEvent struct {
	Type string
	Data string
}

func newSSEDecoder(r io.Reader) *sseDecoder {
	return &sseDecoder{reader: bufio.NewReader(r)}
}

func (d *sseDecoder) Next() (*sseEvent, error) {
	event := &sseEvent{}
	for {
		line, err := d.reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		if line == "" && event.Type != "" {
			return event, nil
		}

		if strings.HasPrefix(line, "event:") {
			event.Type = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				return &sseEvent{Type: "message_stop"}, nil
			}
			event.Data = data
		}

		if err == io.EOF {
			if event.Type != "" {
				return event, nil
			}
			return nil, io.EOF
		}
		if err != nil {
			return nil, err
		}
	}
}
