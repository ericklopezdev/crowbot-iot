package services

import "context"

// MockTTS returns a minimal valid 16-bit PCM WAV (44-byte header + silence).
type MockTTS struct {
	Audio []byte
}

func (m *MockTTS) Synthesize(_ context.Context, _ string) ([]byte, error) {
	if m.Audio != nil {
		return m.Audio, nil
	}
	return silentWAV(), nil
}

func (m *MockTTS) PlayAudio(_ context.Context, _ []byte) error {
	return nil
}

// silentWAV returns a minimal 16kHz mono WAV with 100ms of silence.
func silentWAV() []byte {
	const sampleRate = 16000
	const durationMs = 100
	const numSamples = sampleRate * durationMs / 1000
	const dataSize = numSamples * 2

	h := make([]byte, 44+dataSize)
	copy(h[0:], []byte("RIFF"))
	le32(h[4:], uint32(36+dataSize))
	copy(h[8:], []byte("WAVE"))
	copy(h[12:], []byte("fmt "))
	le32(h[16:], 16)
	le16(h[20:], 1) // PCM
	le16(h[22:], 1) // mono
	le32(h[24:], sampleRate)
	le32(h[28:], sampleRate*2)
	le16(h[32:], 2)  // block align
	le16(h[34:], 16) // bits per sample
	copy(h[36:], []byte("data"))
	le32(h[40:], uint32(dataSize))
	// samples are zero (silence)
	return h
}

func le16(b []byte, v uint16) { b[0] = byte(v); b[1] = byte(v >> 8) }
func le32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}
