//go:build portaudio

package services

import (
	"log"
	"time"

	"github.com/gordonklaus/portaudio"
)

// playPCM plays raw 16-bit little-endian PCM at 16kHz via PortAudio.
func playPCM(pcmBytes []byte) error {
	portaudio.Initialize()
	defer portaudio.Terminate()

	samples := make([]int16, len(pcmBytes)/2)
	for i := range samples {
		samples[i] = int16(pcmBytes[i*2]) | int16(pcmBytes[i*2+1])<<8
	}

	const sampleRate = 16000
	stream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(samples), samples)
	if err != nil {
		return err
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		return err
	}
	time.Sleep(time.Duration(len(samples)/sampleRate) * time.Second)
	if err := stream.Stop(); err != nil {
		return err
	}

	log.Println("[TTS] playback finished")
	return nil
}
