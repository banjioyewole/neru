package services_test

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"slices"
	"testing"

	"github.com/y3owk1n/neru/internal/adapter/logger"
	"github.com/y3owk1n/neru/internal/app/services"
	"github.com/y3owk1n/neru/internal/config"
	"github.com/y3owk1n/neru/internal/derrors"
	"github.com/y3owk1n/neru/internal/domain"
	"github.com/y3owk1n/neru/internal/domain/element"
	"github.com/y3owk1n/neru/internal/domain/hint"
	"github.com/y3owk1n/neru/internal/domain/hint/vocab"
	"github.com/y3owk1n/neru/internal/ports"
	"github.com/y3owk1n/neru/internal/ports/mocks"
)

const (
	successfulHide   = "successful hide"
	overlayHideError = "overlay hide error"
)

func TestHintService_ShowHints(t *testing.T) {
	// Create test elements
	testElements := []*element.Element{
		mustNewElement("elem1", image.Rect(10, 10, 50, 50)),
		mustNewElement("elem2", image.Rect(60, 10, 100, 50)),
		mustNewElement("elem3", image.Rect(10, 60, 50, 100)),
	}

	tests := []struct {
		name          string
		setupMocks    func(*mocks.MockAccessibilityPort, *mocks.MockOverlayPort)
		setupGen      func() hint.Generator
		config        config.HintsConfig
		wantErr       bool
		wantHintCount int
		checkHints    func(*testing.T, []*hint.Interface)
		checkOverlay  func(*testing.T, *mocks.MockOverlayPort)
	}{
		{
			name: "successful hint display",
			setupMocks: func(acc *mocks.MockAccessibilityPort, _ *mocks.MockOverlayPort) {
				acc.ClickableElementsFunc = func(_ context.Context, _ ports.ElementFilter) ([]*element.Element, error) {
					return testElements, nil
				}
			},
			setupGen: func() hint.Generator {
				gen, _ := hint.NewWordGenerator()

				return gen
			},
			wantErr:       false,
			wantHintCount: 3, // We have 3 test elements
			checkHints: func(t *testing.T, hints []*hint.Interface) {
				t.Helper()

				if len(hints) != 3 {
					t.Errorf("Expected 3 hints, got %d", len(hints))

					return
				}
				// Check that hints have labels (exact labels depend on generator)
				if hints[0].Label() == "" || hints[1].Label() == "" || hints[2].Label() == "" {
					t.Error("Hints should have non-empty labels")
				}
			},
			checkOverlay: func(t *testing.T, ov *mocks.MockOverlayPort) {
				t.Helper()

				if !ov.IsVisible() {
					t.Error("Overlay should be visible after ShowHints")
				}
			},
		},
		{
			name: "no elements found",
			setupMocks: func(acc *mocks.MockAccessibilityPort, _ *mocks.MockOverlayPort) {
				acc.ClickableElementsFunc = func(_ context.Context, _ ports.ElementFilter) ([]*element.Element, error) {
					return []*element.Element{}, nil
				}
			},
			setupGen: func() hint.Generator {
				gen, _ := hint.NewWordGenerator()

				return gen
			},
			wantErr:       false,
			wantHintCount: 0, // No elements means no hints
			checkHints: func(t *testing.T, hints []*hint.Interface) {
				t.Helper()

				if len(hints) != 0 {
					t.Errorf("Expected 0 hints, got %d", len(hints))
				}
			},
		},
		{
			name: "accessibility error",
			setupMocks: func(acc *mocks.MockAccessibilityPort, _ *mocks.MockOverlayPort) {
				acc.ClickableElementsFunc = func(_ context.Context, _ ports.ElementFilter) ([]*element.Element, error) {
					return nil, derrors.New(
						derrors.CodeAccessibilityFailed,
						"accessibility permission denied",
					)
				}
			},
			setupGen: func() hint.Generator {
				gen, _ := hint.NewWordGenerator()

				return gen
			},
			wantErr:       true,
			wantHintCount: 0,
		},
		{
			name: "large element set",
			setupMocks: func(acc *mocks.MockAccessibilityPort, _ *mocks.MockOverlayPort) {
				acc.ClickableElementsFunc = func(_ context.Context, _ ports.ElementFilter) ([]*element.Element, error) {
					// Create 100 elements
					elements := make([]*element.Element, 100)

					for index := range 100 {
						elem, _ := element.NewElement(
							element.ID(fmt.Sprintf("elem%d", index)),
							image.Rect(index*10, index*10, index*10+40, index*10+40),
							nativeButtonRole,
						)
						elements[index] = elem
					}

					return elements, nil
				}
			},
			setupGen: func() hint.Generator {
				// Use larger character set for more hints
				gen, _ := hint.NewWordGenerator()

				return gen
			},
			wantErr:       false,
			wantHintCount: 100,
			checkHints: func(t *testing.T, hints []*hint.Interface) {
				t.Helper()

				if len(hints) != 100 {
					t.Errorf("Expected 100 hints, got %d", len(hints))
				}
			},
		},
		{
			name: "single element",
			setupMocks: func(acc *mocks.MockAccessibilityPort, _ *mocks.MockOverlayPort) {
				acc.ClickableElementsFunc = func(_ context.Context, _ ports.ElementFilter) ([]*element.Element, error) {
					return []*element.Element{testElements[0]}, nil
				}
			},
			setupGen: func() hint.Generator {
				gen, _ := hint.NewWordGenerator()

				return gen
			},
			wantErr:       false,
			wantHintCount: 1,
			checkHints: func(t *testing.T, hints []*hint.Interface) {
				t.Helper()

				if len(hints) != 1 {
					t.Errorf("Expected 1 hint, got %d", len(hints))
				}
			},
		},
		{
			name: "config-driven filtering",
			setupMocks: func(acc *mocks.MockAccessibilityPort, _ *mocks.MockOverlayPort) {
				acc.ClickableElementsFunc = func(_ context.Context, filter ports.ElementFilter) ([]*element.Element, error) {
					// Verify that config values are properly applied to filter
					if !filter.IncludeMenubar {
						t.Error("IncludeMenubar should be true based on config")
					}

					if !filter.IncludeDock {
						t.Error("IncludeDock should be true based on config")
					}

					if !filter.IncludeNotificationCenter {
						t.Error("IncludeNotificationCenter should be true based on config")
					}

					if !filter.IncludeStageManager {
						t.Error("IncludeStageManager should be true based on config")
					}

					if !filter.IncludePIP {
						t.Error("IncludePIP should be true based on config")
					}

					if !filter.IncludeScreenCapture {
						t.Error("IncludeScreenCapture should be true based on config")
					}

					return testElements, nil
				}
			},
			setupGen: func() hint.Generator {
				gen, _ := hint.NewWordGenerator()

				return gen
			},
			config: config.HintsConfig{
				IncludeMenubarHints:       true,
				IncludeDockHints:          true,
				IncludeNCHints:            true,
				IncludeStageManagerHints:  true,
				IncludePIPHints:           true,
				IncludeScreenCaptureHints: true,
			},
			wantErr:       false,
			wantHintCount: 3,
		},
		{
			name: "app-specific clickable roles are applied",
			setupMocks: func(acc *mocks.MockAccessibilityPort, _ *mocks.MockOverlayPort) {
				acc.FocusedAppBundleIDFunc = func(context.Context) (string, error) {
					return "net.imput.helium", nil
				}
				acc.ClickableElementsFunc = func(_ context.Context, filter ports.ElementFilter) ([]*element.Element, error) {
					roles := make(map[element.Role]bool, len(filter.Roles))
					for _, role := range filter.Roles {
						roles[role] = true
					}

					// The filter carries native role names for the running
					// platform, so expectations are resolved the same way.
					wantGlobal := element.ResolveRolesForCurrentPlatform([]string{"button"}).Native
					wantApp := element.ResolveRolesForCurrentPlatform([]string{"heading"}).Native

					for _, role := range wantGlobal {
						if !roles[element.Role(role)] {
							t.Errorf("expected global role %q to be present", role)
						}
					}

					for _, role := range wantApp {
						if !roles[element.Role(role)] {
							t.Errorf("expected app-specific role %q to be present", role)
						}
					}

					if len(roles) != len(wantGlobal)+len(wantApp) {
						t.Errorf(
							"expected exactly %d roles, got %v",
							len(wantGlobal)+len(wantApp), filter.Roles,
						)
					}

					return testElements, nil
				}
			},
			setupGen: func() hint.Generator {
				gen, _ := hint.NewWordGenerator()

				return gen
			},
			config: config.HintsConfig{
				ClickableRoles: []string{"button"},
				AppConfigs: []config.AppConfig{
					{
						BundleID:            "net.imput.helium",
						AdditionalClickable: []string{"heading"},
					},
				},
			},
			wantErr:       false,
			wantHintCount: 3,
		},
		{
			name: "too many elements shows max hints without error",
			setupMocks: func(acc *mocks.MockAccessibilityPort, _ *mocks.MockOverlayPort) {
				acc.ClickableElementsFunc = func(_ context.Context, _ ports.ElementFilter) ([]*element.Element, error) {
					// One more element than the vocabulary can label, to force
					// truncation without a generator error.
					elements := make([]*element.Element, vocab.Len()+1)

					for index := range elements {
						elements[index] = mustNewElement(
							fmt.Sprintf("elem%d", index),
							image.Rect(index*10, index*10, index*10+40, index*10+40),
						)
					}

					return elements, nil
				}
			},
			setupGen: func() hint.Generator {
				gen, _ := hint.NewWordGenerator()

				return gen
			},
			wantErr:       false,
			wantHintCount: vocab.Len(),
			checkHints: func(t *testing.T, hints []*hint.Interface) {
				t.Helper()

				if len(hints) != vocab.Len() {
					t.Errorf("Expected %d hints, got %d", vocab.Len(), len(hints))
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mockAcc := &mocks.MockAccessibilityPort{}
			mockOverlay := &mocks.MockOverlayPort{}

			if testCase.setupMocks != nil {
				testCase.setupMocks(mockAcc, mockOverlay)
			}

			generator := testCase.setupGen()
			logger := logger.Get()

			service := services.NewHintService(
				mockAcc,
				mockOverlay,
				&mocks.MockSystemPort{},
				generator,
				testCase.config,
				logger,
				nil,
			)

			ctx := context.Background()

			// Act
			hints, hintsErr := service.ShowHints(ctx, nil, nil)

			// Assert
			if testCase.wantErr && hintsErr == nil {
				t.Error("ShowHints() expected error, got nil")
			}

			if !testCase.wantErr && hintsErr != nil {
				t.Errorf("ShowHints() unexpected error: %v", hintsErr)
			}

			if testCase.checkHints != nil {
				testCase.checkHints(t, hints)
			}

			if testCase.checkOverlay != nil {
				testCase.checkOverlay(t, mockOverlay)
			}
		})
	}
}

func TestHintService_HideHints(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*mocks.MockOverlayPort)
		wantErr    bool
	}{
		{
			name: successfulHide,
			setupMocks: func(ov *mocks.MockOverlayPort) {
				ov.HideFunc = func(_ context.Context) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: overlayHideError,
			setupMocks: func(ov *mocks.MockOverlayPort) {
				ov.HideFunc = func(_ context.Context) error {
					return derrors.New(
						derrors.CodeOverlayFailed,
						"failed to hide overlay",
					)
				}
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mockAcc := &mocks.MockAccessibilityPort{}
			mockOverlay := &mocks.MockOverlayPort{}
			generator, _ := hint.NewWordGenerator()
			logger := logger.Get()

			if testCase.setupMocks != nil {
				testCase.setupMocks(mockOverlay)
			}

			service := services.NewHintService(
				mockAcc,
				mockOverlay,
				&mocks.MockSystemPort{},
				generator,
				config.HintsConfig{},
				logger,
				nil,
			)

			ctx := context.Background()
			hideHintsErr := service.HideHints(ctx)

			if (hideHintsErr != nil) != testCase.wantErr {
				t.Errorf("HideHints() error = %v, wantErr %v", hideHintsErr, testCase.wantErr)
			}

			// Only check visibility for successful hide
			if !testCase.wantErr && mockOverlay.IsVisible() {
				t.Error("Overlay should not be visible after successful HideHints")
			}
		})
	}
}

