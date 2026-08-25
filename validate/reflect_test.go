package validate

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// failures is what an error says failed, as "field:rule", in the order the
// fields first failed.
func failures(t *testing.T, err error) []string {
	t.Helper()
	if err == nil {
		return nil
	}
	var bad *Errors
	if !errors.As(err, &bad) {
		t.Fatalf("not a validation error: %v", err)
	}

	var out []string
	for field, rules := range bad.Fields() {
		for _, r := range rules {
			out = append(out, field+":"+r.Rule)
		}
	}
	return out
}

func wantFailures(t *testing.T, value any, want ...string) {
	t.Helper()
	err := Struct(context.Background(), value)
	if got := failures(t, err); !slices.Equal(got, want) {
		t.Errorf("%T failed %v, want %v", value, got, want)
	}
}

// wantTagError is a struct this cannot check at all, which is a mistake in the
// program and not in a request.
func wantTagError(t *testing.T, value any, says string) {
	t.Helper()

	err := Struct(context.Background(), value)
	if err == nil {
		t.Fatalf("%T came back with no error", value)
	}
	if kind := errs.KindOf(err); kind != errs.Internal {
		t.Errorf("%T came back %v, want internal", value, kind)
	}
	if !strings.Contains(err.Error(), says) {
		t.Errorf("%T said %q, want it to mention %q", value, err, says)
	}
}

type post struct {
	Title string   `json:"title" validate:"required,min=3,max=200"`
	Body  string   `json:"body" validate:"required"`
	Site  string   `json:"site" validate:"omitempty,url"`
	Tags  []string `json:"tags" validate:"max=2,dive,email"`
	Notes string
}

func TestStruct(t *testing.T) {
	wantFailures(t, post{Title: "Hello", Body: "Words."})

	wantFailures(t, post{Title: "Hi", Site: "nope", Tags: []string{"a@b.com", "nope"}},
		"title:min", "body:required", "site:url", "tags.1:email")

	// max on the field runs before dive does, and a chain that has failed does
	// not go on to check the elements of a list that is already too long.
	wantFailures(t, post{Title: "Hello", Body: "Words.", Tags: []string{"a", "b", "c"}},
		"tags:max")
}

func TestStructTakesAStructOrAPointerToOne(t *testing.T) {
	in := post{Title: "Hello", Body: "Words."}
	wantFailures(t, in)
	wantFailures(t, &in)

	p := &in
	wantFailures(t, &p)
}

func TestStructOnSomethingThatIsNotAStruct(t *testing.T) {
	for _, value := range []any{nil, 7, "words", []int{1}, (*post)(nil)} {
		err := Struct(context.Background(), value)
		if errs.KindOf(err) != errs.Internal {
			t.Errorf("Struct(%#v) came back %v, want internal", value, errs.KindOf(err))
		}
		if !strings.Contains(err.Error(), "not a struct") {
			t.Errorf("Struct(%#v) said %q", value, err)
		}
	}
}

func TestAStructWithNoTagsPasses(t *testing.T) {
	type plain struct {
		Title string
		When  time.Time
	}
	wantFailures(t, plain{})
}

// The name a failure comes back under is the name the request used, which is
// the name web.Bind reads the field from.
func TestFieldNames(t *testing.T) {
	type named struct {
		FromJSON   string `json:"from_json" validate:"required"`
		FromForm   string `form:"from-form" validate:"required"`
		FromQuery  string `query:"q" validate:"required"`
		FromPath   string `path:"id" validate:"required"`
		FromHeader string `header:"X-Trace" validate:"required"`
		FromCookie string `cookie:"sid" validate:"required"`
		FromBare   string `form:"" validate:"required"`
		PerPage    string `validate:"required"`
		IDToken    string `validate:"required"`
		Skipped    string `json:"-" validate:"required"`
		Excused    string `validate:"-"`
		Handled    string `bind:"-" validate:"required"`
		Ignored    string `form:"-" validate:"required"`
	}

	wantFailures(t, named{},
		"from_json:required",
		"from-form:required",
		"q:required",
		"id:required",
		"X-Trace:required",
		"sid:required",
		"from_bare:required",
		"per_page:required",
		"id_token:required",
		"handled:required",
	)
}

