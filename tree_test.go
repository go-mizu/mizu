package mizu

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A compiled binary is easy to commit by accident and hard to notice
// afterwards. go build with no -o leaves one next to the source, named after
// the directory, and git add -A picks it up along with the change that was
// meant. Nobody reads a diff that says "Binary files differ".
//
// The root .gitignore has a rule per place a build lands, and this is the test
// that says so. A rule holds until somebody adds a command in a new directory,
// and then it is a rule with a hole in it.

// magic is the first bytes of the executable formats a Go build produces on the
// platforms anybody develops this on.
//
// The Windows one is not in the list. Its signature is the two bytes MZ, which
// is also how a fair number of the error codes in this repository start, and a
// .exe is covered by .gitignore already.
var magic = map[string][]byte{
	"an ELF executable":        {0x7f, 'E', 'L', 'F'},
	"a Mach-O executable":      {0xcf, 0xfa, 0xed, 0xfe},
	"a 32 bit Mach-O":          {0xce, 0xfa, 0xed, 0xfe},
	"a universal binary":       {0xca, 0xfe, 0xba, 0xbe},
	"a WebAssembly module":     {0x00, 'a', 's', 'm'},
	"a static library archive": {'!', '<', 'a', 'r', 'c', 'h', '>'},
}

func TestNothingInTheTreeIsACompiledBinary(t *testing.T) {
	out, err := exec.Command("git", "ls-files", "-z").Output()
	if err != nil {
		t.Skipf("git ls-files did not run, so there is no file list to check: %v", err)
	}

	for name := range strings.SplitSeq(strings.TrimRight(string(out), "\x00"), "\x00") {
		if name == "" {
			continue
		}

		// testdata is where fixtures live, and a fixture is allowed to be
		// whatever the code it is a fixture for reads.
		if slices.Contains(strings.Split(filepath.ToSlash(name), "/"), "testdata") {
			continue
		}

		if what := format(t, name); what != "" {
			t.Errorf("%s is %s and it is committed, so delete it and give .gitignore a rule for it", name, what)
		}
	}
}

// format names the executable format the file starts with, or the empty string
// for anything else, which is almost every file.
func format(t *testing.T, name string) string {
	t.Helper()

	f, err := os.Open(name)
	if err != nil {
		// A file git knows about and the filesystem does not is a deletion
		// somebody has staged, and that is not what this is looking for.
		return ""
	}
	defer f.Close()

	var head [8]byte
	n, err := io.ReadFull(f, head[:])
	if err != nil && err != io.ErrUnexpectedEOF {
		return ""
	}

	for what, prefix := range magic {
		if bytes.HasPrefix(head[:n], prefix) {
			return what
		}
	}
	return ""
}
