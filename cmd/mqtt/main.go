package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ErickLopezDev/cwlb-server/internal/config"
	"github.com/ErickLopezDev/cwlb-server/internal/core"
	"github.com/ErickLopezDev/cwlb-server/internal/mqtt"
	"github.com/ErickLopezDev/cwlb-server/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config:", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	stt, err := buildSTT(cfg)
	if err != nil {
		log.Fatal("STT init:", err)
	}
	tts, err := buildTTS(cfg)
	if err != nil {
		log.Fatal("TTS init:", err)
	}
	llm, err := buildLLM(cfg)
	if err != nil {
		log.Fatal("LLM init:", err)
	}

	orchestrator := &core.Orchestrator{STT: stt, LLM: llm, TTS: tts}

	client := mqtt.NewClient(ctx, cfg.MQTTBroker, orchestrator)
	defer client.Disconnect(250)

	log.Printf("MQTT server running (broker=%s, STT=%s, TTS=%s, LLM=%s)",
		cfg.MQTTBroker, cfg.STTProvider, cfg.TTSProvider, cfg.LLMProvider)

	<-ctx.Done()
	log.Println("shutting down")
}

func buildSTT(cfg *config.Config) (services.STTService, error) {
	if cfg.STTProvider == "mock" {
		return &services.MockSTT{}, nil
	}
	return services.NewGCPSTT()
}

func buildTTS(cfg *config.Config) (services.TTSService, error) {
	if cfg.TTSProvider == "mock" {
		return &services.MockTTS{}, nil
	}
	return services.NewGCPTTS()
}

func buildLLM(cfg *config.Config) (services.LLMService, error) {
	if cfg.LLMProvider == "mock" {
		return &services.MockLLM{}, nil
	}
	return services.NewGeminiLLM(cfg.GeminiAPIKey)
}
