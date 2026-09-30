package cli

import (
	"github.com/spf13/cobra"
)

// BuildHighlightCellCommand creates a highlight_cell cobra command that draws a
// recursive-grid cell as the one about to be picked, without picking it.
func BuildHighlightCellCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "highlight_cell [key]",
		Short: "Highlight a recursive-grid cell without selecting it",
		Long: `Draw the recursive-grid cell labelled key as the one about to be picked,
without narrowing to it. With no key, clear the highlight. Requires
recursive_grid mode to be active.

For a caller steering toward a cell some other way than typing its key — a
remote whose finger chooses the cell, a moment before it is chosen. The
highlight follows the key as the grid narrows, until it is changed or cleared.

Examples:
  neru action highlight_cell r
  neru action highlight_cell`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return requiresRunningInstance()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendCommand(cmd, "action", append([]string{"highlight_cell"}, args...))
		},
	}
}
