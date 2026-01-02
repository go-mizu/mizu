package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed static views
var FS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(FS, "static")
	return sub
}

// Views returns the views filesystem for the default theme.
func Views() fs.FS {
	sub, _ := fs.Sub(FS, "views/default")
	return sub
}

// ViewsForTheme returns the views filesystem for a specific theme.
// Themes inherit from default - theme files override default files.
func ViewsForTheme(theme string) fs.FS {
	if theme == "" || theme == "default" {
		return Views()
	}
	return &themeFS{
		theme:    theme,
		base:     FS,
		default_: "views/default",
		overlay:  "views/" + theme,
	}
}

// themeFS implements fs.FS with theme inheritance.
// Files from the overlay (theme) directory take precedence over the default.
type themeFS struct {
	theme    string
	base     fs.FS
	default_ string
	overlay  string
}

func (t *themeFS) Open(name string) (fs.File, error) {
	if f, err := t.base.Open(t.overlay + "/" + name); err == nil {
		return f, nil
	}
	return t.base.Open(t.default_ + "/" + name)
}

func (t *themeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries := make(map[string]fs.DirEntry)

	if dir, ok := t.base.(fs.ReadDirFS); ok {
		defaultPath := t.default_
		if name != "." {
			defaultPath = t.default_ + "/" + name
		}
		if list, err := dir.ReadDir(defaultPath); err == nil {
			for _, e := range list {
				entries[e.Name()] = e
			}
		}
	}

	if dir, ok := t.base.(fs.ReadDirFS); ok {
		overlayPath := t.overlay
		if name != "." {
			overlayPath = t.overlay + "/" + name
		}
		if list, err := dir.ReadDir(overlayPath); err == nil {
			for _, e := range list {
				entries[e.Name()] = e
			}
		}
	}

	result := make([]fs.DirEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, e)
	}
	return result, nil
}

func (t *themeFS) ReadFile(name string) ([]byte, error) {
	if f, ok := t.base.(fs.ReadFileFS); ok {
		if data, err := f.ReadFile(t.overlay + "/" + name); err == nil {
			return data, nil
		}
	}
	if f, ok := t.base.(fs.ReadFileFS); ok {
		return f.ReadFile(t.default_ + "/" + name)
	}
	return nil, fs.ErrNotExist
}

func (t *themeFS) Glob(pattern string) ([]string, error) {
	matches := make(map[string]bool)

	if g, ok := t.base.(fs.GlobFS); ok {
		if list, err := g.Glob(t.default_ + "/" + pattern); err == nil {
			for _, m := range list {
				rel := m[len(t.default_)+1:]
				matches[rel] = true
			}
		}
	}

	if g, ok := t.base.(fs.GlobFS); ok {
		if list, err := g.Glob(t.overlay + "/" + pattern); err == nil {
			for _, m := range list {
				rel := m[len(t.overlay)+1:]
				matches[rel] = true
			}
		}
	}

	result := make([]string, 0, len(matches))
	for m := range matches {
		result = append(result, m)
	}
	return result, nil
}

// templateFuncs returns the template function map.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime":         formatTime,
		"formatTimeRelative": formatTimeRelative,
		"formatNumber":       formatNumber,
		"formatScore":        formatScore,
		"truncate":           truncate,
		"slugify":            slugify,
		"add":                add,
		"sub":                sub,
		"mul":                mul,
		"dict":               dict,
		"list":               list,
		"contains":           contains,
		"hasPrefix":          hasPrefix,
		"hasSuffix":          hasSuffix,
		"default":            defaultVal,
		"safeHTML":           safeHTML,
		"staticURL":          StaticURL,
	}
}

var (
	staticHashesOnce sync.Once
	staticHashes     map[string]string
)

func loadStaticHashes() map[string]string {
	staticHashesOnce.Do(func() {
		staticHashes = make(map[string]string)
		staticFS := Static()
		_ = fs.WalkDir(staticFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			data, err := fs.ReadFile(staticFS, path)
			if err != nil {
				return nil
			}
			sum := sha256.Sum256(data)
			staticHashes[path] = hex.EncodeToString(sum[:8])
			return nil
		})
	})
	return staticHashes
}

// StaticURL returns a versioned static asset URL with a content hash.
func StaticURL(path string) string {
	if path == "" {
		return path
	}
	base := path
	query := ""
	if idx := strings.Index(path, "?"); idx != -1 {
		base = path[:idx]
		query = path[idx+1:]
		if strings.Contains(query, "v=") {
			return path
		}
	}

	rel := strings.TrimPrefix(base, "/static/")
	if rel == base {
		rel = strings.TrimPrefix(base, "/")
	}
	if rel == "" {
		return path
	}

	hash := loadStaticHashes()[rel]
	if hash == "" {
		return path
	}

	url := "/static/" + rel
	if query != "" {
		return url + "?" + query + "&v=" + hash
	}
	return url + "?v=" + hash
}

