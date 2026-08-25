package web

import (
	"encoding/binary"
	"errors"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"reflect"
	"strings"

	"github.com/go-mizu/mizu/errs"

	// The three formats the standard library decodes, registered so that
	// [Upload.Image] answers for the files people actually upload. A format
	// nobody registered comes back as an error saying so.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// An Upload is one file a form sent.
//
// It is what a *Upload or a []*Upload field binds to, and it is a handle rather
// than the bytes: a file under the in-memory limit is held in memory and a
// larger one is on disk in a temporary file that net/http removes when the
// request is over. Anything that has to outlive the request has to be copied
// somewhere first.
//
//	type avatar struct {
//		Image *web.Upload `form:"image"`
//	}
//
// Storing one is [github.com/go-mizu/mizu/storage]'s job and arrives with it.
type Upload struct {
	// Filename is what the client called the file, with any directory in front
	// of it removed.
	//
	// It is a label to show back to the person who uploaded it. It is not a
	// name to write to disk with: it comes from the client, two clients can
	// send the same one, and a client that sends nothing sensible leaves it
	// empty.
	Filename string

	// Size is the file's length in bytes.
	Size int64

	// MIME is what the file's first bytes say it is.
	//
	// It is sniffed rather than taken from the part's own Content-Type header,
	// which is whatever the client felt like sending. A file whose bytes say
	// nothing recognisable is application/octet-stream.
	MIME string

	// Header is the part's headers, for the rare case that needs one.
	//
	// The Content-Type in it is the client's claim about the file. MIME is what
	// the bytes say, and where the two disagree it is the client that is wrong.
	Header textproto.MIMEHeader

	fh *multipart.FileHeader
}

// Open opens the file for reading.
//
// The caller closes it. Opening the same upload twice gives two readers, each
// starting at the beginning, which is what a handler that sniffs a file and
// then stores it needs.
func (u *Upload) Open() (multipart.File, error) {
	return u.fh.Open()
}

// Bytes is the whole file.
//
// It holds the file in memory, so it is for the small ones: an avatar, a CSV
// somebody is importing, a signature to check. A file that might be large is
// read through [Upload.Open] and copied where it is going.
func (u *Upload) Bytes() ([]byte, error) {
	f, err := u.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Image is the size and colour model of an image, without decoding it.
//
// Reading the header rather than the picture is the point. A decompression
// bomb is a small file that says it is forty thousand pixels square, and this
// answers that question for the cost of the first few hundred bytes.
//
// GIF, JPEG and PNG are read out of the box. Another format is read once the
// program imports a decoder for it, such as golang.org/x/image/webp, in the
// usual way with a blank import.
func (u *Upload) Image() (image.Config, error) {
	f, err := u.Open()
	if err != nil {
		return image.Config{}, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return image.Config{}, errs.Wrap(err, errs.Invalid, "upload.image",
			"That file is not an image in a format this server reads.")
	}
	return cfg, nil
}

var (
	uploadType  = reflect.TypeFor[*Upload]()
	uploadValue = reflect.TypeFor[Upload]()
)

// newUpload is one part of a multipart form as an Upload.
//
// The sniff happens here rather than the first time somebody reads MIME, so
// that a field is a field rather than a call that might fail. It reads the
// first 512 bytes, which for a file under the in-memory limit is a slice of
// something already in memory and for a larger one is one read of a temporary
// file.
func newUpload(fh *multipart.FileHeader) *Upload {
	return &Upload{
		Filename: cleanName(fh.Filename),
		Size:     fh.Size,
		MIME:     sniffFile(fh),
		Header:   fh.Header,
		fh:       fh,
	}
}

// sniffFile is what the first bytes of a file say it is.
//
// A file whose bytes cannot be read says nothing, which is a stream of bytes.
// Binding is not the place to report that: the handler is about to read the
// file properly and will get the real error with the real reason on it, where
// swallowing it here and reporting a bad field would say the client sent
// something wrong when it did not.
func sniffFile(fh *multipart.FileHeader) string {
	f, err := fh.Open()
	if err != nil {
		return unknownType
	}
	defer f.Close()
	return sniffReader(f)
}

// unknownType is what a file whose bytes say nothing recognisable is, and what
// a file whose bytes could not be read is.
const unknownType = "application/octet-stream"

// sniffReader is [sniff] over the first 512 bytes of a reader.
//
// A file shorter than that is not a short read to complain about, it is a short
// file, which is why the two ends of one are not errors here.
func sniffReader(r io.Reader) string {
	var head [512]byte
	n, err := io.ReadFull(r, head[:])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return unknownType
	}
	return sniff(head[:n])
}

// cleanName is a client's filename with everything that makes it a path taken
// out of it.
//
// A client can send anything here, including ../../etc/passwd, a Windows path
// with backslashes in it, and a name with a newline in it that would split a
// log line in half. What is left is the last segment with the control
// characters removed, and a name that was only a path is left empty rather than
// turned into something that looks like a file.
func cleanName(s string) string {
	if i := strings.LastIndexAny(s, `/\`); i >= 0 {
		s = s[i+1:]
	}

	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)

	s = strings.TrimSpace(s)
	if s == "." || s == ".." {
		return ""
	}
	return s
}

// sniff is what a file's first bytes say it is.
//
// [net/http.DetectContentType] is the WHATWG sniffing algorithm and it covers
// nearly everything, including webp, ogg and wasm, which it did not when this
// was first written down. What it does not cover is the ISO base media file
// format, where the type is a four letter brand inside the first box rather
// than a signature at a fixed offset, and that is what an iPhone photo is.
func sniff(head []byte) string {
	if kind := sniffISO(head); kind != "" {
		return kind
	}
	return http.DetectContentType(head)
}

// isoBrands are the ISO base media file format brands worth telling apart.
//
// A file carries a major brand and then a list of the brands it is also
// compatible with, so a HEIC photo says heic first and mif1 after it. Anything
// not in here is left to the sniffer, which calls it a stream of bytes.
var isoBrands = map[string]string{
	"avif": "image/avif",
	"avis": "image/avif",
	"heic": "image/heic",
	"heim": "image/heic",
	"heis": "image/heic",
	"heix": "image/heic",
	"hevc": "image/heic",
	"hevx": "image/heic",
	"mif1": "image/heif",
	"msf1": "image/heif",
	"qt  ": "video/quicktime",
	"avc1": "video/mp4",
	"dash": "video/mp4",
	"iso2": "video/mp4",
	"isom": "video/mp4",
	"mp41": "video/mp4",
	"mp42": "video/mp4",
}

// sniffISO reads the brand out of an ftyp box, and answers with nothing when
// the bytes are not one.
//
// The box is a four byte length, the word ftyp, the major brand, a four byte
// version nobody reads, and then the compatible brands until the box ends. The
// length is the file's claim about itself, so it is clamped to what actually
// arrived before anything indexes with it.
func sniffISO(head []byte) string {
	if len(head) < 12 || string(head[4:8]) != "ftyp" {
		return ""
	}
	if kind := isoBrands[string(head[8:12])]; kind != "" {
		return kind
	}

	size := int(binary.BigEndian.Uint32(head[:4]))
	if size > len(head) {
		size = len(head)
	}
	for i := 16; i+4 <= size; i += 4 {
		if kind := isoBrands[string(head[i:i+4])]; kind != "" {
			return kind
		}
	}
	return ""
}
