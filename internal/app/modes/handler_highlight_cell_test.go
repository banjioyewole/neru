package modes

import (
	"image"
	"testing"

	"go.uber.org/zap"

	"github.com/y3owk1n/neru/internal/app/components"
	componentrecursivegrid "github.com/y3owk1n/neru/internal/app/components/recursivegrid"
	"github.com/y3owk1n/neru/internal/config"
	"github.com/y3owk1n/neru/internal/derrors"
	"github.com/y3owk1n/neru/internal/domain"
	"github.com/y3owk1n/neru/internal/domain/state"
)

func newHighlightHandler(mode domain.Mode) *Handler {
	appState := state.NewAppState()
	appState.SetMode(mode)

	handler := newHandlerWithState(handlerState{
		appState: appState,
		config: &config.Config{
			RecursiveGrid: config.RecursiveGridConfig{
				Enabled:       true,
				GridCols:      3,
				GridRows:      3,
				Keys:          "rtyfghvbn",
				MinSizeWidth:  1,
				MinSizeHeight: 1,
				MaxDepth:      10,
				Hotkeys:       map[string]config.StringOrStringArray{},
			},
		},
		logger: zap.NewNop(),
		recursiveGrid: &components.RecursiveGridComponent{
			Context: &componentrecursivegrid.Context{},
		},
		screenBounds: image.Rect(0, 0, 900, 900),
	})
	handler.initializeRecursiveGridManager(image.Rect(0, 0, 900, 900))

	return handler
}

func TestHandler_HighlightRecursiveGridCell_ReachesTheFrameAndClears(t *testing.T) {
	t.Parallel()

	handler := newHighlightHandler(domain.ModeRecursiveGrid)

	err := handler.HighlightRecursiveGridCell("g")
	if err != nil {
		t.Fatalf("highlight: %v", err)
	}

	if got := handler.recursiveGridFrame().Highlighted; got != "g" {
		t.Fatalf("frame highlight = %q, want g", got)
	}

	// The highlight is a key, so it survives the grid narrowing.
	handler.handleRecursiveGridKey("r")

	if got := handler.recursiveGridFrame().Highlighted; got != "g" {
		t.Fatalf("frame highlight after narrowing = %q, want g", got)
	}

	err = handler.HighlightRecursiveGridCell("")
	if err != nil {
		t.Fatalf("clear: %v", err)
	}

	if got := handler.recursiveGridFrame().Highlighted; got != "" {
		t.Fatalf("frame highlight after clearing = %q, want none", got)
	}
}

func TestHandler_HighlightRecursiveGridCell_RefusesAKeyWithNoCell(t *testing.T) {
	t.Parallel()

	handler := newHighlightHandler(domain.ModeRecursiveGrid)

	for _, key := range []string{"z", "rt"} {
		err := handler.HighlightRecursiveGridCell(key)
		if !derrors.IsCode(err, derrors.CodeInvalidInput) {
			t.Errorf("key %q: expected CodeInvalidInput, got %v", key, err)
		}
	}
}

func TestHandler_HighlightRecursiveGridCell_RequiresRecursiveGridMode(t *testing.T) {
	t.Parallel()

	handler := newHighlightHandler(domain.ModeHints)

	err := handler.HighlightRecursiveGridCell("r")
	if !derrors.IsCode(err, derrors.CodeInvalidInput) {
		t.Errorf("expected CodeInvalidInput, got %v", err)
	}
}
