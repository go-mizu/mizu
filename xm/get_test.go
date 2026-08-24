package xm_test

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/xm"
)

func TestGetOr(t *testing.T) {
	settings := map[string]int{"port": 9090}

	if got := xm.GetOr(settings, "port", 8080); got != 9090 {
		t.Errorf("GetOr gave %d, want the value in the map, 9090", got)
	}
	if got := xm.GetOr(settings, "timeout", 30); got != 30 {
		t.Errorf("GetOr gave %d, want the fallback, 30", got)
	}
}

// TestGetOrKeepsAZeroThatIsReallyThere is the whole reason the function is not
// a plain map read, so it is the case worth pinning hardest.
func TestGetOrKeepsAZeroThatIsReallyThere(t *testing.T) {
	settings := map[string]int{"port": 0}

	if got := xm.GetOr(settings, "port", 8080); got != 0 {
		t.Errorf("GetOr gave %d, want the 0 that is in the map", got)
	}
}

func TestGetOrOnANilMap(t *testing.T) {
	var settings map[string]string

	if got := xm.GetOr(settings, "host", "localhost"); got != "localhost" {
		t.Errorf("GetOr gave %q, want the fallback, localhost", got)
	}
}

func TestUpdate(t *testing.T) {
	counts := map[string]int{"go": 2}

	xm.Update(counts, "go", func(n int) int { return n + 1 })
	if counts["go"] != 3 {
		t.Errorf("Update left %d under go, want 3", counts["go"])
	}
}

// TestUpdateStartsFromTheZeroValue is what makes a counter work without a
// lookup first.
func TestUpdateStartsFromTheZeroValue(t *testing.T) {
	counts := map[string]int{}

	for _, word := range strings.Fields("go go mizu") {
		xm.Update(counts, word, func(n int) int { return n + 1 })
	}

	if counts["go"] != 2 || counts["mizu"] != 1 {
		t.Errorf("Update gave %v, want go at 2 and mizu at 1", counts)
	}
}

func TestUpdatePutsAKeyThereThatWasNot(t *testing.T) {
	m := map[string]string{}

	xm.Update(m, "greeting", func(s string) string { return s + "hello" })
	if _, there := m["greeting"]; !there {
		t.Error("Update did not put the key in the map")
	}
}

// TestUpdateWritesIntoTheMapItWasGiven says out loud that this one is the
// exception in a package that otherwise copies.
func TestUpdateWritesIntoTheMapItWasGiven(t *testing.T) {
	m := map[string]int{"a": 1}
	same := m

	xm.Update(m, "a", func(n int) int { return n * 10 })
	if same["a"] != 10 {
		t.Errorf("the other reference to the map sees %d, want 10", same["a"])
	}
}

func TestUpdateOnANamedMapType(t *testing.T) {
	type counters map[string]int

	c := counters{}
	xm.Update(c, "hits", func(n int) int { return n + 1 })

	if c["hits"] != 1 {
		t.Errorf("Update left %d under hits, want 1", c["hits"])
	}
}
