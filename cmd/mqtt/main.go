package main

import (
	"log"
	"os"

	"github.com/ErickLopezDev/cwlb-server/internal/core"
	"github.com/ErickLopezDev/cwlb-server/internal/mqtt"
	"github.com/ErickLopezDev/cwlb-server/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "n8n-testing-469619-979325bcaed2.json")

	apiKey := os.Getenv("GEMINI_API_KEY")

	log.Println("Initializing GCP STT...")
	stt, err := services.NewGCPSTT()
	if err != nil {
		log.Fatal("STT init failed:", err)
	}
	log.Println("GCP STT initialized successfully")

	log.Println("Initializing GCP TTS...")
	tts, err := services.NewGCPTTS()
	if err != nil {
		log.Fatal("TTS init failed:", err)
	}
	log.Println("GCP TTS initialized successfully")

	log.Println("Initializing Gemini LLM...")
	llm, err := services.NewGeminiLLM(apiKey)
	if err != nil {
		log.Fatal("LLM init failed:", err)
	}
	log.Println("Gemini LLM initialized successfully")

	orchestrator := &core.Orchestrator{
		STT: stt,
		LLM: llm,
		TTS: tts,
	}

	broker := "tcp://localhost:1883" 
	_ = mqtt.NewClient(broker, orchestrator)

	log.Println("MQTT server running...")

	// Keep running
	select {}
}
