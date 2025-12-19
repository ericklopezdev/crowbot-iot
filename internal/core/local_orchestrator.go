package core

import (
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
	return &LocalOrchestrator{
		STT: stt,
		LLM: llm,
		TTS: tts,
	}
}

// ProcessAudio receives already processed text
// pipeline: text → response → final audio
func (lo *LocalOrchestrator) ProcessAudio(inputText string) ([]byte, error) {
	log.Println("[LocalOrchestrator] Processing text:", inputText)

	// Send text to the LLM model
	responseText, err := lo.LLM.Ask(inputText)
	if err != nil {
		return nil, err
	}
	log.Println("[LocalOrchestrator] LLM response:", responseText)

	// Convert text to audio
	audio, err := lo.TTS.Synthesize(responseText)
	if err != nil {
		return nil, err
	}

	return audio, nil
}

// HandleAudio receives recorded audio bytes, saves them temporarily
// pipeline: STT → LLM → TTS
func (lo *LocalOrchestrator) HandleAudio(audioData []byte) ([]byte, error) {
	log.Println("[LocalOrchestrator] Starting complete audio pipeline...")

	// Create temporary file for STT
	tmpDir := os.TempDir()
	filePath := filepath.Join(tmpDir, "input_audio_"+time.Now().Format("150405")+".wav")

	saveWAV(audioData, filePath)
	log.Println("[LocalOrchestrator] Temporary audio saved at:", filePath)

	// Convert audio to text
	text, err := lo.STT.ConvertAudio(filePath)
	if err != nil {
		return nil, err
	}
	log.Println("[LocalOrchestrator] Recognized text:", text)

	// Pass text to LLM and get response
	responseText, err := lo.LLM.Ask(text)
	if err != nil {
		return nil, err
	}
	log.Println("[LocalOrchestrator] Generated response:", responseText)

	// Convert response to audio
	outputAudio, err := lo.TTS.Synthesize(responseText)
	if err != nil {
		return nil, err
	}

	// Save response audio for debugging
	responseFile := "local_response.wav"
	saveWAV(outputAudio, responseFile)
	log.Printf("[LocalOrchestrator] Response audio saved to: %s", responseFile)

	log.Println("[LocalOrchestrator] Complete pipeline OK")
	return outputAudio, nil
}

// ProcessLocalRecord processes the local_record.wav file through the full pipeline: STT -> LLM -> TTS
func (lo *LocalOrchestrator) ProcessLocalRecord() error {
	log.Println("[LocalOrchestrator] Starting local record processing...")

	// Check if local_record.wav exists
	if _, err := os.Stat("local_record.wav"); os.IsNotExist(err) {
		return logError("local_record.wav not found")
	}

	// Step 1: STT - Convert audio to text
	text, err := lo.STT.ConvertAudio("local_record.wav")
	if err != nil {
		return logError("STT conversion failed: %v", err)
	}
	log.Printf("[LocalOrchestrator] Transcribed text: %s", text)

	// Step 2: LLM - Generate response
	responseText, err := lo.LLM.Ask(text)
	if err != nil {
		return logError("LLM request failed: %v", err)
	}
	log.Printf("[LocalOrchestrator] LLM response: %s", responseText)

	// Step 3: TTS - Synthesize response to audio
	audioData, err := lo.TTS.Synthesize(responseText)
	if err != nil {
		return logError("TTS synthesis failed: %v", err)
	}
	log.Printf("[LocalOrchestrator] Synthesized audio: %d bytes", len(audioData))

	// Save the output audio
	outputFile := "local_response.wav"
	err = os.WriteFile(outputFile, audioData, 0644)
	if err != nil {
		return logError("Failed to save output audio: %v", err)
	}
	log.Printf("[LocalOrchestrator] Output audio saved to: %s", outputFile)

	// Optionally play the audio
	err = lo.TTS.PlayAudio(audioData)
	if err != nil {
		log.Printf("[LocalOrchestrator] Warning: Failed to play audio: %v", err)
	} else {
		log.Println("[LocalOrchestrator] Audio playback completed")
	}

	log.Println("[LocalOrchestrator] Local record processing completed successfully")
	return nil
}

func (lo *LocalOrchestrator) PlayAudio(audio []byte) error {
	return lo.TTS.PlayAudio(audio)
}

func saveWAV(data []byte, fileName string) {
	const sampleRate = 16000
	buf := &audio.IntBuffer{
		Data:           bytesToInt16(data),
		Format:         &audio.Format{NumChannels: 1, SampleRate: sampleRate},
		SourceBitDepth: 16,
	}
	f, _ := os.Create(fileName)
	defer f.Close()
	enc := wav.NewEncoder(f, sampleRate, 16, 1, 1)
	enc.Write(buf)
	enc.Close()
}

func bytesToInt16(b []byte) []int {
	out := make([]int, len(b)/2)
	for i := 0; i < len(out); i++ {
		out[i] = int(int16(b[i*2]) | int16(b[i*2+1])<<8)
	}
	return out
}

func logError(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	log.Printf("[LocalOrchestrator] Error: %v", err)
	return err
}
