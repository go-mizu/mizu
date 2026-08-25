package validate

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

func TestLabel(t *testing.T) {
	cases := map[string]string{
		"":                  "This field",
		"title":             "Title",
		"publish_at":        "Publish at",
		"items.0.quantity":  "Items 0 quantity",
		"password_confirm":  "Password confirm",
		"Title":             "Title",
		"vat":               "Vat",
		"_leading":          " leading",
		"ünicode_is_a_word": "Ünicode is a word",
	}
	for field, want := range cases {
		if got := Label(field); got != want {
			t.Errorf("Label(%q) = %q, want %q", field, got, want)
		}
	}
}

func TestEnglishWritesTheSentence(t *testing.T) {
	cases := []struct {
		field string
		r     RuleError
		want  string
	}{
		{"title", Failed("required"), "Title is required."},
		{"title", Failed("min", 3).Of("string"), "Title must be at least 3 characters."},
		{"age", Failed("min", 18).Of("numeric"), "Age must be at least 18."},
		{"tags", Failed("max", 5).Of("array"), "Tags must not have more than 5 items."},
		{"avatar", Failed("max", 512).Of("file"), "Avatar must not be larger than 512 kilobytes."},
		{"wait", Failed("min", "1s").Of("duration"), "Wait must be at least 1s."},
		{"tags", Failed("between", 1, 5).Of("array"), "Tags must have between 1 and 5 items."},
		{"code", Failed("size", 6).Of("string"), "Code must be 6 characters."},
		{"publish_at", Failed("required"), "Publish at is required."},
	}
	for _, c := range cases {
		if got := English.Message(c.field, c.r); got != c.want {
			t.Errorf("Message(%q, %+v) = %q,\nwant %q", c.field, c.r, got, c.want)
		}
	}
}

// A rule with no entry is a missing translation and not a broken request, so
// it gets a sentence rather than a blank or the rule name shown to somebody
// who has never heard of it.
func TestARuleWithNoEntryStillGetsASentence(t *testing.T) {
	cases := []RuleError{
		Failed("vat", "GB"),                // a custom rule nobody registered a message for
		Failed("min", 3),                   // a size rule that forgot to say what it counts
		Failed("min", 3).Of("nonsense"),    // a subject that is not one of the five
		Failed("min"),                      // too few parameters for the template
		Failed("min", 3, 4).Of("string"),   // too many
		Failed("between", 1).Of("numeric"), // one short
		Failed("required", "unasked-for"),  // a rule that takes none
		Failed(""),                         //
		Failed("max", 5).Of("file").Of(""), // Of cleared again
	}
	for _, r := range cases {
		got := English.Message("publish_at", r)
		if got != "Publish at is not valid." {
			t.Errorf("Message(%+v) = %q, want the fallback", r, got)
		}
	}
}

// The parameter count is written down next to each template so that a rule
// handing over the wrong number produces a plain sentence instead of
// %!v(MISSING) in front of a user. This is what keeps the two agreeing.
func TestTemplateParamsMatchTheVerbs(t *testing.T) {
	for key, tpl := range templates {
		verbs := strings.Count(tpl.text, "%") - 2*strings.Count(tpl.text, "%%")
		if verbs != tpl.params+1 {
			t.Errorf("%s has %d verbs and says it takes %d parameters, and the first verb is the label",
				key, verbs, tpl.params)
		}
	}
}

// Every sentence is a sentence: it starts with the field and ends with a stop,
// because these are read next to an input and not in a log.
func TestEveryTemplateReadsLikeASentence(t *testing.T) {
	for _, key := range slices.Sorted(maps.Keys(templates)) {
		text := templates[key].text
		if !strings.HasPrefix(text, "%s ") {
			t.Errorf("%s does not start with the field", key)
		}
		if !strings.HasSuffix(text, ".") {
			t.Errorf("%s does not end in a full stop", key)
		}
	}
}

// A size rule has one sentence per thing it can count, and a table with four
// of the five is a rule that writes "is not valid" for the fifth.
func TestEverySizeRuleCoversEverySubject(t *testing.T) {
	subjects := []string{"string", "numeric", "array", "file", "duration"}
	for _, rule := range []string{"min", "max", "between", "size"} {
		for _, subject := range subjects {
			if _, ok := templates[rule+"."+subject]; !ok {
				t.Errorf("%s over a %s has no sentence", rule, subject)
			}
		}
	}
}

// Msgs is the seam, so an Errors that has one uses it everywhere a sentence
// comes out and not only in the place it was first needed.
func TestMsgsIsUsedEverywhere(t *testing.T) {
	var e Errors
	e.Msgs = fixed("whatever")
	e.Add("title", Failed("required"))

	if got := e.First("title"); got != "whatever" {
		t.Errorf("First = %q", got)
	}
	if got := e.All()["title"]; !slices.Equal(got, []string{"whatever"}) {
		t.Errorf("All = %v", got)
	}
	if got := e.Error(); got != "validate: whatever" {
		t.Errorf("Error = %q", got)
	}
	if fields := errs.Fields(e.OrNil()); len(fields) != 1 || fields[0].Msg != "whatever" {
		t.Errorf("OrNil wrote %v", fields)
	}
}

type fixed string

func (f fixed) Message(string, RuleError) string { return string(f) }
