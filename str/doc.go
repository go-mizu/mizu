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
// # When something is not there
//
// [Before], [BeforeLast], [After] and [AfterLast] hand back the whole string
// when they do not find what they were given. That is the one surprise in the
// package, and it is there because it makes the four of them chain: After of a
// path with no slash in it is the path, which is usually what the next call
// wants. Code that has to tell a match from a miss wants [strings.Cut], which
// returns a bool for exactly that.
//
// [Excerpt] goes the other way and returns nothing when the phrase is absent,
// because an excerpt of something that is not there is nothing rather than
// everything.
//
// # Naming
//
// Two functions are named differently from the spec. Ucfirst and Lcfirst are
// [UpperFirst] and [LowerFirst], and Swap is [SwapCase], because the first two
// are names from PHP and the third says nothing about what it swaps.
//
// Plural takes a count through [PluralN] rather than an optional argument,
// because Go does not have optional arguments and pretending otherwise with a
// variadic reads worse than a second name.
//
// Eight were dropped for being a name over a line of [strings]. WordCount is
// len(strings.Fields(s)), Lower and Upper are [strings.ToLower] and
// [strings.ToUpper], Squish is strings.Join(strings.Fields(s), " "), Replace and
// Remove are [strings.ReplaceAll] with and without a replacement, ReplaceFirst
// is [strings.Replace] with a count of one, and PluralStudly is Pascal of
// Plural. The hand-written Squish was measured against that line and came out
// slower, so there was nothing left to argue for it.
//
// Two more went for being too narrow to carry a name. Transliterate is [Ascii]
// with a table the caller supplies, and a caller with a table already has
// [strings.NewReplacer]. ReplaceArray walks a list of replacements into
// successive occurrences of one search, which is a parameterised query with the
// safety taken out.
//
// # English
//
// [Plural] and [Singular] are English and nothing else. Inflection outside
// English is not a rule table, and a package that guessed at it would be wrong
// in a way that looked right. Anything that has to read as a sentence in more
// than one language wants plural rules keyed by locale rather than a spelling
// change, and that lives with the translations rather than here.
//
// Inside English they are a heuristic with a word list behind it, sized for the
// words that end up in the name of a type or a table. [Singular] names the two
// ways it is known to be wrong.
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
// The cutting functions are [strings.Index] and a slice, so they allocate
// nothing and cost what a search costs. The slicing functions pay for the walk
// that finds the character offsets, which is linear in the part of the string
// they have to cross: [Take] of the first ten characters is ten clusters of
// work whatever the string after it looks like, and [Substr] with a negative
// start has to measure the whole string before it can count back from the end.
//
// Timings came from an Apple M4 with other work running on it, so read them as
// ceilings. The allocation counts do not move.
package str
