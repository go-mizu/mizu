package openlibrarydump

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilenameFromURL(t *testing.T) {
	t.Parallel()
	got := filenameFromURL("https://archive.org/download/ol_dump_2026-01-31/ol_dump_works_2026-01-31.txt.gz")
	want := "ol_dump_works_2026-01-31.txt.gz"
	if got != want {
		t.Fatalf("filename mismatch: got %q want %q", got, want)
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()
	cases := map[int64]string{
		10:                     "10 B",
		1024:                   "1.0 KiB",
		1024 * 1024:            "1.0 MiB",
		5 * 1024 * 1024:        "5.0 MiB",
		3 * 1024 * 1024 * 1024: "3.0 GiB",
	}
	for in, want := range cases {
		if got := FormatBytes(in); got != want {
			t.Fatalf("FormatBytes(%d) = %q want %q", in, got, want)
		}
	}
}

func TestEnsureReusableTargetFromLatestAlias(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	alias := filepath.Join(dir, "ol_dump_authors_latest.txt.gz")
	target := filepath.Join(dir, "ol_dump_authors_2026-01-31.txt.gz")
	if err := os.WriteFile(alias, []byte("abcd"), 0o644); err != nil {
		t.Fatalf("write alias: %v", err)
	}
	done, err := ensureReusableTarget(dir, DumpSpec{Name: "authors", SizeBytes: 4}, target)
	if err != nil {
		t.Fatalf("ensure reusable: %v", err)
	}
	if !done {
		t.Fatal("expected reusable target from alias")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target missing: %v", err)
	}
}
