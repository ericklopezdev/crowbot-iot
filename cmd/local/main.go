package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ErickLopezDev/cwlb-server/internal/core"
	"github.com/ErickLopezDev/cwlb-server/internal/services"
	"github.com/ErickLopezDev/cwlb-server/internal/utils"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "n8n-testing-469619-979325bcaed2.json")

	apiKey := os.Getenv("GEMINI_API_KEY")

	log.Println("Initializing GCP STT...")
	stt, err := services.NewGCPSTT()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("GCP STT initialized successfully")

	log.Println("Initializing GCP TTS...")
	tts, err := services.NewGCPTTS()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("GCP TTS initialized successfully")

	log.Println("Initializing Gemini LLM...")
	llm, err := services.NewGeminiLLM(apiKey)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Gemini LLM initialized successfully")

	orchestrator := core.NewLocalOrchestrator(stt, llm, tts)

	recorder := utils.NewLocalRecorder(orchestrator)

	// Captura señal Ctrl+C para salir
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("Exiting...")
		os.Exit(0)
	}()

	log.Println("Press 'R' then Enter to start/stop recording")

	var input string
	for {
		_, err := fmt.Scanln(&input)
		if err != nil {
			log.Println("Error reading input:", err)
			continue
		}
		log.Println("Read input:", input)
		if input == "R" || input == "r" {
			recorder.StartStopRecording()
		}
	}
}
