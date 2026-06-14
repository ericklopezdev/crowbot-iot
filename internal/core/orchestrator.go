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
	HandleAudio(ctx context.Context, audioData []byte) (*TurnResult, error)
	ProcessAudio(ctx context.Context, inputText string) ([]byte, error)
	PlayAudio(ctx context.Context, audio []byte) error
}

// TurnResult carries everything one STT→LLM→TTS turn produced, so callers can
// both reply with the audio and persist the raw turn.
type TurnResult struct {
	Audio          []byte
	Transcript     string
	ResponseText   string
	STTLatencyMs   int32
	LLMLatencyMs   int32
	TTSLatencyMs   int32
	TotalLatencyMs int32
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

// HandleAudio runs the full STT → LLM → TTS pipeline from raw PCM bytes and
// returns the response audio plus the transcript, response text and per-stage
// latencies so the caller can persist the turn.
func (o *Orchestrator) HandleAudio(ctx context.Context, audioData []byte) (*TurnResult, error) {
	log.Println("[Orchestrator] starting audio pipeline")
	start := time.Now()

	filePath := filepath.Join(os.TempDir(), "cwlb_input_"+time.Now().Format("150405.000")+".wav")
	if err := saveWAV(audioData, filePath); err != nil {
		return nil, fmt.Errorf("save temp WAV: %w", err)
	}
	defer os.Remove(filePath)

	t0 := time.Now()
	text, err := o.STT.ConvertAudio(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("STT: %w", err)
	}
	sttLatency := time.Since(t0)
	log.Println("[Orchestrator] recognized:", text)

	t1 := time.Now()
	responseText, err := o.LLM.Ask(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("LLM: %w", err)
	}
	llmLatency := time.Since(t1)
	log.Println("[Orchestrator] response:", responseText)

	t2 := time.Now()
	outputAudio, err := o.TTS.Synthesize(ctx, responseText)
	if err != nil {
		return nil, fmt.Errorf("TTS: %w", err)
	}
	ttsLatency := time.Since(t2)

	log.Println("[Orchestrator] pipeline complete")
	return &TurnResult{
		Audio:          outputAudio,
		Transcript:     text,
		ResponseText:   responseText,
		STTLatencyMs:   int32(sttLatency.Milliseconds()),
		LLMLatencyMs:   int32(llmLatency.Milliseconds()),
		TTSLatencyMs:   int32(ttsLatency.Milliseconds()),
		TotalLatencyMs: int32(time.Since(start).Milliseconds()),
	}, nil
}

func (o *Orchestrator) PlayAudio(ctx context.Context, audio []byte) error {
	return o.TTS.PlayAudio(ctx, audio)
}
