package micro

import (
	"testing"

	"github.com/go-mizu/mizu/validate"
)

func init() {
	register("validate/report", benchValidateReport)
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
