package bench

import (
	"embed"
	"strings"
)

// The corpora are embedded rather than opened so that a benchmark does not
// touch the disk in the middle of a measurement and does not care which
// directory it was started from. README.md comes along with them because
// benchrun lint reads it to check that every file here is accounted for.
//
//go:embed testdata
var corpus embed.FS

// Read returns the corpus file called name, which is a path under testdata
// without the directory: bench.Read("hashes.txt").
//
// It panics when there is no such file, because a benchmark reads its corpus
// while it is being set up and a missing one is a typo rather than a condition
// to handle.
func Read(name string) []byte {
	b, err := corpus.ReadFile("testdata/" + name)
	if err != nil {
		panic("bench: " + err.Error())
	}
	return b
}

// Values returns the named values in a corpus file, which are one to a line as
// a name, whitespace, and the value. Blank lines and lines starting with a hash
// are skipped, which is where the header saying what changing the file costs
// lives.
func Values(name string) map[string]string {
	out := map[string]string{}
	for line := range strings.Lines(string(Read(name))) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "\t")
		if !ok {
			key, value, _ = strings.Cut(line, " ")
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}