func TestHintService_RefreshHints(t *testing.T) {
	tests := []struct {
		name           string
		overlayVisible bool
		expectRefresh  bool
		refreshError   error
		wantErr        bool
	}{
		{
			name:           "refresh when visible",
			overlayVisible: true,
			expectRefresh:  true,
			refreshError:   nil,
			wantErr:        false,
		},
		{
			name:           "skip refresh when not visible",
			overlayVisible: false,
			expectRefresh:  false,
			refreshError:   nil,
			wantErr:        false,
		},
		{
			name:           "refresh error when visible",
			overlayVisible: true,
			expectRefresh:  true,
			refreshError:   derrors.New(derrors.CodeOverlayFailed, "overlay refresh failed"),
			wantErr:        true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mockAcc := &mocks.MockAccessibilityPort{}
			mockOverlay := &mocks.MockOverlayPort{}

			refreshCalled := false
			mockOverlay.IsVisibleFunc = func() bool {
				return testCase.overlayVisible
			}
			mockOverlay.RefreshFunc = func(_ context.Context) error {
				refreshCalled = true

				return testCase.refreshError
			}

			generator, _ := hint.NewWordGenerator()
			logger := logger.Get()

			service := services.NewHintService(
				mockAcc,
				mockOverlay,
				&mocks.MockSystemPort{},
				generator,
				config.HintsConfig{},
				logger,
				nil,
			)

			ctx := context.Background()
			refreshHintsErr := service.RefreshHints(ctx)

			if (refreshHintsErr != nil) != testCase.wantErr {
				t.Errorf("RefreshHints() error = %v, wantErr %v", refreshHintsErr, testCase.wantErr)
			}

			if refreshCalled != testCase.expectRefresh {
				t.Errorf("Refresh called = %v, want %v", refreshCalled, testCase.expectRefresh)
			}
		})
	}
}