// A form tag with a comma after the name is the name in front of the comma,
// which is how encoding/json spells options and how the binder reads them.
func TestATagNameStopsAtItsFirstComma(t *testing.T) {
	type opts struct {
		A string `json:"a,omitempty" validate:"required"`
		B string `json:",omitempty" validate:"required"`
	}
	wantFailures(t, opts{}, "a:required", "b:required")
}

func TestNestedStructs(t *testing.T) {
	type address struct {
		City string `json:"city" validate:"required"`
	}
	type inner struct {
		Note string `json:"note" validate:"required"`
	}
	type order struct {
		inner
		Ship  address  `json:"ship"`
		Bill  *address `json:"bill" validate:"required"`
		Where time.Time
	}

	// An embedded struct's fields are named as if they were the outer
	// struct's, and a named one's are named under it.
	wantFailures(t, order{}, "note:required", "ship.city:required", "bill:required")

	// A struct that is there is checked whether or not the field holding it
	// passed its own rule. bill and bill.city are two names a form has two
	// inputs for, and marking both says which one to fill in.
	wantFailures(t, order{inner: inner{Note: "n"}, Ship: address{City: "c"}, Bill: &address{}},
		"bill:required", "bill.city:required")

	wantFailures(t, order{inner: inner{Note: "n"}, Ship: address{City: "c"}, Bill: &address{City: "c"}})
}

// A nil pointer stops there. Whether it was allowed to be nil is the pointer's
// own rule to answer, and the fields under it are not there to be wrong.
func TestANilStructPointerLeavesTheFieldsUnderItAlone(t *testing.T) {
	type address struct {
		City string `json:"city" validate:"required"`
	}
	type order struct {
		Bill *address `json:"bill"`
	}

	wantFailures(t, order{})
	wantFailures(t, order{Bill: &address{}}, "bill.city:required")
}

func TestATypeThatHoldsOneOfItselfStops(t *testing.T) {
	type node struct {
		Name string `json:"name" validate:"required"`
		Next *node  `json:"next"`
	}
	wantFailures(t, node{Next: &node{}}, "name:required")
}

func TestDive(t *testing.T) {
	type basket struct {
		Codes []string  `json:"codes" validate:"dive,required,min=2"`
		Ports [2]string `json:"ports" validate:"dive,port"`
		Sites *[]string `json:"sites" validate:"dive,url"`
	}

	sites := []string{"https://example.com", "nope"}
	wantFailures(t, basket{
		Codes: []string{"ab", "", "c"},
		Ports: [2]string{"80", "0"},
		Sites: &sites,
	},
		"codes.1:required",
		"codes.2:min",
		"ports.1:port",
		"sites.1:url",
	)

	// An array always has its slots, so diving into one checks every element
	// whether anything was sent or not. A slot that may be blank writes
	// omitempty in front of its rules, the same as a field that may be blank.
	wantFailures(t, basket{}, "ports.0:port", "ports.1:port")
}

func TestDiveIntoStructs(t *testing.T) {
	type line struct {
		SKU   string `json:"sku" validate:"required"`
		Count int    `json:"count" validate:"min=1"`
	}
	type cart struct {
		Lines []line  `json:"lines" validate:"required,dive"`
		Extra []*line `json:"extra" validate:"dive"`
	}

	wantFailures(t, cart{
		Lines: []line{{SKU: "a", Count: 1}, {Count: 0}},
		Extra: []*line{nil, {SKU: "b"}},
	},
		"lines.1.sku:required",
		"lines.1.count:min",
		"extra.1.count:min",
	)
}

// A type holding a list of itself is checked all the way down. The plan for
// the element is the plan holding it, so the recursion runs over the values
// that arrived and an empty list is where it stops.
func TestDiveIntoAListOfItself(t *testing.T) {
	type tree struct {
		Name string `json:"name" validate:"required"`
		Kids []tree `json:"kids" validate:"dive"`
	}

	wantFailures(t, tree{Kids: []tree{{Kids: []tree{{}}}}},
		"name:required", "kids.0.name:required", "kids.0.kids.0.name:required")
	wantFailures(t, tree{Name: "root", Kids: []tree{{Name: "leaf"}}})
}

