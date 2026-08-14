package cli

import "testing"

// TestLaunchOptions_NoSystray pins the flag to the option the daemon reads.
// Without it, --no-systray can go on parsing cleanly while reaching nothing.
func TestLaunchOptions_NoSystray(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "absent", args: []string{}, want: false},
		{name: "present", args: []string{"--no-systray"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				_ = LaunchCmd.Flags().Set("no-systray", "false")
			})

			err := LaunchCmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("unexpected error parsing flags: %v", err)
			}

			got := launchOptions()
			if got.NoSystray != tt.want {
				t.Errorf("NoSystray = %v, want %v", got.NoSystray, tt.want)
			}
		})
	}
}