func TestHintService_GenerateHintsVisionCombinesSupplementaryAndWindowElements(
	t *testing.T,
) {
	supplementElement := mustNewElement("menubar", image.Rect(10, 0, 60, 20))
	windowElement := mustNewElement("window", image.Rect(10, 40, 60, 90))

	mockAcc := &mocks.MockAccessibilityPort{}
	mockAcc.ClickableElementsFunc = func(
		_ context.Context,
		filter ports.ElementFilter,
	) ([]*element.Element, error) {
		if !filter.SkipWindowElements {
			t.Error("accessibility should not collect window elements when using vision strategy")

			return nil, nil
		}

		return []*element.Element{supplementElement}, nil
	}

	mockSystem := &mocks.MockSystemPort{}
	mockSystem.FocusedWindowBoundsFunc = func(context.Context) (image.Rectangle, bool, error) {
		return image.Rect(0, 0, 200, 200), true, nil
	}

	generator, _ := hint.NewWordGenerator()
	service := services.NewHintService(
		mockAcc,
		&mocks.MockOverlayPort{},
		mockSystem,
		generator,
		config.HintsConfig{
			ClickableRoles:                []string{string(element.SemanticButton)},
			IncludeMenubarHints:           true,
			AdditionalMenubarHintsTargets: []string{"Clock"},
			IncludeDockHints:              true,
			IncludeNCHints:                true,
			IncludeStageManagerHints:      true,
			IncludePIPHints:               true,
			IncludeScreenCaptureHints:     true,
		},
		logger.Get(),
		&mockVisionPort{
			detectedElements: []*element.Element{windowElement},
		},
	)

	hints, err := service.GenerateHints(
		context.Background(),
		nil,
		nil,
		"com.example.app",
		domain.StrategyVision,
		false,
	)
	if err != nil {
		t.Fatalf("GenerateHints() unexpected error: %v", err)
	}

	if len(hints) != 2 {
		t.Fatalf("GenerateHints() returned %d hints, want 2", len(hints))
	}

	seen := map[element.ID]int{}
	for _, generatedHint := range hints {
		seen[generatedHint.Element().ID()]++
	}

	if seen[supplementElement.ID()] != 1 {
		t.Errorf("supplementary element count = %d, want 1", seen[supplementElement.ID()])
	}

	if seen[windowElement.ID()] != 1 {
		t.Errorf("window element count = %d, want 1", seen[windowElement.ID()])
	}
}

