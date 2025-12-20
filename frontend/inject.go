package frontend

import (
	"bytes"
	"encoding/json"
	"html/template"
	"os"
	"strings"
)

// InjectEnv injects environment variables into HTML.
// Variables are exposed as window.__ENV__ in a script tag.
func InjectEnv(html []byte, vars []string) []byte {
	env := make(map[string]string)
	for _, key := range vars {
		if val := os.Getenv(key); val != "" {
			env[key] = val
		}
	}

	if len(env) == 0 {
		return html
	}

	data, err := json.Marshal(env)
	if err != nil {
		return html
	}

	script := "<script>window.__ENV__=" + string(data) + ";</script>"
	return insertBeforeTag(html, "</head>", script)
}

// InjectMeta injects meta tags into HTML.
func InjectMeta(html []byte, meta map[string]string) []byte {
	if len(meta) == 0 {
		return html
	}

	var b strings.Builder
	for name, content := range meta {
		b.WriteString(`<meta name="`)
		b.WriteString(template.HTMLEscapeString(name))
		b.WriteString(`" content="`)
		b.WriteString(template.HTMLEscapeString(content))
		b.WriteString(`">`)
		b.WriteByte('\n')
	}

	return insertBeforeTag(html, "</head>", b.String())
}

// InjectScript injects a script tag into HTML.
func InjectScript(html []byte, script string, beforeBody bool) []byte {
	tag := "<script>" + script + "</script>"
	if beforeBody {
		return insertBeforeTag(html, "</body>", tag)
	}
	return insertBeforeTag(html, "</head>", tag)
}

// InjectPreload injects preload link tags into HTML.
func InjectPreload(html []byte, assets []PreloadAsset) []byte {
	if len(assets) == 0 {
		return html
	}

	var b strings.Builder
	for _, asset := range assets {
		b.WriteString(`<link rel="`)
		b.WriteString(asset.Rel)
		b.WriteString(`" href="`)
		b.WriteString(template.HTMLEscapeString(asset.Href))
		b.WriteByte('"')
		if asset.As != "" {
			b.WriteString(` as="`)
			b.WriteString(asset.As)
			b.WriteByte('"')
		}
		if asset.Type != "" {
			b.WriteString(` type="`)
			b.WriteString(asset.Type)
			b.WriteByte('"')
		}
		if asset.CrossOrigin != "" {
			b.WriteString(` crossorigin="`)
			b.WriteString(asset.CrossOrigin)
			b.WriteByte('"')
		}
		b.WriteString(">\n")
	}

	return insertBeforeTag(html, "</head>", b.String())
}

// PreloadAsset represents an asset to preload.
type PreloadAsset struct {
	Href        string // Asset URL
	Rel         string // preload, modulepreload, prefetch
	As          string // script, style, font, image, etc.
	Type        string // MIME type (optional)
	CrossOrigin string // anonymous, use-credentials (optional)
}

// insertBeforeTag inserts content before the first occurrence of a tag.
func insertBeforeTag(html []byte, tag, content string) []byte {
	tagBytes := []byte(tag)
	idx := bytes.Index(bytes.ToLower(html), bytes.ToLower(tagBytes))
	if idx == -1 {
		return html
	}

	result := make([]byte, 0, len(html)+len(content))
	result = append(result, html[:idx]...)
	result = append(result, content...)
	result = append(result, html[idx:]...)
	return result
}

