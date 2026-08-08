package vocab

// aliases maps common ASR mis-transcriptions and alternate spellings to the
// canonical vocabulary word they mean. Keys are folded (uppercase, accents
// stripped, punctuation removed) before lookup. Seeded from the NATO
// alphabet's well-known variants; grow this table from real transcripts.
var aliases = map[string]string{
	"ALFA":     "ALPHA",
	"JULIETT":  "JULIET",
	"JULIETTE": "JULIET",
	"WHISKY":   "WHISKEY",
}
