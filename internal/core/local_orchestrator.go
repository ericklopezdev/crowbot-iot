package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ErickLopezDev/cwlb-server/internal/services"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

type LocalOrchestrator struct {
	STT services.STTService
	LLM services.LLMService
	TTS services.TTSService
}

func NewLocalOrchestrator(stt services.STTService, llm services.LLMService, tts services.TTSService) *LocalOrchestrator {
	return &LocalOrchestrator{STT: stt, LLM: llm, TTS: tts}
}

func (lo *LocalOrchestrator) ProcessAudio(ctx context.Context, inputText string) ([]byte, error) {
	log.Println("[LocalOrchestrator] processing text:", inputText)

	responseText, err := lo.LLM.Ask(ctx, inputText)
	if err != nil {
		return nil, fmt.Errorf("LLM: %w", err)
	}

	out, err := lo.TTS.Synthesize(ctx, responseText)
	if err != nil {
		return nil, fmt.Errorf("TTS: %w", err)
	}
	return out, nil
}

func (lo *LocalOrchestrator) HandleAudio(ctx context.Context, audioData []byte) (*TurnResult, error) {
	log.Println("[LocalOrchestrator] starting audio pipeline")
	start := time.Now()

	filePath := filepath.Join(os.TempDir(), "cwlb_local_"+time.Now().Format("150405.000")+".wav")
	if err := saveWAV(audioData, filePath); err != nil {
		return nil, fmt.Errorf("save temp WAV: %w", err)
	}
	defer os.Remove(filePath)

	t0 := time.Now()
	text, err := lo.STT.ConvertAudio(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("STT: %w", err)
	}
	sttLatency := time.Since(t0)
	log.Printf("[LocalOrchestrator] transcript: %s", text)

	t1 := time.Now()
	responseText, err := lo.LLM.Ask(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("LLM: %w", err)
	}
	llmLatency := time.Since(t1)
	log.Printf("[LocalOrchestrator] response: %s", responseText)

	t2 := time.Now()
	outputAudio, err := lo.TTS.Synthesize(ctx, responseText)
	if err != nil {
		return nil, fmt.Errorf("TTS: %w", err)
	}
	ttsLatency := time.Since(t2)

	if err := os.WriteFile("local_response.wav", outputAudio, 0644); err != nil {
		log.Printf("[LocalOrchestrator] warning: could not save response WAV: %v", err)
	}

	log.Println("[LocalOrchestrator] pipeline complete")
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

func (lo *LocalOrchestrator) ProcessLocalRecord(ctx context.Context) error {
	if _, err := os.Stat("local_record.wav"); os.IsNotExist(err) {
		return fmt.Errorf("local_record.wav not found")
	}

	text, err := lo.STT.ConvertAudio(ctx, "local_record.wav")
	if err != nil {
		return fmt.Errorf("STT: %w", err)
	}
	log.Printf("[LocalOrchestrator] transcript: %s", text)

	responseText, err := lo.LLM.Ask(ctx, text)
	if err != nil {
		return fmt.Errorf("LLM: %w", err)
	}

	audioData, err := lo.TTS.Synthesize(ctx, responseText)
	if err != nil {
		return fmt.Errorf("TTS: %w", err)
	}

	if err := os.WriteFile("local_response.wav", audioData, 0644); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	if err := lo.TTS.PlayAudio(ctx, audioData); err != nil {
		log.Printf("[LocalOrchestrator] warning: playback failed: %v", err)
	}
	return nil
}

func (lo *LocalOrchestrator) PlayAudio(ctx context.Context, audio []byte) error {
	return lo.TTS.PlayAudio(ctx, audio)
}

func saveWAV(data []byte, fileName string) error {
	const sampleRate = 16000
	buf := &audio.IntBuffer{
		Data:           bytesToInt16(data),
		Format:         &audio.Format{NumChannels: 1, SampleRate: sampleRate},
		SourceBitDepth: 16,
	}
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := wav.NewEncoder(f, sampleRate, 16, 1, 1)
	if err := enc.Write(buf); err != nil {
		return err
	}
	return enc.Close()
}

func bytesToInt16(b []byte) []int {
	out := make([]int, len(b)/2)
	for i := range out {
		out[i] = int(int16(b[i*2]) | int16(b[i*2+1])<<8)
	}
	return out
}
