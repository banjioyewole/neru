package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/y3owk1n/neru/internal/derrors"
)

// BuildSelectHintCommand creates a select_hint cobra command that selects the
// hint matching a spoken/typed word.
func BuildSelectHintCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "select_hint <word>",
		Short: "Select the hint matching a spoken or typed word",
		Long: `Select the hint whose label matches a spoken or typed word, resolved through
the hint vocabulary (NATO-style words, e.g. sierra, delta). Requires hints
mode to be active.

Intended for voice-driven selection: the caller (e.g. a dictation bridge)
passes the raw transcript, which is resolved the same way whether it came
through cleanly ("sierra") or wrapped in filler ("uh, sierra.").

Examples:
  neru action select_hint sierra
  neru action select_hint "uh, sierra."`,
		Args: validateSelectHintArgs,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return requiresRunningInstance()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			actionArgs := []string{"select_hint", strings.Join(args, " ")}

			return sendCommand(cmd, "action", actionArgs)
		},
	}
}

func validateSelectHintArgs(_ *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(strings.Join(args, " ")) == "" {
		return derrors.New(
			derrors.CodeInvalidInput,
			"select_hint requires a word (e.g., neru action select_hint sierra)",
		)
	}

	return nil
}