func TestHintService_GenerateHintsVisionWithNilPortReturnsSupplementaryElements(
	t *testing.T,
) {
	supplementElement := mustNewElement("menubar", image.Rect(10, 0, 60, 20))

	mockAcc := &mocks.MockAccessibilityPort{}
	mockAcc.ClickableElementsFunc = func(
		_ context.Context,
		filter ports.ElementFilter,
	) ([]*element.Element, error) {
		if !filter.SkipWindowElements {
			t.Error("nil vision port should not trigger window AX collection")
		}

		return []*element.Element{supplementElement}, nil
	}

	generator, _ := hint.NewWordGenerator()
	service := services.NewHintService(
		mockAcc,
		&mocks.MockOverlayPort{},
		&mocks.MockSystemPort{},
		generator,
		config.HintsConfig{
			IncludeMenubarHints: true,
		},
		logger.Get(),
		nil,
	)

	hints, err := service.GenerateHints(
		context.Background(),
		nil,
		nil,
		"com.example.app",
		domain.StrategyVision,
		false,
	)
	if err != nil {
		t.Fatalf("GenerateHints() unexpected error: %v", err)
	}

	if len(hints) != 1 {
		t.Fatalf("GenerateHints() returned %d hints, want 1", len(hints))
	}

	if hints[0].Element().ID() != supplementElement.ID() {
		t.Errorf("hint element = %q, want %q", hints[0].Element().ID(), supplementElement.ID())
	}
}

