package log

import (
	"compress/gzip"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// RotateOptions says when a log file is renamed out of the way and what happens
// to the ones already renamed. The zero value keeps a hundred megabytes per
// file, ten files, and two weeks of them, without compressing.
type RotateOptions struct {
	// MaxSizeMB is how large the file gets before the next record starts a new
	// one, and defaults to 100.
	MaxSizeMB int

	// MaxAge is how long a renamed file is kept, and defaults to two weeks.
	MaxAge time.Duration

	// MaxFiles is how many renamed files are kept, whatever their age, and
	// defaults to 10.
	MaxFiles int

	// Compress gzips a file once it has been renamed. A log compresses to about
	// a tenth of itself, so this is most of the disk back for some work on a
	// goroutine that nothing waits for.
	Compress bool
}

// NewFileHandler writes JSON records to a file that rotates, and returns the
// handler and the file to close on the way out.
//
//	h, closer, err := log.NewFileHandler("/var/log/blog/app.log", log.RotateOptions{Compress: true})
//	if err != nil {
//		return err
//	}
//	defer closer.Close()
//	slog.SetDefault(slog.New(h))
//
// The handler is [NewJSONHandler] with its defaults. To choose the format or
// the level, open the file with [NewFile] and build the handler around it, or
// let [New] read both out of a configuration.
func NewFileHandler(path string, o RotateOptions) (slog.Handler, io.Closer, error) {
	f, err := NewFile(path, o)
	if err != nil {
		return nil, nil, err
	}
	return NewJSONHandler(f, JSONOptions{}), f, nil
}

// NewFile opens path for appending and rotates it as it grows.
//
// The file and the directories above it are created if they are not there. It
// opens now rather than at the first record, so a path a program cannot write
// is a failure to start rather than a surprise at three in the morning.
//
// A renamed file keeps the name it had with the time it was retired in the
// middle of it, so app.log becomes app-2026-08-24T10-44-02.113.log, and the
// names sort in the order they were written. The time is UTC, so they keep
// sorting when the clock changes for summer time.
func NewFile(path string, o RotateOptions) (*File, error) {
	f := &File{
		path:     path,
		ext:      filepath.Ext(path),
		max:      int64(o.MaxSizeMB) << 20,
		age:      o.MaxAge,
		keep:     o.MaxFiles,
		compress: o.Compress,
		now:      time.Now,
	}
	f.prefix = strings.TrimSuffix(path, f.ext)
	if o.MaxSizeMB <= 0 {
		f.max = 100 << 20
	}
	if o.MaxAge <= 0 {
		f.age = 14 * 24 * time.Hour
	}
	if o.MaxFiles <= 0 {
		f.keep = 10
	}
	if err := f.open(); err != nil {
		return nil, err
	}
	return f, nil
}

// File is a log file that renames itself out of the way when it gets large. It
// is an [io.Writer], so it goes under any handler, and several handlers can
// share one.
//
// Records are written straight through without buffering. Nothing is lost when
// a program is killed, and there is nothing to flush.
type File struct {
	path     string
	prefix   string // path without its extension
	ext      string
	max      int64
	age      time.Duration
	keep     int
	compress bool

	// now is the clock, which a test replaces to make the names of retired
	// files something it can write down.
	now func() time.Time

	// wg is the compressing and deleting that is happening on its own
	// goroutine, which Close waits for.
	wg sync.WaitGroup

	mu   sync.Mutex
	f    *os.File
	size int64
}

// Write appends p, rotating first when p would take the file past its size.
//
// A record is never split across two files, so a file goes over its size rather
// than a line being cut in half, and a single record larger than the size is
// written whole into an empty file.
func (f *File) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.size > 0 && f.size+int64(len(p)) > f.max {
		if err := f.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := f.f.Write(p)
	f.size += int64(n)
	return n, err
}

// Rotate renames the file out of the way and starts a new one, whatever its
// size. It is what a program does when it is told to, on a signal or on a
// schedule, rather than waiting for the size to decide.
func (f *File) Rotate() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rotate()
}

