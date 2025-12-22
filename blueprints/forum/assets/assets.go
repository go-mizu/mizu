// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"strings"
)

//go:embed static views
var FS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(FS, "static")
	return sub
}

// Views returns the views filesystem.
func Views() fs.FS {
	sub, _ := fs.Sub(FS, "views")
	return sub
}

// Templates parses all view templates.
func Templates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"dict":    dict,
		"add":     add,
		"sub":     sub,
		"toUpper": strings.ToUpper,
	}
	return template.New("").Funcs(funcMap).ParseFS(Views(), "layouts/*.html", "pages/*.html", "components/*.html")
}

// dict creates a map from key-value pairs for passing to templates.
func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, nil
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, nil
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

// add adds two integers.
func add(a, b int) int {
	return a + b
}

// sub subtracts b from a.
func sub(a, b int) int {
	return a - b
}
