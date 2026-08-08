package modes

import (
	"context"
	"image"

	"go.uber.org/zap"
)

// HintElement is one on-screen element paired with the label hints mode would
// give it, in a shape that survives JSON.
//
// This is the machine-readable counterpart to DebugProbeHints, which formats a
// short human sample. The difference matters to callers that are not people:
// an agent driving Neru needs the label to act on (via `action select_hint`),
// the geometry to reason about layout, and the text to decide what an element
// is — none of which can be recovered from a prose summary.
type HintElement struct {
	// Label is the hint word, and the thing to pass to `action select_hint`.
	Label string `json:"label"`
	Role  string `json:"role"`

	// Title and Description identify the element. Value is omitted unless the
	// caller explicitly asks for it — see ProbeHintElements.
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`

	Bounds Rect  `json:"bounds"`
	Center Point `json:"center"`

	Clickable bool `json:"clickable"`
	// VisionOnly marks elements found by the vision strategy rather than the
	// accessibility tree. They have no AX reference, so they can only be acted
	// on by coordinate.
	VisionOnly bool `json:"vision_only"`
}

// Rect is an axis-aligned rectangle in global top-left-origin pixels, matching
// the coordinate convention used everywhere outside the darwin adapter.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Point is a position in the same coordinate space as Rect.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// HintElementsResult is the whole answer: the elements plus the context needed
// to interpret them.
type HintElementsResult struct {
	// FocusedApp is the bundle identifier the elements were collected from.
	// Elements are gathered from whatever window is focused when this runs, so
	// a caller invoking it from a terminal gets that terminal's elements.
	FocusedApp string        `json:"focused_app"`
	Screen     Rect          `json:"screen"`
	Total      int           `json:"total"`
	OnScreen   int           `json:"on_screen"`
	Elements   []HintElement `json:"elements"`
}

// ProbeHintElements answers what hints mode would target, as data.
//
// It does not draw an overlay or enter a mode, so a caller can look at the
// screen without taking it over — which is what makes it usable as the "eyes"
// half of driving Neru programmatically, alongside the actions that already
// serve as its hands.
//
// includeValues controls whether element values are returned. They are withheld
// by default: a value is the *content* of a field rather than its identity, so
// it can hold whatever the user has typed, and the caller for this API is
// typically about to forward the result to a model. Titles and descriptions are
// enough to identify an element; values are opt-in for callers that genuinely
// need to read the screen's contents.
func (h *Handler) ProbeHintElements(
	ctx context.Context,
	filterRoles []string,
	filterTextContains []string,
	strategy string,
	splitWord bool,
	includeValues bool,
) (*HintElementsResult, error) {
	bundleID, bundleErr := h.actionService.FocusedAppBundleID(ctx)
	if bundleErr != nil {
		h.logger.Debug("hint elements: failed to get focused app id", zap.Error(bundleErr))
	}

	screenBounds, boundsErr := h.actionService.ScreenBounds(ctx)
	if boundsErr != nil {
		return nil, boundsErr
	}

	generated, genErr := h.hintService.GenerateHints(
		ctx,
		filterRoles,
		filterTextContains,
		bundleID,
		strategy,
		splitWord,
	)
	if genErr != nil {
		return nil, genErr
	}

	onScreen := filterHintsForScreen(generated, screenBounds)

	result := &HintElementsResult{
		FocusedApp: bundleID,
		Screen:     rectFrom(screenBounds),
		Total:      len(generated),
		OnScreen:   len(onScreen),
		Elements:   make([]HintElement, 0, len(onScreen)),
	}

	for _, hintItem := range onScreen {
		elem := hintItem.Element()
		if elem == nil {
			continue
		}

		entry := HintElement{
			Label:       hintItem.Label(),
			Role:        string(elem.Role()),
			Title:       elem.Title(),
			Description: elem.Description(),
			Bounds:      rectFrom(elem.Bounds()),
			Center:      pointFrom(elem.Center()),
			Clickable:   elem.IsClickable(),
			VisionOnly:  elem.IsVisionOnly(),
		}

		if includeValues {
			entry.Value = elem.Value()
		}

		result.Elements = append(result.Elements, entry)
	}

	// Counts and durations only — never the element text itself, per the
	// logging rules in AGENTS.md. The payload leaves over the socket; it does
	// not go to the log.
	h.logger.Debug("hint elements probed",
		zap.String("bundle_id", bundleID),
		zap.Int("total", result.Total),
		zap.Int("on_screen", result.OnScreen),
		zap.Bool("include_values", includeValues),
	)

	return result, nil
}

func rectFrom(r image.Rectangle) Rect {
	return Rect{X: r.Min.X, Y: r.Min.Y, Width: r.Dx(), Height: r.Dy()}
}

func pointFrom(p image.Point) Point {
	return Point{X: p.X, Y: p.Y}
}
