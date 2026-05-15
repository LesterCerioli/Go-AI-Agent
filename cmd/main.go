package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-ai-agent/internal/config"
	"go-ai-agent/internal/handlers"
	"go-ai-agent/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

// @title           Go AI Agent API
// @version         1.0
// @description     AI-powered backend agent built with Go 1.24+ and Fiber
// @host            localhost:8080
// @BasePath        /api/v1
func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()

	app := fiber.New(fiber.Config{
		AppName:      "Go AI Agent v1.0",
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		ErrorHandler: middleware.ErrorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	app.Static("/", "./web/dist", fiber.Static{
		Compress:  true,
		ByteRange: true,
		Browse:    false,
		Index:     "index.html",
	})

	api := app.Group("/api/v1")
	api.Use(middleware.RequestID())

	api.Get("/health", handlers.HealthCheck)

	agentHandler := handlers.NewAgentHandler(cfg)
	agent := api.Group("/agent")
	agent.Post("/chat", agentHandler.Chat)
	agent.Post("/stream", agentHandler.StreamChat)
	agent.Get("/sessions", agentHandler.ListSessions)
	agent.Get("/sessions/:id", agentHandler.GetSession)
	agent.Delete("/sessions/:id", agentHandler.DeleteSession)

	toolsHandler := handlers.NewToolsHandler(cfg)
	tools := api.Group("/tools")
	tools.Get("/", toolsHandler.ListTools)
	tools.Post("/execute", toolsHandler.ExecuteTool)

	modelsHandler := handlers.NewModelsHandler(cfg)
	api.Get("/models", modelsHandler.ListModels)

	go func() {
		addr := cfg.Host + ":" + cfg.Port
		log.Printf("🚀 Go AI Agent running on http://%s\n", addr)
		log.Printf("📖 API Docs: http://%s/api/v1/health\n", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited cleanly.")
}
