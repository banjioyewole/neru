package hint

import (
	"context"
	"sort"

	"github.com/y3owk1n/neru/internal/derrors"
	"github.com/y3owk1n/neru/internal/domain/element"
	"github.com/y3owk1n/neru/internal/domain/hint/vocab"
)

// WordGenerator generates hint labels from the spoken/typed word vocabulary
// (see vocab). Unlike AlphabetGenerator, there is exactly one vocabulary, so
// there is no label-direction axis to configure.
type WordGenerator struct{}

// NewWordGenerator creates a new word-based hint generator. It errors if the
// vocabulary is not prefix-free, since the manager's unique-prefix selection
// rule (manager.go) depends on that invariant.
func NewWordGenerator() (*WordGenerator, error) {
	if !vocab.IsPrefixFree() {
		return nil, derrors.New(derrors.CodeInvalidInput, "hint vocabulary is not prefix-free")
	}

	return &WordGenerator{}, nil
}

// sortAndCap sorts elements into reading order (top-to-bottom, left-to-right)
// and truncates to maxHints. It copies the input slice rather than sorting
// in place.
func sortAndCap(elements []*element.Element, maxHints int) []*element.Element {
	sorted := make([]*element.Element, len(elements))
	copy(sorted, elements)

	sort.Slice(sorted, func(i, j int) bool {
		boundI, boundJ := sorted[i].Bounds(), sorted[j].Bounds()
		if boundI.Min.Y != boundJ.Min.Y {
			return boundI.Min.Y < boundJ.Min.Y
		}

		return boundI.Min.X < boundJ.Min.X
	})

	if len(sorted) > maxHints {
		sorted = sorted[:maxHints]
	}

	return sorted
}

// Generate creates hints for the given elements, one vocabulary word each in
// reading order.
func (g *WordGenerator) Generate(
	ctx context.Context,
	elements []*element.Element,
) ([]*Interface, error) {
	if len(elements) == 0 {
		return nil, nil
	}

	select {
	case <-ctx.Done():
		return nil, derrors.Wrap(ctx.Err(), derrors.CodeContextCanceled, "operation canceled")
	default:
	}

	sorted := sortAndCap(elements, g.MaxHints())

	hints := make([]*Interface, len(sorted))

	for index, elem := range sorted {
		label, _ := vocab.Word(index)
		position := elem.Center()

		hint, err := NewHint(label, elem, position)
		if err != nil {
			return nil, derrors.Wrapf(
				err,
				derrors.CodeHintGenerationFailed,
				"failed to create hint %d: %v",
				index,
				err,
			)
		}

		hints[index] = hint
	}

	return hints, nil
}

// MaxHints returns the maximum number of hints this generator can create.
func (g *WordGenerator) MaxHints() int {
	return vocab.Len()
}
