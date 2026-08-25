package web

import (
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu/errs"
)

// File sends a file from disk.
//
//	return c.File("/var/lib/app/reports/2026-q1.pdf")
//
// Range requests, If-Modified-Since and If-None-Match are all answered, so a
// download resumes and a browser that already has the file gets a 304. That is
// [net/http.ServeContent] doing the work, and it means the status is whatever
// the conditional and range rules decided rather than whatever [Ctx.Status] was
// given. The content type comes from the extension, falling back to the first
// bytes.
//
// A path with a .. element in it is refused. A path built from the request is
// the reason that rule exists, and refusing here is not the whole answer:
// [Ctx.FileFS] under [os.Root.FS] is, because it holds a symlink to the
// filesystem's answer as well as a name.
//
// A missing file is [github.com/go-mizu/mizu/errs.NotFound] and a directory is
// too, because there is no listing here and the client asked for a file. The
// error carries the path for the log and keeps it out of the message.
func (c *Ctx) File(name string) error {
	c.live("File")

	f, err := openPath(name)
	if err != nil {
		return err
	}
	defer f.Close()

	body, info, err := statFile(f)
	if err != nil {
		return err
	}

	http.ServeContent(c.res, c.r, info.Name(), info.ModTime(), body)
	return nil
}

// FileFS sends a file out of an fs.FS.
//
//	//go:embed assets
//	var assets embed.FS
//
//	func serve(c *web.Ctx) error {
//		return c.FileFS(assets, "assets/"+c.Param("path"))
//	}
//
// This is the one to reach for when the name came from the request. An fs.FS
// path cannot climb out of its own tree, so a client that asks for ../../etc
// gets an error rather than a file, and under [os.Root.FS] a symlink pointing
// out of the tree does not work either.
//
//	root, err := os.OpenRoot("/var/www")
//	...
//	return c.FileFS(root.FS(), c.Param("path"))
//
// A leading slash is taken off, since an fs.FS path is always relative to the
// root of the fs and a client sends the leading one anyway. Anything else that
// [io/fs.ValidPath] refuses is an error.
//
// The file has to be seekable, which is what serving a range means and what
// [net/http.ServeFileFS] asks for as well. [embed.FS] and [os.DirFS] both are.
func (c *Ctx) FileFS(fsys fs.FS, name string) error {
	c.live("FileFS")

	name = strings.TrimPrefix(name, "/")
	if !fs.ValidPath(name) {
		return errs.New(errs.Invalid, "file.path", "web: not a path this serves")
	}

	f, err := fsys.Open(name)
	if err != nil {
		return openError(err)
	}
	defer f.Close()

	body, info, err := statFile(f)
	if err != nil {
		return err
	}

	http.ServeContent(c.res, c.r, info.Name(), info.ModTime(), body)
	return nil
}

// Download sends a reader as a file the browser saves rather than shows.
//
//	return c.Download("invoice-"+id+".csv", rows)
//
// name is what the file is called on the way down, and everything that makes it
// a path is taken out of it first, so a name a user chose cannot walk a download
// manager into another directory. A name that survives none of that becomes
// "download". The content type comes from the extension, or is
// application/octet-stream when nothing recognises it.
//
// Nothing here closes r, since this did not open it, and nothing here answers a
// range request, since a reader cannot seek. [Ctx.Attachment] is the one that
// can, for a file that is already on disk.
func (c *Ctx) Download(name string, r io.Reader) error {
	c.live("Download")

	c.res.Header().Set("Content-Disposition", disposition(name))
	c.head(typeOf(name))
	_, err := io.Copy(c.res, r)
	return err
}

