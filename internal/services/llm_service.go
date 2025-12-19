package services

type LLMService interface {
	Ask(text string) (string, error)
}