// StaticHash returns a short hash for a static asset path.
func StaticHash(path string) string {
	if path == "" {
		return ""
	}
	rel := strings.TrimPrefix(path, "/static/")
	if rel == path {
		rel = strings.TrimPrefix(path, "/")
	}
	if rel == "" {
		return ""
	}
	return loadStaticHashes()[rel]
}

// Templates loads and returns all templates as a map keyed by page name.
// Each page gets its own isolated template to avoid content block collisions.
func Templates() (map[string]*template.Template, error) {
	return TemplatesForTheme("default")
}

// TemplatesForTheme loads templates for a specific theme with inheritance from default.
func TemplatesForTheme(theme string) (map[string]*template.Template, error) {
	views := ViewsForTheme(theme)

	baseTmpl := template.New("").Funcs(templateFuncs())

	layoutFiles := []string{"layouts/default.html"}
	componentFiles := []string{
		"components/nav.html",
		"components/question_card.html",
		"components/answer.html",
		"components/comment.html",
	}

	for _, f := range layoutFiles {
		content, err := fs.ReadFile(views, f)
		if err != nil {
			return nil, err
		}
		_, err = baseTmpl.New(filepath.Base(f)).Parse(string(content))
		if err != nil {
			return nil, err
		}
	}

	for _, f := range componentFiles {
		content, err := fs.ReadFile(views, f)
		if err != nil {
			return nil, err
		}
		_, err = baseTmpl.New(filepath.Base(f)).Parse(string(content))
		if err != nil {
			return nil, err
		}
	}

	pageFiles, err := fs.Glob(views, "pages/*.html")
	if err != nil {
		return nil, err
	}

	templates := make(map[string]*template.Template)
	for _, pageFile := range pageFiles {
		pageName := filepath.Base(pageFile)
		pageContent, err := fs.ReadFile(views, pageFile)
		if err != nil {
			return nil, err
		}

		pageTmpl, err := baseTmpl.Clone()
		if err != nil {
			return nil, err
		}

		if _, err := pageTmpl.New(pageName).Parse(string(pageContent)); err != nil {
			return nil, err
		}

		templates[pageName] = pageTmpl
	}

	return templates, nil
}

func formatTime(t time.Time) string {
	return t.Format("Jan 2, 2006 at 3:04pm")
}

func formatTimeRelative(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return formatInt(int(d.Minutes())) + " mins ago"
	}
	if d < 24*time.Hour {
		return formatInt(int(d.Hours())) + " hours ago"
	}
	if d < 30*24*time.Hour {
		return formatInt(int(d.Hours()/24)) + " days ago"
	}
	if d < 365*24*time.Hour {
		return formatInt(int(d.Hours()/(24*30))) + " months ago"
	}
	return formatInt(int(d.Hours()/(24*365))) + " years ago"
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return formatFloat(float64(n)/1000000.0, 1) + "m"
	}
	if n >= 1000 {
		return formatFloat(float64(n)/1000.0, 1) + "k"
	}
	return formatInt(int(n))
}

func formatScore(n int64) string {
	if n > 0 {
		return "+" + formatNumber(n)
	}
	return formatNumber(n)
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length < 4 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

func slugify(s string) string {
	out := make([]rune, 0, len(s))
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
			lastDash = false
			continue
		}
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+32)
			lastDash = false
			continue
		}
		if !lastDash && len(out) > 0 {
			out = append(out, '-')
			lastDash = true
		}
	}
	if len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	return string(out)
}

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func mul(a, b int) int {
	return a * b
}

func dict(values ...any) map[string]any {
	m := make(map[string]any)
	for i := 0; i < len(values); i += 2 {
		if i+1 >= len(values) {
			break
		}
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		m[key] = values[i+1]
	}
	return m
}

func list(values ...any) []any {
	return values
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && findIndex(s, substr) >= 0)
}

func hasPrefix(s, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	if len(suffix) > len(s) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}

func defaultVal(value, fallback any) any {
	switch v := value.(type) {
	case string:
		if v == "" {
			return fallback
		}
	case int:
		if v == 0 {
			return fallback
		}
	case int64:
		if v == 0 {
			return fallback
		}
	case nil:
		return fallback
	}
	return value
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [32]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func formatFloat(f float64, decimals int) string {
	if decimals <= 0 {
		return formatInt(int(f))
	}
	factor := 1.0
	for i := 0; i < decimals; i++ {
		factor *= 10
	}
	val := float64(int(f*factor+0.5)) / factor
	intPart := int(val)
	frac := int((val - float64(intPart)) * factor)
	intStr := formatInt(intPart)
	fracStr := formatInt(frac)
	for len(fracStr) < decimals {
		fracStr = "0" + fracStr
	}
	return intStr + "." + fracStr
}

func findIndex(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
