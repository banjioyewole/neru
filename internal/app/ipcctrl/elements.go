package ipcctrl

import (
	"context"
	"strconv"

	"github.com/y3owk1n/neru/internal/adapter/ipc"
)

// flagIncludeValues opts into returning element values. Off by default — see
// modes.ProbeHintElements for why.
const flagIncludeValues = "--include-values"

// handleElements answers with the on-screen elements hints mode would target,
// as structured data rather than prose.
//
// This is what `hints-probe` could not be. That command formats a ten-element
// human sample with no labels and leaves Data unset, which is unusable to a
// caller that needs to choose an element and act on it. The label returned here
// is exactly what `action select_hint` accepts, so reading the screen and acting
// on it close the loop without going through pixels.
func (h *ModesHandler) handleElements(ctx context.Context, cmd ipc.Command) ipc.Response {
	if h.modes == nil {
		return h.modesUnavailableResponse()
	}

	// extractProbeOptions parses the mode-command grammar shared with
	// hints-probe and rejects anything it does not recognise, so this flag is
	// taken out before handing the rest over rather than being added to a
	// grammar the other command has no use for.
	includeValues := false
	remaining := make([]string, 0, len(cmd.Args))

	for _, arg := range cmd.Args {
		if arg == flagIncludeValues {
			includeValues = true

			continue
		}

		remaining = append(remaining, arg)
	}

	probeCmd := cmd
	probeCmd.Args = remaining

	opts, errResp := h.extractProbeOptions(probeCmd)
	if errResp != nil {
		return *errResp
	}

	result, probeErr := h.modes.ProbeHintElements(
		ctx,
		opts.FilterRoles,
		opts.FilterTextContains,
		opts.Strategy,
		opts.SplitWord,
		includeValues,
	)
	if probeErr != nil {
		return ipc.Response{
			Success: false,
			Message: "elements probe failed: " + probeErr.Error(),
			Code:    ipc.CodeActionFailed,
		}
	}

	return ipc.Response{
		Success: true,
		Message: elementsSummary(result.OnScreen, result.Total),
		Code:    ipc.CodeOK,
		Data:    result,
	}
}

func elementsSummary(onScreen, total int) string {
	if onScreen == total {
		return strconv.Itoa(total) + " elements"
	}

	return strconv.Itoa(onScreen) + " of " + strconv.Itoa(total) + " elements on the active screen"
}
