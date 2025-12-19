package core

import (
	"log"
	"path/filepath"
	"time"

	"github.com/ErickLopezDev/cwlb-server/internal/services"
)

type OrchestratorInterface interface {
	HandleAudio(audioData []byte) ([]byte, error)
	ProcessAudio(inputText string) ([]byte, error)
	PlayAudio(audio []byte) error
}

type Orchestrator struct {
	STT services.STTService
	LLM services.LLMService
	TTS services.TTSService
}

// ProcessAudio receives already processed text
// pipeline: text → response → final audio
func (o *Orchestrator) ProcessAudio(inputText string) ([]byte, error) {
	log.Println("[Orchestrator] Processing text:", inputText)

	// Send text to the LLM model
	responseText, err := o.LLM.Ask(inputText)
	if err != nil {
		return nil, err
	}
	log.Println("[Orchestrator] LLM response:", responseText)

	// Convert text to audio
	audio, err := o.TTS.Synthesize(responseText)
	if err != nil {
		return nil, err
	}

	return audio, nil
}

// HandleAudio receives recorded audio bytes, saves them temporarily
// pipeline: STT → LLM → TTS
func (o *Orchestrator) HandleAudio(audioData []byte) ([]byte, error) {
	log.Println("[Orchestrator] Starting complete audio pipeline...")

	// Create temporary file for STT
	tmpDir := "."
	filePath := filepath.Join(tmpDir, "input_audio_"+time.Now().Format("150405")+".wav")

	saveWAV(audioData, filePath)
	log.Println("[Orchestrator] Temporary audio saved at:", filePath)

	// Convert audio to text
	text, err := o.STT.ConvertAudio(filePath)
	if err != nil {
		return nil, err
	}
	log.Println("[Orchestrator] Recognized text:", text)

	// Pass text to LLM and get response
	responseText, err := o.LLM.Ask(text)
	if err != nil {
		return nil, err
	}
	log.Println("[Orchestrator] Generated response:", responseText)

	// Convert response to audio
	outputAudio, err := o.TTS.Synthesize(responseText)
	if err != nil {
		return nil, err
	}

	log.Println("[Orchestrator] Complete pipeline OK")
	return outputAudio, nil
}

func (o *Orchestrator) PlayAudio(audio []byte) error {
	return o.TTS.PlayAudio(audio)
}
