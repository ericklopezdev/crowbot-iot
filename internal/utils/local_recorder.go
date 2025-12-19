package utils

import (
	"bytes"
	"log"
	"os"
	"sync"

	"github.com/ErickLopezDev/cwlb-server/internal/core"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
)

type LocalRecorder struct {
	Orchestrator *core.LocalOrchestrator
	Chunks       [][]byte
	Recording    bool
	mu           sync.Mutex
}

func NewLocalRecorder(orchestrator *core.LocalOrchestrator) *LocalRecorder {
	return &LocalRecorder{
		Orchestrator: orchestrator,
		Chunks:       [][]byte{},
	}
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

func (r *LocalRecorder) recordLoop() {
	portaudio.Initialize()
	defer portaudio.Terminate()

	const sampleRate = 16000
	const framesPerBuffer = 1024
	input := make([]int16, framesPerBuffer)

	stream, err := portaudio.OpenDefaultStream(1, 0, sampleRate, len(input), input)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		log.Fatal(err)
	}
	defer stream.Stop()

	for r.Recording {
		if err := stream.Read(); err != nil {
			log.Println("Error reading from mic:", err)
			continue
		}
		// Save chunk
		chunkCopy := make([]byte, len(input)*2)
		for i, v := range input {
			chunkCopy[i*2] = byte(v)
			chunkCopy[i*2+1] = byte(v >> 8)
		}
		r.mu.Lock()
		r.Chunks = append(r.Chunks, chunkCopy)
		r.mu.Unlock()
	}
}

// Concat chunks & send to orchestrator
func (r *LocalRecorder) processAudio() {
	r.mu.Lock()
	fullAudio := bytes.Join(r.Chunks, []byte{})
	r.Chunks = [][]byte{} // limpiar
	r.mu.Unlock()

	// Save temporal WAV
	tmpFile := "local_record.wav"
	saveWAV(fullAudio, tmpFile)

	// Process complete audio
	respAudio, err := r.Orchestrator.HandleAudio(fullAudio)
	if err != nil {
		log.Println("Error processing audio:", err)
		return
	}

	// Reproduce answer
	r.Orchestrator.PlayAudio(respAudio)
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
	for i := range out {
		out[i] = int(int16(b[i*2]) | int16(b[i*2+1])<<8)
	}
	return out
}
