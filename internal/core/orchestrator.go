package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ErickLopezDev/cwlb-server/internal/services"
)

type OrchestratorInterface interface {
	HandleAudio(ctx context.Context, audioData []byte) ([]byte, error)
	ProcessAudio(ctx context.Context, inputText string) ([]byte, error)
	PlayAudio(ctx context.Context, audio []byte) error
}

type Orchestrator struct {
	STT services.STTService
	LLM services.LLMService
	TTS services.TTSService
}

// ProcessAudio runs the text → LLM → TTS pipeline.
func (o *Orchestrator) ProcessAudio(ctx context.Context, inputText string) ([]byte, error) {
	log.Println("[Orchestrator] processing text:", inputText)

	responseText, err := o.LLM.Ask(ctx, inputText)
	if err != nil {
		return nil, fmt.Errorf("LLM: %w", err)
	}

	audio, err := o.TTS.Synthesize(ctx, responseText)
	if err != nil {
		return nil, fmt.Errorf("TTS: %w", err)
	}
	return audio, nil
}

// HandleAudio runs the full STT → LLM → TTS pipeline from raw PCM bytes.
func (o *Orchestrator) HandleAudio(ctx context.Context, audioData []byte) ([]byte, error) {
	log.Println("[Orchestrator] starting audio pipeline")

	filePath := filepath.Join(os.TempDir(), "cwlb_input_"+time.Now().Format("150405.000")+".wav")
	if err := saveWAV(audioData, filePath); err != nil {
		return nil, fmt.Errorf("save temp WAV: %w", err)
	}
	defer os.Remove(filePath)

	text, err := o.STT.ConvertAudio(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("STT: %w", err)
	}
	log.Println("[Orchestrator] recognized:", text)

	responseText, err := o.LLM.Ask(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("LLM: %w", err)
	}
	log.Println("[Orchestrator] response:", responseText)

	outputAudio, err := o.TTS.Synthesize(ctx, responseText)
	if err != nil {
		return nil, fmt.Errorf("TTS: %w", err)
	}

	log.Println("[Orchestrator] pipeline complete")
	return outputAudio, nil
}

func (o *Orchestrator) PlayAudio(ctx context.Context, audio []byte) error {
	return o.TTS.PlayAudio(ctx, audio)
}
