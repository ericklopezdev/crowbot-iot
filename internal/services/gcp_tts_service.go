package services

import (
	"context"
	"log"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gordonklaus/portaudio"
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

func (t *GCPTTS) Synthesize(text string) ([]byte, error) {
	ctx := context.Background()

	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{InputSource: &texttospeechpb.SynthesisInput_Text{Text: text}},
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

	log.Printf("[TTS] Synthesized audio length: %d bytes", len(resp.AudioContent))
	return resp.AudioContent, nil
}

func (t *GCPTTS) PlayAudio(pcmBytes []byte) error {
	portaudio.Initialize()
	defer portaudio.Terminate()

	// Convert raw PCM bytes to int16 samples
	samples := make([]int16, len(pcmBytes)/2)
	for i := range samples {
		samples[i] = int16(pcmBytes[i*2]) | int16(pcmBytes[i*2+1])<<8
	}

	const sampleRate = 16000
	stream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(samples), samples)
	if err != nil {
		return err
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		return err
	}
	time.Sleep(time.Duration(len(samples)/sampleRate) * time.Second)
	if err := stream.Stop(); err != nil {
		return err
	}
	log.Println("[TTS] Playback finished")
	return nil
}
