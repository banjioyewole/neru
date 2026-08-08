//go:build linux

package linux

import (
	"image"
	"testing"
)

// Unit tests for the pure hint-label matched-prefix split math (no cgo,
// unlike the Cairo draw calls it feeds). See splitHintLabel in manager.go.
func TestSplitHintLabel_NoMatchReturnsNotOK(t *testing.T) {
	badgeRect := image.Rect(0, 0, 100, 20)

	if _, _, _, ok := splitHintLabel("ALPHA", "", badgeRect, 14); ok {
		t.Fatal("empty matchedPrefix should not split")
	}
}

func TestSplitHintLabel_FullMatchReturnsNotOK(t *testing.T) {
	badgeRect := image.Rect(0, 0, 100, 20)

	if _, _, _, ok := splitHintLabel("ALPHA", "ALPHA", badgeRect, 14); ok {
		t.Fatal("a fully matched label should not split — caller draws it as one run")
	}
}

func TestSplitHintLabel_PartialMatchSplitsHeadAndTail(t *testing.T) {
	badgeRect := image.Rect(0, 0, 100, 20)

	headRect, tailRect, tail, ok := splitHintLabel("ALPHA", "AL", badgeRect, 14)
	if !ok {
		t.Fatal("a partial match should split")
	}

	if tail != "PHA" {
		t.Errorf("tail = %q, want %q", tail, "PHA")
	}

	if headRect.Dx() <= 0 {
		t.Errorf("headRect has non-positive width: %v", headRect)
	}

	if tailRect.Dx() <= 0 {
		t.Errorf("tailRect has non-positive width: %v", tailRect)
	}

	if headRect.Max.X != tailRect.Min.X {
		t.Errorf("head and tail rects are not adjacent: head=%v tail=%v", headRect, tailRect)
	}

	// The combined head+tail block is centered within badgeRect, matching how
	// the unsplit label would have been centered.
	totalWidth := tailRect.Max.X - headRect.Min.X
	wantStartX := badgeRect.Min.X + (badgeRect.Dx()-totalWidth)/2
	if headRect.Min.X != wantStartX {
		t.Errorf("split block start X = %d, want %d (centered in badgeRect)", headRect.Min.X, wantStartX)
	}
}

func TestSplitHintLabel_ResultFitsWithinBadgeHeight(t *testing.T) {
	badgeRect := image.Rect(10, 5, 110, 25)

	headRect, tailRect, _, ok := splitHintLabel("NOVEMBER", "NO", badgeRect, 14)
	if !ok {
		t.Fatal("expected a split")
	}

	if headRect.Min.Y != badgeRect.Min.Y || headRect.Max.Y != badgeRect.Max.Y {
		t.Errorf("headRect Y span = [%d,%d], want badgeRect's [%d,%d]",
			headRect.Min.Y, headRect.Max.Y, badgeRect.Min.Y, badgeRect.Max.Y)
	}

	if tailRect.Min.Y != badgeRect.Min.Y || tailRect.Max.Y != badgeRect.Max.Y {
		t.Errorf("tailRect Y span = [%d,%d], want badgeRect's [%d,%d]",
			tailRect.Min.Y, tailRect.Max.Y, badgeRect.Min.Y, badgeRect.Max.Y)
	}
}
