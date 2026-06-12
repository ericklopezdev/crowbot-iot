//go:build portaudio

package utils

import (
	"log"

	"github.com/gordonklaus/portaudio"
)

// recordLoop captures mic audio via PortAudio until Recording is set false,
// appending 16-bit little-endian PCM chunks to the recorder.
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

	for {
		r.mu.Lock()
		recording := r.Recording
		r.mu.Unlock()
		if !recording {
			return
		}

		if err := stream.Read(); err != nil {
			log.Println("Error reading from mic:", err)
			continue
		}
		chunk := make([]byte, len(input)*2)
		for i, v := range input {
			chunk[i*2] = byte(v)
			chunk[i*2+1] = byte(v >> 8)
		}
		r.mu.Lock()
		r.Chunks = append(r.Chunks, chunk)
		r.mu.Unlock()
	}
}
