package services

import (
	"context"
	"slices"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/y3owk1n/neru/internal/config"
	"github.com/y3owk1n/neru/internal/derrors"
	"github.com/y3owk1n/neru/internal/domain"
	"github.com/y3owk1n/neru/internal/domain/element"
	"github.com/y3owk1n/neru/internal/domain/hint"
	"github.com/y3owk1n/neru/internal/ports"
)

// HintService orchestrates hint generation and display.
// It coordinates between the accessibility system, vision detection,
// hint generator, and overlay.
type HintService struct {
	BaseService

	mu        sync.RWMutex
	generator hint.Generator
	config    config.HintsConfig
	logger    *zap.Logger
	vision    ports.VisionPort
}

// NewHintService creates a new hint service with the given dependencies.
func NewHintService(
	accessibility ports.AccessibilityPort,
	overlay ports.OverlayPort,
	system ports.SystemPort,
	generator hint.Generator,
	config config.HintsConfig,
	logger *zap.Logger,
	vision ports.VisionPort,
) *HintService {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &HintService{
		BaseService: NewBaseService(accessibility, overlay, system),
		generator:   generator,
		config:      config,
		logger:      logger.Named("service.hints"),
		vision:      vision,
	}
}

// ShowHints displays hints for clickable elements on the screen.
// If filterRoles or filterTextContains are provided, they override the configured values.
func (s *HintService) ShowHints(
	ctx context.Context,
	filterRoles []string,
	filterTextContains []string,
) ([]*hint.Interface, error) {
	s.logger.Debug("Showing hints")

	hints, err := s.GenerateHints(ctx, filterRoles, filterTextContains, "", "", false)
	if err != nil {
		return nil, err
	}

	if len(hints) == 0 {
		return hints, nil
	}

	showHintsErr := s.overlay.ShowHints(ctx, hints)
	if showHintsErr != nil {
		s.logger.Error("Failed to show hints overlay", zap.Error(showHintsErr))

		return nil, derrors.WrapOverlayFailed(showHintsErr, "show hints")
	}

	s.logger.Debug("Hints displayed successfully", zap.Int("count", len(hints)))

	return hints, nil
}

// GenerateHints collects clickable elements and generates labels without
// drawing them, so mode handlers can filter and position hints before the
// first render. A non-empty bundleID skips the AX lookup; non-empty overrides
// win over the config-derived strategy and label direction.
func (s *HintService) GenerateHints(
	ctx context.Context,
	filterRoles []string,
	filterTextContains []string,
	bundleID string,
	strategyOverride string,
	splitWord bool,
) ([]*hint.Interface, error) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if bundleID == "" {
		var bundleIDErr error

		bundleID, bundleIDErr = s.accessibility.FocusedAppBundleID(ctx)
		if bundleIDErr != nil {
			s.logger.Debug(
				"Failed to get focused app bundle ID for hints roles",
				zap.Error(bundleIDErr),
			)
		}
	}

	filter, usable := s.hintFilter(cfg, bundleID, filterRoles, filterTextContains)
	if !usable {
		return nil, nil
	}

	strategy := cfg.StrategyForApp(bundleID)
	if strategyOverride != "" {
		strategy = strategyOverride
	}

	if splitWord && strategy != domain.StrategyVision {
		return nil, derrors.New(
			derrors.CodeInvalidInput,
			"--split-word is only supported when resolved strategy is 'vision'",
		)
	}

	var (
		elements []*element.Element
		genErr   error
	)

	switch strategy {
	case domain.StrategyVision:
		elements = s.generateHintsVision(ctx, bundleID, filter, splitWord)
	default:
		elements, genErr = s.generateHintsAX(ctx, filter)
	}

	if genErr != nil {
		return nil, genErr
	}

	if len(elements) == 0 {
		s.logger.Debug("No clickable elements found")

		return nil, nil
	}

	s.logger.Debug("Found clickable elements", zap.Int("count", len(elements)))

	return s.labelElements(ctx, elements)
}

// HideHints removes the hint overlay from the screen.
func (s *HintService) HideHints(ctx context.Context) error {
	s.logger.Debug("Hiding hints")

	err := s.HideOverlay(ctx, "hide hints")
	if err != nil {
		s.logger.Error("Failed to hide overlay", zap.Error(err))

		return err
	}

	s.logger.Debug("Hints hidden successfully")

	return nil
}

