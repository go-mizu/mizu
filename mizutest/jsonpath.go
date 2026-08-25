package mizutest

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// The path syntax is the part of JSONPath that assertions in tests actually
// use, and no more:
//
//	$.data[0].id        a member, then an element, then a member
//	data[0].id          the leading $ is optional
//	$.data[-1]          a negative index counts from the end
//	$["content-type"]   a bracketed name, for a key a dot cannot spell
//	$                   the whole document
//
// Filters, wildcards, slices and recursive descent are not here. Every one of
// them turns an assertion about a value into an assertion about a set, and a
// test that says "some element somewhere matches" is a test that keeps passing
// after the thing it was written for is gone.

// step is one move through the document, either into a member or into an
// element.
type step struct {
	name  string
	index int
	byKey bool
}

func (s step) String() string {
	if s.byKey {
		if plainName(s.name) {
			return "." + s.name
		}
		return "[" + strconv.Quote(s.name) + "]"
	}
	return "[" + strconv.Itoa(s.index) + "]"
}

// evaluate walks doc along path and returns what is there.
//
// The error names the prefix of the path that worked and says what was found
// instead of what was wanted, since "no such member" without either of those is
// a hint rather than an answer.
func evaluate(doc any, path string) (any, error) {
	steps, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	at := doc
	var walked strings.Builder
	walked.WriteByte('$')

	for _, s := range steps {
		switch {
		case s.byKey:
			m, ok := at.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s%s: %s is %s, not an object", walked.String(), s, walked.String(), describe(at))
			}
			v, ok := m[s.name]
			if !ok {
				return nil, fmt.Errorf("%s%s: %s has no member %q, and has %s",
					walked.String(), s, walked.String(), s.name, members(m))
			}
			at = v
		default:
			list, ok := at.([]any)
			if !ok {
				return nil, fmt.Errorf("%s%s: %s is %s, not an array", walked.String(), s, walked.String(), describe(at))
			}
			i := s.index
			if i < 0 {
				i += len(list)
			}
			if i < 0 || i >= len(list) {
				return nil, fmt.Errorf("%s%s: %s has %d elements", walked.String(), s, walked.String(), len(list))
			}
			at = list[i]
		}
		walked.WriteString(s.String())
	}
	return at, nil
}

// parsePath splits a path into steps. It is strict, because a path with a typo
// in it that quietly means something else is a test that passes for the wrong
// reason.
func parsePath(path string) ([]step, error) {
	rest := path
	rest = strings.TrimPrefix(rest, "$")

	var steps []step
	for rest != "" {
		switch rest[0] {
		case '.':
			rest = rest[1:]
			i := strings.IndexAny(rest, ".[")
			if i < 0 {
				i = len(rest)
			}
			if i == 0 {
				return nil, fmt.Errorf("mizutest: %q: a dot with no name after it", path)
			}
			steps = append(steps, step{name: rest[:i], byKey: true})
			rest = rest[i:]

		case '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return nil, fmt.Errorf("mizutest: %q: a [ with no ] after it", path)
			}
			inner := rest[1:end]
			rest = rest[end+1:]

			if len(inner) >= 2 && (inner[0] == '"' || inner[0] == '\'') && inner[len(inner)-1] == inner[0] {
				steps = append(steps, step{name: inner[1 : len(inner)-1], byKey: true})
				continue
			}
			n, err := strconv.Atoi(inner)
			if err != nil {
				return nil, fmt.Errorf("mizutest: %q: %q is neither an index nor a quoted name", path, inner)
			}
			steps = append(steps, step{index: n})

		default:
			// A name at the very start, so that data[0] works as well as
			// $.data[0]. Anywhere else it is a missing dot.
			if len(steps) > 0 {
				return nil, fmt.Errorf("mizutest: %q: expected . or [ before %q", path, rest)
			}
			i := strings.IndexAny(rest, ".[")
			if i < 0 {
				i = len(rest)
			}
			steps = append(steps, step{name: rest[:i], byKey: true})
			rest = rest[i:]
		}
	}
	return steps, nil
}

// plainName reports whether a name can be written after a dot, which decides
// how a path is spelled back in an error message.
func plainName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

// describe says what a value is, in the words a failure message wants.
func describe(v any) string {
	switch v := v.(type) {
	case nil:
		return "null"
	case bool:
		return fmt.Sprintf("the boolean %v", v)
	case string:
		return fmt.Sprintf("the string %q", v)
	case json.Number:
		return "the number " + v.String()
	case float64:
		return fmt.Sprintf("the number %v", v)
	case map[string]any:
		if len(v) == 0 {
			return "an empty object"
		}
		return "an object " + members(v)
	case []any:
		return fmt.Sprintf("an array of %d", len(v))
	default:
		return fmt.Sprintf("%T", v)
	}
}

// members lists an object's keys, sorted and cut off before it stops being
// something a person reads.
func members(m map[string]any) string {
	if len(m) == 0 {
		return "no members"
	}
	keys := slices.Sorted(maps.Keys(m))
	if len(keys) == 1 {
		return "the member " + keys[0]
	}
	if len(keys) > 12 {
		return fmt.Sprintf("members %s and %d more", strings.Join(keys[:12], ", "), len(keys)-12)
	}
	return "members " + strings.Join(keys, ", ")
}