// Close closes the file and waits for the compressing and deleting that
// rotating started.
func (f *File) Close() error {
	f.mu.Lock()
	err := f.f.Close()
	f.mu.Unlock()

	// Outside the lock: the goroutine does not take it, and a caller closing
	// while another one writes should not deadlock on the difference.
	f.wg.Wait()
	return err
}

// rotate is Rotate with the lock already held.
func (f *File) rotate() error {
	// Whatever Close says, the writes it was covering have gone to the file
	// system, and the name is about to move. It is reported with the rest.
	closed := f.f.Close()

	if err := os.Rename(f.path, f.free()); err != nil {
		// Nothing moved, so there is nothing to tidy. Opening the same path
		// again keeps the program logging somewhere while the operator works
		// out what happened to the directory.
		return errors.Join(closed, err, f.open())
	}

	err := errors.Join(closed, f.open())
	f.tidy()
	return err
}

// open opens the path for appending, creating the directories above it, and
// picks up the size of whatever was already there. A failure leaves the file
// as it was, so a write after one says the file is closed rather than finding
// nothing there at all.
func (f *File) open() error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(f.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	f.f, f.size = file, 0
	if fi, err := file.Stat(); err == nil {
		f.size = fi.Size()
	}
	return nil
}

// tidy compresses and deletes on its own goroutine. Compressing a hundred
// megabytes takes longer than a request should wait to log a line, and none of
// it is work the record depends on. Failures leave the files as they are: a
// full disk is not a reason to lose the log as well.
func (f *File) tidy() {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		for _, r := range f.retired() {
			if f.compress && !strings.HasSuffix(r.name, ".gz") {
				compress(r.name)
			}
		}
		f.prune()
	}()
}

// prune deletes the retired files that are too old or too many.
func (f *File) prune() {
	files := f.retired()
	cutoff := f.now().Add(-f.age)
	for i, r := range files {
		if r.when.Before(cutoff) || len(files)-i > f.keep {
			os.Remove(r.name)
		}
	}
}

// stamp is the time in the name of a retired file. It sorts in the order the
// files were written and holds nothing a file system objects to.
const stamp = "2006-01-02T15-04-05.000"

// free is a name for the file about to be retired that nothing else has. Two
// rotations in the same millisecond are rare and a lost file is not, so the
// second one takes the next millisecond rather than the same name.
func (f *File) free() string {
	for t := f.now().UTC(); ; t = t.Add(time.Millisecond) {
		name := f.prefix + "-" + t.Format(stamp) + f.ext
		if !exists(name) && !exists(name+".gz") {
			return name
		}
	}
}

// rotated is a file this one has already retired, and when it was.
type rotated struct {
	name string
	when time.Time
}

// retired is the files this one has already renamed out of the way, oldest
// first.
//
// It reads the directory rather than remembering, so a program at startup
// still tidies up after the one that ran before it. A name
// that does not parse is not one of ours and is left alone, which is what keeps
// this from deleting a file somebody put in the directory by hand.
func (f *File) retired() []rotated {
	dir := filepath.Dir(f.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	base := filepath.Base(f.prefix) + "-"
	var out []rotated
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), base) {
			continue
		}
		rest := strings.TrimSuffix(strings.TrimPrefix(e.Name(), base), ".gz")
		when, err := time.Parse(stamp, strings.TrimSuffix(rest, f.ext))
		if err != nil {
			continue
		}
		out = append(out, rotated{filepath.Join(dir, e.Name()), when})
	}
	slices.SortFunc(out, func(a, b rotated) int { return a.when.Compare(b.when) })
	return out
}

// compress writes name.gz and removes name.
func compress(name string) error {
	in, err := os.Open(name)
	if err != nil {
		return err
	}
	defer in.Close()

	// Exclusively, so that two of these racing over one file cannot both write
	// the same archive.
	out, err := os.OpenFile(name+".gz", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}

	zw := gzip.NewWriter(out)
	_, err = io.Copy(zw, in)
	err = errors.Join(err, zw.Close(), out.Close())
	if err != nil {
		os.Remove(name + ".gz")
		return err
	}

	// Windows will not remove a file that is still open, and this one is about
	// to be removed.
	in.Close()
	return os.Remove(name)
}

func exists(name string) bool {
	_, err := os.Lstat(name)
	return err == nil
}
