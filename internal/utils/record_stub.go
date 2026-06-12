//go:build !portaudio

package utils

import "log"

// recordLoop is a stub for builds without the `portaudio` tag. Real mic capture
// needs `go build -tags portaudio ./cmd/local` + the PortAudio system library.
func (r *LocalRecorder) recordLoop() {
	log.Println("[recorder] mic capture unavailable: rebuild cmd/local with -tags portaudio")
	r.mu.Lock()
	r.Recording = false
	r.mu.Unlock()
}
