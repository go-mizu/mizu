package bench

import (
	"strings"
	"testing"
)

func TestReadFindsACorpus(t *testing.T) {
	got := Read("hashes.txt")

	if !strings.Contains(string(got), "argon2id") {
		t.Errorf("hashes.txt does not hold what it says:\n%s", got)
	}
}

func TestReadSaysWhenThereIsNoSuchCorpus(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("reading a corpus that does not exist did not panic")
		}
	}()

	Read("nothing-like-this.txt")
}

func TestValuesReadsTheHashes(t *testing.T) {
	got := Values("hashes.txt")

	for _, name := range []string{"argon2id", "bcrypt"} {
		if !strings.HasPrefix(got[name], "$") {
			t.Errorf("%s is %q, want an encoded hash", name, got[name])
		}
	}
	if len(got) != 2 {
		t.Errorf("hashes.txt holds %v, want the two hashes and nothing from the header", got)
	}
}

// TestValuesIgnoresTheHeader is the part that matters, because the header is
// where the warning about changing the file lives and it is most of the file.
func TestValuesIgnoresTheHeader(t *testing.T) {
	for name := range Values("hashes.txt") {
		if strings.HasPrefix(name, "#") {
			t.Errorf("%q came out of the header", name)
		}
	}
}

// TestEveryCorpusIsListed is the same rule benchrun lint applies, run here so
// that a corpus added without a line in the README fails an ordinary go test
// rather than waiting for the command to be run.
func TestEveryCorpusIsListed(t *testing.T) {
	names, err := corpus.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	index := string(Read("README.md"))

	for _, e := range names {
		if e.Name() == "README.md" {
			continue
		}
		if !strings.Contains(index, e.Name()) {
			t.Errorf("testdata/%s is not listed in testdata/README.md", e.Name())
		}
	}
}
