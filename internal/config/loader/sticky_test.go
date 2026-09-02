package loader_test

import (
	"testing"

	"go.uber.org/zap"

	"github.com/y3owk1n/neru/internal/config"
	"github.com/y3owk1n/neru/internal/config/loader"
)

// TestStickyModifiersDisabledForDaemon covers the flag's whole contract: a
// config asking for sticky modifiers loses them, a daemon without the flag
// keeps them, and — the case the hotkey suppression had to learn too — a
// config bad enough to be refused does not bring them back with the defaults.
func TestStickyModifiersDisabledForDaemon(t *testing.T) {
	const enabled = "[sticky_modifiers]\nenabled = true\n"

	// Refused at load: a duplicate normalized hotkey, which falls back to the
	// defaults, where sticky modifiers are on.
	const refused = enabled + "\n[hotkeys]\n\"Primary+Shift+G\" = \"hints\"\n" +
		"\"primary+shift+g\" = \"grid\"\n"

	tests := []struct {
		name     string
		body     string
		disabled bool
		want     bool
	}{
		{name: "flag clears an enabled config", body: enabled, disabled: true, want: false},
		{name: "no flag leaves it alone", body: enabled, disabled: false, want: true},
		{name: "flag survives a refused config", body: refused, disabled: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeSuppressionConfig(t, tt.body)

			service := loader.NewService(
				config.DefaultConfig(),
				path,
				zap.NewNop(),
				nil,
			).WithStickyModifiersDisabled(tt.disabled)

			result := service.LoadWithValidation(path)

			got := result.Config.StickyModifiers.Enabled
			if got != tt.want {
				t.Errorf("StickyModifiers.Enabled = %v, want %v", got, tt.want)
			}
		})
	}
}

// A runtime field change derives from the written half of the load, so the
// flag has to hold there as well or the first `neru config set` turns sticky
// modifiers back on.
func TestStickyModifiersDisabledSurvivesARuntimeFieldChange(t *testing.T) {
	path := writeSuppressionConfig(t, "[sticky_modifiers]\nenabled = true\n")

	service := loader.NewService(config.DefaultConfig(), path, zap.NewNop(), nil).
		WithStickyModifiersDisabled(true)

	result := service.LoadWithValidation(path)
	if result.ValidationError != nil {
		t.Fatalf("load: %v", result.ValidationError)
	}

	changed, _, err := loader.ApplyFieldChange(result.Written, "hints.max_depth", "30")
	if err != nil {
		t.Fatalf("ApplyFieldChange: %v", err)
	}

	if changed.StickyModifiers.Enabled {
		t.Error("sticky modifiers came back after a runtime field change")
	}
}
