// Package vocab provides the spoken/typed word vocabulary used to label
// hints. Labels are drawn round-robin over the alphabet — the first 26 are
// the NATO phonetic alphabet (ALPHA, BRAVO, ...), chosen because automatic
// speech recognition transcribes them near-perfectly; further rounds add a
// curated word per letter (AMBER, BASIL, ...) capped at six characters.
//
// The vocabulary is prefix-free by construction: no word is a prefix of
// another. This lets the hint matching path treat any unique filtered
// result as a selection without requiring the full label to be typed or
// spoken.
package vocab
