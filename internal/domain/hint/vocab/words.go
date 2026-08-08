package vocab

import "sync"

// maxWordLength caps every word beyond the first NATO round (indices 26+)
// so labels stay short enough to type and to fit the overlay.
const maxWordLength = 6

// pools holds, per starting letter, the ordered list of candidate words for
// that letter across rounds. pools[letter][0] is always the NATO phonetic
// word for that letter, exempt from maxWordLength (CHARLIE, NOVEMBER, and
// WHISKEY all run longer). Every later entry must be <= maxWordLength and
// no entry may be a prefix of another entry in the same pool — vocab_test.go
// enforces both.
var pools = map[byte][]string{
	'A': {"ALPHA", "AMBER", "ANCHOR", "ARBOR", "ASHEN", "AVID", "AZURE", "APRIL", "AUGUST", "ATLAS"},
	'B': {"BRAVO", "BASIL", "BERRY", "BISON", "BLAZE", "BUGLE", "BARGE", "BADGE", "BONUS", "BEACH"},
	'C': {"CHARLIE", "CEDAR", "CIVIC", "CORAL", "CRISP", "CANDY", "CABIN", "COMET", "CURVE", "CYCLE"},
	'D': {"DELTA", "DOZEN", "DAISY", "DERBY", "DODGE", "DRAFT", "DUSTY", "DIVER", "DONOR", "DELVE"},
	'E': {"ECHO", "EMBER", "EAGLE", "EARTH", "ELBOW", "ENJOY", "ERROR", "ETHIC", "EVENT", "EXTRA"},
	'F': {"FOXTROT", "FROST", "FABLE", "FALCON", "FERRY", "FIBER", "FLASK", "FORUM", "FUSION", "FUNKY"},
	'G': {"GOLF", "GRAPE", "GECKO", "GIANT", "GLOBE", "GRAVY", "GUEST", "GUILD", "GYPSY", "GENIE"},
	'H': {"HOTEL", "HAVEN", "HABIT", "HARSH", "HEDGE", "HINGE", "HONEY", "HUMOR", "HUSKY", "HYENA"},
	'I': {"INDIA", "IGLOO", "ICING", "IDEAL", "IMAGE", "INBOX", "IONIC", "IRISH", "ISSUE", "IVORY"},
	'J': {"JULIET", "JOKER", "JADE", "JAZZY", "JELLY", "JERKY", "JOLLY", "JUMBO", "JUNGLE", "JUSTLY"},
	'K': {"KILO", "KNOLL", "KAYAK", "KEBAB", "KETTLE", "KHAKI", "KIOSK", "KOALA", "KRAFT", "KOREA"},
	'L': {"LIMA", "LUNAR", "LADLE", "LAGER", "LATCH", "LEGACY", "LEMON", "LIGHT", "LOTUS", "LUCKY"},
	'M': {"MIKE", "MANGO", "MAPLE", "MARSH", "MEDAL", "MELON", "MERIT", "MODEM", "MOTOR", "MUSIC"},
	'N': {"NOVEMBER", "NORTH", "NAIVE", "NASAL", "NEEDY", "NEON", "NICHE", "NOBLE", "NUDGE", "NURSE"},
	'O': {"OSCAR", "OTTER", "OASIS", "OCEAN", "OFTEN", "OLIVE", "ONION", "OPERA", "ORBIT", "OUNCE"},
	'P': {"PAPA", "PEARL", "PANDA", "PATIO", "PEACH", "PIVOT", "PIXEL", "PLAZA", "PROXY", "PUZZLE"},
	'Q': {"QUEBEC", "QUAD", "QUILT", "QUEEN", "QUERY", "QUICK", "QUIRK", "QUOTA", "QUOTE", "QUIET"},
	'R': {"ROMEO", "ROBIN", "RADIO", "RAPID", "REACH", "RIDGE", "RIVER", "ROCKY", "ROYAL", "RUSTY"},
	'S': {"SIERRA", "SUNNY", "SABLE", "SALSA", "SATIN", "SAVOR", "SCOUT", "SHARK", "SILLY", "SOLAR"},
	'T': {"TANGO", "TIGER", "TABLE", "TALON", "TEMPO", "TEXAS", "THORN", "TONIC", "TRUCK", "TULIP"},
	'U': {"UNIFORM", "ULTRA", "UNCLE", "UNDUE", "UNION", "UNITY", "URBAN", "USAGE", "USUAL", "UTTER"},
	'V': {"VICTOR", "VIVID", "VALET", "VALOR", "VAPOR", "VELVET", "VENOM", "VIOLA", "VIRUS", "VOCAL"},
	'W': {"WHISKEY", "WALTZ", "WAFER", "WAGON", "WATCH", "WHEAT", "WIDOW", "WITTY", "WOVEN", "WORDY"},
	'X': {"XRAY", "XENON", "XEROX"},
	'Y': {"YANKEE", "YIELD", "YEAST", "YOUNG", "YUMMY", "YEARN", "YODEL", "YOGA", "YOLK", "YUCCA"},
	'Z': {"ZULU", "ZEBRA", "ZESTY", "ZIGZAG", "ZONAL", "ZOOM", "ZIPPY", "ZINC", "ZONE", "ZEAL"},
}

var (
	wordsOnce sync.Once
	words     []string
)

// buildWords computes the round-robin word list: round 0 is A-Z NATO, each
// following round adds up to one more word per letter (skipping letters
// whose pool is exhausted) until every pool is drained.
func buildWords() []string {
	maxRounds := 0
	for letter := byte('A'); letter <= 'Z'; letter++ {
		if n := len(pools[letter]); n > maxRounds {
			maxRounds = n
		}
	}

	result := make([]string, 0, 26*maxRounds)

	for round := range maxRounds {
		for letter := byte('A'); letter <= 'Z'; letter++ {
			pool := pools[letter]
			if round >= len(pool) {
				continue
			}

			result = append(result, pool[round])
		}
	}

	return result
}

// Words returns the ordered, uppercase, ASCII, prefix-free vocabulary.
func Words() []string {
	wordsOnce.Do(func() {
		words = buildWords()
	})

	return words
}

// Len returns the number of words in the vocabulary.
func Len() int {
	return len(Words())
}

// Word returns the word at index i, if any.
func Word(i int) (string, bool) {
	all := Words()
	if i < 0 || i >= len(all) {
		return "", false
	}

	return all[i], true
}

// IsPrefixFree reports whether no word in the vocabulary is a prefix of
// another word in the vocabulary.
func IsPrefixFree() bool {
	all := Words()
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}

			if len(a) <= len(b) && b[:len(a)] == a {
				return false
			}
		}
	}

	return true
}
