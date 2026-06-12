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
}

func Load() (*Config, error) {
	cfg := &Config{
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		MQTTBroker:   getEnvOrDefault("MQTT_BROKER", "tcp://localhost:1883"),
		STTProvider:  getEnvOrDefault("CROWBOT_STT_PROVIDER", "gcp"),
		TTSProvider:  getEnvOrDefault("CROWBOT_TTS_PROVIDER", "gcp"),
		LLMProvider:  getEnvOrDefault("CROWBOT_LLM_PROVIDER", "gemini"),
		SystemPrompt: os.Getenv("PROMPT"),
	}

	if (cfg.STTProvider == "gcp" || cfg.TTSProvider == "gcp") && os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS required when STT or TTS provider is gcp")
	}
	if cfg.LLMProvider == "gemini" && cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY required when LLM provider is gemini")
	}

	return cfg, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
