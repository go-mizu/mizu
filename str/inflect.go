package str

import (
	"strings"
	"unicode"
)

// irregulars are the nouns whose plural is not their singular with an ending
// changed, so no rule reaches them. The list is the ones that turn up in the
// names of things: the old English plurals, the Latin and Greek endings that
// English kept, the f words that voice their f, and the handful of nouns that
// end in s while being singular.
var irregulars = map[string]string{
	"man": "men", "woman": "women", "child": "children", "person": "people",
	"tooth": "teeth", "foot": "feet", "goose": "geese", "mouse": "mice",
	"louse": "lice", "ox": "oxen", "die": "dice", "penny": "pence",

	"cactus": "cacti", "focus": "foci", "fungus": "fungi", "nucleus": "nuclei",
	"radius": "radii", "syllabus": "syllabi", "alumnus": "alumni",
	"stimulus": "stimuli",

	"bacterium": "bacteria", "curriculum": "curricula", "datum": "data",
	"medium": "media", "memorandum": "memoranda", "stratum": "strata",

	"analysis": "analyses", "basis": "bases", "crisis": "crises",
	"diagnosis": "diagnoses", "hypothesis": "hypotheses", "oasis": "oases",
	"parenthesis": "parentheses", "synopsis": "synopses", "thesis": "theses",
	"axis": "axes", "ellipsis": "ellipses",

	"criterion": "criteria", "phenomenon": "phenomena",

	"index": "indices", "matrix": "matrices", "vertex": "vertices",
	"appendix": "appendices",

	"life": "lives", "knife": "knives", "wife": "wives", "leaf": "leaves",
	"loaf": "loaves", "half": "halves", "shelf": "shelves", "wolf": "wolves",
	"self": "selves", "thief": "thieves", "calf": "calves", "elf": "elves",
	"hoof": "hooves", "scarf": "scarves",

	"bus": "buses", "gas": "gases", "quiz": "quizzes", "virus": "viruses",
	"status": "statuses", "campus": "campuses", "alias": "aliases",
	"atlas": "atlases", "lens": "lenses",

	"potato": "potatoes", "tomato": "tomatoes", "hero": "heroes",
	"echo": "echoes", "veto": "vetoes", "volcano": "volcanoes",
	"torpedo": "torpedoes", "buffalo": "buffaloes", "mosquito": "mosquitoes",
}

// singulars is irregulars read the other way round, built once at start up so
// that Singular is a lookup rather than a search.
var singulars = func() map[string]string {
	m := make(map[string]string, len(irregulars))
	for one, many := range irregulars {
		m[many] = one
	}
	return m
}()

// uncountables are the nouns that are the same in both numbers. Asking for the
// plural of one gives it back unchanged, which is the right answer and not a
// failure to find a rule.
var uncountables = map[string]bool{
	"advice": true, "aircraft": true, "art": true, "bison": true,
	"chassis": true, "corps": true, "deer": true, "equipment": true,
	"evidence": true, "fish": true, "furniture": true, "happiness": true,
	"information": true, "jeans": true, "knowledge": true, "luggage": true,
	"means": true, "money": true, "moose": true, "music": true,
	"news": true, "offspring": true, "police": true, "rice": true,
	"salmon": true, "series": true, "sheep": true, "software": true,
	"species": true, "swine": true, "trout": true,
}

