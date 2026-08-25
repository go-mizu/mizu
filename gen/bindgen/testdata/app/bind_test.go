package app

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/router"
	"github.com/go-mizu/mizu/web"
)

// The generated binder and the reflective one have to answer the same request
// the same way, which is the promise the whole generator rests on.
//
// A defined type over a request struct has the same fields with the same tags
// and none of the methods, so binding one of those is binding the same struct
// by reflection. That is what these compare against.
type (
	listingRef Listing
	orderRef   Order
	profileRef Profile
	webhookRef Webhook
	treeRef    Tree
)

// bind runs one request through one handler and hands back what Bind said.
//
// The request is made rather than passed, because each of these is bound twice
// and a body is only readable once.
func bind(t *testing.T, pattern string, make func() *http.Request, into func(*web.Ctx) error) error {
	t.Helper()

	var err error
	rt := router.New()
	rt.Handle(pattern, web.H(func(c *web.Ctx) error {
		err = into(c)
		return nil
	}))
	rt.ServeHTTP(httptest.NewRecorder(), make())
	return err
}

// ok is bind for a request that is expected to work.
func ok(t *testing.T, pattern string, make func() *http.Request, into func(*web.Ctx) error) {
	t.Helper()
	if err := bind(t, pattern, make, into); err != nil {
		t.Fatalf("bind: %v", err)
	}
}