// RefreshHints updates the hint display (e.g., after screen changes).
func (s *HintService) RefreshHints(ctx context.Context) error {
	s.logger.Debug("Refreshing hints")

	if !s.overlay.IsVisible() {
		s.logger.Debug("Overlay not visible, skipping refresh")

		return nil
	}

	refreshOverlayErr := s.overlay.Refresh(ctx)
	if refreshOverlayErr != nil {
		s.logger.Error("Failed to refresh overlay", zap.Error(refreshOverlayErr))

		return derrors.WrapOverlayFailed(refreshOverlayErr, "refresh hints")
	}

	s.logger.Debug("Hints refreshed successfully")

	return nil
}

// UpdateConfig updates the hints configuration.
// Hint filters can therefore change without a restart.
func (s *HintService) UpdateConfig(config config.HintsConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config

	s.logger.Debug("Hints configuration updated",
		zap.Bool("include_menubar", config.IncludeMenubarHints),
		zap.Bool("include_dock", config.IncludeDockHints),
		zap.Bool("include_nc", config.IncludeNCHints),
		zap.Bool("include_stage_manager", config.IncludeStageManagerHints),
		zap.Bool("include_pip", config.IncludePIPHints),
		zap.Bool("include_screen_capture", config.IncludeScreenCaptureHints))
}

// Generator returns the registered hint generator.
func (s *HintService) Generator() hint.Generator {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.generator
}

