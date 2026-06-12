package utils

import (
	"bytes"
	"context"
	"log"
	"os"
	"sync"

	"github.com/ErickLopezDev/cwlb-server/internal/core"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

type LocalRecorder struct {
	Orchestrator *core.LocalOrchestrator
	Chunks       [][]byte
	Recording    bool
	mu           sync.Mutex
}

func NewLocalRecorder(orchestrator *core.LocalOrchestrator) *LocalRecorder {
	return &LocalRecorder{Orchestrator: orchestrator}
}

func (r *LocalRecorder) StartStopRecording() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.Recording {
		log.Println("Recording started")
		r.Recording = true
		go r.recordLoop()
	} else {
		log.Println("Recording ended")
		r.Recording = false
		go r.processAudio()
	}
}

func (r *LocalRecorder) processAudio() {
	r.mu.Lock()
	fullAudio := bytes.Join(r.Chunks, nil)
	r.Chunks = nil
	r.mu.Unlock()

	saveWAV(fullAudio, "local_record.wav")

	ctx := context.Background()
	respAudio, err := r.Orchestrator.HandleAudio(ctx, fullAudio)
	if err != nil {
		log.Println("Error processing audio:", err)
		return
	}

	if err := r.Orchestrator.PlayAudio(ctx, respAudio); err != nil {
		log.Println("Error playing audio:", err)
	}
}

func saveWAV(data []byte, fileName string) {
	const sampleRate = 16000
	buf := &audio.IntBuffer{
		Data:           bytesToInt16(data),
		Format:         &audio.Format{NumChannels: 1, SampleRate: sampleRate},
		SourceBitDepth: 16,
	}
	f, err := os.Create(fileName)
	if err != nil {
		log.Printf("saveWAV: %v", err)
		return
	}
	defer f.Close()
	enc := wav.NewEncoder(f, sampleRate, 16, 1, 1)
	enc.Write(buf)
	enc.Close()
}

func bytesToInt16(b []byte) []int {
	out := make([]int, len(b)/2)
	for i := range out {
		out[i] = int(int16(b[i*2]) | int16(b[i*2+1])<<8)
	}
	return out
}
