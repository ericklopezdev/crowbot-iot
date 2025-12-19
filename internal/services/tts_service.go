package services

type TTSService interface {
	Synthesize(text string) ([]byte, error)
	PlayAudio(wavBytes []byte) error
}