func TestHintService_UpdateGenerator(t *testing.T) {
	mockAcc := &mocks.MockAccessibilityPort{}
	mockOverlay := &mocks.MockOverlayPort{}
	log := logger.Get()

	initialGen, err := hint.NewWordGenerator()
	if err != nil {
		t.Fatalf("NewWordGenerator() error = %v", err)
	}

	service := services.NewHintService(
		mockAcc,
		mockOverlay,
		&mocks.MockSystemPort{},
		initialGen,
		config.HintsConfig{},
		log,
		nil,
	)

	replacement, err := hint.NewWordGenerator()
	if err != nil {
		t.Fatalf("NewWordGenerator() error = %v", err)
	}

	service.UpdateGenerator(context.Background(), replacement)

	got := service.Generator()
	if got == nil {
		t.Fatal("Generator() = nil after UpdateGenerator")
	}

	if got != hint.Generator(replacement) {
		t.Error("Generator() did not return the generator passed to UpdateGenerator")
	}

	// A nil generator must be ignored rather than wiping a live one.
	service.UpdateGenerator(context.Background(), nil)

	if service.Generator() == nil {
		t.Error("UpdateGenerator(nil) cleared the previously registered generator")
	}
}

func TestHintService_Health(t *testing.T) {
	mockAcc := &mocks.MockAccessibilityPort{}
	mockOverlay := &mocks.MockOverlayPort{}
	generator, _ := hint.NewWordGenerator()
	logger := logger.Get()

	service := services.NewHintService(
		mockAcc,
		mockOverlay,
		&mocks.MockSystemPort{},
		generator,
		config.HintsConfig{},
		logger,
		nil,
	)

	// Setup mocks
	mockAcc.HealthFunc = func(_ context.Context) error {
		return nil
	}
	mockOverlay.HealthFunc = func(_ context.Context) error {
		return derrors.New(derrors.CodeOverlayFailed, "overlay unhealthy")
	}

	ctx := context.Background()
	health := service.Health(ctx)

	// Check that health map has both keys
	if len(health) != 2 {
		t.Errorf("Health() returned %d entries, want 2", len(health))
	}

	if _, ok := health["accessibility"]; !ok {
		t.Error("Health() missing 'accessibility' key")
	}

	if _, ok := health["overlay"]; !ok {
		t.Error("Health() missing 'overlay' key")
	}

	// Check that overlay has error
	if health["overlay"] == nil {
		t.Error("Health() overlay should have error")
	}

	if health["accessibility"] != nil {
		t.Error("Health() accessibility should not have error")
	}
}

