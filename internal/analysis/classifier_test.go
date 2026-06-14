package analysis

import (
	"context"
	"testing"

	"github.com/ErickLopezDev/cwlb-server/internal/services"
)

func TestClassify_ParsesFencedJSON(t *testing.T) {
	llm := &services.MockLLM{Response: "```json\n" +
		`{"cognitive_area":"science","question_type":"curiosity","complexity":3,"sentiment":"positive","topics":[{"slug":"dinosaurs","label":"Dinosaurs"}]}` +
		"\n```"}
	c := NewClassifier(llm)

	cl, err := c.Classify(context.Background(), "why are dinosaurs extinct?")
	if err != nil {
		t.Fatal(err)
	}
	if cl.CognitiveArea != "science" {
		t.Errorf("cognitive_area = %q, want science", cl.CognitiveArea)
	}
	if cl.QuestionType != "curiosity" {
		t.Errorf("question_type = %q, want curiosity", cl.QuestionType)
	}
	if len(cl.Topics) != 1 || cl.Topics[0].Slug != "dinosaurs" {
		t.Errorf("topics = %+v, want one dinosaurs topic", cl.Topics)
	}
}

func TestParseClassification_ClampsComplexity(t *testing.T) {
	cl, err := parseClassification(`{"complexity":9}`)
	if err != nil {
		t.Fatal(err)
	}
	if cl.Complexity != 5 {
		t.Errorf("complexity = %d, want clamped to 5", cl.Complexity)
	}

	cl, err = parseClassification(`{"complexity":0}`)
	if err != nil {
		t.Fatal(err)
	}
	if cl.Complexity != 1 {
		t.Errorf("complexity = %d, want clamped to 1", cl.Complexity)
	}
}

func TestParseClassification_Invalid(t *testing.T) {
	if _, err := parseClassification("not json at all"); err == nil {
		t.Error("expected error for non-JSON input")
	}
}