// UpdateGenerator replaces the registered hint generator. A nil generator is
// ignored to avoid replacing a live generator with nothing.
func (s *HintService) UpdateGenerator(_ context.Context, generator hint.Generator) {
	if generator == nil {
		s.logger.Warn("Attempted to set nil generator, ignoring")

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.generator = generator

	s.logger.Debug("Hint generator updated")
}

// generateHintsAX collects elements using the AX tree (default strategy).
func (s *HintService) generateHintsAX(
	ctx context.Context,
	filter ports.ElementFilter,
) ([]*element.Element, error) {
	axStart := time.Now()
	elements, err := s.accessibility.ClickableElements(ctx, filter)
	s.logger.Debug("TIMING: ClickableElements (axtree)",
		zap.Duration("elapsed", time.Since(axStart)),
		zap.Int("element_count", len(elements)),
		zap.Error(err))

	if err != nil {
		s.logger.Error("Failed to get clickable elements via AX", zap.Error(err))

		return nil, derrors.WrapAccessibilityFailed(err, "get clickable elements")
	}

	return elements, nil
}

// generateHintsVision collects window elements via vision detection and
// supplementary elements (menubar, dock, etc.) via AX. This hybrid approach
// ensures system UI is always detected while the frontmost window content
// uses vision-based detection for apps with poor AX trees.
func (s *HintService) generateHintsVision(
	ctx context.Context,
	_ string,
	filter ports.ElementFilter,
	splitWord bool,
) []*element.Element {
	// Collect supplementary elements (menubar, dock, NC, etc.) via AX.
	// These are system-level components that vision should not attempt to detect.
	var allElements []*element.Element

	supplementStart := time.Now()
	supplementFilter := filter
	supplementFilter.Roles = nil               // no role filtering for supplementary elements
	supplementFilter.SkipWindowElements = true // vision handles the window

	supplementElements, err := s.accessibility.ClickableElements(ctx, supplementFilter)
	if err != nil {
		s.logger.Debug("Failed to get supplementary elements via AX", zap.Error(err))
	} else {
		allElements = append(allElements, supplementElements...)
	}

	s.logger.Debug("TIMING: Supplementary elements (AX)",
		zap.Duration("elapsed", time.Since(supplementStart)),
		zap.Int("count", len(supplementElements)))

	if s.vision == nil {
		s.logger.Warn("Vision strategy selected but vision port is unavailable")

		return allElements
	}

	// Get focused window bounds for vision detection
	windowBounds, found, boundsErr := s.system.FocusedWindowBounds(ctx)
	if boundsErr != nil || !found {
		s.logger.Debug(
			"No focused window bounds, falling back to full screen",
			zap.Error(boundsErr),
		)

		windowBounds, boundsErr = s.system.ScreenBounds(ctx)
		if boundsErr != nil {
			s.logger.Error("Failed to get screen bounds for vision detection", zap.Error(boundsErr))

			return allElements
		}
	}

	// Detect window elements via vision
	visionStart := time.Now()
	windowElements, visionErr := s.vision.DetectElements(
		ctx,
		windowBounds,
		s.config.Vision,
		splitWord,
	)
	s.logger.Debug("TIMING: Window elements (vision)",
		zap.Duration("elapsed", time.Since(visionStart)),
		zap.Int("count", len(windowElements)),
		zap.Error(visionErr))

	if visionErr != nil {
		s.logger.Error("Failed to detect elements via vision", zap.Error(visionErr))

		return allElements
	}

	// Filter vision-detected elements by configured roles
	for _, element := range windowElements {
		if len(filter.Roles) == 0 {
			allElements = append(allElements, element)

			continue
		}

		if slices.Contains(filter.Roles, element.Role()) {
			allElements = append(allElements, element)
		}
	}

	return allElements
}

// hintFilter builds the element filter for one activation. The second result
// is false when every requested role belongs to another platform: hinting
// everything would hide that misconfiguration, so nothing is hinted instead
// (`neru roles --explain` and `neru doctor` report the cause).
func (s *HintService) hintFilter(
	cfg config.HintsConfig,
	bundleID string,
	filterRoles []string,
	filterTextContains []string,
) (ports.ElementFilter, bool) {
	filter := ports.DefaultElementFilter()

	// requested holds the entries as written, before resolution, so "no filter
	// at all" stays distinguishable from "a filter that resolved to nothing".
	var roles, requested []string

	if len(filterRoles) > 0 {
		// `neru hints --role ...` accepts the same vocabulary as the config
		// and is resolved the same way.
		requested = filterRoles

		resolution := element.ResolveRolesForCurrentPlatform(filterRoles)
		roles = resolution.Native

		s.logger.Debug("Using override roles from activation options",
			zap.Int("requested", len(requested)),
			zap.Int("role_count", len(roles)))

		for _, message := range resolution.FatalMessages() {
			s.logger.Warn("Ignoring role filter entry", zap.String("reason", message))
		}
	} else {
		requested = cfg.MergedForApp(bundleID).ClickableRoles
		roles = cfg.ClickableRolesForApp(bundleID)

		s.logger.Debug("Resolved clickable roles for hints",
			zap.String("bundle_id", bundleID),
			zap.Int("requested", len(requested)),
			zap.Int("role_count", len(roles)))
	}

	filter.Roles = make([]element.Role, 0, len(roles))
	for _, role := range roles {
		if role == "" {
			continue
		}

		filter.Roles = append(filter.Roles, element.Role(role))
	}

	if len(filter.Roles) == 0 && len(requested) > 0 {
		s.logger.Warn(
			"No configured role applies on this platform; showing no hints",
			zap.Int("requested", len(requested)),
		)

		return filter, false
	}

	filter.IncludeMenubar = cfg.IncludeMenubarHints
	filter.AdditionalMenubarTargets = cfg.AdditionalMenubarHintsTargets
	filter.IncludeDock = cfg.IncludeDockHints
	filter.IncludeNotificationCenter = cfg.IncludeNCHints
	filter.IncludeStageManager = cfg.IncludeStageManagerHints
	filter.IncludePIP = cfg.IncludePIPHints
	filter.IncludeScreenCapture = cfg.IncludeScreenCaptureHints

	// Text filter: an element matches when any term matches.
	if len(filterTextContains) > 0 {
		filter.TitleContains = filterTextContains[0]
		filter.DescriptionContains = filterTextContains[0]

		filter.ValueContains = filterTextContains[0]
		if len(filterTextContains) > 1 {
			filter.TextContainsList = filterTextContains[1:]
		}

		s.logger.Debug("Applying text filter",
			zap.Int("term_count", len(filterTextContains)))
	}

	return filter, true
}

// labelElements turns collected elements into labeled hints.
func (s *HintService) labelElements(
	ctx context.Context,
	elements []*element.Element,
) ([]*hint.Interface, error) {
	gen := s.Generator()

	maxHints := gen.MaxHints()
	if maxHints > 0 && len(elements) > maxHints {
		s.logger.Warn(
			"Clickable element count exceeds available hint key combinations; showing as many as possible",
			zap.Int("element_count", len(elements)),
			zap.Int("max_hints", maxHints),
			zap.Int("omitted_count", len(elements)-maxHints),
		)
	}

	genStart := time.Now()
	hints, elementsErr := gen.Generate(ctx, elements)
	s.logger.Debug("TIMING: HintGenerator.Generate",
		zap.Duration("elapsed", time.Since(genStart)),
		zap.Int("element_count", len(elements)),
		zap.Int("hint_count", len(hints)),
		zap.Error(elementsErr))

	if elementsErr != nil {
		s.logger.Error("Failed to generate hints", zap.Error(elementsErr))

		return nil, derrors.WrapInternalFailed(elementsErr, "generate hints")
	}

	s.logger.Debug("Generated hints", zap.Int("count", len(hints)))

	return hints, nil
}