// nativeButtonRole is the accessibility role a button reports on the platform
// running the tests. Configured roles resolve to native names, so an element
// built with a role from another platform would silently stop matching a
// config that asks for "button".
var nativeButtonRole = func() element.Role {
	native := element.ResolveRolesForCurrentPlatform(
		[]string{string(element.SemanticButton)},
	).Native
	if len(native) == 0 {
		return element.RoleButton
	}

	return element.Role(native[0])
}()

func mustNewElement(id string, bounds image.Rectangle) *element.Element {
	element, elementErr := element.NewElement(element.ID(id), bounds, nativeButtonRole)
	if elementErr != nil {
		panic(elementErr)
	}

	return element
}

type mockVisionPort struct {
	detectedElements []*element.Element
	detectErr        error
}

func (m *mockVisionPort) DetectElements(
	context.Context,
	image.Rectangle,
	config.HintsVisionConfig,
	bool,
) ([]*element.Element, error) {
	if m.detectErr != nil {
		return nil, m.detectErr
	}

	return m.detectedElements, nil
}

func (m *mockVisionPort) CaptureScreen(context.Context) (*image.RGBA, error) {
	return nil, derrors.New(derrors.CodeBridgeFailed, "capture screen not implemented")
}

func (m *mockVisionPort) Health(context.Context) error {
	return nil
}

