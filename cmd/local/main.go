package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ErickLopezDev/cwlb-server/internal/config"
	"github.com/ErickLopezDev/cwlb-server/internal/core"
	"github.com/ErickLopezDev/cwlb-server/internal/services"
	"github.com/ErickLopezDev/cwlb-server/internal/utils"
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

	var stt services.STTService
	if cfg.STTProvider == "mock" {
		stt = &services.MockSTT{}
	} else {
		stt, err = services.NewGCPSTT()
		if err != nil {
			log.Fatal("STT init:", err)
		}
	}

	var tts services.TTSService
	if cfg.TTSProvider == "mock" {
		tts = &services.MockTTS{}
	} else {
		tts, err = services.NewGCPTTS()
		if err != nil {
			log.Fatal("TTS init:", err)
		}
	}

	var llm services.LLMService
	if cfg.LLMProvider == "mock" {
		llm = &services.MockLLM{}
	} else {
		llm, err = services.NewGeminiLLM(cfg.GeminiAPIKey)
		if err != nil {
			log.Fatal("LLM init:", err)
		}
	}

	orchestrator := core.NewLocalOrchestrator(stt, llm, tts)
	recorder := utils.NewLocalRecorder(orchestrator)

	go func() {
		<-ctx.Done()
		log.Println("exiting")
		os.Exit(0)
	}()

	log.Printf("ready (STT=%s, TTS=%s, LLM=%s) — press 'R' then Enter to start/stop recording",
		cfg.STTProvider, cfg.TTSProvider, cfg.LLMProvider)

	var input string
	for {
		if _, err := fmt.Scanln(&input); err != nil {
			log.Println("read error:", err)
			continue
		}
		if input == "R" || input == "r" {
			recorder.StartStopRecording()
		}
	}
}
