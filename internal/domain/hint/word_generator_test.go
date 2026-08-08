package hint_test

import (
	"context"
	"fmt"
	"image"
	"slices"
	"testing"

	"github.com/y3owk1n/neru/internal/domain/element"
	"github.com/y3owk1n/neru/internal/domain/hint"
	"github.com/y3owk1n/neru/internal/domain/hint/vocab"
)

func TestNewWordGenerator_Succeeds(t *testing.T) {
	if _, err := hint.NewWordGenerator(); err != nil {
		t.Fatalf("NewWordGenerator: %v", err)
	}
}

// TestWordGenerator_Generate_AssignsLabelsInReadingOrder pins the same
// top-to-bottom, left-to-right contract AlphabetGenerator honored — users
// navigate by position, so the sort must not scramble it.
func TestWordGenerator_Generate_AssignsLabelsInReadingOrder(t *testing.T) {
	generator, err := hint.NewWordGenerator()
	if err != nil {
		t.Fatalf("NewWordGenerator: %v", err)
	}

	type placed struct {
		id   string
		x, y int
	}

	input := []placed{
		{"row2-right", 300, 200},
		{"row1-right", 300, 100},
		{"row2-left", 100, 200},
		{"row1-left", 100, 100},
		{"row1-middle", 200, 100},
	}

	elements := make([]*element.Element, 0, len(input))

	for _, spec := range input {
		created, elemErr := element.NewElement(
			element.ID(spec.id),
			image.Rect(spec.x, spec.y, spec.x+50, spec.y+20),
			element.RoleButton,
		)
		if elemErr != nil {
			t.Fatalf("NewElement(%s): %v", spec.id, elemErr)
		}

		elements = append(elements, created)
	}

	hints, err := generator.Generate(context.Background(), elements)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(hints) != len(elements) {
		t.Fatalf("Generate returned %d hints, want %d", len(hints), len(elements))
	}

	wantOrder := []string{"row1-left", "row1-middle", "row1-right", "row2-left", "row2-right"}

	gotOrder := make([]string, 0, len(hints))
	for _, generated := range hints {
		gotOrder = append(gotOrder, string(generated.Element().ID()))
	}

	if !slices.Equal(gotOrder, wantOrder) {
		t.Errorf("elements labeled in order %v, want reading order %v", gotOrder, wantOrder)
	}

	wantLabels := []string{"ALPHA", "BRAVO", "CHARLIE", "DELTA", "ECHO"}

	gotLabels := make([]string, 0, len(hints))
	for _, generated := range hints {
		gotLabels = append(gotLabels, generated.Label())
	}

	if !slices.Equal(gotLabels, wantLabels) {
		t.Errorf("labels = %v, want %v", gotLabels, wantLabels)
	}
}

// TestWordGenerator_Generate_TruncatesAtMaxHints ensures elements beyond
// vocab.Len() are dropped, exactly as AlphabetGenerator truncated at its
// character-derived MaxHints.
func TestWordGenerator_Generate_TruncatesAtMaxHints(t *testing.T) {
	generator, err := hint.NewWordGenerator()
	if err != nil {
		t.Fatalf("NewWordGenerator: %v", err)
	}

	maxHints := generator.MaxHints()

	elements := make([]*element.Element, 0, maxHints+5)

	for i := range maxHints + 5 {
		created, elemErr := element.NewElement(
			element.ID(fmt.Sprintf("elem-%d", i)),
			image.Rect(i, i, i+10, i+10),
			element.RoleButton,
		)
		if elemErr != nil {
			t.Fatalf("NewElement: %v", elemErr)
		}

		elements = append(elements, created)
	}

	hints, err := generator.Generate(context.Background(), elements)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(hints) != maxHints {
		t.Fatalf("Generate returned %d hints, want %d (MaxHints)", len(hints), maxHints)
	}
}

func TestWordGenerator_Generate_EmptyInput(t *testing.T) {
	generator, err := hint.NewWordGenerator()
	if err != nil {
		t.Fatalf("NewWordGenerator: %v", err)
	}

	hints, err := generator.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(hints) != 0 {
		t.Fatalf("Generate(nil) returned %d hints, want 0", len(hints))
	}
}

func TestWordGenerator_MaxHints_MatchesVocabLen(t *testing.T) {
	generator, err := hint.NewWordGenerator()
	if err != nil {
		t.Fatalf("NewWordGenerator: %v", err)
	}

	if generator.MaxHints() != vocab.Len() {
		t.Fatalf("MaxHints() = %d, want vocab.Len() = %d", generator.MaxHints(), vocab.Len())
	}
}