// form is a request carrying an urlencoded body.
func form(target string, values url.Values) *http.Request {
	r := httptest.NewRequest("POST", target, strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestListingReadsTheQueryString(t *testing.T) {
	target := "/search?q=water&sort=new&tags=a&tags=b&cursor=abc&limit=50" +
		"&score=1.5&draft=true&since=2026-08-25T09:30:00Z&page=3&per_page=25" +
		"&internal=no&secret=no"
	make := func() *http.Request { return httptest.NewRequest("GET", target, nil) }

	var got Listing
	var ref listingRef
	ok(t, "GET /search", make, func(c *web.Ctx) error { return c.Bind(&got) })
	ok(t, "GET /search", make, func(c *web.Ctx) error { return c.Bind(&ref) })

	if !reflect.DeepEqual(got, Listing(ref)) {
		t.Errorf("the two binders disagree\ngenerated %+v\nreflected %+v", got, Listing(ref))
	}
	if got.Q != "water" || got.Sort != "new" {
		t.Errorf("q is %q and sort is %q, want water and new", got.Q, got.Sort)
	}
	if !slices.Equal(got.Tags, []string{"a", "b"}) {
		t.Errorf("tags are %q, want a and b", got.Tags)
	}
	if string(got.Cursor) != "abc" {
		t.Errorf("the cursor is %q, want abc", got.Cursor)
	}
	if got.Limit != 50 || got.Score != 1.5 {
		t.Errorf("the limit is %d and the score is %v, want 50 and 1.5", got.Limit, got.Score)
	}
	if got.Draft == nil || !*got.Draft {
		t.Errorf("draft is %v, want a pointer to true", got.Draft)
	}
	if got.Since.Year() != 2026 {
		t.Errorf("since is %v, want a time in 2026", got.Since)
	}
	if got.Page != 3 || got.PerPage != 25 {
		t.Errorf("the page is %d of %d, want 3 of 25, so an embedded struct adds nothing to a name", got.Page, got.PerPage)
	}
	if got.Internal != "" || got.Secret != "" {
		t.Errorf("a field tagged to be left alone was filled in: %+v", got)
	}
}

// A name the request never used leaves its field where it was, which is what a
// handler filling in defaults before it binds depends on.
func TestListingLeavesWhatTheRequestDidNotSend(t *testing.T) {
	make := func() *http.Request { return httptest.NewRequest("GET", "/search?q=water", nil) }

	got := Listing{Sort: "old", Tags: []string{"kept"}, Paging: Paging{PerPage: 10}}
	ref := listingRef{Sort: "old", Tags: []string{"kept"}, Paging: Paging{PerPage: 10}}
	ok(t, "GET /search", make, func(c *web.Ctx) error { return c.Bind(&got) })
	ok(t, "GET /search", make, func(c *web.Ctx) error { return c.Bind(&ref) })

	if !reflect.DeepEqual(got, Listing(ref)) {
		t.Errorf("the two binders disagree\ngenerated %+v\nreflected %+v", got, Listing(ref))
	}
	if got.Sort != "old" || !slices.Equal(got.Tags, []string{"kept"}) || got.PerPage != 10 {
		t.Errorf("read %+v, want the fields the request left out untouched", got)
	}
}

// A name the request did send with nothing under it empties the list, which is
// how a form clears a set of checkboxes.
func TestListingEmptiesAListTheRequestSentNothingFor(t *testing.T) {
	make := func() *http.Request { return httptest.NewRequest("GET", "/search?tags=", nil) }

	got := Listing{Tags: []string{"kept"}}
	ref := listingRef{Tags: []string{"kept"}}
	ok(t, "GET /search", make, func(c *web.Ctx) error { return c.Bind(&got) })
	ok(t, "GET /search", make, func(c *web.Ctx) error { return c.Bind(&ref) })

	if !reflect.DeepEqual(got, Listing(ref)) {
		t.Errorf("the two binders disagree\ngenerated %+v\nreflected %+v", got, Listing(ref))
	}
	if got.Tags == nil || len(got.Tags) != 1 || got.Tags[0] != "" {
		t.Errorf("tags are %#v, want one empty string, since a string takes an empty value", got.Tags)
	}
}

func TestOrderReadsEveryPartOfTheRequest(t *testing.T) {
	body := url.Values{
		"note":            {"a gift"},
		"wait":            {"1500ms"},
		"codes":           {"1", "2", "3"},
		"labels":          {"gift", "fragile"},
		"sizes":           {"7", "", "9"},
		"names":           {"box", ""},
		"origin":          {"192.0.2.7"},
		"address.city":    {"Kyoto"},
		"address.country": {"JP"},
		"address.zone":    {"198.51.100.9"},
		"ship.city":       {"Osaka"},
		"coupon_code":     {"WATER"},
	}
	make := func() *http.Request {
		r := form("/orders/42?coupon_code=from-the-query", body)
		r.Header.Set("X-Request-Id", "req-1")
		r.Header.Add("X-Trace", "one")
		r.Header.Add("X-Trace", "two")
		r.AddCookie(&http.Cookie{Name: "locale", Value: "ja"})
		r.AddCookie(&http.Cookie{Name: "seen", Value: "welcome"})
		return r
	}

	var got Order
	var ref orderRef
	ok(t, "POST /orders/{id:int}", make, func(c *web.Ctx) error { return c.Bind(&got) })
	ok(t, "POST /orders/{id:int}", make, func(c *web.Ctx) error { return c.Bind(&ref) })

	if !reflect.DeepEqual(got, Order(ref)) {
		t.Errorf("the two binders disagree\ngenerated %+v\nreflected %+v", got, Order(ref))
	}
	if got.ID != 42 {
		t.Errorf("the id is %d, want 42 out of the path", got.ID)
	}
	if got.Ref != "req-1" || got.Locale != "ja" {
		t.Errorf("the header is %q and the cookie is %q, want req-1 and ja", got.Ref, got.Locale)
	}
	if !slices.Equal(got.Traces, []string{"one", "two"}) {
		t.Errorf("the traces are %q, want one and two", got.Traces)
	}
	if got.Note == nil || *got.Note != "a gift" {
		t.Errorf("the note is %v, want a pointer to a gift", got.Note)
	}
	if got.Wait.String() != "1.5s" {
		t.Errorf("the wait is %v, want 1.5s", got.Wait)
	}
	if !slices.Equal(got.Codes, []int{1, 2, 3}) {
		t.Errorf("the codes are %v, want 1, 2 and 3", got.Codes)
	}
	if !slices.Equal(got.Labels, []string{"gift", "fragile"}) {
		t.Errorf("the labels are %q, want gift and fragile", got.Labels)
	}
	if !slices.Equal(got.Seen, []string{"welcome"}) {
		t.Errorf("what was seen is %q, want the one cookie", got.Seen)
	}
	if len(got.Sizes) != 2 || *got.Sizes[0] != 7 || *got.Sizes[1] != 9 {
		t.Errorf("the sizes are %v, want 7 and 9 with the empty one left out", got.Sizes)
	}
	if len(got.Names) != 2 || *got.Names[0] != "box" || *got.Names[1] != "" {
		t.Errorf("the names are %v, want box and an empty one, since a string keeps its place", got.Names)
	}
	if got.Origin.String() != "192.0.2.7" {
		t.Errorf("the origin is %v, want 192.0.2.7", got.Origin)
	}
	if got.Address.City != "Kyoto" || got.Address.Country != "JP" {
		t.Errorf("the address is %+v, want Kyoto in JP", got.Address)
	}
	if got.Address.Zone == nil || got.Address.Zone.String() != "198.51.100.9" {
		t.Errorf("the zone is %v, want 198.51.100.9", got.Address.Zone)
	}
	if got.Ship == nil || got.Ship.City != "Osaka" || got.Ship.Zone != nil {
		t.Errorf("the shipping address is %+v, want one allocated with only a city in it", got.Ship)
	}
	if got.CouponCode != "WATER" {
		t.Errorf("the coupon is %q, want WATER, since the body wins over the query string", got.CouponCode)
	}
	if got.hidden != "" {
		t.Errorf("an unexported field was filled in: %q", got.hidden)
	}
}

// The same struct off a multipart body, which is the other way a form arrives
// and the one net/http has to build a map for.
func TestOrderReadsAMultipartForm(t *testing.T) {
	make := func() *http.Request {
		var body strings.Builder
		w := multipart.NewWriter(&body)
		for name, all := range map[string][]string{
			"address.city": {"Kyoto"},
			"codes":        {"4", "5"},
			"coupon_code":  {"WATER"},
		} {
			for _, v := range all {
				if err := w.WriteField(name, v); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest("POST", "/orders/7?note=from-the-query", strings.NewReader(body.String()))
		r.Header.Set("Content-Type", w.FormDataContentType())
		return r
	}

	var got Order
	var ref orderRef
	ok(t, "POST /orders/{id:int}", make, func(c *web.Ctx) error { return c.Bind(&got) })
	ok(t, "POST /orders/{id:int}", make, func(c *web.Ctx) error { return c.Bind(&ref) })

	if !reflect.DeepEqual(got, Order(ref)) {
		t.Errorf("the two binders disagree\ngenerated %+v\nreflected %+v", got, Order(ref))
	}
	if got.ID != 7 || got.Address.City != "Kyoto" || !slices.Equal(got.Codes, []int{4, 5}) {
		t.Errorf("read %+v, want 7, Kyoto, and 4 and 5", got)
	}
	if got.Note == nil || *got.Note != "from-the-query" {
		t.Errorf("the note is %v, want the query string's value, since the body left it out", got.Note)
	}
}

func TestProfileReadsFiles(t *testing.T) {
	make := func() *http.Request {
		var body strings.Builder
		w := multipart.NewWriter(&body)
		if err := w.WriteField("name", "Mizu"); err != nil {
			t.Fatal(err)
		}
		for _, f := range []struct{ field, name, data string }{
			{"avatar", "face.png", "\x89PNG\r\n\x1a\n"},
			{"photos", "one.txt", "one"},
			{"photos", "two.txt", "two"},
			{"extra", "three.txt", "three"},
		} {
			part, err := w.CreateFormFile(f.field, f.name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write([]byte(f.data)); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest("POST", "/profile", strings.NewReader(body.String()))
		r.Header.Set("Content-Type", w.FormDataContentType())
		return r
	}

	var got Profile
	var ref profileRef
	ok(t, "POST /profile", make, func(c *web.Ctx) error { return c.Bind(&got) })
	ok(t, "POST /profile", make, func(c *web.Ctx) error { return c.Bind(&ref) })

	// Two uploads of one part are two different pointers, so this compares what
	// they say rather than what they are.
	names := func(p Profile) []string {
		out := []string{p.Name}
		if p.Avatar != nil {
			out = append(out, p.Avatar.Filename)
		}
		for _, f := range p.Photos {
			out = append(out, f.Filename)
		}
		for _, f := range p.Extra {
			out = append(out, f.Filename)
		}
		return out
	}
	if a, b := names(got), names(Profile(ref)); !slices.Equal(a, b) {
		t.Errorf("the two binders disagree\ngenerated %q\nreflected %q", a, b)
	}
	if got.Name != "Mizu" {
		t.Errorf("the name is %q, want Mizu", got.Name)
	}
	if got.Avatar == nil || got.Avatar.Filename != "face.png" {
		t.Fatalf("the avatar is %v, want face.png", got.Avatar)
	}
	if got.Avatar.MIME != "image/png" {
		t.Errorf("the avatar is a %q, want image/png read off the bytes", got.Avatar.MIME)
	}
	if len(got.Photos) != 2 || got.Photos[0].Filename != "one.txt" || got.Photos[1].Filename != "two.txt" {
		t.Errorf("the photos are %v, want one.txt and two.txt", got.Photos)
	}
	if len(got.Extra) != 1 || got.Extra[0].Filename != "three.txt" {
		t.Errorf("the extra files are %v, want three.txt in a named slice", got.Extra)
	}
}

func TestWebhookTakesAMemberNothingDeclared(t *testing.T) {
	make := func() *http.Request {
		r := httptest.NewRequest("POST", "/hooks", strings.NewReader(
			`{"event":"order.paid","kind":"live","from_a_later_version":true}`))
		r.Header.Set("Content-Type", "application/json")
		return r
	}

	var got Webhook
	var ref webhookRef
	ok(t, "POST /hooks", make, func(c *web.Ctx) error { return c.Bind(&got) })
	ok(t, "POST /hooks", make, func(c *web.Ctx) error { return c.Bind(&ref) })

	if !reflect.DeepEqual(got, Webhook(ref)) {
		t.Errorf("the two binders disagree\ngenerated %+v\nreflected %+v", got, Webhook(ref))
	}
	if got.Event != "order.paid" || got.Kind != "live" {
		t.Errorf("read %+v, want order.paid and live", got)
	}
}

// A struct with no AllowUnknown in it rejects a member nobody declared, and the
// generated binder hands the body to the same decoder, so it does too.
func TestTreeRejectsAMemberNothingDeclared(t *testing.T) {
	make := func() *http.Request {
		r := httptest.NewRequest("POST", "/trees", strings.NewReader(`{"name":"root","nope":1}`))
		r.Header.Set("Content-Type", "application/json")
		return r
	}

	gotErr := bind(t, "POST /trees", make, func(c *web.Ctx) error { var v Tree; return c.Bind(&v) })
	refErr := bind(t, "POST /trees", make, func(c *web.Ctx) error { var v treeRef; return c.Bind(&v) })

	if gotErr == nil || refErr == nil {
		t.Fatalf("generated %v and reflected %v, want both to refuse the member", gotErr, refErr)
	}
	if a, b := errs.CodeOf(gotErr), errs.CodeOf(refErr); a != b {
		t.Errorf("the codes are %q and %q, want the same one", a, b)
	}
}

// The tree binds the body it was given even though nothing in it can arrive on
// a query string, since the decoder handles the nesting without a plan.
func TestTreeBindsTheBody(t *testing.T) {
	make := func() *http.Request {
		r := httptest.NewRequest("POST", "/trees", strings.NewReader(
			`{"name":"root","children":[{"name":"leaf"}]}`))
		r.Header.Set("Content-Type", "application/json")
		return r
	}

	var got Tree
	var ref treeRef
	ok(t, "POST /trees", make, func(c *web.Ctx) error { return c.Bind(&got) })
	ok(t, "POST /trees", make, func(c *web.Ctx) error { return c.Bind(&ref) })

	if !reflect.DeepEqual(got, Tree(ref)) {
		t.Errorf("the two binders disagree\ngenerated %+v\nreflected %+v", got, Tree(ref))
	}
	if got.Name != "root" || len(got.Children) != 1 || got.Children[0].Name != "leaf" {
		t.Errorf("read %+v, want a root with one leaf under it", got)
	}
}

func TestBadValuesAreReportedTheSameWay(t *testing.T) {
	make := func() *http.Request {
		return httptest.NewRequest("GET", "/search?page=x&limit=y&since=nope&score=z", nil)
	}

	gotErr := bind(t, "GET /search", make, func(c *web.Ctx) error { var v Listing; return c.Bind(&v) })
	refErr := bind(t, "GET /search", make, func(c *web.Ctx) error { var v listingRef; return c.Bind(&v) })

	if gotErr == nil || refErr == nil {
		t.Fatalf("generated %v and reflected %v, want both to report the values", gotErr, refErr)
	}
	if a, b := errs.KindOf(gotErr), errs.KindOf(refErr); a != b {
		t.Errorf("the kinds are %v and %v, want the same one", a, b)
	}

	// The generated binder reports in the order the request sent the names and
	// the reflective one in the order the struct declares them, so this compares
	// them as sets.
	byName := func(err error) []errs.Field {
		out := slices.Clone(errs.Fields(err))
		slices.SortFunc(out, func(a, b errs.Field) int { return strings.Compare(a.Name, b.Name) })
		return out
	}
	if a, b := byName(gotErr), byName(refErr); !reflect.DeepEqual(a, b) {
		t.Errorf("the two binders disagree\ngenerated %+v\nreflected %+v", a, b)
	}
	if n := len(errs.Fields(gotErr)); n != 4 {
		t.Errorf("%d fields were reported, want all 4, rather than stopping at the first", n)
	}
}
