package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/y3owk1n/neru/internal/config"
	"github.com/y3owk1n/neru/internal/config/loader"
)

// writeSuppressionConfig writes a config file and returns its path.
func writeSuppressionConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")

	err := os.WriteFile(path, []byte(body), 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

// hasNormalizedBinding reports whether any binding normalizes to chord.
func hasNormalizedBinding(bindings map[string][]string, chord string) bool {
	want := config.NormalizeKeyForComparison(chord)

	for key := range bindings {
		if config.NormalizeKeyForComparison(key) == want {
			return true
		}
	}

	return false
}

func TestWithSuppressedHotkeysDropsDefaultBinding(t *testing.T) {
	path := writeSuppressionConfig(t, "[hotkeys]\n\"Primary+Shift+H\" = \"hints\"\n")

	service := loader.NewService(config.DefaultConfig(), path, zap.NewNop(), nil).
		WithSuppressedHotkeys([]string{"Primary+Shift+G"})

	result := service.LoadWithValidation(path)
	if result.ValidationError != nil {
		t.Fatalf("load: %v", result.ValidationError)
	}

	if hasNormalizedBinding(result.Config.Hotkeys.Bindings, "Primary+Shift+G") {
		t.Error("Primary+Shift+G survived suppression")
	}

	// The chord next to it is untouched, so this is a suppression and not a
	// wipe of the whole table.
	if !hasNormalizedBinding(result.Config.Hotkeys.Bindings, "Primary+Shift+H") {
		t.Error("Primary+Shift+H was dropped along with the suppressed chord")
	}
}

func TestWithSuppressedHotkeysMatchesRegardlessOfSpelling(t *testing.T) {
	// The user's own spelling of the chord, which is not the flag's.
	path := writeSuppressionConfig(t, "[hotkeys]\n\"Shift+Primary+g\" = \"grid\"\n")

	service := loader.NewService(config.DefaultConfig(), path, zap.NewNop(), nil).
		WithSuppressedHotkeys([]string{"Primary+Shift+G"})

	result := service.LoadWithValidation(path)
	if result.ValidationError != nil {
		t.Fatalf("load: %v", result.ValidationError)
	}

	if hasNormalizedBinding(result.Config.Hotkeys.Bindings, "Primary+Shift+G") {
		t.Error("a differently spelled binding of the suppressed chord survived")
	}
}

func TestWithSuppressedHotkeysDropsPerAppOverride(t *testing.T) {
	path := writeSuppressionConfig(t, `[hotkeys]
"Primary+Shift+G" = "grid"

[[app_configs]]
bundle_id = "com.apple.finder"
hotkeys = { "Primary+Shift+G" = "hints" }
`)

	service := loader.NewService(config.DefaultConfig(), path, zap.NewNop(), nil).
		WithSuppressedHotkeys([]string{"Primary+Shift+G"})

	result := service.LoadWithValidation(path)
	if result.ValidationError != nil {
		t.Fatalf("load: %v", result.ValidationError)
	}

	for _, appConfig := range result.Config.AppConfigs {
		for key := range appConfig.Hotkeys {
			if config.NormalizeKeyForComparison(key) ==
				config.NormalizeKeyForComparison("Primary+Shift+G") {
				t.Errorf("per-app override for %q survived suppression", appConfig.BundleID)
			}
		}
	}
}

func TestWithSuppressedHotkeysAppliesToRefusedConfig(t *testing.T) {
	// A config bad enough to be refused drops the daemon back onto the stock
	// bindings, which is where the suppressed chord lives by default.
	path := writeSuppressionConfig(t, "[hotkeys]\n\"Primary+Shift+G\" = 42\n")

	service := loader.NewService(config.DefaultConfig(), path, zap.NewNop(), nil).
		WithSuppressedHotkeys([]string{"Primary+Shift+G"})

	result := service.LoadWithValidation(path)
	if result.ValidationError == nil {
		t.Fatal("expected the malformed config to be refused")
	}

	if hasNormalizedBinding(result.Config.Hotkeys.Bindings, "Primary+Shift+G") {
		t.Error("the fallback defaults brought the suppressed chord back")
	}
}

func TestWithoutSuppressedHotkeysKeepsTheDefault(t *testing.T) {
	path := writeSuppressionConfig(t, "[general]\nexec_shell = \"/bin/sh\"\n")

	service := loader.NewService(config.DefaultConfig(), path, zap.NewNop(), nil)

	result := service.LoadWithValidation(path)
	if result.ValidationError != nil {
		t.Fatalf("load: %v", result.ValidationError)
	}

	if !hasNormalizedBinding(result.Config.Hotkeys.Bindings, "Primary+Shift+G") {
		t.Error("Primary+Shift+G is missing without any suppression asked for")
	}
}
