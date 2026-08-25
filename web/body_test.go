package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

// jsonRequest is a POST carrying a JSON body.
func jsonRequest(body string) *http.Request {
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// xmlRequest is a POST carrying an XML body.
func xmlRequest(body string) *http.Request {
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/xml")
	return r
}

// post is a article the way a client sends one, in every shape it sends it.
type post struct {
	Title string   `json:"title" xml:"title"`
	Body  string   `json:"body" xml:"body"`
	Tags  []string `json:"tags" xml:"tags"`
	Draft bool     `json:"draft" xml:"draft"`
}

// firstField is the one field error in err, and it fails the test when there is
// not exactly one.
func firstField(t *testing.T, err error) errs.Field {
	t.Helper()

	got := errs.Fields(err)
	if len(got) != 1 {
		t.Fatalf("the error carries %d fields, want one: %v", len(got), err)
	}
	return got[0]
}

func TestBindReadsAJSONBody(t *testing.T) {
	in, err := bind[post](t, jsonRequest(`{"title":"water","body":"is water","tags":["go"],"draft":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "water" || in.Body != "is water" || !in.Draft {
		t.Errorf("the body read as %+v", in)
	}
	if len(in.Tags) != 1 || in.Tags[0] != "go" {
		t.Errorf("the tags are %v, want one go", in.Tags)
	}
}

func TestBindReadsAnXMLBody(t *testing.T) {
	in, err := bind[post](t, xmlRequest(`<post><title>water</title><body>is water</body></post>`))
	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "water" || in.Body != "is water" {
		t.Errorf("the body read as %+v", in)
	}
}

func TestAJSONBodyWinsOverTheQueryString(t *testing.T) {
	r := httptest.NewRequest("POST", "/?title=from-query&body=from-query",
		strings.NewReader(`{"title":"from-body"}`))
	r.Header.Set("Content-Type", "application/json")

	in, err := bind[post](t, r)
	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "from-body" {
		t.Errorf("the title is %q, want the body to have won", in.Title)
	}
	if in.Body != "from-query" {
		t.Errorf("the body field is %q, want the query to have filled what the JSON left out", in.Body)
	}
}

func TestAMemberTheStructHasNoFieldForIsRejected(t *testing.T) {
	_, err := bind[post](t, jsonRequest(`{"titl":"water"}`))
	if errs.KindOf(err) != errs.Invalid {
		t.Fatalf("the error is %v, want one of kind Invalid", err)
	}

	f := firstField(t, err)
	if f.Name != "titl" || f.Code != "unknown_field" {
		t.Errorf("the field error is %+v, want titl and unknown_field", f)
	}
}

func TestAStructThatSaysSoTakesMembersItHasNoFieldFor(t *testing.T) {
	type webhook struct {
		AllowUnknown

		Event string `json:"event"`
	}

	in, err := bind[webhook](t, jsonRequest(`{"event":"paid","invented_last_tuesday":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if in.Event != "paid" {
		t.Errorf("the event is %q, want paid", in.Event)
	}
}

func TestAValueOfTheWrongTypeIsReportedAgainstItsMember(t *testing.T) {
	_, err := bind[post](t, jsonRequest(`{"title":7}`))
	if errs.KindOf(err) != errs.Invalid {
		t.Fatalf("the error is %v, want one of kind Invalid", err)
	}

	f := firstField(t, err)
	if f.Name != "title" || f.Code != "invalid_value" {
		t.Errorf("the field error is %+v, want title and invalid_value", f)
	}
}

func TestAMemberInsideAnObjectIsNamedTheWayANestedFieldIs(t *testing.T) {
	type address struct {
		City string `json:"city"`
	}
	type order struct {
		Ship address `json:"ship"`
	}

	_, err := bind[order](t, jsonRequest(`{"ship":{"city":7}}`))
	if f := firstField(t, err); f.Name != "ship.city" {
		t.Errorf("the field error names %q, want ship.city", f.Name)
	}
}

func TestABodyThatIsNotJSONAtAllSaysSoWithoutNamingAField(t *testing.T) {
	_, err := bind[post](t, jsonRequest(`{`))
	if errs.KindOf(err) != errs.Invalid || errs.CodeOf(err) != "bind.body" {
		t.Fatalf("the error is %v, want Invalid and bind.body", err)
	}
	if len(errs.Fields(err)) != 0 {
		t.Errorf("a body that would not parse named %d fields", len(errs.Fields(err)))
	}
}

func TestAnythingAfterTheJSONValueIsRefused(t *testing.T) {
	for _, body := range []string{
		`{"title":"water"}{"title":"again"}`,
		`{"title":"water"} }}}`,
	} {
		_, err := bind[post](t, jsonRequest(body))
		if errs.CodeOf(err) != "bind.body" {
			t.Errorf("%s: the error is %v, want bind.body", body, err)
		}
	}
}

func TestAnXMLBodyThatWillNotParseSaysSo(t *testing.T) {
	_, err := bind[post](t, xmlRequest(`<post><title>water`))
	if errs.KindOf(err) != errs.Invalid || errs.CodeOf(err) != "bind.body" {
		t.Fatalf("the error is %v, want Invalid and bind.body", err)
	}
}

func TestAContentTypeNothingReadsIsRefused(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("water"))
	r.Header.Set("Content-Type", "text/plain")

	_, err := bind[post](t, r)
	if errs.KindOf(err) != errs.Unsupported || errs.CodeOf(err) != "bind.content_type" {
		t.Fatalf("the error is %v, want Unsupported and bind.content_type", err)
	}
}

func TestABodyNothingSaidTheTypeOfIsLeftWhereItIs(t *testing.T) {
	r := httptest.NewRequest("POST", "/?title=from-query", strings.NewReader("water"))

	in, err := bind[post](t, r)
	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "from-query" {
		t.Errorf("the title is %q, want the query string to have been read", in.Title)
	}
}

func TestABodyOnAGetIsNotRead(t *testing.T) {
	r := httptest.NewRequest("GET", "/?title=from-query", strings.NewReader(`{"title":"from-body"}`))
	r.Header.Set("Content-Type", "application/json")

	in, err := bind[post](t, r)
	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "from-query" {
		t.Errorf("the title is %q, want a body on a GET to have been ignored", in.Title)
	}
}

func TestAJSONBodyOverTheLimitSaysSoRatherThanBindingNothing(t *testing.T) {
	r := jsonRequest(`{"title":"water is water is water is water"}`)

	var err error
	serve(t, r, func(c *Ctx) error {
		c.Request().Body = http.MaxBytesReader(c.Writer(), c.Request().Body, 8)
		_, err = Bind[post](c)
		return nil
	})

	if errs.KindOf(err) != errs.TooLarge || errs.CodeOf(err) != "bind.too_large" {
		t.Fatalf("the error is %v, want TooLarge and bind.too_large", err)
	}
}

func TestTheSuffixTypesAreReadAsWhatTheyEndIn(t *testing.T) {
	for _, c := range []struct {
		kind string
		body string
		want string
	}{
		{"application/vnd.example.v2+json", `{"title":"water"}`, "water"},
		{"application/atom+xml", `<post><title>water</title></post>`, "water"},
		{"text/xml; charset=utf-8", `<post><title>water</title></post>`, "water"},
		{"APPLICATION/JSON", `{"title":"water"}`, "water"},
	} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(c.body))
		r.Header.Set("Content-Type", c.kind)

		in, err := bind[post](t, r)
		if err != nil {
			t.Errorf("%s: %v", c.kind, err)
			continue
		}
		if in.Title != c.want {
			t.Errorf("%s: the title is %q, want %q", c.kind, in.Title, c.want)
		}
	}
}

func TestJSONReadsTheBodyAndNothingElse(t *testing.T) {
	r := httptest.NewRequest("POST", "/?title=from-query", strings.NewReader(`{"title":"from-body"}`))
	r.Header.Set("Content-Type", "application/json")

	var in post
	var err error
	serve(t, r, func(c *Ctx) error {
		err = c.JSON(&in)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "from-body" {
		t.Errorf("the title is %q, want from-body", in.Title)
	}
}

func TestJSONReadsABodySomethingElseAlreadyRead(t *testing.T) {
	var in post
	var err error
	serve(t, jsonRequest(`{"title":"water"}`), func(c *Ctx) error {
		if _, err = c.BodyBytes(); err != nil {
			return nil
		}
		err = c.JSON(&in)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "water" {
		t.Errorf("the title is %q, want the body to have been read twice", in.Title)
	}
}

func TestJSONReadsWhateverTheRequestSaidItWas(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":"water"}`))
	r.Header.Set("Content-Type", "text/plain")

	var in post
	var err error
	serve(t, r, func(c *Ctx) error {
		err = c.JSON(&in)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if in.Title != "water" {
		t.Errorf("the title is %q, want the body read as JSON anyway", in.Title)
	}
}

func TestJSONReportsAMemberTheTargetHasNoFieldFor(t *testing.T) {
	var in post
	var err error
	serve(t, jsonRequest(`{"titl":"water"}`), func(c *Ctx) error {
		err = c.JSON(&in)
		return nil
	})

	if f := firstField(t, err); f.Name != "titl" || f.Code != "unknown_field" {
		t.Errorf("the field error is %+v, want titl and unknown_field", f)
	}
}

func TestJSONIntoSomethingThatIsNotAStruct(t *testing.T) {
	var in map[string]any
	var err error
	serve(t, jsonRequest(`{"anything":"at all"}`), func(c *Ctx) error {
		err = c.JSON(&in)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if in["anything"] != "at all" {
		t.Errorf("the map read as %v", in)
	}
}

func TestJSONSaysSoWhenTheBodyWillNotParse(t *testing.T) {
	var in post
	var err error
	serve(t, jsonRequest(`{`), func(c *Ctx) error {
		err = c.JSON(&in)
		return nil
	})

	if errs.CodeOf(err) != "bind.body" {
		t.Fatalf("the error is %v, want bind.body", err)
	}
}

func TestTheAnswerAboutAllowUnknownIsRememberedPerType(t *testing.T) {
	type lax struct {
		AllowUnknown

		Q string `json:"q"`
	}
	type strict struct {
		Q string `json:"q"`
	}

	for range 2 {
		if !laxFor(reflect.TypeFor[lax]()) {
			t.Error("a struct embedding AllowUnknown was read as strict")
		}
		if laxFor(reflect.TypeFor[strict]()) {
			t.Error("a struct embedding nothing was read as lax")
		}
	}
	if laxFor(nil) {
		t.Error("a nil type was read as lax")
	}
	if laxFor(reflect.TypeFor[int]()) {
		t.Error("an int was read as lax")
	}
}

func TestAllowUnknownIsNotAFieldToBind(t *testing.T) {
	type lax struct {
		AllowUnknown

		Q string `query:"q"`
	}

	in, err := bind[lax](t, httptest.NewRequest("GET", "/?q=water&allow_unknown=x", nil))
	if err != nil {
		t.Fatal(err)
	}
	if in.Q != "water" {
		t.Errorf("the query is %q, want water", in.Q)
	}
}