// Plural returns the plural of an English noun.
//
//	str.Plural("user")     // users
//	str.Plural("person")   // people
//	str.Plural("city")     // cities
//	str.Plural("sheep")    // sheep
//
// The case of the input is kept, so Person gives People and PERSON gives
// PEOPLE. A word that is already plural comes back unchanged, which makes this
// safe to call on a name that may have been pluralised already.
//
// See [Singular] for what this does not know.
func Plural(s string) string {
	if s == "" {
		return s
	}

	lower := strings.ToLower(s)
	if uncountables[lower] {
		return s
	}
	if many, ok := irregulars[lower]; ok {
		return matchCase(s, many)
	}
	if singulars[lower] != "" {
		return s
	}

	switch {
	case hasAnySuffix(lower, "ss", "x", "z", "ch", "sh"):
		return s + suffixCase(s, "es")
	case strings.HasSuffix(lower, "s"):
		// Already plural as far as the rules can tell. Adding another s here is
		// what turns a table name into users s the second time something
		// pluralises it.
		return s
	case len(lower) > 1 && strings.HasSuffix(lower, "y") && !isVowel(lower[len(lower)-2]):
		return s[:len(s)-1] + suffixCase(s, "ies")
	}
	return s + suffixCase(s, "s")
}

// PluralN returns the plural of s when n is anything but one, and the singular
// when it is one.
//
//	str.PluralN("item", 1)   // item
//	str.PluralN("item", 3)   // items
//	str.PluralN("items", 1)  // item
//
// Zero takes the plural, because English says no items rather than no item.
// The answer does not depend on whether s was handed over singular or plural,
// which is what makes this usable on a name that came from somewhere else.
func PluralN(s string, n int) string {
	if n == 1 || n == -1 {
		return Singular(s)
	}
	return Plural(s)
}

// Singular returns the singular of an English noun.
//
//	str.Singular("users")    // user
//	str.Singular("people")   // person
//	str.Singular("cities")   // city
//
// English inflection is not a solved problem and this does not pretend to
// solve it. It knows the endings that follow a rule and a list of the nouns
// that do not, which together cover the words that end up in the name of a type
// or a table. Two things it gets wrong are worth knowing about: a compound
// ending in an irregular noun is not recognised, so Singular of chairmen is
// chairmen rather than chairman, and a plural that could have come from two
// different singulars picks one, so bases is basis rather than base.
//
// Code that has a word this gets wrong should say so at the call site rather
// than hope. There is no registry to add to, on purpose: a package level table
// that any package can write to is a data race waiting for a second goroutine.
func Singular(s string) string {
	if s == "" {
		return s
	}

	lower := strings.ToLower(s)
	if uncountables[lower] {
		return s
	}
	if one, ok := singulars[lower]; ok {
		return matchCase(s, one)
	}
	if irregulars[lower] != "" {
		return s
	}

	switch {
	case len(lower) > 4 && strings.HasSuffix(lower, "ies"):
		// Two letters have to be left in front of the ies, because every four
		// letter word ending in it came from an ie rather than a y: ties is the
		// plural of tie, and cries is the shortest one that came from a y.
		return s[:len(s)-3] + suffixCase(s, "y")
	case hasAnySuffix(lower, "sses", "xes", "zes", "ches", "shes"):
		return s[:len(s)-2]
	case strings.HasSuffix(lower, "s") && !hasAnySuffix(lower, "ss", "us", "is"):
		return s[:len(s)-1]
	}
	return s
}

// matchCase returns word shaped like sample, so that a rule written in
// lowercase can answer a question asked in any case. It is for the irregulars,
// where the whole word is being swapped for another one.
//
// sample has to have something in it. Both callers return early on an empty
// string, since there is no plural of nothing.
func matchCase(sample, word string) string {
	if isAllUpper(sample) {
		return strings.ToUpper(word)
	}
	if unicode.IsUpper(rune(sample[0])) {
		return UpperFirst(word)
	}
	return word
}

// suffixCase returns ending in the case the rest of sample is written in. This
// is the other half of matchCase: an ending glued onto a word that is already
// there follows that word rather than starting a new one, so Box takes es and
// not Es.
func suffixCase(sample, ending string) string {
	if isAllUpper(sample) {
		return strings.ToUpper(ending)
	}
	return ending
}

// isAllUpper reports whether sample has letters in it and none of them are
// lowercase, which is the test for a name written in shouting case.
func isAllUpper(sample string) bool {
	return sample == strings.ToUpper(sample) && sample != strings.ToLower(sample)
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func isVowel(c byte) bool {
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}
