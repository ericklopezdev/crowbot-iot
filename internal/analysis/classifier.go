// Package analysis turns persisted interactions into developmental signal:
// it classifies each child utterance into a cognitive area, question type,
// complexity, sentiment and topics, and stores the result (PRODUCT.md P2).
package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ErickLopezDev/cwlb-server/internal/services"
)

// Topic is one subject the utterance touched. Slug is a stable kebab-case key
// (the graph node id); Label is the human-readable name.
type Topic struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// Classification is the structured signal extracted from one transcript.
type Classification struct {
	CognitiveArea string  `json:"cognitive_area"`
	QuestionType  string  `json:"question_type"`
	Complexity    int16   `json:"complexity"`
	Sentiment     string  `json:"sentiment"`
	Topics        []Topic `json:"topics"`
}

// Classifier asks the (vendor-neutral) LLM to label a transcript.
type Classifier struct {
	LLM services.LLMService
}

func NewClassifier(llm services.LLMService) *Classifier {
	return &Classifier{LLM: llm}
}

const classifyPrompt = `You analyze a child's utterance to a learning robot and turn it into developmental signal.
Reply with ONLY a JSON object (no prose, no markdown fences) with exactly these fields:
{
  "cognitive_area": one of ["language","logic-math","science","socio-emotional","creativity","social-world"],
  "question_type": one of ["curiosity","homework","emotional","play"],
  "complexity": integer 1-5,
  "sentiment": one of ["positive","neutral","negative"],
  "topics": [{"slug":"kebab-case-key","label":"Human Readable"}]
}
Child said: %q`

// Classify sends the transcript to the LLM and parses the structured result.
func (c *Classifier) Classify(ctx context.Context, transcript string) (*Classification, error) {
	raw, err := c.LLM.Ask(ctx, fmt.Sprintf(classifyPrompt, transcript))
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}
	cl, err := parseClassification(raw)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", raw, err)
	}
	return cl, nil
}

// parseClassification tolerates LLMs that wrap JSON in ``` fences or surround it
// with stray prose, then clamps complexity into the documented 1-5 range.
func parseClassification(raw string) (*Classification, error) {
	s := extractJSON(raw)
	var cl Classification
	if err := json.Unmarshal([]byte(s), &cl); err != nil {
		return nil, err
	}
	switch {
	case cl.Complexity < 1:
		cl.Complexity = 1
	case cl.Complexity > 5:
		cl.Complexity = 5
	}
	return &cl, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	if start := strings.IndexByte(s, '{'); start >= 0 {
		if end := strings.LastIndexByte(s, '}'); end >= start {
			return s[start : end+1]
		}
	}
	return strings.TrimSpace(s)
}
