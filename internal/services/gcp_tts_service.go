package services

import (
	"context"
	"log"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

type GCPTTS struct {
	client *texttospeech.Client
}

func NewGCPTTS() (*GCPTTS, error) {
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCPTTS{client: client}, nil
}

func (t *GCPTTS) Synthesize(ctx context.Context, text string) ([]byte, error) {
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "es-ES",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_FEMALE,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   texttospeechpb.AudioEncoding_LINEAR16,
			SampleRateHertz: 16000,
		},
	}

	resp, err := t.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Printf("[TTS] synthesized %d bytes", len(resp.AudioContent))
	return resp.AudioContent, nil
}

// PlayAudio plays raw 16kHz PCM locally. Requires building with `-tags portaudio`
// (and the PortAudio system library); otherwise it returns an error. The server
// path never calls this — it sends audio back over MQTT instead.
func (t *GCPTTS) PlayAudio(_ context.Context, pcmBytes []byte) error {
	return playPCM(pcmBytes)
}
