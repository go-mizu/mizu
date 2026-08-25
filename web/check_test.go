package web

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/validate"
)

// A signup is a form with rules on it, written the way a handler would write
// one: the tag that names the field for the request and the tag that says what
// it takes, on the same line.
type signup struct {
	Email string `form:"email" validate:"required,email"`
	Name  string `form:"name" validate:"required,min=2"`
	Site  string `form:"site" validate:"omitempty,url"`
	Age   int    `form:"age" validate:"omitempty,between=13 120"`
}

func TestBindChecksTheTags(t *testing.T) {
	_, err := bind[signup](t, form("POST", url.Values{
		"email": {"not an address"},
		"name":  {"a"},
		"age":   {"9"},
	}))
	if err == nil {
		t.Fatal("a form with three things wrong with it came back with nothing wrong with it")
	}
	if !errors.Is(err, errs.Unprocessable) {
		t.Errorf("the kind is %v, want Unprocessable, which is a 422", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != validate.Code {
		t.Errorf("the code is %q, want %q", got, validate.Code)
	}

	want := map[string]string{"email": "email", "name": "min", "age": "between"}
	got := map[string]string{}
	for _, f := range errs.Fields(err) {
		got[f.Name] = f.Code
		if f.Msg == "" {
			t.Errorf("%s came back with no sentence on it", f.Name)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("reported %v, want %v", got, want)
	}
	for name, code := range want {
		if got[name] != code {
			t.Errorf("%s failed %q, want %q", name, got[name], code)
		}
	}
}

// A form the rules are happy with binds, which is the case that has to stay
// quiet however many rules are on the struct.
func TestBindPassesAFormThatIsRight(t *testing.T) {
	in, err := bind[signup](t, form("POST", url.Values{
		"email": {"sam@example.com"},
		"name":  {"Sam"},
		"site":  {"https://example.com"},
		"age":   {"31"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if in.Name != "Sam" || in.Age != 31 {
		t.Errorf("read %+v, want the form back as it was sent", in)
	}
}

// A field the request did not bind is at its zero value, so checking the struct
// would say a box is empty that somebody filled in. The 400 is the answer, and
// it is the only one.
func TestBindDoesNotCheckWhatItCouldNotBind(t *testing.T) {
	_, err := bind[signup](t, form("POST", url.Values{
		"email": {"sam@example.com"},
		"age":   {"twelve"},
	}))
	if err == nil {
		t.Fatal("a form with a word where a number goes bound without complaint")
	}
	if !errors.Is(err, errs.Invalid) {
		t.Errorf("the kind is %v, want Invalid, which is the 400 binding gives", errs.KindOf(err))
	}

	for _, f := range errs.Fields(err) {
		if f.Name != "age" {
			t.Errorf("%s failed %q, and the only thing wrong with this request is age", f.Name, f.Code)
		}
	}
}

// A checked is a struct with a method of its own, which is what mizu
// gen:validate writes and what somebody writes by hand for a rule that reads
// two fields at once.
type checked struct {
	From int `query:"from" validate:"required"`
	To   int `query:"to"`

	ran *bool
}

func (v checked) Validate(ctx context.Context) error {
	if v.ran != nil {
		*v.ran = true
	}
	var bad validate.Errors
	if v.To < v.From {
		bad.Add("to", validate.Failed("after", "from"))
	}
	return bad.OrNil()
}

// The method is the whole answer. The tag on From says required and the method
// says nothing about it, so a request with no from in it passes: running both
// would report a generated method's failures twice, and a hand-written one is
// somebody taking the field rules over.
func TestBindAsksTheMethodAndNotTheTags(t *testing.T) {
	ran := false
	var in checked
	serve(t, httptest.NewRequest("GET", "/?to=3", nil), func(c *Ctx) error {
		in.ran = &ran
		if err := BindInto(c, &in); err != nil {
			t.Errorf("the method said nothing was wrong and Bind said %v", err)
		}
		return nil
	})
	if !ran {
		t.Error("the method did not run, so the tags were read instead")
	}
}

func TestBindReportsWhatTheMethodReports(t *testing.T) {
	var in checked
	var err error
	serve(t, httptest.NewRequest("GET", "/?from=9&to=3", nil), func(c *Ctx) error {
		err = BindInto(c, &in)
		return nil
	})
	if err == nil {
		t.Fatal("a range that runs backwards came back with nothing wrong with it")
	}
	if f := errs.Fields(err); len(f) != 1 || f[0].Name != "to" || f[0].Code != "after" {
		t.Errorf("the failures are %v, want one on to", f)
	}
}

// A method that could not answer is not a request that was wrong. Whatever it
// returns travels up as it is, so an outage behind a rule is not a 422 telling
// somebody to fix a field that is fine.
type unreachable struct {
	Q string `query:"q"`
}

var errRegistryDown = errors.New("the registry did not answer")

func (v unreachable) Validate(ctx context.Context) error { return errRegistryDown }

func TestBindHandsBackWhatTheMethodCouldNotAnswer(t *testing.T) {
	_, err := bind[unreachable](t, httptest.NewRequest("GET", "/?q=water", nil))
	if !errors.Is(err, errRegistryDown) {
		t.Errorf("the error is %v, want the one the method returned", err)
	}
}

// A binder writes the fields and the rules are read afterwards, so a struct
// gets the same answer whichever half of the pair is generated.
type stocked struct {
	Q   string `query:"q" validate:"required,min=2"`
	Ran bool
}

func (v *stocked) BindRequest(c *Ctx) error {
	b := c.Binding()
	v.Ran = true
	for name, value := range b.Values() {
		if name == "q" {
			v.Q = value
		}
	}
	b.Body(v)
	return b.Err()
}

func TestAGeneratedBinderIsCheckedToo(t *testing.T) {
	in, err := bind[stocked](t, httptest.NewRequest("GET", "/?q=a", nil))
	if err == nil {
		t.Fatalf("read %+v, want the one character to have been refused", in)
	}
	if !errors.Is(err, errs.Unprocessable) {
		t.Errorf("the kind is %v, want Unprocessable", errs.KindOf(err))
	}
	if f := errs.Fields(err); len(f) != 1 || f[0].Name != "q" || f[0].Code != "min" {
		t.Errorf("the failures are %v, want one on q", f)
	}
}

// Most structs have no rules on them, and every one of them goes through this
// step on every request. It has to cost nothing, so the number is pinned here
// rather than left to the benchmark to notice a year later.
func TestAStructWithNoRulesCostsNothing(t *testing.T) {
	r := httptest.NewRequest("GET", "/?q=water&page=3", nil)
	serve(t, r, func(c *Ctx) error {
		var in search
		if n := testing.AllocsPerRun(100, func() { _ = c.check(&in) }); n != 0 {
			t.Errorf("checking a struct with no rules allocated %v times, want 0", n)
		}
		return nil
	})
}
