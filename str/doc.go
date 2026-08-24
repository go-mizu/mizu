// Package str is the string handling that [strings] leaves out.
//
//	str.Snake("HTTPServer")   // http_server
//	str.Headline("user_id")   // User Id
//	str.Length("👨‍👩‍👧‍👦")           // 1
//
// Anything [strings] already does is not here. There is no str.Contains and no
// str.ToUpper, because [strings.Contains] and [strings.ToUpper] are the same
// function under a different name and having it twice is worse than having it
// once.
//
// # Characters, not bytes
//
// Every function here counts in grapheme clusters. A grapheme cluster is what a
// reader calls a character, and it is often more than one code point: an e with
// an accent written as two, an emoji with a skin tone, a flag made of two
// regional indicators, a family made of four people and three joiners.
//
//	len("👨‍👩‍👧‍👦")                       // 25, bytes
//	utf8.RuneCountInString("👨‍👩‍👧‍👦")     // 7, code points
//	str.Length("👨‍👩‍👧‍👦")                 // 1, characters
//
// This is the difference that makes the package worth having. Cutting a string
// at a byte offset splits a character down the middle and leaves a replacement
// mark on screen; cutting at a rune offset splits a family into a father and a
// leftover. [Graphemes] and [Length] are the two functions everything else is
// built on, and [Graphemes] is worth using directly whenever a loop needs to
// walk characters rather than runes.
//
// # How the segmenter works
//
// The boundaries come from Unicode annex 29, implemented against the category
// tables in [unicode] plus a handful of ranges the standard library does not
// name: the regional indicators, the emoji skin tone modifiers, the two zero
// width joiners, and the marks that attach forwards rather than backwards.
// There is no public grapheme segmenter in golang.org/x/text to call instead,
// which is why this one is here.
//
// It follows the annex with two deliberate differences, both of which join more
// than the annex does rather than less:
//
//   - A zero width joiner holds together whatever is on either side of it. The
//     annex only joins when both sides are emoji. A joiner between two things
//     that are not emoji is a writer asking for them to stay together, so
//     keeping them together is the answer either way.
//   - Every spacing mark counts as a spacing mark. The annex excludes about a
//     dozen of them, mostly in Burmese and Khmer, where the mark can start a
//     cluster of its own.
//
// The rest of it, the Hangul syllable rules, the regional indicator pairing,
// the carriage return and newline pair, the marks that attach forwards, is the
// annex as written.
//
// # Naming
//
// Two functions are named differently from the spec. Ucfirst and Lcfirst are
// [UpperFirst] and [LowerFirst], and Swap is [SwapCase], because the first two
// are names from PHP and the third says nothing about what it swaps.
//
// # Cost
//
// [Length] and [Graphemes] walk the string once and allocate nothing. On a
// thousand characters of English that is about 2 microseconds, against 0.35 for
// [utf8.RuneCountInString], so knowing where the characters really are costs
// about six times what counting code points costs. Accented Latin is about 3
// microseconds and emoji about 15, because those go through the category tables
// rather than down the fast path for text below the combining diacritics.
//
// The fast path is worth knowing about, because it is the difference between 2
// microseconds and 97. Without it every character costs two searches of a
// Unicode range table.
//
// The case functions allocate: they cut the string into words and put it back
// together, which is around ten allocations for a name of three or four words.
// They are for names and headings, which is code that runs once, not code in a
// loop.
//
// Timings came from an Apple M4 with other work running on it, so read them as
// ceilings. The allocation counts do not move.
package str
