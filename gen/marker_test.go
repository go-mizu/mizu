package gen

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"testing"
)

func TestParseMarker(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want Marker
	}{
		{
			"name only",
			"//mizu:enum",
			Marker{Name: "enum"},
		},
		{
			"keys and values",
			"//mizu:rpc method=POST path=/v1/orders ability=order.create",
			Marker{Name: "rpc", Args: []Arg{
				{"method", "POST"},
				{"path", "/v1/orders"},
				{"ability", "order.create"},
			}},
		},
		{
			"a value with a colon in it",
			"//mizu:rpc idempotent=header:Idempotency-Key",
			Marker{Name: "rpc", Args: []Arg{{"idempotent", "header:Idempotency-Key"}}},
		},
		{
			"bare words are positional",
			"//mizu:response 409 ConflictBody",
			Marker{Name: "response", Args: []Arg{{"", "409"}, {"", "ConflictBody"}}},
		},
		{
			"a bare word mixed in with keys is a flag",
			"//mizu:command name=prune standalone isolated=true",
			Marker{Name: "command", Args: []Arg{
				{"name", "prune"},
				{"", "standalone"},
				{"isolated", "true"},
			}},
		},
		{
			"a quoted value holds spaces",
			`//mizu:command name="users:prune" desc="Delete users who never verified"`,
			Marker{Name: "command", Args: []Arg{
				{"name", "users:prune"},
				{"desc", "Delete users who never verified"},
			}},
		},
		{
			"a quoted value holds escapes",
			`//mizu:command desc="say \"hi\"\tnow"`,
			Marker{Name: "command", Args: []Arg{{"desc", "say \"hi\"\tnow"}}},
		},
		{
			"a quoted word with no key is positional",
			`//mizu:response 409 "Conflict body"`,
			Marker{Name: "response", Args: []Arg{{"", "409"}, {"", "Conflict body"}}},
		},
		{
			"a hyphen is part of the name",
			"//mizu:rpc-renamed from=old_name",
			Marker{Name: "rpc-renamed", Args: []Arg{{"from", "old_name"}}},
		},
		{
			"an empty value is allowed",
			"//mizu:rpc prefix=",
			Marker{Name: "rpc", Args: []Arg{{"prefix", ""}}},
		},
		{
			"extra spaces are not arguments",
			"//mizu:model   table=posts  ",
			Marker{Name: "model", Args: []Arg{{"table", "posts"}}},
		},
		{"not a marker at all", "// an ordinary comment", Marker{}},
		{"someone else's directive", "//go:generate mizu gen", Marker{}},
		{"a block comment is never a marker", "/*mizu:model*/", Marker{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMarker(tc.text, token.Position{})
			if err != nil {
				t.Fatalf("parseMarker(%q): %v", tc.text, err)
			}
			if tc.want.Name != "" {
				tc.want.Text = tc.text
			}
			if got.Name != tc.want.Name || !slices.Equal(got.Args, tc.want.Args) || got.Text != tc.want.Text {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseMarkerErrors(t *testing.T) {
	for _, tc := range []struct{ name, text, want string }{
		{"no name", "//mizu:", "no name"},
		{"punctuation after the name", "//mizu:model! table=posts", "not a name character"},
		{"a repeated key", "//mizu:model table=a table=b", "table is given twice"},
		{"an unclosed quote", `//mizu:command name="posts:prune`, "closing quote"},
		{"a bad escape", `//mizu:command desc="\q"`, "not a valid quoted string"},
		{"no key", "//mizu:model =posts", "no key"},
		{"an unclosed quote with no key", `//mizu:response "Conflict body`, "closing quote"},
		{"the missing directive", "// mizu:model table=posts", "space after the slashes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMarker(tc.text, token.Position{})
			if err == nil {
				t.Fatalf("parseMarker(%q) gave no error", tc.text)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("got %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// A sentence that happens to start with the word mizu and a colon is prose,
// and saying otherwise would be an error message nobody can act on.
func TestNearMissLeavesProseAlone(t *testing.T) {
	for _, text := range []string{
		"// mizu:model is what marks a type as a model.",
		"// mizu:rpc Markers go on the method, not the type",
		"// mizu:",
		"//  mizu:model table=posts is the way to write it.",
		"// nothing to do with markers",
	} {
		if _, err := parseMarker(text, token.Position{}); err != nil {
			t.Errorf("parseMarker(%q) = %v, want no error", text, err)
		}
	}
}

func TestMarkerAccessors(t *testing.T) {
	m, err := parseMarker("//mizu:rpc method=POST transport=grpc,connect,http standalone quiet=true loud=false", token.Position{})
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := m.Get("method"); !ok || v != "POST" {
		t.Errorf(`Get("method") = %q, %v`, v, ok)
	}
	if _, ok := m.Get("nope"); ok {
		t.Error(`Get("nope") found something`)
	}
	if got := m.List("transport"); !slices.Equal(got, []string{"grpc", "connect", "http"}) {
		t.Errorf(`List("transport") = %v`, got)
	}
	if got := m.List("nope"); got != nil {
		t.Errorf(`List("nope") = %v, want nil`, got)
	}
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"standalone", true}, // a bare word
		{"quiet", true},      // spelled out
		{"loud", false},      // spelled out the other way
		{"method", false},    // a value that is not a boolean
		{"nope", false},
	} {
		if got := m.Flag(tc.name); got != tc.want {
			t.Errorf("Flag(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
	if got := m.Words(); !slices.Equal(got, []string{"standalone"}) {
		t.Errorf("Words() = %v", got)
	}
	if m.String() != m.Text {
		t.Error("String should be the comment as written")
	}
}

// The list splitter is given a value with spaces and empties in it, because
// people write lists that way and dropping the whitespace is cheaper than
// explaining a rule about it.
func TestListTrimsAndDropsEmpties(t *testing.T) {
	m, err := parseMarker(`//mizu:rpc transport="grpc, connect, ,http,"`, token.Position{})
	if err != nil {
		t.Fatal(err)
	}
	if got := m.List("transport"); !slices.Equal(got, []string{"grpc", "connect", "http"}) {
		t.Errorf("got %v", got)
	}
}

func scanFixture(t *testing.T, path string) ([]Target, []Error) {
	t.Helper()
	p := loadFixture(t, nil)[path]
	if p == nil {
		t.Fatalf("%s was not loaded", path)
	}
	if err := p.Err(); err != nil {
		t.Fatalf("%s does not load cleanly: %v", path, err)
	}
	return Scan(p)
}

func TestScan(t *testing.T) {
	targets, errs := scanFixture(t, "mizu.test/gen/markers")
	if len(errs) != 0 {
		t.Fatalf("the fixture should have no bad markers: %v", errs)
	}

	// Source order, so the same input always produces the same output.
	var got []string
	for _, tr := range targets {
		names := make([]string, len(tr.Markers))
		for i, m := range tr.Markers {
			names[i] = m.Name
		}
		got = append(got, tr.Name()+": "+strings.Join(names, " "))
	}
	want := []string{
		"markers: manual", // the package comment
		"Post: model searchable",
		"Body: ts", // a struct field
		"Publish: rpc response",
		"Status: enum",
		"DefaultQueue: config",
		"Prune: command",
		": embed",    // an embedded field has no name to resolve
		"Watch: rpc", // an interface method
		"A: api",     // the group's own comment is not copied down
	}
	if !slices.Equal(got, want) {
		t.Errorf("got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func TestScanResolvesObjects(t *testing.T) {
	targets, _ := scanFixture(t, "mizu.test/gen/markers")
	byName := map[string]Target{}
	for _, tr := range targets {
		byName[tr.Name()] = tr
	}

	post := byName["Post"]
	tn, ok := post.Object.(*types.TypeName)
	if !ok {
		t.Fatalf("Post resolved to %T, want a *types.TypeName", post.Object)
	}
	if _, ok := tn.Type().Underlying().(*types.Struct); !ok {
		t.Errorf("Post is %s, want a struct", tn.Type().Underlying())
	}
	if _, ok := post.Node.(*ast.TypeSpec); !ok {
		t.Errorf("Post's node is %T, want an *ast.TypeSpec", post.Node)
	}

	if _, ok := byName["Publish"].Object.(*types.Func); !ok {
		t.Errorf("Publish resolved to %T, want a *types.Func", byName["Publish"].Object)
	}
	if _, ok := byName["Body"].Object.(*types.Var); !ok {
		t.Errorf("Body resolved to %T, want a *types.Var", byName["Body"].Object)
	}
	if _, ok := byName["DefaultQueue"].Object.(*types.Const); !ok {
		t.Errorf("DefaultQueue resolved to %T, want a *types.Const", byName["DefaultQueue"].Object)
	}

	// The package comment has nothing to name.
	pkg := byName["markers"]
	if pkg.Object != nil {
		t.Errorf("the package comment resolved to %v", pkg.Object)
	}
	if _, ok := pkg.Node.(*ast.File); !ok {
		t.Errorf("the package comment's node is %T, want an *ast.File", pkg.Node)
	}
}

func TestScanReadsArguments(t *testing.T) {
	targets, _ := scanFixture(t, "mizu.test/gen/markers")
	for _, tr := range targets {
		for _, m := range tr.Markers {
			switch {
			case tr.Name() == "Post" && m.Name == "model":
				if v, _ := m.Get("table"); v != "posts" {
					t.Errorf("table = %q, want posts", v)
				}
			case tr.Name() == "Prune":
				if v, _ := m.Get("name"); v != "posts:prune" {
					t.Errorf("name = %q", v)
				}
				if v, _ := m.Get("desc"); v != "Delete drafts older than a year" {
					t.Errorf("desc = %q", v)
				}
				if !m.Flag("standalone") {
					t.Error("standalone should be on")
				}
			case tr.Name() == "Watch":
				if got := m.List("transport"); !slices.Equal(got, []string{"grpc", "connect", "http"}) {
					t.Errorf("transport = %v", got)
				}
			case tr.Name() == "Publish" && m.Name == "response":
				if got := m.Words(); !slices.Equal(got, []string{"409", "ConflictBody"}) {
					t.Errorf("words = %v", got)
				}
			}
		}
	}
}

func TestScanPositions(t *testing.T) {
	targets, _ := scanFixture(t, "mizu.test/gen/markers")
	for _, tr := range targets {
		if tr.Name() != "Post" {
			continue
		}
		m := tr.Markers[0]
		if !strings.HasSuffix(m.Pos.Filename, "markers.go") {
			t.Errorf("marker filename is %q", m.Pos.Filename)
		}
		// The marker sits on the line above the declaration it belongs to.
		if m.Pos.Line >= tr.Pos().Line {
			t.Errorf("marker at line %d, declaration at line %d", m.Pos.Line, tr.Pos().Line)
		}
		return
	}
	t.Error("Post was not found")
}

func TestScanReportsBadMarkers(t *testing.T) {
	targets, errs := scanFixture(t, "mizu.test/gen/badmarkers")
	if len(targets) != 0 {
		t.Errorf("nothing in that package is a valid marker, got %d targets", len(targets))
	}
	want := []string{
		"space after the slashes",
		"table is given twice",
		"closing quote",
		"no key",
		"not a name character",
	}
	if len(errs) != len(want) {
		t.Fatalf("got %d errors, want %d: %v", len(errs), len(want), errs)
	}
	for i, w := range want {
		if !strings.Contains(errs[i].Msg, w) {
			t.Errorf("error %d is %q, want it to mention %q", i, errs[i].Msg, w)
		}
		if errs[i].Kind != MarkerError {
			t.Errorf("error %d is a %s error, want marker", i, errs[i].Kind)
		}
		if !strings.Contains(errs[i].Pos, "badmarkers.go:") {
			t.Errorf("error %d has no useful position: %q", i, errs[i].Pos)
		}
	}
}

// A package that did not type-check still has markers in it, and that is
// exactly when a generator has work to do.
func TestScanWorksWithoutTypes(t *testing.T) {
	p := loadFixture(t, nil)["mizu.test/gen/markers"]
	stripped := &Package{PkgPath: p.PkgPath, Fset: p.Fset, Syntax: p.Syntax}
	targets, errs := Scan(stripped)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(targets) == 0 {
		t.Fatal("no markers found")
	}
	for _, tr := range targets {
		if tr.Object != nil {
			t.Errorf("%v resolved an object without type information", tr.Node)
		}
	}
}

func TestScanAcrossPackages(t *testing.T) {
	pkgs := loadFixture(t, nil)
	targets, _ := Scan(pkgs["mizu.test/gen/markers"], pkgs["mizu.test/gen/model"])
	var seen []string
	for _, tr := range targets {
		seen = append(seen, tr.Pkg.PkgPath+"."+tr.Name())
	}
	if len(seen) == 0 || seen[0] != "mizu.test/gen/markers.markers" {
		t.Fatalf("packages should come back in the order given, got %v", seen)
	}
	if !slices.Contains(seen, "mizu.test/gen/model.User") {
		t.Errorf("the marker on model.User was not found: %v", seen)
	}
}

// A package with no markers produces nothing, even when it does not compile.
// The bootstrap fixture is both, which makes it the right one to ask.
func TestScanFindsNothingInAPlainPackage(t *testing.T) {
	targets, errs := Scan(loadFixture(t, nil)["mizu.test/gen/bootstrap"])
	if len(targets) != 0 || len(errs) != 0 {
		t.Errorf("got %d targets and %d errors, want none", len(targets), len(errs))
	}
}

func TestTargetNameFallsBackToNothing(t *testing.T) {
	if got := (Target{Node: &ast.TypeSpec{}}).Name(); got != "" {
		t.Errorf("got %q, want the empty string", got)
	}
}
