package services

import "context"

type STTService interface {
	ConvertAudio(ctx context.Context, filePath string) (string, error)
}
