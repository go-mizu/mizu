package log

import (
	"compress/gzip"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// fileClock is the time a file under test runs on, so that the names of retired
// files are something a test can write down, and a fortnight can pass without
// one waiting for it.
type fileClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fileClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fileClock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// openFile is a file in a directory of its own, with a clock the test moves.
func openFile(t *testing.T, o RotateOptions) (*File, *fileClock) {
	t.Helper()

	f, err := NewFile(filepath.Join(t.TempDir(), "app.log"), o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })

	c := &fileClock{t: time.Date(2026, 8, 24, 10, 44, 2, 113_000_000, time.UTC)}
	f.now = c.now
	return f, c
}

// read is what ended up in a file, compressed or not.
func read(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(name, ".gz") {
		return string(b)
	}

	zr, err := gzip.NewReader(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestFile(t *testing.T) {
	f, _ := openFile(t, RotateOptions{})
	log := slog.New(NewJSONHandler(f, JSONOptions{}))

	log.Info("one")
	log.Info("two")
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	got := read(t, f.path)
	if lines := strings.Count(got, "\n"); lines != 2 {
		t.Errorf("the file has %d lines, want 2:\n%s", lines, got)
	}
	if !strings.Contains(got, `"msg":"one"`) || !strings.Contains(got, `"msg":"two"`) {
		t.Errorf("the file holds:\n%s", got)
	}
}

// TestFileDefaults checks the zero options are the documented ones, since they
// are what a caller who has not thought about rotation gets.
func TestFileDefaults(t *testing.T) {
	f, _ := openFile(t, RotateOptions{})

	if f.max != 100<<20 {
		t.Errorf("max %d, want a hundred megabytes", f.max)
	}
	if f.age != 14*24*time.Hour {
		t.Errorf("age %v, want two weeks", f.age)
	}
	if f.keep != 10 {
		t.Errorf("keep %d, want 10", f.keep)
	}
}

// TestFileMegabytes is the unit the option is written in.
func TestFileMegabytes(t *testing.T) {
	f, _ := openFile(t, RotateOptions{MaxSizeMB: 7})
	if f.max != 7<<20 {
		t.Errorf("max %d, want seven megabytes", f.max)
	}
}

func TestFileRotatesBySize(t *testing.T) {
	f, _ := openFile(t, RotateOptions{})
	f.max = 64

	line := strings.Repeat("a", 39) + "\n"
	for range 3 {
		if _, err := f.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}

	old := f.retired()
	if len(old) != 2 {
		t.Fatalf("%d files were retired, want 2", len(old))
	}
	for _, r := range old {
		if got := read(t, r.name); got != line {
			t.Errorf("%s holds %q, want one line", filepath.Base(r.name), got)
		}
	}
	if got := read(t, f.path); got != line {
		t.Errorf("the open file holds %q, want the last line", got)
	}
}

// TestFileWholeRecords is the promise that keeps a JSON log readable: a record
// larger than the size goes into an empty file whole rather than in two halves.
func TestFileWholeRecords(t *testing.T) {
	f, _ := openFile(t, RotateOptions{})
	f.max = 8

	big := strings.Repeat("a", 100) + "\n"
	if _, err := f.Write([]byte(big)); err != nil {
		t.Fatal(err)
	}
	if got := read(t, f.path); got != big {
		t.Errorf("the file holds %d bytes, want the whole record", len(got))
	}
	if old := f.retired(); len(old) != 0 {
		t.Errorf("a write to an empty file retired %d files", len(old))
	}
}

// TestFileNames is what an operator sees in the directory. The names sort in
// the order the files were written, which is what makes ls enough.
func TestFileNames(t *testing.T) {
	f, c := openFile(t, RotateOptions{})

	for range 3 {
		if err := f.Rotate(); err != nil {
			t.Fatal(err)
		}
		c.add(time.Minute)
	}
	f.wg.Wait()

	var names []string
	for _, r := range f.retired() {
		names = append(names, filepath.Base(r.name))
	}
	want := []string{
		"app-2026-08-24T10-44-02.113.log",
		"app-2026-08-24T10-45-02.113.log",
		"app-2026-08-24T10-46-02.113.log",
	}
	for i, name := range want {
		if i >= len(names) || names[i] != name {
			t.Fatalf("the directory holds %v, want %v", names, want)
		}
	}
}

// TestFileNameCollision is two rotations in the same millisecond, which is what
// a program rotating on a signal in a loop does. The second one takes the next
// millisecond rather than the name of the first.
func TestFileNameCollision(t *testing.T) {
	f, _ := openFile(t, RotateOptions{})

	for range 2 {
		if err := f.Rotate(); err != nil {
			t.Fatal(err)
		}
	}
	f.wg.Wait()

	old := f.retired()
	if len(old) != 2 {
		t.Fatalf("%d files were retired, want 2", len(old))
	}
	if old[0].name == old[1].name {
		t.Errorf("both rotations wrote %s", old[0].name)
	}
}

func TestFileCompress(t *testing.T) {
	f, _ := openFile(t, RotateOptions{Compress: true})

	if _, err := f.Write([]byte("a line\n")); err != nil {
		t.Fatal(err)
	}
	if err := f.Rotate(); err != nil {
		t.Fatal(err)
	}
	f.wg.Wait()

	old := f.retired()
	if len(old) != 1 {
		t.Fatalf("%d files were retired, want 1", len(old))
	}
	if !strings.HasSuffix(old[0].name, ".log.gz") {
		t.Fatalf("the retired file is %s, want a .gz", filepath.Base(old[0].name))
	}
	if exists(strings.TrimSuffix(old[0].name, ".gz")) {
		t.Error("the file was compressed and the original left behind")
	}
	if got := read(t, old[0].name); got != "a line\n" {
		t.Errorf("the archive holds %q", got)
	}
}

// TestFileMaxFiles is the disk staying the size it was told to be.
func TestFileMaxFiles(t *testing.T) {
	f, c := openFile(t, RotateOptions{MaxFiles: 2})

	for range 5 {
		if err := f.Rotate(); err != nil {
			t.Fatal(err)
		}
		c.add(time.Minute)
		f.wg.Wait()
	}

	old := f.retired()
	if len(old) != 2 {
		t.Fatalf("%d files were kept, want 2", len(old))
	}
	if got := filepath.Base(old[1].name); got != "app-2026-08-24T10-48-02.113.log" {
		t.Errorf("the newest kept file is %s, want the last one written", got)
	}
}

// TestFileMaxAge is the other half of it, for a program that logs a little and
// runs for a year.
func TestFileMaxAge(t *testing.T) {
	f, c := openFile(t, RotateOptions{MaxAge: time.Hour})

	if err := f.Rotate(); err != nil {
		t.Fatal(err)
	}
	f.wg.Wait()
	if len(f.retired()) != 1 {
		t.Fatal("the first rotation kept nothing")
	}

	c.add(2 * time.Hour)
	if err := f.Rotate(); err != nil {
		t.Fatal(err)
	}
	f.wg.Wait()

	old := f.retired()
	if len(old) != 1 {
		t.Fatalf("%d files were kept, want the one inside the hour", len(old))
	}
	if got := filepath.Base(old[0].name); got != "app-2026-08-24T12-44-02.113.log" {
		t.Errorf("the kept file is %s, want the newer one", got)
	}
}

// TestFileLeavesStrangersAlone is what keeps this from deleting something an
// operator put in the directory by hand.
func TestFileLeavesStrangersAlone(t *testing.T) {
	f, _ := openFile(t, RotateOptions{MaxFiles: 1, MaxAge: time.Nanosecond})

	stranger := filepath.Join(filepath.Dir(f.path), "app-notes.log")
	if err := os.WriteFile(stranger, []byte("mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := f.Rotate(); err != nil {
		t.Fatal(err)
	}
	f.wg.Wait()

	if !exists(stranger) {
		t.Error("a file whose name is not a rotation was deleted")
	}
}

// TestFileReopen is a program restarting. The file it left is appended to, and
// its size is what decides the next rotation.
func TestFileReopen(t *testing.T) {
	f, _ := openFile(t, RotateOptions{})
	if _, err := f.Write([]byte("before\n")); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := NewFile(f.path, RotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()

	if again.size != int64(len("before\n")) {
		t.Errorf("the file reopened at size %d, want the size it was left at", again.size)
	}
	if _, err := again.Write([]byte("after\n")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, f.path); got != "before\nafter\n" {
		t.Errorf("the file holds %q, want both lines", got)
	}
}

// TestFileConcurrent is for the race detector, and for the promise that a
// record from one goroutine never lands inside a record from another.
func TestFileConcurrent(t *testing.T) {
	f, _ := openFile(t, RotateOptions{MaxFiles: 1000})
	f.max = 512

	line := strings.Repeat("a", 63) + "\n"
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				if _, err := f.Write([]byte(line)); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()
	f.wg.Wait()

	lines := strings.Count(read(t, f.path), "\n")
	for _, r := range f.retired() {
		got := read(t, r.name)
		lines += strings.Count(got, "\n")
		if len(got)%len(line) != 0 {
			t.Errorf("%s is %d bytes, which is not whole lines", filepath.Base(r.name), len(got))
		}
	}
	if lines != 400 {
		t.Errorf("%d lines were written, want 400", lines)
	}
}

func TestFileOpenFailures(t *testing.T) {
	dir := t.TempDir()
	notADir := filepath.Join(dir, "app.log")
	if err := os.WriteFile(notADir, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"a directory that is a file": filepath.Join(notADir, "deeper", "app.log"),
		"a path that is a directory": dir,
	}
	for name, path := range cases {
		if f, err := NewFile(path, RotateOptions{}); err == nil {
			f.Close()
			t.Errorf("%s: NewFile(%q) opened", name, path)
		}
	}
}

// TestFileRotateFailure is the directory an operator has taken the write bit
// off, which is the way rotation fails in practice. The rename fails, the error
// says so, and the program keeps logging to the file it already had.
func TestFileRotateFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows does not take the write bit off a directory")
	}
	if os.Geteuid() == 0 {
		t.Skip("root writes to a directory whatever its mode says")
	}

	f, _ := openFile(t, RotateOptions{})
	dir := filepath.Dir(f.path)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	if err := f.Rotate(); err == nil {
		t.Error("rotating into a directory that cannot be written worked")
	}
	if _, err := f.Write([]byte("still here\n")); err != nil {
		t.Errorf("the file was not reopened: %v", err)
	}
	if got := read(t, f.path); got != "still here\n" {
		t.Errorf("the file holds %q", got)
	}

	// A write that has to rotate first says what went wrong rather than
	// writing into a file that was supposed to have been retired.
	f.max = 4
	if _, err := f.Write([]byte("and again\n")); err == nil {
		t.Error("a write that could not rotate first worked")
	}
}

// TestFileTidyIsBestEffort is what happens when the tidying cannot run: the
// records are already written, and that is the part that matters.
func TestFileTidyIsBestEffort(t *testing.T) {
	dir := t.TempDir()
	gone := &File{path: filepath.Join(dir, "missing", "app.log")}
	if old := gone.retired(); old != nil {
		t.Errorf("a directory that is not there listed %v", old)
	}

	if err := compress(filepath.Join(dir, "not-there.log")); err == nil {
		t.Error("compressing a file that is not there worked")
	}

	// A directory opens and does not read, which is the shape of a compression
	// that fails halfway.
	if err := compress(dir); err == nil {
		t.Error("compressing a directory worked")
	}
	if exists(dir + ".gz") {
		t.Error("a failed compression left an archive behind")
	}

	// An archive that is already there is left where it is, since the one thing
	// worse than not compressing is overwriting.
	kept := filepath.Join(dir, "app-2026-08-24T10-44-02.113.log")
	if err := os.WriteFile(kept, []byte("plain\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kept+".gz", []byte("not really gzip\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := compress(kept); err == nil {
		t.Error("compressing over an archive that was there worked")
	}
	if got, _ := os.ReadFile(kept + ".gz"); string(got) != "not really gzip\n" {
		t.Error("the archive that was there was overwritten")
	}
}

func TestNewFileHandler(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	h, closer, err := NewFileHandler(path, RotateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	slog.New(h).Info("started", "port", 8080)
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	got := read(t, path)
	if !strings.Contains(got, `"msg":"started"`) || !strings.Contains(got, `"port":8080`) {
		t.Errorf("the file holds %s", got)
	}

	if _, _, err := NewFileHandler(t.TempDir(), RotateOptions{}); err == nil {
		t.Error("a handler on a path that is a directory opened")
	}
}
