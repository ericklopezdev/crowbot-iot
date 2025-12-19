package services

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
)

type GeminiLLM struct {
	client *genai.Client
}

func NewGeminiLLM(apiKey string) (*GeminiLLM, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, err
	}

	return &GeminiLLM{client: client}, nil
}

func (g *GeminiLLM) Ask(prompt string) (string, error) {
	ctx := context.Background()

	log.Printf("[Gemini] Sending prompt: %q", prompt)

	systemPrompt := os.Getenv("PROMPT")
	resp, err := g.client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash",
		genai.Text(systemPrompt+"\nUser prompt: "+prompt),
		nil,
	)

	if err != nil {
		return "", fmt.Errorf("gemini request failed: %w", err)
	}

	return resp.Text(), nil
}
