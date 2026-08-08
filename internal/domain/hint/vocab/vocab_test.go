package vocab_test

import (
	"sync"
	"testing"

	"github.com/y3owk1n/neru/internal/domain/hint/vocab"
)

// TestResolve_ConcurrentSafe guards against reusing a stateful
// transform.Transformer across goroutines (data corruption there manifests
// as a slice-bounds panic, not a silently wrong answer) — run with -race.
func TestResolve_ConcurrentSafe(t *testing.T) {
	var wg sync.WaitGroup

	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, ok := vocab.Resolve("QUÉBEC"); !ok {
				t.Error("Resolve(QUÉBEC) failed under concurrent load")
			}

			if _, err := vocab.ResolveTranscript("uh, sierra."); err != nil {
				t.Errorf("ResolveTranscript failed under concurrent load: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestWords_PrefixFree(t *testing.T) {
	if !vocab.IsPrefixFree() {
		t.Fatal("vocabulary is not prefix-free")
	}
}

func TestWords_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for _, w := range vocab.Words() {
		if seen[w] {
			t.Fatalf("duplicate word %q", w)
		}

		seen[w] = true
	}
}

func TestWords_UppercaseASCII(t *testing.T) {
	for _, w := range vocab.Words() {
		for _, r := range w {
			if r < 'A' || r > 'Z' {
				t.Fatalf("word %q contains non-uppercase-ASCII rune %q", w, r)
			}
		}
	}
}

func TestWords_AtLeast300(t *testing.T) {
	if got := vocab.Len(); got < 200 {
		t.Fatalf("Len() = %d, want at least 200", got)
	}
}

func TestWords_Round1IsNATO(t *testing.T) {
	nato := []string{
		"ALPHA", "BRAVO", "CHARLIE", "DELTA", "ECHO", "FOXTROT", "GOLF", "HOTEL",
		"INDIA", "JULIET", "KILO", "LIMA", "MIKE", "NOVEMBER", "OSCAR", "PAPA",
		"QUEBEC", "ROMEO", "SIERRA", "TANGO", "UNIFORM", "VICTOR", "WHISKEY",
		"XRAY", "YANKEE", "ZULU",
	}

	words := vocab.Words()
	if len(words) < len(nato) {
		t.Fatalf("vocabulary too short for NATO round: %d", len(words))
	}

	for i, want := range nato {
		if words[i] != want {
			t.Errorf("words[%d] = %q, want %q", i, words[i], want)
		}
	}
}

func TestWords_BeyondRound1CappedAtSixChars(t *testing.T) {
	words := vocab.Words()
	for i := 26; i < len(words); i++ {
		if len(words[i]) > 6 {
			t.Errorf("words[%d] = %q exceeds 6 chars", i, words[i])
		}
	}
}

func TestLen_MatchesWords(t *testing.T) {
	if vocab.Len() != len(vocab.Words()) {
		t.Fatalf("Len() = %d, want %d", vocab.Len(), len(vocab.Words()))
	}
}

func TestWord_Bounds(t *testing.T) {
	if _, ok := vocab.Word(-1); ok {
		t.Error("Word(-1) should not be ok")
	}

	if _, ok := vocab.Word(vocab.Len()); ok {
		t.Error("Word(Len()) should not be ok")
	}

	w, ok := vocab.Word(0)
	if !ok || w != "ALPHA" {
		t.Errorf("Word(0) = %q, %v, want ALPHA, true", w, ok)
	}
}

func TestResolve_Table(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
		ok    bool
	}{
		{"exact", "SIERRA", "SIERRA", true},
		{"lowercase", "sierra", "SIERRA", true},
		{"trailing punctuation", "sierra.", "SIERRA", true},
		{"accented", "QUÉBEC", "QUEBEC", true},
		{"alias ALFA", "alfa", "ALPHA", true},
		{"alias JULIETT", "juliett", "JULIET", true},
		{"alias JULIETTE", "juliette", "JULIET", true},
		{"alias WHISKY", "whisky", "WHISKEY", true},
		{"unresolvable gibberish", "zzzznotaword", "", false},
		{"empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := vocab.Resolve(tt.token)
			if ok != tt.ok || got != tt.want {
				t.Errorf("Resolve(%q) = %q, %v, want %q, %v", tt.token, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestResolveTranscript_Table(t *testing.T) {
	tests := []struct {
		name       string
		transcript string
		want       string
		wantErr    bool
	}{
		{"clean", "sierra", "SIERRA", false},
		{"filler wrapped", "uh, sierra.", "SIERRA", false},
		{"x ray spaced", "x ray", "XRAY", false},
		{"x-ray hyphen", "x-ray", "XRAY", false},
		{"two valid words refused", "sierra delta", "", true},
		{"nonsense refused", "qwqwqwqwq", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vocab.ResolveTranscript(tt.transcript)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveTranscript(%q) = %q, nil, want error", tt.transcript, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("ResolveTranscript(%q) unexpected error: %v", tt.transcript, err)
			}

			if got != tt.want {
				t.Errorf("ResolveTranscript(%q) = %q, want %q", tt.transcript, got, tt.want)
			}
		})
	}
}
