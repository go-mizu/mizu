package web

import (
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/router"
)

// bind runs one request through Bind and hands back what came out.
//
// It goes through H rather than building a Ctx by hand, so every test here is
// binding the same Ctx a handler is given, guard counter and all.
func bind[T any](t *testing.T, r *http.Request) (T, error) {
	t.Helper()

	var in T
	var err error
	serve(t, r, func(c *Ctx) error {
		in, err = Bind[T](c)
		return nil
	})
	return in, err
}

// form is a request carrying an urlencoded body.
func form(method string, values url.Values) *http.Request {
	r := httptest.NewRequest(method, "/", strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// multipartOf is a request carrying a multipart body with those fields in it.
func multipartOf(t *testing.T, values url.Values) *http.Request {
	t.Helper()

	var body strings.Builder
	w := multipart.NewWriter(&body)
	for key, all := range values {
		for _, v := range all {
			if err := w.WriteField(key, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/", strings.NewReader(body.String()))
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}

type search struct {
	Q       string `query:"q"`
	Page    int    `query:"page"`
	PerPage int
}

func TestBindReadsTheQueryString(t *testing.T) {
	in, err := bind[search](t, httptest.NewRequest("GET", "/?q=water&page=3&per_page=25", nil))
	if err != nil {
		t.Fatal(err)
	}
	if in.Q != "water" {
		t.Errorf("q is %q, want water", in.Q)
	}
	if in.Page != 3 {
		t.Errorf("page is %d, want 3", in.Page)
	}
	if in.PerPage != 25 {
		t.Errorf("per_page is %d, want 25, so an untagged field takes its name in snake case", in.PerPage)
	}
}

func TestBindReadsAFormBody(t *testing.T) {
	r := form("POST", url.Values{"q": {"water"}, "page": {"2"}})

	in, err := bind[search](t, r)
	if err != nil {
		t.Fatal(err)
	}
	if in.Q != "water" || in.Page != 2 {
		t.Errorf("read %+v, want water and 2", in)
	}
}

func TestBindReadsAMultipartForm(t *testing.T) {
	r := multipartOf(t, url.Values{"q": {"water"}, "page": {"2"}})

	in, err := bind[search](t, r)
	if err != nil {
		t.Fatal(err)
	}
	if in.Q != "water" || in.Page != 2 {
		t.Errorf("read %+v, want water and 2", in)
	}
}

// The precedence is worth a test of its own for both body shapes, because
// net/http merges them into Form in one order for an urlencoded body and the
// other order for a multipart one.
func TestTheBodyWinsOverTheQueryString(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func(*testing.T) *http.Request
	}{
		{"urlencoded", func(*testing.T) *http.Request {
			return form("POST", url.Values{"q": {"body"}})
		}},
		{"multipart", func(t *testing.T) *http.Request {
			return multipartOf(t, url.Values{"q": {"body"}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.make(t)
			r.URL.RawQuery = "q=query&page=9"

			in, err := bind[search](t, r)
			if err != nil {
				t.Fatal(err)
			}
			if in.Q != "body" {
				t.Errorf("q is %q, want body, since the body outranks the query string", in.Q)
			}
			if in.Page != 9 {
				t.Errorf("page is %d, want 9, since the query string fills what the body left out", in.Page)
			}
		})
	}
}

func TestBindReadsThePathTheHeaderAndTheCookie(t *testing.T) {
	type show struct {
		ID    int    `path:"id"`
		Trace string `header:"X-Trace-Id"`
		Seen  string `cookie:"seen"`
	}

	var in show
	var err error

	rt := router.New()
	rt.Handle("GET /posts/{id:int}", H(func(c *Ctx) error {
		in, err = Bind[show](c)
		return nil
	}))

	r := httptest.NewRequest("GET", "/posts/12", nil)
	r.Header.Set("X-Trace-Id", "abc")
	r.AddCookie(&http.Cookie{Name: "seen", Value: "yes"})
	rt.ServeHTTP(httptest.NewRecorder(), r)

	if err != nil {
		t.Fatal(err)
	}
	if in.ID != 12 {
		t.Errorf("id is %d, want 12", in.ID)
	}
	if in.Trace != "abc" {
		t.Errorf("the trace header is %q, want abc", in.Trace)
	}
	if in.Seen != "yes" {
		t.Errorf("the cookie is %q, want yes", in.Seen)
	}
}

// A name that was not sent leaves the field alone, so a struct with something
// already in it keeps it. That is what makes BindInto worth having.
func TestANameTheRequestDidNotSendLeavesTheFieldAlone(t *testing.T) {
	in := search{Q: "already here", Page: 1}

	serve(t, httptest.NewRequest("GET", "/?page=4", nil), func(c *Ctx) error {
		if err := BindInto(c, &in); err != nil {
			t.Fatal(err)
		}
		return nil
	})

	if in.Q != "already here" {
		t.Errorf("q is %q, want it left alone", in.Q)
	}
	if in.Page != 4 {
		t.Errorf("page is %d, want 4", in.Page)
	}
}

func TestAnEmptyValueIsOnlySentForAFieldThatTakesOne(t *testing.T) {
	type blank struct {
		Note string `query:"note"`
		Page int    `query:"page"`
		When *int   `query:"when"`
	}

	in := blank{Page: 7}
	serve(t, httptest.NewRequest("GET", "/?note=&page=&when=", nil), func(c *Ctx) error {
		if err := BindInto(c, &in); err != nil {
			t.Fatal(err)
		}
		return nil
	})

	if in.Note != "" {
		t.Errorf("note is %q, want empty, since an empty string is a string", in.Note)
	}
	if in.Page != 7 {
		t.Errorf("page is %d, want 7, since a blank number field is one nobody filled in", in.Page)
	}
	if in.When != nil {
		t.Error("a blank number field behind a pointer was allocated")
	}
}

func TestBindReadsEveryValueSentUnderOneName(t *testing.T) {
	type tags struct {
		Tag  []string `query:"tag"`
		Size []int    `query:"size"`
	}

	in, err := bind[tags](t, httptest.NewRequest("GET", "/?tag=a&tag=&tag=b&size=1&size=&size=2", nil))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(in.Tag, ",") != "a,,b" {
		t.Errorf("the tags are %q, want a, an empty one and b", in.Tag)
	}
	if len(in.Size) != 2 || in.Size[0] != 1 || in.Size[1] != 2 {
		t.Errorf("the sizes are %v, want [1 2] with the blank one dropped", in.Size)
	}
}

func TestBindReadsTheTypesItKnows(t *testing.T) {
	type every struct {
		S   string        `query:"s"`
		B   bool          `query:"b"`
		I8  int8          `query:"i8"`
		I64 int64         `query:"i64"`
		U16 uint16        `query:"u16"`
		F32 float32       `query:"f32"`
		F64 float64       `query:"f64"`
		Raw []byte        `query:"raw"`
		D   time.Duration `query:"d"`
		At  time.Time     `query:"at"`
		IP  netip.Addr    `query:"ip"`
		P   *int          `query:"p"`
	}

	q := "s=mizu&b=true&i8=-8&i64=90071992547409&u16=65535&f32=1.5&f64=2.25" +
		"&raw=bytes&d=90s&at=2026-08-25T10:00:00Z&ip=192.0.2.7&p=42"

	in, err := bind[every](t, httptest.NewRequest("GET", "/?"+q, nil))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name      string
		got, want any
	}{
		{"s", in.S, "mizu"},
		{"b", in.B, true},
		{"i8", in.I8, int8(-8)},
		{"i64", in.I64, int64(90071992547409)},
		{"u16", in.U16, uint16(65535)},
		{"f32", in.F32, float32(1.5)},
		{"f64", in.F64, 2.25},
		{"raw", string(in.Raw), "bytes"},
		{"d", in.D, 90 * time.Second},
		{"at", in.At.Format(time.RFC3339), "2026-08-25T10:00:00Z"},
		{"ip", in.IP.String(), "192.0.2.7"},
		{"p", *in.P, 42},
	} {
		if tc.got != tc.want {
			t.Errorf("%s is %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

func TestADateArrivesInEveryShapeAFormSends(t *testing.T) {
	type when struct {
		At time.Time `query:"at"`
	}

	for _, tc := range []struct{ sent, want string }{
		{"2026-08-25T10:00:00Z", "2026-08-25T10:00:00Z"},
		{"2026-08-25T10:00:00", "2026-08-25T10:00:00Z"},
		{"2026-08-25T10:00", "2026-08-25T10:00:00Z"},
		{"2026-08-25", "2026-08-25T00:00:00Z"},
	} {
		in, err := bind[when](t, httptest.NewRequest("GET", "/?at="+url.QueryEscape(tc.sent), nil))
		if err != nil {
			t.Fatalf("%s: %v", tc.sent, err)
		}
		if got := in.At.Format(time.RFC3339); got != tc.want {
			t.Errorf("%s read as %s, want %s", tc.sent, got, tc.want)
		}
	}
}

func TestACheckboxBinds(t *testing.T) {
	type prefs struct {
		Mail bool `form:"mail"`
	}

	for _, tc := range []struct {
		sent string
		want bool
	}{
		{"on", true},
		{"yes", true},
		{"true", true},
		{"1", true},
		{"off", false},
		{"no", false},
		{"false", false},
		{"0", false},
	} {
		in, err := bind[prefs](t, form("POST", url.Values{"mail": {tc.sent}}))
		if err != nil {
			t.Fatalf("%s: %v", tc.sent, err)
		}
		if in.Mail != tc.want {
			t.Errorf("%s read as %v, want %v", tc.sent, in.Mail, tc.want)
		}
	}
}

// A checkbox that is not ticked is not sent at all, which is the one thing
// about form booleans that surprises everybody once.
func TestACheckboxNobodyTickedIsNotSent(t *testing.T) {
	type prefs struct {
		Mail bool `form:"mail"`
	}

	in := prefs{Mail: true}
	serve(t, form("POST", url.Values{}), func(c *Ctx) error {
		return BindInto(c, &in)
	})
	if !in.Mail {
		t.Error("an unticked checkbox cleared the field, which means it cannot be told from one that was never on the form")
	}
}

func TestEveryValueThatWouldNotDecodeIsReported(t *testing.T) {
	type wrong struct {
		Page int           `query:"page"`
		Rate float64       `query:"rate"`
		On   bool          `query:"on"`
		D    time.Duration `query:"d"`
		At   time.Time     `query:"at"`
		IP   netip.Addr    `query:"ip"`
		Big  int8          `query:"big"`
		Neg  uint          `query:"neg"`
	}

	q := "page=one&rate=lots&on=maybe&d=soon&at=yesterday&ip=here&big=999&neg=-1"
	_, err := bind[wrong](t, httptest.NewRequest("GET", "/?"+q, nil))
	if err == nil {
		t.Fatal("a request full of nonsense bound without complaint")
	}
	if !errors.Is(err, errs.Invalid) {
		t.Errorf("the error is %v, want one of kind Invalid", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "bind.invalid" {
		t.Errorf("the code is %q, want bind.invalid", got)
	}

	want := map[string]string{
		"page": "invalid_number",
		"rate": "invalid_number",
		"on":   "invalid_boolean",
		"d":    "invalid_duration",
		"at":   "invalid_time",
		"ip":   "invalid_value",
		"big":  "out_of_range",
		"neg":  "invalid_number",
	}

	got := map[string]string{}
	for _, f := range errs.Fields(err) {
		got[f.Name] = f.Code
		if f.Msg == "" {
			t.Errorf("%s came back with no message on it", f.Name)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("reported %d fields, want %d: %v", len(got), len(want), got)
	}
	for name, code := range want {
		if got[name] != code {
			t.Errorf("%s is %q, want %q", name, got[name], code)
		}
	}
}

// A value that does not fit is reported for the field it was sent for and the
// rest of the slice is not bound, since half a list is not a list.
func TestOneBadValueInAListStopsTheList(t *testing.T) {
	type tags struct {
		Size []int `query:"size"`
	}

	in, err := bind[tags](t, httptest.NewRequest("GET", "/?size=1&size=two&size=3", nil))
	if err == nil {
		t.Fatal("a list with a word in it bound without complaint")
	}
	if in.Size != nil {
		t.Errorf("the list is %v, want nothing", in.Size)
	}
	if f := errs.Fields(err); len(f) != 1 || f[0].Name != "size" || f[0].Code != "invalid_number" {
		t.Errorf("reported %v, want one invalid_number on size", f)
	}
}

func TestBindSkipsWhatItIsToldTo(t *testing.T) {
	type post struct {
		Title    string `form:"title"`
		AuthorID int    `bind:"-"`
		Secret   string `json:"-"`
		Skipped  int    `form:"-"`
		hidden   string
	}

	in, err := bind[post](t, form("POST", url.Values{
		"title":     {"water"},
		"author_id": {"9"},
		"secret":    {"leaked"},
		"skipped":   {"3"},
		"hidden":    {"no"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "water" {
		t.Errorf("title is %q, want water", in.Title)
	}
	if in.AuthorID != 0 {
		t.Errorf("author_id is %d, and a field marked bind:\"-\" is the whole defence against mass assignment", in.AuthorID)
	}
	if in.Secret != "" {
		t.Errorf("secret is %q, and json:\"-\" says the field is not part of the wire shape", in.Secret)
	}
	if in.Skipped != 0 {
		t.Errorf("skipped is %d, want it left alone", in.Skipped)
	}
	if in.hidden != "" {
		t.Errorf("an unexported field was filled in, which reflect should not have allowed")
	}
}

func TestTheNameComesFromTheFirstTagThatHasOne(t *testing.T) {
	type named struct {
		A string `form:"a_form" query:"a_query" json:"a_json"`
		B string `query:"b_query" json:"b_json"`
		C string `json:"c_json"`
		D string
		E string `form:",omitempty"`
	}

	in, err := bind[named](t, httptest.NewRequest("GET",
		"/?a_form=1&a_query=x&a_json=x&b_query=2&b_json=x&c_json=3&d=4&e=5", nil))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, got, want string }{
		{"form beats query", in.A, "1"},
		{"query beats json", in.B, "2"},
		{"json beats the field name", in.C, "3"},
		{"the field name in snake case", in.D, "4"},
		{"a tag with only options on it", in.E, "5"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: read %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

type page struct {
	Page    int `query:"page"`
	PerPage int `query:"per_page"`
}

type address struct {
	City string `form:"city"`
}

func TestAnEmbeddedStructIsFlattenedAndANestedOneIsNot(t *testing.T) {
	type order struct {
		page
		Ship  address  `form:"ship"`
		Bill  *address `form:"bill"`
		Where address
	}

	in, err := bind[order](t, form("POST", url.Values{
		"page":       {"2"},
		"ship.city":  {"Kyoto"},
		"where.city": {"Osaka"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if in.Page != 2 {
		t.Errorf("page is %d, want 2, since an embedded struct has no prefix", in.Page)
	}
	if in.Ship.City != "Kyoto" {
		t.Errorf("the shipping city is %q, want Kyoto", in.Ship.City)
	}
	if in.Where.City != "Osaka" {
		t.Errorf("the city is %q, want Osaka, so an untagged nested struct is named after its field", in.Where.City)
	}
	if in.Bill != nil {
		t.Error("a nested struct behind a pointer was made for a request that named nothing in it")
	}
}

// The prefix is for the query and the form, which is where a nested name is a
// name. A header, a cookie and a path parameter are flat namespaces the whole
// request shares.
func TestAHeaderInsideANestedStructIsNotNamedUnderIt(t *testing.T) {
	type place struct {
		Country string `header:"X-Country"`
	}
	type order struct {
		Ship place `form:"ship"`
	}

	r := form("POST", url.Values{})
	r.Header.Set("X-Country", "JP")

	in, err := bind[order](t, r)
	if err != nil {
		t.Fatal(err)
	}
	if in.Ship.Country != "JP" {
		t.Errorf("the country is %q, want JP", in.Ship.Country)
	}
}

func TestANestedStructBehindAPointerIsMadeWhenSomethingNamesIt(t *testing.T) {
	type order struct {
		Bill *address `form:"bill"`
	}

	in, err := bind[order](t, form("POST", url.Values{"bill.city": {"Nara"}}))
	if err != nil {
		t.Fatal(err)
	}
	if in.Bill == nil || in.Bill.City != "Nara" {
		t.Errorf("the billing address is %+v, want Nara", in.Bill)
	}
}

// node holds one of itself, which is a shape a query string has no way to
// carry. The plan stops rather than going round, and the scalar next to it
// still binds.
type node struct {
	Name string `form:"name"`
	Next *node  `form:"next"`
}

func TestAStructThatHoldsOneOfItselfDoesNotGoRound(t *testing.T) {
	in, err := bind[node](t, form("POST", url.Values{"name": {"root"}}))
	if err != nil {
		t.Fatal(err)
	}
	if in.Name != "root" {
		t.Errorf("name is %q, want root", in.Name)
	}
	if in.Next != nil {
		t.Error("the recursive field was bound")
	}
}

func TestAFieldNothingReadsIsOnlyAMistakeWhenSomebodyTaggedIt(t *testing.T) {
	type quiet struct {
		Title string         `form:"title"`
		Meta  map[string]any // no tag, so binding has nothing to do with it
	}
	if _, err := bind[quiet](t, form("POST", url.Values{"title": {"water"}})); err != nil {
		t.Fatalf("an untagged field of a type nothing reads was treated as a mistake: %v", err)
	}

	type loud struct {
		Meta map[string]any `form:"meta"`
	}
	_, err := bind[loud](t, form("POST", url.Values{}))
	if err == nil {
		t.Fatal("a tag on a type nothing reads bound without complaint")
	}
	if !errors.Is(err, errs.Internal) {
		t.Errorf("the kind is %v, want Internal, since this is a mistake in the program", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "bind.field" {
		t.Errorf("the code is %q, want bind.field", got)
	}
	if !strings.Contains(err.Error(), "loud.Meta") {
		t.Errorf("the message is %q, want the field named in it", err)
	}
}

// The first one is the one reported. A second message would say the same thing
// about a different field and neither is more use than the other, since both
// are fixed by reading the struct.
func TestTwoFieldsNothingReadsReportTheFirst(t *testing.T) {
	type loud struct {
		Meta map[string]any `form:"meta"`
		Rows [][]string     `form:"rows"`
	}

	_, err := bind[loud](t, form("POST", url.Values{}))
	if err == nil {
		t.Fatal("two tags on types nothing reads bound without complaint")
	}
	if !strings.Contains(err.Error(), "loud.Meta") {
		t.Errorf("the message is %q, want the first of the two named in it", err)
	}
}

func TestAPathHeaderOrCookieTheRequestDidNotSendLeavesTheFieldAlone(t *testing.T) {
	type show struct {
		ID    int    `path:"id"`
		Trace string `header:"X-Trace-Id"`
		Seen  string `cookie:"seen"`
	}

	in := show{ID: 3, Trace: "kept", Seen: "kept"}
	serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return BindInto(c, &in)
	})

	if in.ID != 3 || in.Trace != "kept" || in.Seen != "kept" {
		t.Errorf("read %+v, want all three left as they were", in)
	}
}

func TestBindIntoSomethingThatIsNotAPointerToAStruct(t *testing.T) {
	var s search

	for _, tc := range []struct {
		name string
		dst  any
	}{
		{"a value", s},
		{"a nil pointer", (*search)(nil)},
		{"a pointer to something else", new(int)},
		{"nothing at all", nil},
	} {
		serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
			err := c.Bind(tc.dst)
			if err == nil {
				t.Errorf("%s: bound without complaint", tc.name)
				return nil
			}
			if !errors.Is(err, errs.Internal) {
				t.Errorf("%s: the kind is %v, want Internal", tc.name, errs.KindOf(err))
			}
			return nil
		})
	}
}

func TestABodyOverTheLimitSaysSoRatherThanBindingNothing(t *testing.T) {
	r := form("POST", url.Values{"q": {strings.Repeat("water", 100)}})

	serve(t, r, func(c *Ctx) error {
		c.r.Body = http.MaxBytesReader(c.Writer(), c.r.Body, 16)

		_, err := Bind[search](c)
		if err == nil {
			t.Fatal("a body over the limit bound to an empty struct")
		}
		if !errors.Is(err, errs.TooLarge) {
			t.Errorf("the kind is %v, want TooLarge", errs.KindOf(err))
		}
		if got := errs.CodeOf(err); got != "bind.too_large" {
			t.Errorf("the code is %q, want bind.too_large", got)
		}
		return nil
	})
}

func TestAFormThatWillNotParseSaysSo(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("q=%zz"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := bind[search](t, r)
	if err == nil {
		t.Fatal("a body that is not a form bound to an empty struct")
	}
	if !errors.Is(err, errs.Invalid) {
		t.Errorf("the kind is %v, want Invalid", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "bind.unreadable" {
		t.Errorf("the code is %q, want bind.unreadable", got)
	}
}

// A struct that reads nothing from the query or the form never parses one, so a
// handler binding two path parameters does not touch the body.
func TestAStructWithNoFormFieldsNeverReadsTheBody(t *testing.T) {
	type headers struct {
		Trace string `header:"X-Trace-Id"`
	}

	r := httptest.NewRequest("POST", "/", strings.NewReader("q=%zz"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-Trace-Id", "abc")

	in, err := bind[headers](t, r)
	if err != nil {
		t.Fatalf("a body nobody asked for was parsed anyway: %v", err)
	}
	if in.Trace != "abc" {
		t.Errorf("the trace header is %q, want abc", in.Trace)
	}
}

func TestThePlanForATypeIsBuiltOnce(t *testing.T) {
	type once struct {
		Q string `query:"q"`
	}

	first := planFor(reflect.TypeFor[once]())
	if second := planFor(reflect.TypeFor[once]()); second != first {
		t.Error("the second bind of a type built a second plan")
	}
}

func TestBindReadsEveryValueOfAHeaderSentTwice(t *testing.T) {
	type accept struct {
		Langs []string `header:"Accept-Language"`
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Add("Accept-Language", "ja")
	r.Header.Add("Accept-Language", "en")

	in, err := bind[accept](t, r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(in.Langs, ",") != "ja,en" {
		t.Errorf("the languages are %v, want ja and en", in.Langs)
	}
}

func TestAFieldNameBecomesTheNameARequestSpells(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Page", "page"},
		{"PerPage", "per_page"},
		{"ID", "id"},
		{"UserID", "user_id"},
		{"HTTPServer", "http_server"},
		{"OAuth2Token", "o_auth2_token"},
		{"A", "a"},
		{"", ""},
		{"already_snake", "already_snake"},
		{"Ωmega", "Ωmega"},
	} {
		if got := snake(c.in); got != c.want {
			t.Errorf("snake(%q) is %q, want %q", c.in, got, c.want)
		}
	}
}
