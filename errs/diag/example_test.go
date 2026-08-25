package diag_test

import (
	"fmt"
	"os"

	"github.com/go-mizu/mizu/errs/diag"
)

// A loader reports what it found and lets the caller decide how to say it.
func Example() {
	source := func(string) ([]byte, error) {
		return []byte("[database]\nurl = \"postgres://localhost/blog\"\npool_size = 25\n"), nil
	}

	l := diag.List{{
		Code:    "MZ1042",
		Message: `unknown config key "database.pool_size"`,
		File:    "config/app.toml",
		Range:   diag.Span(3, 1, 9),
		Detail:  "no such field in Config.Database",
		Suggestions: []diag.Suggestion{{
			Message:    `did you mean "max_open_conns"?`,
			Confidence: diag.High,
			Edits: []diag.Edit{{
				File:    "config/app.toml",
				Range:   diag.Span(3, 1, 9),
				NewText: "max_open_conns",
			}},
		}},
		Fix: "mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns",
	}}

	diag.Text(os.Stdout, l, diag.WithSource(source))
	// Output:
	// error[MZ1042]: unknown config key "database.pool_size"
	//  --> config/app.toml:3:1
	//   |
	// 3 | pool_size = 25
	//   | ^^^^^^^^^ no such field in Config.Database
	//   |
	//   = did you mean "max_open_conns"?
	//   = fix: mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns
	//   = explain: mizu explain MZ1042
}

// A package that finds problems returns them as an ordinary error, and the
// caller that wants them one at a time asks for them back.
func ExampleOf() {
	err := load()

	for _, d := range diag.Of(err) {
		fmt.Println(d.Severity, d.Error())
	}
	// Output:
	// error config/app.toml:14:1: unknown config key "database.pool_size"
	// warning config/app.toml:2:1: app.name is empty
}

func load() error {
	return fmt.Errorf("loading configuration: %w", diag.List{
		{
			Code:    "MZ1042",
			Message: `unknown config key "database.pool_size"`,
			File:    "config/app.toml",
			Range:   diag.Span(14, 1, 9),
		},
		{
			Severity: diag.Warning,
			Message:  "app.name is empty",
			File:     "config/app.toml",
			Range:    diag.Span(2, 1, 4),
		},
	}.Err())
}

// The registry is what a code means, separately from any one occurrence of it.
func ExampleLookup() {
	e, ok := diag.Lookup("MZ1042")
	if !ok {
		return
	}
	s, _ := diag.SubsystemOf(e.Code)
	fmt.Println(e.Summary)
	fmt.Println(s.Name, "(doc", s.Doc+")")
	// Output:
	// a setting is written down that nothing asked for
	// configuration (doc 05)
}

// An empty list is not a failure, which is what lets a loader end with one
// line rather than with a length check.
func ExampleList_Err() {
	var found diag.List
	fmt.Println(found.Err() == nil)

	found = append(found, diag.Diagnostic{Message: "unknown config key"})
	fmt.Println(found.Err())
	// Output:
	// true
	// unknown config key
}
