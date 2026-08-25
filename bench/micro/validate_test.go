package micro

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/validate"
)

func init() {
	register("validate/report", benchValidateReport)
	register("validate/format", benchValidateFormat)
	register("validate/build", benchValidateBuild)
	register("validate/reflect", benchValidateReflect)
}

// benchValidateBuild is a builder doing what a handler would: four fields, ten
// rules, everything passing.
//
// The builder is the mode for rules a tag cannot say, so it is the one a
// handler writes by hand and the one nobody generates. Every rule reads the
// value's type at runtime, where a generated validator knows it at build time,
// and that is the cost the row is here to show.
//
// It comes to 3 allocations, which is the V, the slice literal in the
// benchmark, and the closure passed to When. The Check a chain runs through
// does not escape, so a field costs nothing to start. That is worth pinning:
// one more field on Check, or one method that takes its address somewhere the
// compiler cannot see, turns 3 into one allocation per field.
func benchValidateBuild(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		v := validate.New()
		v.Field("email", "first.last@example.com").Required().Email()
		v.Field("title", "A reasonable title").Required().Min(3).Max(120)
		v.Field("tags", []string{"go", "web"}).Required().Max(5)
		v.Field("website", "").Optional().URL()
		v.When(true, func(v *validate.V) {
			v.Field("id", "01ARZ3NDEKTSV4RRFFQ69G5FAV").Required().ULID()
		})
		if err := v.Err(); err != nil {
			b.Fatal(err)
		}
	}
}

// signup is the struct the reflective interpreter and the generated validator
// are both measured on: 12 fields and 20 rules, which is a form somebody would
// actually post.
type signup struct {
	Email string   `json:"email" validate:"required,email"`
	Title string   `json:"title" validate:"required,min=3,max=120"`
	Slug  string   `json:"slug" validate:"required"`
	Site  string   `json:"site" validate:"omitempty,url"`
	IP    string   `json:"ip" validate:"omitempty,ip"`
	ID    string   `json:"id" validate:"required,ulid"`
	Ref   string   `json:"ref" validate:"uuid"`
	Body  string   `json:"body" validate:"required,min=10"`
	Tags  []string `json:"tags" validate:"required,max=5"`
	Count int      `json:"count" validate:"between=1 10"`
	Phone string   `json:"phone" validate:"e164"`
	Host  string   `json:"host" validate:"hostname"`
}

// benchValidateReflect is the same work as validate/gen through struct tags
// read at runtime, and the pair is the whole argument for the generator.
//
// The rules a tag names are resolved once per type and kept, so what a call
// costs is walking the plan, pulling each field out by index into an interface,
// and running the same Check chain a handler would have written. The reflection
// that is left is the field access, and that is what the row measures.
//
// There are 16 allocations for 12 fields, and most of them are a field's value
// going into an interface so that a Check can hold it. That is the cost the
// generator does not pay, and it is the whole of the difference between this
// row and validate/gen. A count that climbs faster than the field count means
// the plan is being rebuilt on every call, which would be a bug rather than a
// cost.
func benchValidateReflect(b *testing.B) {
	ctx := context.Background()
	in := signup{
		Email: "first.last@example.com",
		Title: "A reasonable title",
		Slug:  "a-reasonable-title",
		Site:  "https://example.com/a/b?q=1",
		IP:    "2001:db8::1",
		ID:    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Ref:   "f47ac10b-58cc-4372-a567-0e02b2c3d479",
		Body:  "Long enough to pass the minimum.",
		Tags:  []string{"go", "web"},
		Count: 3,
		Phone: "+14155552671",
		Host:  "mail.example.com",
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := validate.Struct(ctx, in); err != nil {
			b.Fatal(err)
		}
	}
}

// benchValidateReport is what a rejected form costs after the checks have run:
// three failures collected, three sentences written, and one errs.Error built
// with a field per rule.
//
// It is the failure path, and the failure path of validation is not rare. A
// public form gets more bad submissions than good ones, and a request that is
// rejected has still been read, decoded and bound before it arrives here.
//
// What the number is watching for is a message table that starts allocating,
// or an Errors that starts copying its map. The passing path is measured by
// validate/gen and validate/reflect, which arrive with the rules.
func benchValidateReport(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var bad validate.Errors
		bad.Add("title", validate.Failed("min", 3).Of("string"))
		bad.Add("body", validate.Failed("required"))
		bad.Add("publish_at", validate.Failed("required"))
		if err := bad.OrNil(); err == nil {
			b.Fatal("nothing failed")
		}
	}
}

// benchValidateFormat is the passing path of the four format checks a signup
// form is most likely to run: an address, a link, an identifier and a network
// address.
//
// Three of the four are scans over the bytes and allocate nothing. The two
// allocations are net/url building a URL, which is the one check here that
// hands off to the standard library, and they are the reason the whole group
// costs what it does.
//
// What the number is watching for is a regexp. That is the obvious way to
// write any of these, it costs an order of magnitude more, and it allocates on
// every call. A budget is how that stays a decision somebody makes on purpose
// rather than one that arrives in a patch.
func benchValidateFormat(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if !validate.IsEmail("first.last@mail.example.com") {
			b.Fatal("email")
		}
		if !validate.IsURL("https://example.com:8443/a/b?q=1") {
			b.Fatal("url")
		}
		if !validate.IsUUID("f47ac10b-58cc-4372-a567-0e02b2c3d479") {
			b.Fatal("uuid")
		}
		if !validate.IsIP("2001:db8::1") {
			b.Fatal("ip")
		}
	}
}
