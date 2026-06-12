package services

import (
	"context"
	"fmt"
	"log"
	"os"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
)

type GCPSTT struct {
	client *speech.Client
}

func NewGCPSTT() (*GCPSTT, error) {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCPSTT{client: client}, nil
}

func (s *GCPSTT) ConvertAudio(ctx context.Context, filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// skip 44-byte WAV header to get raw PCM
	if len(data) > 44 {
		data = data[44:]
	}

	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "es-ES",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: data},
		},
	}

	resp, err := s.client.Recognize(ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Results) == 0 {
		return "", fmt.Errorf("no transcription results")
	}

	text := resp.Results[0].Alternatives[0].Transcript
	log.Printf("[STT] transcription: %s", text)
	return text, nil
}
