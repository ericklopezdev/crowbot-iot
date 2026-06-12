package services

import "context"

type TTSService interface {
	Synthesize(ctx context.Context, text string) ([]byte, error)
	PlayAudio(ctx context.Context, wavBytes []byte) error
}
