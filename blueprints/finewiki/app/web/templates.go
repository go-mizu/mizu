package web

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed views/**/*
var viewsFS embed.FS

// Renderer is the interface for template rendering.
type Renderer interface {
	Render(w any, name string, data any) error
}

// Templates wraps the HTML templates and implements Renderer.
type Templates struct {
	t *template.Template
}

// NewTemplates loads templates from embedded filesystem.
func NewTemplates() (*Templates, error) {
	funcs := template.FuncMap{
		"dict":         dict,
		"formatNumber": formatNumber,
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

// formatNumber formats a number with thousand separators.
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