// Attachment sends a file from disk as a download.
//
//	return c.Attachment("/var/lib/app/reports/q1.pdf", "Q1 report.pdf")
//
// It is [Ctx.File] and [Ctx.Download] together: the path is opened and served
// with ranges and conditional requests the way File does it, and the response
// says to save it under name the way Download does.
//
// An empty name means the file keeps the one it has on disk. Where the two
// differ it is name that decides the content type, so a report written to q1.dat
// and sent as q1.csv arrives as a spreadsheet.
func (c *Ctx) Attachment(path, name string) error {
	c.live("Attachment")

	f, err := openPath(path)
	if err != nil {
		return err
	}
	defer f.Close()

	body, info, err := statFile(f)
	if err != nil {
		return err
	}
	if name == "" {
		name = info.Name()
	}

	c.res.Header().Set("Content-Disposition", disposition(name))
	http.ServeContent(c.res, c.r, name, info.ModTime(), body)
	return nil
}

// openPath opens a path from the filesystem, or says why it will not.
func openPath(name string) (*os.File, error) {
	if hasDotDot(name) {
		return nil, errs.New(errs.Invalid, "file.path", "web: not a path this serves")
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, openError(err)
	}
	return f, nil
}

// statFile is what serving an open file needs, or the reason it cannot be
// served.
//
// Both openers come through here, so a directory, an unreadable handle and a
// file that will not seek mean the same thing whether the file came off the disk
// or out of an fs.FS.
func statFile(f fs.File) (io.ReadSeeker, fs.FileInfo, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, nil, errs.Wrap(err, errs.Internal, "file.unreadable", "web: cannot read the file")
	}
	if info.IsDir() {
		return nil, nil, errs.New(errs.NotFound, "file.is_directory", "web: no such file")
	}

	body, ok := f.(io.ReadSeeker)
	if !ok {
		return nil, nil, errs.New(errs.Internal, "file.unseekable", "web: cannot serve a file that does not seek")
	}
	return body, info, nil
}

// openError is what a failed open means to a client.
//
// A file that is not there is the client's answer and anything else is the
// server's problem: a permission the deploy got wrong is a 500 that somebody has
// to fix, not a 404 that looks like nothing happened. Either way the original
// error is still under this one, so the log has the path and the response does
// not.
func openError(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return errs.Wrap(err, errs.NotFound, "file.not_found", "web: no such file")
	}
	return errs.Wrap(err, errs.Internal, "file.unreadable", "web: cannot read the file")
}

// hasDotDot reports whether any segment of the path is "..".
//
// Both separators count, because a Windows path can use either and a check that
// only knew about one would pass the other straight through. The test is on the
// path as given rather than on a cleaned one: cleaning turns /srv/../etc into
// /etc, which has no .. left in it and is still not the directory the caller
// meant to serve out of.
func hasDotDot(name string) bool {
	if !strings.Contains(name, "..") {
		return false
	}
	for seg := range strings.FieldsFuncSeq(name, isSeparator) {
		if seg == ".." {
			return true
		}
	}
	return false
}

func isSeparator(r rune) bool { return r == '/' || r == '\\' }

// disposition is the Content-Disposition header for a download called name.
//
// The name goes through the same cleaning a client's upload filename does, so a
// path, a control character or a newline that would have split the header in
// half does not reach the header. What is left is formatted rather than
// concatenated, which is what puts a space or a non-ASCII letter in the encoding
// RFC 6266 asks for.
//
// There is no fallback for [mime.FormatMediaType] answering with nothing,
// because a cleaned name is always something it can format: the attribute is a
// constant, the value is printable, and a value it cannot write as a token it
// percent-encodes instead. A test holds that to be true over the names worth
// worrying about rather than a branch here defending against it.
func disposition(name string) string {
	name = cleanName(name)
	if name == "" {
		name = "download"
	}
	return mime.FormatMediaType("attachment", map[string]string{"filename": name})
}

// typeOf is the content type a filename implies.
//
// The table is the standard library's, which reads the system's list on top of
// its own built-in one, so an extension a machine knows about is answered even
// though nothing here has heard of it. An extension nothing recognises is
// application/octet-stream, which is the honest answer and also the one that
// stops a browser from guessing and rendering.
func typeOf(name string) string {
	if t := mime.TypeByExtension(filepath.Ext(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}
