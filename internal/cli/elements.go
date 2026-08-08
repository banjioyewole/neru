package cli

import (
	"github.com/spf13/cobra"

	"github.com/y3owk1n/neru/internal/cli/cliutil"
	"github.com/y3owk1n/neru/internal/domain"
)

// ElementsCmd reports the on-screen elements hints mode would target, as data.
//
// The counterpart to `neru hints --debug`, which prints a short human sample.
// This one is for callers that are not people: it returns every element with the
// hint label attached, and that label is exactly what `neru action select_hint`
// accepts — so a script or an agent can read the screen and act on it without
// ever drawing an overlay or reading pixels.
var ElementsCmd = &cobra.Command{
	Use:   "elements",
	Short: "List the elements hints mode would target, as JSON",
	Long: `List the on-screen elements hints mode would target, without drawing an
overlay or entering a mode.

Each element carries the hint label that would select it, its role, its text,
and its bounds and centre in global top-left-origin pixels. The label is what
'neru action select_hint' accepts, so reading the screen and acting on it are
the same vocabulary.

Elements are collected from whatever window is focused when the command runs —
invoked from a terminal, it reports that terminal's elements.

Element values are withheld unless --include-values is passed. A value is the
content of a field rather than its identity, so it can hold whatever has been
typed into it; titles and descriptions are enough to identify an element.

Examples:
  neru elements --json
  neru elements --json --role button,link
  neru elements --json --text "search"
  neru elements --json --include-values`,
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return requiresRunningInstance()
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		asJSON, flagErr := cmd.Flags().GetBool("json")
		if flagErr != nil {
			return flagErr
		}

		var args []string

		if roles, _ := cmd.Flags().GetString("role"); roles != "" {
			args = append(args, "--role", roles)
		}

		if text, _ := cmd.Flags().GetString("text"); text != "" {
			args = append(args, "--text", text)
		}

		if strategy, _ := cmd.Flags().GetString("strategy"); strategy != "" {
			args = append(args, "--strategy", strategy)
		}

		if includeValues, _ := cmd.Flags().GetBool("include-values"); includeValues {
			args = append(args, "--include-values")
		}

		communicator := cliutil.NewIPCCommunicator(timeoutSec)

		ipcResponse, err := communicator.SendCommand(domain.CommandElements, args)
		if err != nil {
			return err
		}

		if !ipcResponse.Success {
			return communicator.HandleResponse(cmd, ipcResponse)
		}

		// The payload is the point of this command, so JSON is the default
		// shape. Without --json it prints the one-line summary instead, which
		// is all a human wants when checking whether anything was found.
		if asJSON {
			return formatter.PrintJSON(cmd, ipcResponse.Data)
		}

		cmd.Println(ipcResponse.Message)

		return nil
	},
}

func init() {
	RootCmd.AddCommand(ElementsCmd)

	ElementsCmd.Flags().Bool("json", false, "Print the elements as a JSON object")
	ElementsCmd.Flags().String("role", "", "Only elements with these roles (comma-separated)")
	ElementsCmd.Flags().String("text", "", "Only elements whose text contains this")
	ElementsCmd.Flags().String("strategy", "", "Element source: axtree or vision")
	ElementsCmd.Flags().Bool(
		"include-values",
		false,
		"Include element values (the content of fields, not just their labels)",
	)
}
