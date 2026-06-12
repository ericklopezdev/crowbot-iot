package services

import "context"

type LLMService interface {
	Ask(ctx context.Context, text string) (string, error)
}
