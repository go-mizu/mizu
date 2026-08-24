package str

import "strings"

// Before returns what comes before the first search in s.
//
//	str.Before("user@example.com", "@")   // user
//
// When search is not in s at all, the whole of s comes back. That is the one
// thing to know about this family of six, because it means a result that looks
// fine can be the input unchanged. Code that has to tell the difference should
// use [strings.Cut], which reports whether it found anything.
//
// An empty search is not found, so s comes back.
func Before(s, search string) string {
	if search == "" {
		return s
	}
	if i := strings.Index(s, search); i >= 0 {
		return s[:i]
	}
	return s
}

// BeforeLast returns what comes before the last search in s.
//
//	str.BeforeLast("a/b/c.txt", "/")   // a/b
//
// When search is not in s, the whole of s comes back. See [Before].
func BeforeLast(s, search string) string {
	if search == "" {
		return s
	}
	if i := strings.LastIndex(s, search); i >= 0 {
		return s[:i]
	}
	return s
}

// After returns what comes after the first search in s.
//
//	str.After("user@example.com", "@")   // example.com
//
// When search is not in s, the whole of s comes back. See [Before].
func After(s, search string) string {
	if search == "" {
		return s
	}
	if i := strings.Index(s, search); i >= 0 {
		return s[i+len(search):]
	}
	return s
}

// AfterLast returns what comes after the last search in s.
//
//	str.AfterLast("a/b/c.txt", "/")   // c.txt
//	str.AfterLast("archive.tar.gz", ".")   // gz
//
// When search is not in s, the whole of s comes back. See [Before].
func AfterLast(s, search string) string {
	if search == "" {
		return s
	}
	if i := strings.LastIndex(s, search); i >= 0 {
		return s[i+len(search):]
	}
	return s
}

// Between returns what lies between the first from and the last to.
//
//	str.Between("[a] and [b]", "[", "]")   // a] and [b
//
// Reaching for the last to is what makes this the one for a value that has the
// closing mark inside it. [BetweenFirst] is the one that stops at the first.
//
// When either mark is missing, the whole of s comes back, the same as the rest
// of this family.
func Between(s, from, to string) string {
	return BeforeLast(After(s, from), to)
}

// BetweenFirst returns what lies between the first from and the first to after
// it.
//
//	str.BetweenFirst("[a] and [b]", "[", "]")   // a
//
// When either mark is missing, the whole of s comes back. See [Before].
func BetweenFirst(s, from, to string) string {
	return Before(After(s, from), to)
}
