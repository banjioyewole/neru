package vocab

import (
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/y3owk1n/neru/internal/derrors"
)

// newFoldTransform builds a transformer that strips diacritics (NFD, drop
// combining marks, NFC) the same way hint.normalizeForSearch does, so
// accented dictation ("QUÉBEC") folds onto the ASCII vocabulary ("QUEBEC").
// transform.Transformer is stateful and not safe for concurrent reuse, so a
// fresh chain is built per call rather than shared as a package variable.
func newFoldTransform() transform.Transformer {
	return transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
}

// fold uppercases s, strips diacritics, and drops everything but letters and
// digits (punctuation, hyphens, whitespace).
func fold(s string) string {
	normalized, _, err := transform.String(newFoldTransform(), s)
	if err != nil {
		normalized = s
	}

	var b strings.Builder

	b.Grow(len(normalized))

	for _, r := range normalized {
		r = unicode.ToUpper(r)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

var (
	wordIndexOnce sync.Once
	wordIndexMap  map[string]struct{}
)

func wordIndex() map[string]struct{} {
	wordIndexOnce.Do(func() {
		m := make(map[string]struct{}, Len())
		for _, w := range Words() {
			m[w] = struct{}{}
		}

		wordIndexMap = m
	})

	return wordIndexMap
}

// Resolve maps a single spoken/typed token to a canonical vocabulary word.
// It folds case and accents, strips punctuation, applies the alias table,
// and falls back to a unique edit-distance-1 neighbour in the vocabulary.
func Resolve(token string) (string, bool) {
	folded := fold(token)
	if folded == "" {
		return "", false
	}

	if _, ok := wordIndex()[folded]; ok {
		return folded, true
	}

	if canonical, ok := aliases[folded]; ok {
		return canonical, true
	}

	return resolveByEditDistance(folded)
}

// resolveByEditDistance returns the unique vocabulary word within edit
// distance 1 of folded, if exactly one exists.
func resolveByEditDistance(folded string) (string, bool) {
	var (
		match   string
		matches int
	)

	for _, w := range Words() {
		if editDistanceAtMost1(folded, w) {
			matches++
			match = w

			if matches > 1 {
				return "", false
			}
		}
	}

	if matches == 1 {
		return match, true
	}

	return "", false
}

// editDistanceAtMost1 reports whether a and b differ by at most one
// insertion, deletion, or substitution. It is a short-circuiting check, not
// a general Levenshtein implementation, since callers only care about the
// distance-1 boundary.
func editDistanceAtMost1(a, b string) bool {
	if a == b {
		return true
	}

	la, lb := len(a), len(b)
	if la > lb {
		a, b = b, a
		la, lb = lb, la
	}

	if lb-la > 1 {
		return false
	}

	if la == lb {
		diff := 0

		for i := range a {
			if a[i] != b[i] {
				diff++
				if diff > 1 {
					return false
				}
			}
		}

		return diff <= 1
	}

	// lb == la+1: check b is a with one rune inserted.
	i, j, diff := 0, 0, 0
	for i < la && j < lb {
		if a[i] == b[j] {
			i++
			j++

			continue
		}

		diff++
		if diff > 1 {
			return false
		}

		j++
	}

	return true
}

// ResolveTranscript maps a noisy spoken phrase to a canonical vocabulary
// word. It first tries the whole transcript with whitespace collapsed (so
// "x ray" and "x-ray" both resolve to XRAY), then splits on whitespace and
// punctuation and succeeds only when exactly one token resolves — "sierra
// delta" is refused, not guessed.
func ResolveTranscript(transcript string) (string, error) {
	if whole := fold(transcript); whole != "" {
		if label, ok := Resolve(whole); ok {
			return label, nil
		}
	}

	fields := strings.FieldsFunc(transcript, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var (
		resolved string
		count    int
	)

	for _, f := range fields {
		if label, ok := Resolve(f); ok {
			resolved = label
			count++
		}
	}

	if count != 1 {
		return "", derrors.Newf(
			derrors.CodeInvalidInput,
			"transcript %q did not resolve to exactly one hint label", transcript,
		)
	}

	return resolved, nil
}
