package config

import (
	"fmt"
	"os"
)

type Config struct {
	GeminiAPIKey string
	MQTTBroker   string
	STTProvider  string
	TTSProvider  string
	LLMProvider  string
	SystemPrompt string
	// DatabaseURL is optional: when empty the MQTT ingest runs without
	// persistence (interactions are not stored).
	DatabaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		MQTTBroker:   getEnvOrDefault("MQTT_BROKER", "tcp://localhost:1883"),
		STTProvider:  getEnvOrDefault("CROWBOT_STT_PROVIDER", "gcp"),
		TTSProvider:  getEnvOrDefault("CROWBOT_TTS_PROVIDER", "gcp"),
		LLMProvider:  getEnvOrDefault("CROWBOT_LLM_PROVIDER", "gemini"),
		SystemPrompt: os.Getenv("PROMPT"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
	}

	if (cfg.STTProvider == "gcp" || cfg.TTSProvider == "gcp") && os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS required when STT or TTS provider is gcp")
	}
	if cfg.LLMProvider == "gemini" && cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY required when LLM provider is gemini")
	}

	return cfg, nil
}

// APIConfig holds the settings for the HTTP API server (cmd/api). It is kept
// separate from the AI-pipeline Config so the API doesn't require GCP creds.
type APIConfig struct {
	DatabaseURL string
	JWTSecret   string
	Addr        string
}

func LoadAPI() (*APIConfig, error) {
	c := &APIConfig{
		DatabaseURL: getEnvOrDefault("DATABASE_URL", "postgres://crowbot:crowbot@localhost:5433/crowbot?sslmode=disable"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Addr:        getEnvOrDefault("API_ADDR", ":8080"),
	}
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	return c, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