func TestHintService_GenerateHintsRejectsSplitWordForNonVisionStrategy(t *testing.T) {
	generator, _ := hint.NewWordGenerator()
	service := services.NewHintService(
		&mocks.MockAccessibilityPort{},
		&mocks.MockOverlayPort{},
		&mocks.MockSystemPort{},
		generator,
		config.HintsConfig{
			Strategy: domain.StrategyAXTree,
		},
		logger.Get(),
		nil,
	)

	ctx := context.Background()

	_, err := service.GenerateHints(
		ctx,
		nil,
		nil,
		"",
		domain.StrategyAXTree,
		true, // splitWord
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !derrors.IsCode(err, derrors.CodeInvalidInput) {
		t.Errorf("expected invalid input error, got: %v", err)
	}
}

// TestHintService_GenerateHintsRoleFilterResolvingToNothing pins the behavior
// of a role filter that is configured but resolves to no native role on this
// platform — for example a config carrying only Linux entries, run on macOS.
//
// An empty ports.ElementFilter.Roles means "match every role", so the naive
// outcome is that an unusable filter hints *everything*. It must hint nothing.
func TestHintService_GenerateHintsRoleFilterResolvingToNothing(t *testing.T) {
	testElements := []*element.Element{
		mustNewElement("elem1", image.Rect(10, 10, 50, 50)),
		mustNewElement("elem2", image.Rect(60, 10, 100, 50)),
	}

	tests := []struct {
		name        string
		roles       []string
		filterRoles []string
	}{
		{
			name:  "configured roles all belong to another platform",
			roles: foreignRolesForCurrentPlatform(),
		},
		{
			name:        "role flag entries are unresolvable",
			roles:       []string{string(element.SemanticButton)},
			filterRoles: []string{"AXButton"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if len(testCase.roles) == 0 {
				t.Skipf("no foreign role set defined for %s", runtime.GOOS)
			}

			mockAcc := &mocks.MockAccessibilityPort{}
			mockAcc.ClickableElementsFunc = func(
				_ context.Context,
				_ ports.ElementFilter,
			) ([]*element.Element, error) {
				return testElements, nil
			}

			generator, _ := hint.NewWordGenerator()
			service := services.NewHintService(
				mockAcc,
				&mocks.MockOverlayPort{},
				&mocks.MockSystemPort{},
				generator,
				config.HintsConfig{ClickableRoles: testCase.roles},
				logger.Get(),
				nil,
			)

			hints, err := service.GenerateHints(
				context.Background(),
				testCase.filterRoles,
				nil,
				"com.example.app",
				"",
				false,
			)
			if err != nil {
				t.Fatalf("GenerateHints() unexpected error: %v", err)
			}

			if len(hints) != 0 {
				t.Errorf(
					"GenerateHints() returned %d hints for an unusable role filter, want 0",
					len(hints),
				)
			}
		})
	}
}

// TestHintService_GenerateHintsRoleFlagOverridesConfig covers the happy path of
// `neru hints --role ...`: the flag replaces the configured roles entirely and
// is resolved through the same vocabulary, so it accepts semantic names and
// vocabulary-prefixed native names alike.
func TestHintService_GenerateHintsRoleFlagOverridesConfig(t *testing.T) {
	testElements := []*element.Element{
		mustNewElement("elem1", image.Rect(10, 10, 50, 50)),
	}

	var captured []element.Role

	mockAcc := &mocks.MockAccessibilityPort{}
	mockAcc.ClickableElementsFunc = func(
		_ context.Context,
		filter ports.ElementFilter,
	) ([]*element.Element, error) {
		captured = filter.Roles

		return testElements, nil
	}

	generator, _ := hint.NewWordGenerator()
	service := services.NewHintService(
		mockAcc,
		&mocks.MockOverlayPort{},
		&mocks.MockSystemPort{},
		generator,
		config.HintsConfig{
			ClickableRoles: []string{string(element.SemanticButton)},
		},
		logger.Get(),
		nil,
	)

	_, err := service.GenerateHints(
		context.Background(),
		[]string{string(element.SemanticLink)},
		nil,
		"com.example.app",
		"",
		false,
	)
	if err != nil {
		t.Fatalf("GenerateHints() unexpected error: %v", err)
	}

	want := element.ResolveRolesForCurrentPlatform(
		[]string{string(element.SemanticLink)},
	).Native
	if len(want) == 0 {
		t.Skip("link has no native role on this platform")
	}

	for _, role := range want {
		if !slices.Contains(captured, element.Role(role)) {
			t.Errorf("filter.Roles = %v, missing overridden role %q", captured, role)
		}
	}

	// The configured role must not survive the override.
	for _, role := range element.ResolveRolesForCurrentPlatform(
		[]string{string(element.SemanticButton)},
	).Native {
		if slices.Contains(captured, element.Role(role)) {
			t.Errorf("filter.Roles = %v, configured role %q leaked past the override",
				captured, role)
		}
	}
}

// foreignRolesForCurrentPlatform returns native role entries that belong to
// platforms other than the one running the tests, so they resolve to nothing
// here. Returns nil on a platform with no accessibility backend.
func foreignRolesForCurrentPlatform() []string {
	return map[string][]string{
		"darwin":  {"atspi:push button", "uia:Button"},
		"linux":   {"ax:AXButton", "uia:Button"},
		"windows": {"ax:AXButton", "atspi:push button"},
	}[runtime.GOOS]
}
