package modes

import (
	"image"
	"testing"
)

func TestRectFrom(t *testing.T) {
	t.Parallel()

	got := rectFrom(image.Rect(10, 20, 110, 70))

	want := Rect{X: 10, Y: 20, Width: 100, Height: 50}
	if got != want {
		t.Errorf("rectFrom() = %+v, want %+v", got, want)
	}
}

// The payload uses width and height rather than a second corner, so a caller
// does not have to know whether the rectangle is inclusive at its far edge.
func TestRectFromEmptyRectangle(t *testing.T) {
	t.Parallel()

	got := rectFrom(image.Rectangle{})
	if got != (Rect{}) {
		t.Errorf("rectFrom(empty) = %+v, want zero", got)
	}
}

func TestPointFrom(t *testing.T) {
	t.Parallel()

	if got := pointFrom(image.Pt(7, 9)); got != (Point{X: 7, Y: 9}) {
		t.Errorf("pointFrom() = %+v, want {7 9}", got)
	}
}
