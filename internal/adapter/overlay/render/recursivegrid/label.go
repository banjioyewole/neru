package recursivegrid

import (
	"image"
	"math"
	"strings"
	"unicode"
)

const (
	// maxLabelFontSize caps the label size on a full-screen cell. Past this the
	// label stops reading as a label and starts reading as a sign.
	maxLabelFontSize = 72
	// labelCellHeightFraction and labelCellWidthFraction size the label to the
	// cell it sits in. The width fraction is the binding one: it is chosen so
	// that the longest phonetic word ("november", eight characters) still fits
	// across the cell with room either side, which is what keeps every cell at
	// one depth showing a word rather than some words and some letters.
	labelCellHeightFraction = 0.18
	labelCellWidthFraction  = 0.12
	// averageGlyphWidth is the fraction of the point size one uppercase glyph
	// takes in the bold system font, measured across the phonetic alphabet.
	// Used to decide whether a word fits without asking the text system, which
	// lives on the other side of cgo from this decision.
	averageGlyphWidth = 0.68
	// labelFillFraction is how much of the cell a label may occupy. The rest is
	// the margin that keeps a word from touching the cell borders.
	labelFillFraction = 0.8
	// labelLineHeight converts a point size to the height a single line of it
	// occupies, ascender to descender.
	labelLineHeight = 1.4
)

// natoWords is the phonetic word for each key a grid can be built from.
//
// The recursive grid is driven by voice as often as by keyboard here, and a
// recognizer asked for a single letter produces whatever it likes — "be" for b,
// "in" for n, nothing at all for h. The phonetic alphabet exists because people
// have needed to say letters over a bad channel for a century. Printing the
// word the speaker is meant to say, rather than the letter the keyboard is
// meant to send, is what makes the overlay the prompt for saying it.
var natoWords = map[rune]string{
	'a': "alpha", 'b': "bravo", 'c': "charlie", 'd': "delta", 'e': "echo",
	'f': "foxtrot", 'g': "golf", 'h': "hotel", 'i': "india", 'j': "juliet",
	'k': "kilo", 'l': "lima", 'm': "mike", 'n': "november", 'o': "oscar",
	'p': "papa", 'q': "quebec", 'r': "romeo", 's': "sierra", 't': "tango",
	'u': "uniform", 'v': "victor", 'w': "whiskey", 'x': "xray", 'y': "yankee",
	'z': "zulu",
}

// LabelFontSize is the size cell labels are drawn at for cells of this size.
//
// Scaled to the cell rather than fixed, because a recursive grid's cells span
// three orders of area between the first depth and the last: one size legible
// at depth four is invisible at depth zero, and one legible at depth zero does
// not fit at depth four. The configured font size is the floor, not the value —
// it is what the scale bottoms out at once the cells are small.
func LabelFontSize(cell image.Rectangle, configured int) int {
	if cell.Empty() {
		return configured
	}

	scaled := math.Min(
		float64(cell.Dy())*labelCellHeightFraction,
		float64(cell.Dx())*labelCellWidthFraction,
	)

	size := int(math.Round(math.Min(scaled, maxLabelFontSize)))
	if size < configured {
		return configured
	}

	return size
}

// CellLabel is the text one cell carries: the phonetic word while it fits
// inside the cell at this font size, the bare key once it does not.
//
// The fallback is the point. Deep in a grid the cells are a few dozen pixels
// across and the font size has bottomed out at the configured floor, so a word
// would either overflow into its neighbours or have to be drawn too small to
// read. A letter still fits there, and by that depth the speaker has the word
// in mind anyway.
func CellLabel(key rune, cell image.Rectangle, fontSize int) string {
	letter := strings.ToUpper(string(key))

	word, ok := natoWords[unicode.ToLower(key)]
	if !ok {
		return letter
	}

	size := float64(fontSize)
	if size*averageGlyphWidth*float64(len(word)) > float64(cell.Dx())*labelFillFraction {
		return letter
	}

	if size*labelLineHeight > float64(cell.Dy())*labelFillFraction {
		return letter
	}

	return strings.ToUpper(word)
}
