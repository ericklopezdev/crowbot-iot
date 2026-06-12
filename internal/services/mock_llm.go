package services

import (
	"context"
	"errors"
)

type MockLLM struct {
	Response string
}

func (m *MockLLM) Ask(_ context.Context, _ string) (string, error) {
	if m.Response == "" {
		return "hola, soy crowbot", nil
	}
	return m.Response, nil
}

// ErrorLLM always returns an error; used in tests.
type ErrorLLM struct{}

func (e *ErrorLLM) Ask(_ context.Context, _ string) (string, error) {
	return "", errors.New("llm unavailable")
}
