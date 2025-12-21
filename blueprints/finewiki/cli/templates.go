package cli

import (
	"embed"
	"errors"
	"html/template"
	"io"
)

//go:embed views/**/*
var viewsFS embed.FS

// Templates wraps the HTML templates.
type Templates struct {
	t *template.Template
}

// NewTemplates loads templates from embedded filesystem.
func NewTemplates() (*Templates, error) {
	funcs := template.FuncMap{
		"dict": dict,
	}

	t := template.New("views").Funcs(funcs)

	patterns := []string{
		"views/layout/*.html",
		"views/component/*.html",
		"views/page/*.html",
	}

	var err error
	for _, p := range patterns {
		t, err = t.ParseFS(viewsFS, p)
		if err != nil {
			return nil, err
		}
	}

	return &Templates{t: t}, nil
}

// Render renders a template to the writer.
func (x *Templates) Render(w any, name string, data any) error {
	ww, ok := w.(io.Writer)
	if !ok {
		return errors.New("templates: writer does not implement io.Writer")
	}
	return x.t.ExecuteTemplate(ww, name, data)
}

func dict(kv ...any) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, errors.New("dict: odd args")
	}
	m := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			return nil, errors.New("dict: key is not string")
		}
		m[k] = kv[i+1]
	}
	return m, nil
}
