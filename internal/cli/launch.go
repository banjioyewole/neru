package cli

import (
	"github.com/spf13/cobra"
)

// noSystray backs --no-systray. Not a config key: it says how this particular
// daemon was launched, not what the user wants Neru to do generally. A
// supervising app that presents its own menu bar item passes it so the two do
// not both appear; the same config run by hand still gets a tray.
var noSystray bool

// LaunchCmd is the CLI launch command.
var LaunchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Start the Neru daemon",
	Long: `Start the Neru daemon process.

This initializes the accessibility engine, overlay, and IPC server
to handle navigation modes and commands. Once launched, Neru runs
in the background until quit from the system tray menu.

Use 'neru stop' to pause functionality (daemon stays running).`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		launchProgram(cmd, configPath, launchOptions())

		return nil
	},
}

// launchOptions collects the parsed flags the daemon needs. Separate from RunE
// so it can be tested without launchProgram, which either becomes the daemon or
// exits when one is already running.
func launchOptions() LaunchOptions {
	return LaunchOptions{NoSystray: noSystray}
}

func init() {
	LaunchCmd.Flags().BoolVar(
		&noSystray,
		"no-systray",
		false,
		"Run without a system tray icon, whatever the config says",
	)

	RootCmd.AddCommand(LaunchCmd)
}
