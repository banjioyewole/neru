package ipcctrl

import "testing"

func TestElementsSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		onScreen int
		total    int
		want     string
	}{
		{"all on screen", 12, 12, "12 elements"},
		{"some off screen", 3, 7, "3 of 7 elements on the active screen"},
		{"nothing found", 0, 0, "0 elements"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := elementsSummary(testCase.onScreen, testCase.total); got != testCase.want {
				t.Errorf("elementsSummary(%d, %d) = %q, want %q",
					testCase.onScreen, testCase.total, got, testCase.want)
			}
		})
	}
}
