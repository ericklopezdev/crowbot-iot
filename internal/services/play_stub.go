//go:build !portaudio

package services

import "errors"

// playPCM is a stub for builds without the `portaudio` tag (server, tests, CI).
// Local audio playback needs `go build -tags portaudio` + the PortAudio library.
func playPCM(_ []byte) error {
	return errors.New("local audio playback unavailable: build with -tags portaudio")
}
