package core_test

import (
	"context"
	"testing"

	"github.com/ErickLopezDev/cwlb-server/internal/core"
	"github.com/ErickLopezDev/cwlb-server/internal/services"
)

func newTestOrchestrator() *core.Orchestrator {
	return &core.Orchestrator{
		STT: &services.MockSTT{Response: "hola crowbot"},
		LLM: &services.MockLLM{Response: "hola humano"},
		TTS: &services.MockTTS{},
	}
}

func TestOrchestrator_ProcessAudio(t *testing.T) {
	o := newTestOrchestrator()
	audio, err := o.ProcessAudio(context.Background(), "hola")
	if err != nil {
		t.Fatal(err)
	}
	if len(audio) == 0 {
		t.Error("expected non-empty audio")
	}
}

func TestOrchestrator_HandleAudio(t *testing.T) {
	o := newTestOrchestrator()
	res, err := o.HandleAudio(context.Background(), make([]byte, 64))
	if err != nil {
		t.Fatal(err)
	}
	if res.Transcript != "hola crowbot" {
		t.Errorf("transcript = %q, want %q", res.Transcript, "hola crowbot")
	}
	if res.ResponseText != "hola humano" {
		t.Errorf("response = %q, want %q", res.ResponseText, "hola humano")
	}
	if len(res.Audio) == 0 {
		t.Error("expected non-empty audio")
	}
}

func TestOrchestrator_ProcessAudio_LLMError(t *testing.T) {
	o := &core.Orchestrator{
		STT: &services.MockSTT{},
		LLM: &services.ErrorLLM{},
		TTS: &services.MockTTS{},
	}
	_, err := o.ProcessAudio(context.Background(), "hi")
	if err == nil {
		t.Error("expected error from LLM")
	}
}
