package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestURLToLocalPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/", "index.html"},
		{"", "index.html"},
		{"/about", "about/index.html"},
		{"/about/", "about/index.html"},
		{"/page.html", "page.html"},
		{"/blog/my-post", "blog/my-post/index.html"},
		{"/css/main.css", "css/main.css"},
		{"/docs/api/v2", "docs/api/v2/index.html"},
	}
	for _, tt := range tests {
		got := URLToLocalPath(tt.input)
		if got != tt.want {
			t.Errorf("URLToLocalPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsSpecialURL(t *testing.T) {
	specials := []string{"", "#", "data:image/png;base64,abc", "javascript:void(0)", "mailto:a@b.com", "tel:+1234"}
	for _, u := range specials {
		if !isSpecialURL(u) {
			t.Errorf("isSpecialURL(%q) = false, want true", u)
		}
	}
	normals := []string{"/about", "https://example.com", "page.html", "../other"}
	for _, u := range normals {
		if isSpecialURL(u) {
			t.Errorf("isSpecialURL(%q) = true, want false", u)
		}
	}
}

func TestExporterWritePage(t *testing.T) {
	dir := t.TempDir()
	exp, err := New(Config{Domain: "example.com", OutDir: dir, Format: "html"})
	if err != nil {
		t.Fatal(err)
	}

	html := `<!DOCTYPE html><html><head><title>Test</title></head><body>
<a href="/about">About</a>
<a href="https://example.com/blog">Blog</a>
<a href="https://other.com/ext">External</a>
<img src="/images/logo.png">
<link rel="stylesheet" href="/css/main.css">
</body></html>`

	localPath, err := exp.WritePage(Page{URL: "https://example.com/", HTML: []byte(html)})
	if err != nil {
		t.Fatal(err)
	}
	if localPath != "index.html" {
		t.Errorf("localPath = %q, want index.html", localPath)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(dir, "html", "example.com", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(content)

	// Internal links should be rewritten to relative paths
	if !strings.Contains(out, `href="about/index.html"`) {
		t.Errorf("expected /about rewritten to relative path, got:\n%s", out)
	}
	if !strings.Contains(out, `href="blog/index.html"`) {
		t.Errorf("expected same-domain link rewritten, got:\n%s", out)
	}

	// External link should be unchanged
	if !strings.Contains(out, `href="https://other.com/ext"`) {
		t.Errorf("expected external link preserved, got:\n%s", out)
	}

	// Asset references should point to _assets/
	if !strings.Contains(out, `_assets/img/`) {
		t.Errorf("expected img rewritten to _assets/img/, got:\n%s", out)
	}
	if !strings.Contains(out, `_assets/css/`) {
		t.Errorf("expected css rewritten to _assets/css/, got:\n%s", out)
	}
}

func TestExporterRawFormat(t *testing.T) {
	dir := t.TempDir()
	exp, err := New(Config{Domain: "example.com", OutDir: dir, Format: "raw"})
	if err != nil {
		t.Fatal(err)
	}

	html := `<html><body><a href="/about">About</a></body></html>`
	_, err = exp.WritePage(Page{URL: "https://example.com/page", HTML: []byte(html)})
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "raw", "example.com", "page", "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	// Raw format should NOT rewrite links
	if !strings.Contains(string(content), `href="/about"`) {
		t.Errorf("raw format should preserve original links, got:\n%s", string(content))
	}
}

func TestExporterWriteIndex(t *testing.T) {
	dir := t.TempDir()
	exp, err := New(Config{Domain: "example.com", OutDir: dir, Format: "html"})
	if err != nil {
		t.Fatal(err)
	}

	_, _ = exp.WritePage(Page{URL: "https://example.com/", HTML: []byte("<html><body>Home</body></html>")})
	_, _ = exp.WritePage(Page{URL: "https://example.com/about", HTML: []byte("<html><body>About</body></html>")})

	if err := exp.WriteIndex(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "html", "example.com", "_index.html"))
	if err != nil {
		t.Fatal(err)
	}

	out := string(content)
	if !strings.Contains(out, "2 pages exported") {
		t.Errorf("expected page count in index, got:\n%s", out)
	}
	if !strings.Contains(out, "example.com") {
		t.Errorf("expected domain in index, got:\n%s", out)
	}
}

func TestExporterSubpageLinkRewriting(t *testing.T) {
	dir := t.TempDir()
	exp, err := New(Config{Domain: "example.com", OutDir: dir, Format: "html"})
	if err != nil {
		t.Fatal(err)
	}

	html := `<html><body><a href="/">Home</a><a href="/about">About</a></body></html>`
	_, err = exp.WritePage(Page{URL: "https://example.com/blog/post-1", HTML: []byte(html)})
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "html", "example.com", "blog", "post-1", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(content)

	// From /blog/post-1/index.html, / should be ../../index.html
	if !strings.Contains(out, `../../index.html`) {
		t.Errorf("expected relative path to root from subpage, got:\n%s", out)
	}
}

func TestRewriteCSSURLs(t *testing.T) {
	dir := t.TempDir()
	exp, err := New(Config{Domain: "example.com", OutDir: dir, Format: "html"})
	if err != nil {
		t.Fatal(err)
	}

	html := `<html><head><style>
body { background: url(/images/bg.png); }
.hero { background-image: url('/images/hero.jpg'); }
</style></head><body>Test</body></html>`

	_, err = exp.WritePage(Page{URL: "https://example.com/", HTML: []byte(html)})
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "html", "example.com", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(content)

	if !strings.Contains(out, `_assets/img/`) {
		t.Errorf("expected CSS url() rewritten to _assets/img/, got:\n%s", out)
	}
}
