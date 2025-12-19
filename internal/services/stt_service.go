package services

type STTService interface {
	ConvertAudio(filePath string) (string, error)
}

