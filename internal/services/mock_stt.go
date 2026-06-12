package services

import "context"

type MockSTT struct {
	Response string
}

func (m *MockSTT) ConvertAudio(_ context.Context, _ string) (string, error) {
	if m.Response == "" {
		return "este es un texto de prueba", nil
	}
	return m.Response, nil
}
