package recursivegrid_test

import (
	"image"
	"testing"

	"github.com/y3owk1n/neru/internal/adapter/overlay/render/recursivegrid"
)

func cell(w, h int) image.Rectangle {
	return image.Rect(0, 0, w, h)
}

// A 3×3 grid over a laptop display: the first depth is a cell the size of a
// window, the fourth is a cell the size of a button.
func TestLabelFontSizeScalesWithTheCell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cell       image.Rectangle
		configured int
		want       int
	}{
		{"first depth is capped rather than enormous", cell(1152, 745), 10, 72},
		{"second depth stays comfortably readable", cell(384, 248), 10, 45},
		{"deep cells bottom out at the configured size", cell(43, 28), 10, 10},
		{"a configured size larger than the scale wins", cell(43, 28), 24, 24},
		{"an empty cell cannot be measured", image.Rectangle{}, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := recursivegrid.LabelFontSize(tt.cell, tt.configured); got != tt.want {
				t.Errorf("LabelFontSize(%v, %d) = %d, want %d", tt.cell, tt.configured, got, tt.want)
			}
		})
	}
}

// The word is what the speaker says, so it is what the cell shows for as long
// as it fits inside one.
func TestCellLabelPrefersThePhoneticWord(t *testing.T) {
	t.Parallel()

	big := cell(1152, 745)
	size := recursivegrid.LabelFontSize(big, 10)

	for key, want := range map[rune]string{'n': "NOVEMBER", 'g': "GOLF", 'f': "FOXTROT"} {
		if got := recursivegrid.CellLabel(key, big, size); got != want {
			t.Errorf("CellLabel(%q) = %q, want %q", key, got, want)
		}
	}
}

// Every cell at one depth must agree: the size is chosen so the longest word
// fits, and a grid showing NOVEMBER beside a bare G would read as a bug.
func TestEveryKeyInAGridShowsAWordAtTheSameDepth(t *testing.T) {
	t.Parallel()

	for _, dims := range [][2]int{{1152, 745}, {384, 248}, {128, 83}} {
		c := cell(dims[0], dims[1])
		size := recursivegrid.LabelFontSize(c, 10)

		for _, key := range "rtyfghvbn" {
			if got := recursivegrid.CellLabel(key, c, size); len(got) == 1 {
				t.Errorf("cell %v at %dpt fell back to %q", c, size, got)
			}
		}
	}
}

// Once the scale has bottomed out the word no longer fits, and a letter that
// can be read beats a word that cannot.
func TestCellLabelFallsBackToTheKeyInSmallCells(t *testing.T) {
	t.Parallel()

	small := cell(43, 28)

	if got := recursivegrid.CellLabel('n', small, recursivegrid.LabelFontSize(small, 10)); got != "N" {
		t.Errorf("CellLabel in a small cell = %q, want %q", got, "N")
	}
}

// A grid configured with digits or symbols has no word to show, and must still
// label its cells.
func TestCellLabelFallsBackForKeysWithNoWord(t *testing.T) {
	t.Parallel()

	if got := recursivegrid.CellLabel('7', cell(1152, 745), 72); got != "7" {
		t.Errorf("CellLabel('7') = %q, want %q", got, "7")
	}
}
