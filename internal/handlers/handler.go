package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go-ai-agent/internal/agent"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/models"
	"time"

	"github.com/gofiber/fiber/v2"
)

// HealthCheck handles GET /api/v1/health
func HealthCheck(c *fiber.Ctx) error {
	apiKey := c.Locals("apiKeyStatus")
	services := map[string]string{
		"server": "ok",
	}
	if apiKey != nil && apiKey.(string) == "configured" {
		services["anthropic_api"] = "configured"
	} else {
		services["anthropic_api"] = "not configured"
	}

	return c.JSON(models.HealthResponse{
		Status:    "ok",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Services:  services,
	})
}

type AgentHandler struct {
	cfg     *config.Config
	service *agent.Service
}

func NewAgentHandler(cfg *config.Config) *AgentHandler {
	return &AgentHandler{
		cfg:     cfg,
		service: agent.New(cfg),
	}
}

func (h *AgentHandler) Chat(c *fiber.Ctx) error {
	var req models.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Message == "" {
		return fiber.NewError(fiber.StatusBadRequest, "message is required")
	}

	resp, err := h.service.Chat(&req)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(resp)
}

func (h *AgentHandler) StreamChat(c *fiber.Ctx) error {
	var req models.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Message == "" {
		return fiber.NewError(fiber.StatusBadRequest, "message is required")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		writeFn := func(chunk models.StreamChunk) error {
			data, err := json.Marshal(chunk)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(w, "data: %s\n\n", data)
			if err != nil {
				return err
			}
			return w.Flush()
		}

		if err := h.service.StreamChat(&req, writeFn); err != nil {
			errChunk := models.StreamChunk{
				Type:  "error",
				Error: err.Error(),
				Done:  true,
			}
			data, _ := json.Marshal(errChunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
		}
	})

	return nil
}

func (h *AgentHandler) ListSessions(c *fiber.Ctx) error {
	sessions := h.service.ListSessions()
	return c.JSON(fiber.Map{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

func (h *AgentHandler) GetSession(c *fiber.Ctx) error {
	id := c.Params("id")
	sess, ok := h.service.GetSession(id)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "session not found")
	}
	return c.JSON(sess)
}

func (h *AgentHandler) DeleteSession(c *fiber.Ctx) error {
	id := c.Params("id")
	if !h.service.DeleteSession(id) {
		return fiber.NewError(fiber.StatusNotFound, "session not found")
	}
	return c.JSON(fiber.Map{"message": "session deleted", "id": id})
}

type ToolsHandler struct {
	cfg   *config.Config
	tools []models.Tool
}

func NewToolsHandler(cfg *config.Config) *ToolsHandler {
	return &ToolsHandler{
		cfg:   cfg,
		tools: builtinTools(),
	}
}

func (h *ToolsHandler) ListTools(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"tools": h.tools,
		"count": len(h.tools),
	})
}

func (h *ToolsHandler) ExecuteTool(c *fiber.Ctx) error {
	var req models.ToolExecuteRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Tool == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tool name is required")
	}

	result, err := executeTool(req.Tool, req.Parameters)
	if err != nil {
		return c.JSON(models.ToolExecuteResponse{
			Tool:    req.Tool,
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(models.ToolExecuteResponse{
		Tool:    req.Tool,
		Result:  result,
		Success: true,
	})
}

type ModelsHandler struct {
	cfg *config.Config
}

func NewModelsHandler(cfg *config.Config) *ModelsHandler {
	return &ModelsHandler{cfg: cfg}
}

func (h *ModelsHandler) ListModels(c *fiber.Ctx) error {
	modelList := []models.ModelInfo{
		{
			ID:          "claude-opus-4-5",
			Name:        "Claude Opus 4.5",
			Description: "Most capable model for complex tasks",
			MaxTokens:   32000,
			Default:     h.cfg.DefaultModel == "claude-opus-4-5",
		},
		{
			ID:          "claude-sonnet-4-20250514",
			Name:        "Claude Sonnet 4",
			Description: "Balanced performance and speed",
			MaxTokens:   16000,
			Default:     h.cfg.DefaultModel == "claude-sonnet-4-20250514",
		},
		{
			ID:          "claude-haiku-4-5-20251001",
			Name:        "Claude Haiku 4.5",
			Description: "Fast and efficient for simple tasks",
			MaxTokens:   8192,
			Default:     h.cfg.DefaultModel == "claude-haiku-4-5-20251001",
		},
	}
	return c.JSON(fiber.Map{"models": modelList})
}

func builtinTools() []models.Tool {
	return []models.Tool{
		{
			Name:        "code_review",
			Description: "Review Go code for best practices and potential issues",
			Category:    "development",
			Parameters: map[string]interface{}{
				"code": map[string]interface{}{"type": "string", "description": "Go code to review"},
			},
		},
		{
			Name:        "generate_struct",
			Description: "Generate Go struct from JSON schema or description",
			Category:    "development",
			Parameters: map[string]interface{}{
				"description": map[string]interface{}{"type": "string"},
				"json_sample": map[string]interface{}{"type": "string"},
			},
		},
		{
			Name:        "explain_error",
			Description: "Explain a Go compiler or runtime error",
			Category:    "debugging",
			Parameters: map[string]interface{}{
				"error": map[string]interface{}{"type": "string"},
				"code":  map[string]interface{}{"type": "string"},
			},
		},
		{
			Name:        "server_time",
			Description: "Get current server time",
			Category:    "utility",
			Parameters:  map[string]interface{}{},
		},
	}
}

func executeTool(name string, params map[string]interface{}) (interface{}, error) {
	switch name {
	case "server_time":
		return map[string]interface{}{
			"utc":   time.Now().UTC().Format(time.RFC3339),
			"local": time.Now().Format(time.RFC3339),
			"unix":  time.Now().Unix(),
		}, nil
	default:
		return map[string]interface{}{
			"note": fmt.Sprintf("Tool '%s' requires AI processing — use /agent/chat with this tool context", name),
			"tool": name,
		}, nil
	}
}
