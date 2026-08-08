package modes

import (
	"context"
	"image"
	"testing"

	"go.uber.org/zap"

	hintscomponent "github.com/y3owk1n/neru/internal/adapter/overlay/render/hints"
	"github.com/y3owk1n/neru/internal/app/components"
	"github.com/y3owk1n/neru/internal/app/services"
	"github.com/y3owk1n/neru/internal/derrors"
	"github.com/y3owk1n/neru/internal/domain"
	"github.com/y3owk1n/neru/internal/domain/element"
	domainhint "github.com/y3owk1n/neru/internal/domain/hint"
	"github.com/y3owk1n/neru/internal/domain/state"
	portmocks "github.com/y3owk1n/neru/internal/ports/mocks"
)

// cursorMoveSpy records the point MoveCursorToPoint was last called with, so
// tests can assert on selection without a dedicated cursor-state accessor.
type cursorMoveSpy struct {
	called bool
	point  image.Point
}

func (s *cursorMoveSpy) record(point image.Point) {
	s.called = true
	s.point = point
}

// selectHintFixture bundles the handler under test with the elements it
// knows about and the cursor-move spy.
type selectHintFixture struct {
	handler *Handler
	spy     *cursorMoveSpy
	sierra  *element.Element
}

// newSelectHintTestHandler builds a Handler in hints mode with two hints
// ("SIERRA", "DELTA") registered on the full collection, mirroring what a
// real hint activation would populate.
func newSelectHintTestHandler(t *testing.T) *selectHintFixture {
	t.Helper()

	appState := state.NewAppState()
	appState.SetMode(domain.ModeHints)

	sierraElem, err := element.NewElement(
		"sierra-elem",
		image.Rect(0, 0, 20, 20),
		element.RoleButton,
		element.WithTitle("Sierra Target"),
	)
	if err != nil {
		t.Fatalf("NewElement: %v", err)
	}

	deltaElem, err := element.NewElement(
		"delta-elem",
		image.Rect(50, 50, 70, 70),
		element.RoleButton,
		element.WithTitle("Delta Target"),
	)
	if err != nil {
		t.Fatalf("NewElement: %v", err)
	}

	spy := &cursorMoveSpy{}

	handler := newHandlerWithState(handlerState{
		ctx:      context.Background(),
		appState: appState,
		logger:   zap.NewNop(),
		actionService: services.NewActionService(
			&portmocks.MockAccessibilityPort{},
			&portmocks.MockOverlayPort{},
			&portmocks.MockSystemPort{
				MoveCursorToPointFunc: func(_ context.Context, point image.Point, _ bool) error {
					spy.record(point)

					return nil
				},
			},
			zap.NewNop(),
		),
		hints: &components.HintsComponent{
			Context: &hintscomponent.Context{},
		},
	})

	collection := domainhint.NewCollection([]*domainhint.Interface{
		mustNewModeHint("SIERRA", sierraElem),
		mustNewModeHint("DELTA", deltaElem),
	})

	handler.mu.Lock()
	manager := domainhint.NewManager(handler.logger, &handler.mu)
	handler.hints.Context.SetManager(manager)

	if setErr := handler.hints.Context.SetHints(collection); setErr != nil {
		handler.mu.Unlock()
		t.Fatalf("SetHints: %v", setErr)
	}
	handler.mu.Unlock()

	return &selectHintFixture{handler: handler, spy: spy, sierra: sierraElem}
}

// The happy path (a resolvable word actually selecting and moving the
// cursor) is covered at the simulation level —
// TestSimulation_HintsDictationJourney in internal/app — because a
// successful selection with no pending action re-activates hints mode via
// activateHintModeInternal, which needs a real accessibility/overlay stack
// to exercise safely. The tests here pin the guard clauses, which return
// before that path is reached.

func TestHandler_SelectHintByLabel_UnresolvableWordRefusesAndLeavesModeUntouched(t *testing.T) {
	t.Parallel()

	fixture := newSelectHintTestHandler(t)

	err := fixture.handler.SelectHintByLabel(context.Background(), "qwqwqwqwq")
	if err == nil {
		t.Fatal("expected an error for an unresolvable word")
	}

	if !derrors.IsCode(err, derrors.CodeInvalidInput) {
		t.Errorf("expected CodeInvalidInput, got %v", err)
	}

	if fixture.handler.appState.CurrentMode() != domain.ModeHints {
		t.Errorf("mode changed to %v, want hints mode untouched", fixture.handler.appState.CurrentMode())
	}

	if fixture.spy.called {
		t.Error("expected no cursor move for a refused selection")
	}
}

func TestHandler_SelectHintByLabel_AmbiguousTranscriptRefused(t *testing.T) {
	t.Parallel()

	fixture := newSelectHintTestHandler(t)

	err := fixture.handler.SelectHintByLabel(context.Background(), "sierra delta")
	if err == nil {
		t.Fatal("expected an error for a transcript containing two valid words")
	}

	if !derrors.IsCode(err, derrors.CodeInvalidInput) {
		t.Errorf("expected CodeInvalidInput, got %v", err)
	}

	if fixture.spy.called {
		t.Error("expected no cursor move for an ambiguous transcript")
	}
}

func TestHandler_SelectHintByLabel_RequiresHintsMode(t *testing.T) {
	t.Parallel()

	appState := state.NewAppState()
	appState.SetMode(domain.ModeGrid)

	handler := newHandlerWithState(handlerState{
		appState: appState,
		logger:   zap.NewNop(),
	})

	err := handler.SelectHintByLabel(context.Background(), "sierra")
	if err == nil {
		t.Fatal("expected an error when not in hints mode")
	}

	if !derrors.IsCode(err, derrors.CodeInvalidInput) {
		t.Errorf("expected CodeInvalidInput, got %v", err)
	}
}
