package web

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

// part is one file in a multipart form.
type part struct {
	field string
	name  string
	kind  string // what the client claims the file is, which nothing trusts
	body  []byte
}

// uploadOf is a request carrying a multipart body with those fields and those
// files in it.
func uploadOf(t *testing.T, values url.Values, parts ...part) *http.Request {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	for key, all := range values {
		for _, v := range all {
			if err := w.WriteField(key, v); err != nil {
				t.Fatal(err)
			}
		}
	}

	for _, p := range parts {
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{
			`form-data; name="` + p.field + `"; filename="` + p.name + `"`,
		}
		if p.kind != "" {
			h["Content-Type"] = []string{p.kind}
		}

		part, err := w.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(p.body); err != nil {
			t.Fatal(err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/", bytes.NewReader(body.Bytes()))
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}

// pngOf is a real PNG of that size, so the image tests read a header something
// wrote rather than one a test made up.
func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})

	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestAFileFieldBindsTheFileThatArrived(t *testing.T) {
	type avatar struct {
		Image *Upload `form:"image"`
	}

	body := pngOf(t, 4, 7)
	in, err := bind[avatar](t, uploadOf(t, nil, part{field: "image", name: "me.png", body: body}))
	if err != nil {
		t.Fatal(err)
	}

	if in.Image == nil {
		t.Fatal("the file did not bind")
	}
	if in.Image.Filename != "me.png" {
		t.Errorf("the filename is %q, want me.png", in.Image.Filename)
	}
	if in.Image.Size != int64(len(body)) {
		t.Errorf("the size is %d, want %d", in.Image.Size, len(body))
	}
	if in.Image.MIME != "image/png" {
		t.Errorf("the type is %q, want image/png", in.Image.MIME)
	}
	if got := in.Image.Header.Get("Content-Disposition"); got == "" {
		t.Error("the part's headers did not come with it")
	}
}

func TestASliceOfUploadsTakesEveryFileSentUnderTheName(t *testing.T) {
	type gallery struct {
		Photos []*Upload `form:"photos"`
	}

	in, err := bind[gallery](t, uploadOf(t, nil,
		part{field: "photos", name: "one.png", body: pngOf(t, 1, 1)},
		part{field: "photos", name: "two.png", body: pngOf(t, 2, 2)},
	))
	if err != nil {
		t.Fatal(err)
	}

	if len(in.Photos) != 2 {
		t.Fatalf("%d files bound, want two", len(in.Photos))
	}
	if in.Photos[0].Filename != "one.png" || in.Photos[1].Filename != "two.png" {
		t.Errorf("the files arrived as %q and %q", in.Photos[0].Filename, in.Photos[1].Filename)
	}
}

func TestASingleFileFieldTakesTheFirstOfSeveral(t *testing.T) {
	type avatar struct {
		Image *Upload `form:"image"`
	}

	in, err := bind[avatar](t, uploadOf(t, nil,
		part{field: "image", name: "first.png", body: pngOf(t, 1, 1)},
		part{field: "image", name: "second.png", body: pngOf(t, 1, 1)},
	))
	if err != nil {
		t.Fatal(err)
	}
	if in.Image.Filename != "first.png" {
		t.Errorf("the file is %q, want first.png", in.Image.Filename)
	}
}

func TestAFileNobodySentLeavesTheFieldAlone(t *testing.T) {
	type avatar struct {
		Name   string    `form:"name"`
		Image  *Upload   `form:"image"`
		Extras []*Upload `form:"extras"`
	}

	in, err := bind[avatar](t, uploadOf(t, url.Values{"name": {"water"}}))
	if err != nil {
		t.Fatal(err)
	}
	if in.Name != "water" {
		t.Errorf("the name is %q, want water", in.Name)
	}
	if in.Image != nil || in.Extras != nil {
		t.Errorf("a form with no files in it bound %v and %v", in.Image, in.Extras)
	}
}

func TestAFileFieldOnAFormWithNoFilesInItIsNotAnError(t *testing.T) {
	type avatar struct {
		Name  string  `form:"name"`
		Image *Upload `form:"image"`
	}

	in, err := bind[avatar](t, form("POST", url.Values{"name": {"water"}}))
	if err != nil {
		t.Fatal(err)
	}
	if in.Name != "water" {
		t.Errorf("the name is %q, want water", in.Name)
	}
	if in.Image != nil {
		t.Error("an urlencoded form bound a file")
	}
}

func TestTheClientsPathIsNotPartOfTheFilename(t *testing.T) {
	type upload struct {
		File *Upload `form:"file"`
	}

	for _, c := range []struct{ sent, want string }{
		{"me.png", "me.png"},
		{"../../etc/passwd", "passwd"},
		{`C:\Users\somebody\evil.exe`, "evil.exe"},
		{"/etc/shadow", "shadow"},
		{"..", ""},
		{"with a space.txt", "with a space.txt"},
	} {
		in, err := bind[upload](t, uploadOf(t, nil, part{field: "file", name: c.sent, body: []byte("x")}))
		if err != nil {
			t.Errorf("%q: %v", c.sent, err)
			continue
		}
		if in.File.Filename != c.want {
			t.Errorf("%q arrived as %q, want %q", c.sent, in.File.Filename, c.want)
		}
	}
}

func TestTheTypeIsSniffedRatherThanBelieved(t *testing.T) {
	type upload struct {
		File *Upload `form:"file"`
	}

	in, err := bind[upload](t, uploadOf(t, nil, part{
		field: "file",
		name:  "photo.png",
		kind:  "image/png",
		body:  []byte("this is not a picture, whatever the part header says"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	if strings.HasPrefix(in.File.MIME, "image/") {
		t.Errorf("the type is %q, want the sniffed one rather than the claim", in.File.MIME)
	}
	if got := in.File.Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("the claim is %q, want it kept on the header where somebody can compare it", got)
	}
}

func TestOpenAndBytesReadTheSameFile(t *testing.T) {
	type upload struct {
		File *Upload `form:"file"`
	}

	body := pngOf(t, 3, 3)
	in, err := bind[upload](t, uploadOf(t, nil, part{field: "file", name: "x.png", body: body}))
	if err != nil {
		t.Fatal(err)
	}

	got, err := in.File.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Error("Bytes did not give back what was sent")
	}

	// Opening it again starts at the beginning, which is what a handler that
	// sniffed a file and then stores it depends on.
	f, err := in.File.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	again, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, body) {
		t.Error("the second read did not start at the beginning")
	}
}

func TestImageReadsTheHeaderWithoutDecodingThePicture(t *testing.T) {
	type upload struct {
		File *Upload `form:"file"`
	}

	in, err := bind[upload](t, uploadOf(t, nil, part{field: "file", name: "x.png", body: pngOf(t, 40, 9)}))
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := in.File.Image()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 40 || cfg.Height != 9 {
		t.Errorf("the image is %dx%d, want 40x9", cfg.Width, cfg.Height)
	}
}

func TestImageSaysSoWhenTheFileIsNotAnImage(t *testing.T) {
	type upload struct {
		File *Upload `form:"file"`
	}

	in, err := bind[upload](t, uploadOf(t, nil, part{field: "file", name: "notes.txt", body: []byte("water")}))
	if err != nil {
		t.Fatal(err)
	}

	_, err = in.File.Image()
	if errs.KindOf(err) != errs.Invalid || errs.CodeOf(err) != "upload.image" {
		t.Fatalf("the error is %v, want Invalid and upload.image", err)
	}
}

func TestAFileTaggedAsSomethingThatIsNotAFormIsAMistake(t *testing.T) {
	type wrong struct {
		File *Upload `header:"x-file"`
	}

	_, err := bind[wrong](t, uploadOf(t, nil))
	if errs.KindOf(err) != errs.Internal || errs.CodeOf(err) != "bind.field" {
		t.Fatalf("the error is %v, want Internal and bind.field", err)
	}
}

func TestAnUploadByValueIsAMistakeRatherThanAStructToWalkInto(t *testing.T) {
	type wrong struct {
		File Upload
	}

	_, err := bind[wrong](t, uploadOf(t, nil))
	if errs.KindOf(err) != errs.Internal || errs.CodeOf(err) != "bind.field" {
		t.Fatalf("the error is %v, want Internal and bind.field", err)
	}
	if !strings.Contains(err.Error(), "*web.Upload") {
		t.Errorf("the error is %q, want it to name the type that works", err)
	}
}

// gone is an upload whose file is not where the request said it was, which is a
// temporary file something else deleted. A zero FileHeader has no bytes in it
// and no name to open, so its Open fails the way that one does.
func gone() *Upload {
	return newUpload(&multipart.FileHeader{Filename: "gone.png", Size: 12})
}

func TestAFileThatCannotBeReadBindsAsAStreamOfBytes(t *testing.T) {
	// Binding does not report this. The client sent a file and the file is
	// fine as far as the client is concerned, so saying the field is wrong
	// would be blaming the wrong end.
	u := gone()
	if u.MIME != "application/octet-stream" {
		t.Errorf("the type is %q, want application/octet-stream", u.MIME)
	}
	if u.Filename != "gone.png" || u.Size != 12 {
		t.Errorf("the rest of the part did not survive: %+v", u)
	}
}

// broken is a file that opens and then will not read, which is a disk going
// away halfway through a request.
type broken struct{}

func (broken) Read([]byte) (int, error) { return 0, errors.New("the disk is gone") }

func TestAFileThatWillNotReadIsAStreamOfBytesToo(t *testing.T) {
	if got := sniffReader(broken{}); got != "application/octet-stream" {
		t.Errorf("the type is %q, want application/octet-stream", got)
	}
}

func TestReadingAFileThatIsNotThereSaysWhy(t *testing.T) {
	u := gone()

	if _, err := u.Bytes(); err == nil {
		t.Error("Bytes read a file that is not there")
	}
	if _, err := u.Image(); err == nil {
		t.Error("Image read a file that is not there")
	}
}

func TestSniffKnowsTheBrandsTheStandardLibraryDoesNot(t *testing.T) {
	iso := func(brand string, compatible ...string) []byte {
		b := []byte("\x00\x00\x00\x00ftyp" + brand + "\x00\x00\x00\x00")
		for _, c := range compatible {
			b = append(b, c...)
		}
		b[3] = byte(len(b))
		return b
	}

	for _, c := range []struct {
		what string
		in   []byte
		want string
	}{
		{"avif", iso("avif", "mif1", "miaf"), "image/avif"},
		{"heic", iso("heic", "mif1", "miaf"), "image/heic"},
		{"heif", iso("mif1", "miaf"), "image/heif"},
		{"mp4", iso("isom", "iso2", "avc1"), "video/mp4"},
		{"quicktime", iso("qt  "), "video/quicktime"},

		// A brand nobody here knows, read from the compatible list rather than
		// from the major brand, which is how a file from a camera that invented
		// its own brand still comes back as a video.
		{"unknown brand, known compatible", iso("zzzz", "mp42"), "video/mp4"},
		{"unknown brand, nothing known", iso("zzzz", "yyyy"), "application/octet-stream"},

		// Everything the standard library already knows, which since this was
		// first written down includes webp, ogg and wasm.
		{"png", []byte("\x89PNG\r\n\x1a\n"), "image/png"},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "image/webp"},
		{"wasm", []byte("\x00asm\x01\x00\x00\x00"), "application/wasm"},
		{"ogg", []byte("OggS\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00"), "application/ogg"},

		// Bytes that are not a box at all, and a box too short to have a brand
		// in it. Neither is allowed to index past what arrived.
		{"empty", nil, "text/plain; charset=utf-8"},
		{"short", []byte("\x00\x00\x00\x18ftyp"), "application/octet-stream"},
		{"not a box", []byte("water is water is water"), "text/plain; charset=utf-8"},
	} {
		if got := sniff(c.in); got != c.want {
			t.Errorf("%s sniffed as %q, want %q", c.what, got, c.want)
		}
	}
}

func TestABoxThatLiesAboutItsLengthIsNotReadPastTheEnd(t *testing.T) {
	// The length says two kilobytes and sixteen bytes arrived. Trusting it is
	// how a sniffer reads somebody else's memory.
	head := []byte("\x00\x00\x08\x00ftypzzzz\x00\x00\x00\x00")
	if got := sniff(head); got != "application/octet-stream" {
		t.Errorf("the sniffed type is %q, want application/octet-stream", got)
	}
}

func TestAFilenameLosesTheCharactersThatWouldSplitALogLine(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"me.png", "me.png"},
		{"me\nGET admin.png", "meGET admin.png"},
		{"me\nGET /admin.png", "admin.png"},
		{"\x00\x01evil", "evil"},
		{"  padded.txt  ", "padded.txt"},
		{".", ""},
		{"", ""},
		{"résumé.pdf", "résumé.pdf"},
	} {
		if got := cleanName(c.in); got != c.want {
			t.Errorf("cleanName(%q) is %q, want %q", c.in, got, c.want)
		}
	}
}
