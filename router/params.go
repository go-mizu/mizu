package router

import "iter"

// Params is the wildcard values of one match, in the order the wildcards appear
// in the pattern.
//
// It is a value rather than a map so that matching a request allocates nothing.
// A route with more wildcards than a match keeps on the stack is the one case
// that allocates, and eight is more than any route anybody writes.
//
// The zero Params has no parameters in it, which is what a route with no
// wildcards matches with.
type Params struct {
	names []string
	vals  values
}

// A Param is one wildcard and what it matched.
type Param struct {
	Name  string
	Value string
}

// Len is how many wildcards the pattern has.
func (p Params) Len() int { return len(p.names) }

// At is the wildcard at index i, counting from the left of the pattern. It
// panics when i is out of range, the way indexing a slice does.
func (p Params) At(i int) Param {
	if i < 0 || i >= len(p.names) {
		panic("router: no parameter at index " + itoa(i))
	}
	return Param{Name: p.names[i], Value: p.vals.at(i)}
}

// Get is the value of the named wildcard, or the empty string when the pattern
// has no wildcard by that name.
//
// A wildcard that matched an empty segment and a wildcard that is not there
// both read as the empty string. Use [Params.Lookup] when the difference
// matters.
func (p Params) Get(name string) string {
	v, _ := p.Lookup(name)
	return v
}

// Lookup is the value of the named wildcard, and whether the pattern has one.
func (p Params) Lookup(name string) (string, bool) {
	for i, n := range p.names {
		if n == name {
			return p.vals.at(i), true
		}
	}
	return "", false
}

// All is every wildcard and its value, left to right.
func (p Params) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for i, n := range p.names {
			if !yield(n, p.vals.at(i)) {
				return
			}
		}
	}
}

// itoa is strconv.Itoa for the small non-negative numbers a panic message
// carries, kept here so that a package on the request path does not link in
// number formatting for one message.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
