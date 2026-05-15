package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	AllowedOrigins string

	AnthropicAPIKey string
	DefaultModel    string
	MaxTokens       int

	MaxSessionAge    time.Duration
	MaxConversations int
}

func Load() *Config {
	return &Config{
		Host:             getEnv("HOST", "localhost"),
		Port:             getEnv("PORT", "8080"),
		ReadTimeout:      getDuration("READ_TIMEOUT", 60*time.Second),
		WriteTimeout:     getDuration("WRITE_TIMEOUT", 60*time.Second),
		AllowedOrigins:   getEnv("ALLOWED_ORIGINS", "*"),
		AnthropicAPIKey:  getEnv("ANTHROPIC_API_KEY", ""),
		DefaultModel:     getEnv("DEFAULT_MODEL", "claude-sonnet-4-20250514"),
		MaxTokens:        getInt("MAX_TOKENS", 8192),
		MaxSessionAge:    getDuration("MAX_SESSION_AGE", 24*time.Hour),
		MaxConversations: getInt("MAX_CONVERSATIONS", 100),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return fallback
}