// Two types that hold lists of each other are the same thing a level further
// out, and they stop the same way.
func TestDiveIntoAPairOfTypesThatHoldEachOther(t *testing.T) {
	type folder struct {
		Name  string   `json:"name" validate:"required"`
		Files []string `json:"files" validate:"dive,required"`
	}
	type drive struct {
		Label   string   `json:"label" validate:"required"`
		Folders []folder `json:"folders" validate:"dive"`
	}

	wantFailures(t, drive{Folders: []folder{{Files: []string{""}}}},
		"label:required", "folders.0.name:required", "folders.0.files.0:required")
}

func TestABoundKeepsTheFieldsType(t *testing.T) {
	type job struct {
		Every time.Duration `json:"every" validate:"min=1h"`
		Tries int           `json:"tries" validate:"between=1 5"`
		Ratio float64       `json:"ratio" validate:"max=0.5"`
	}

	err := Struct(context.Background(), job{Every: time.Minute, Tries: 9, Ratio: 0.9})

	var bad *Errors
	if !errors.As(err, &bad) {
		t.Fatalf("not a validation error: %v", err)
	}
	want := []string{
		"Every must be at least 1h0m0s.",
		"Tries must be between 1 and 5.",
		"Ratio must not be greater than 0.5.",
	}
	var got []string
	for _, field := range []string{"every", "tries", "ratio"} {
		got = append(got, bad.First(field))
	}
	if !slices.Equal(got, want) {
		t.Errorf("said %q, want %q", got, want)
	}
}

func TestAChainFromATagStopsAtItsFirstFailure(t *testing.T) {
	type in struct {
		Email string `json:"email" validate:"required,email,min=5"`
	}
	wantFailures(t, in{}, "email:required")
	wantFailures(t, in{Email: "nope"}, "email:email")
	wantFailures(t, in{Email: "a@b.com"})
}

func TestABadTagIsAnInternalError(t *testing.T) {
	type unknown struct {
		A string `validate:"requird"`
	}
	type malformed struct {
		A string `validate:"min="`
	}
	type notANumber struct {
		A string `validate:"min=x"`
	}
	type notATime struct {
		A time.Duration `validate:"min=x"`
	}
	type tooManyBounds struct {
		A string `validate:"min=1 2"`
	}
	type tooFewBounds struct {
		A string `validate:"between=1"`
	}
	type badLowBound struct {
		A string `validate:"between=x 5"`
	}
	type badHighBound struct {
		A string `validate:"between=1 x"`
	}
	type unwantedParams struct {
		A string `validate:"required=yes"`
	}
	type formatOnANumber struct {
		A int `validate:"email"`
	}
	type sizeOnATime struct {
		A time.Time `validate:"min=3"`
	}
	type diveOnAString struct {
		A string `validate:"dive,email"`
	}
	type diveOnAMap struct {
		A map[string]string `validate:"dive,email"`
	}
	type twoDives struct {
		A [][]string `validate:"dive,dive,email"`
	}
	type badElementTag struct {
		A []int `validate:"dive,email"`
	}
	type badNestedTag struct {
		B struct {
			C string `validate:"nope"`
		}
	}
	type badTagInADivedStruct struct {
		A []struct {
			C string `validate:"nope"`
		} `validate:"dive"`
	}

	cases := []struct {
		value any
		says  string
	}{
		{unknown{}, "there is no rule called requird"},
		{malformed{}, "min= has nothing after it"},
		{notANumber{}, "x is not a number"},
		{notATime{}, "x is not a length of time"},
		{tooManyBounds{}, "min takes one bound"},
		{tooFewBounds{}, "between takes two bounds"},
		{badLowBound{}, "x is not a number"},
		{badHighBound{}, "x is not a number"},
		{unwantedParams{}, "required takes no parameters"},
		{formatOnANumber{}, "email is a check on a string and this field is a int"},
		{sizeOnATime{}, "min counts something and a time.Time has nothing to count"},
		{diveOnAString{}, "dive is for a slice or an array and this field is a string"},
		{diveOnAMap{}, "dive is for a slice or an array and this field is a map"},
		{twoDives{}, "a second dive"},
		{badElementTag{}, "email is a check on a string and this field is a int"},
		{badNestedTag{}, "there is no rule called nope"},
		{badTagInADivedStruct{}, "there is no rule called nope"},
	}

	for _, c := range cases {
		wantTagError(t, c.value, c.says)
	}
}

