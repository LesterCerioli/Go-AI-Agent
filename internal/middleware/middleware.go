package middleware

import (
	"net/http"

	"go-ai-agent/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	} else if err != nil {
		message = err.Error()
	}

	return c.Status(code).JSON(models.ErrorResponse{
		Error:   http.StatusText(code),
		Code:    code,
		Message: message,
	})
}

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("X-Request-ID", id)
		c.Locals("requestID", id)
		return c.Next()
	}
}
