package micro

import (
	"testing"

	"github.com/go-mizu/mizu/validate"
)

func init() {
	register("validate/report", benchValidateReport)
	register("validate/format", benchValidateFormat)
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