// Only the first bad tag is reported. The sentence is for whoever wrote the
// struct and the first field they got wrong is where they start reading.
func TestOnlyTheFirstBadTagIsReported(t *testing.T) {
	type two struct {
		A string `validate:"nope"`
		B string `validate:"alsonope"`
	}
	wantTagError(t, two{}, "there is no rule called nope")
}

func TestAPlanIsBuiltOnceAndKept(t *testing.T) {
	type kept struct {
		A string `validate:"required"`
	}
	t.Cleanup(func() { plans.Delete(reflect.TypeFor[kept]()) })

	first := planFor(reflect.TypeFor[kept]())
	if second := planFor(reflect.TypeFor[kept]()); second != first {
		t.Errorf("planFor built a second plan for the same type")
	}
}

// Every format check is a tag rule under the name it already has, because the
// list a tag reads is the table in format.go rather than a second copy of it.
func TestEveryFormatIsATagRule(t *testing.T) {
	for name := range formats {
		if _, ok := plainRules[name]; !ok {
			t.Errorf("no tag rule called %s", name)
		}
	}
	if got, want := len(plainRules), len(formats)+2; got != want {
		t.Errorf("%d rules take no parameters, want %d, which is the formats plus required and omitempty", got, want)
	}
}

func TestTextualAndCountable(t *testing.T) {
	type word string

	cases := []struct {
		value              any
		textual, countable bool
	}{
		{"", true, true},
		{word(""), true, true},
		{(*string)(nil), true, true},
		{any(nil), true, true},
		{0, false, true},
		{uint8(0), false, true},
		{0.0, false, true},
		{[]int(nil), false, true},
		{map[string]int(nil), false, true},
		{[2]int{}, false, true},
		{time.Duration(0), false, true},
		{time.Time{}, false, false},
		{false, false, false},
		{struct{}{}, false, false},
	}

	for _, c := range cases {
		// A field declared as any arrives here as a nil type, since there is
		// no value to take one from.
		ft := reflect.TypeOf(c.value)
		if ft == nil {
			ft = reflect.TypeFor[any]()
		}
		if got := textual(ft); got != c.textual {
			t.Errorf("textual(%v) = %v, want %v", ft, got, c.textual)
		}
		if got := countable(ft); got != c.countable {
			t.Errorf("countable(%v) = %v, want %v", ft, got, c.countable)
		}
	}
}

func TestKindOf(t *testing.T) {
	cases := []struct {
		t    reflect.Type
		want string
	}{
		{reflect.TypeFor[string](), "string"},
		{reflect.TypeFor[*time.Time](), "time.Time"},
		{reflect.TypeFor[[]string](), "slice"},
		{reflect.TypeFor[map[string]int](), "map"},
		{reflect.TypeFor[struct{ A int }](), "struct"},
	}

	for _, c := range cases {
		if got := kindOf(c.t); got != c.want {
			t.Errorf("kindOf(%v) = %q, want %q", c.t, got, c.want)
		}
	}
}

// A tag is arbitrary text off a struct somebody wrote, and reading one is not
// allowed to bring the program down whatever it says.
func FuzzParseTag(f *testing.F) {
	for _, tag := range []string{
		"", "required", "required,min=3,max=200", "max=5,dive,email",
		"=", ",", "=3", "min=", `a\`, `oneof=a\,b c`, "between=1 5",
	} {
		f.Add(tag)
	}

	f.Fuzz(func(t *testing.T, tag string) {
		rules, err := parseTag(tag)
		if err != nil {
			return
		}
		for _, r := range rules {
			if r.name == "" {
				t.Errorf("parseTag(%q) gave a rule with no name", tag)
			}
			if r.params != nil && len(r.params) == 0 {
				t.Errorf("parseTag(%q) gave %s an equals sign and no parameters", tag, r.name)
			}
		}
	})
}
