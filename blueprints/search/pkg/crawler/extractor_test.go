package crawler

import (
	"strings"
	"testing"
)

func TestExtract(t *testing.T) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <title>Test Page</title>
  <meta name="description" content="A test page for extraction">
  <meta property="og:image" content="https://example.com/image.jpg">
  <meta name="author" content="Test Author">
  <link rel="canonical" href="https://example.com/canonical">
</head>
<body>
  <nav>Navigation links here</nav>
  <main>
    <h1>Hello World</h1>
    <p>This is the main content of the page.</p>
    <a href="/about">About</a>
    <a href="https://example.com/contact">Contact</a>
    <a href="https://other.com/external">External</a>
  </main>
  <footer>Footer content</footer>
  <script>var x = 1;</script>
  <style>.hidden { display: none; }</style>
</body>
</html>`

	res := Extract(strings.NewReader(html), "https://example.com/page")

	if res.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", res.Title, "Test Page")
	}
	if res.Description != "A test page for extraction" {
		t.Errorf("Description = %q, want %q", res.Description, "A test page for extraction")
	}
	if res.Language != "en" {
		t.Errorf("Language = %q, want %q", res.Language, "en")
	}
	if res.Metadata["og:image"] != "https://example.com/image.jpg" {
		t.Errorf("og:image = %q, want %q", res.Metadata["og:image"], "https://example.com/image.jpg")
	}
	if res.Metadata["canonical"] != "https://example.com/canonical" {
		t.Errorf("canonical = %q, want %q", res.Metadata["canonical"], "https://example.com/canonical")
	}
	if res.Metadata["author"] != "Test Author" {
		t.Errorf("author = %q, want %q", res.Metadata["author"], "Test Author")
	}

	// Content should include main text but not nav/footer/script/style
	if !strings.Contains(res.Content, "Hello World") {
		t.Error("Content should contain 'Hello World'")
	}
	if !strings.Contains(res.Content, "main content") {
		t.Error("Content should contain 'main content'")
	}
	if strings.Contains(res.Content, "Navigation links") {
		t.Error("Content should not contain nav text")
	}
	if strings.Contains(res.Content, "Footer content") {
		t.Error("Content should not contain footer text")
	}
	if strings.Contains(res.Content, "var x") {
		t.Error("Content should not contain script text")
	}
	if strings.Contains(res.Content, "display: none") {
		t.Error("Content should not contain style text")
	}

	// Check links
	if len(res.Links) != 3 {
		t.Fatalf("Links = %d, want 3", len(res.Links))
	}
	wantLinks := []string{
		"https://example.com/about",
		"https://example.com/contact",
		"https://other.com/external",
	}
	for i, want := range wantLinks {
		if res.Links[i] != want {
			t.Errorf("Links[%d] = %q, want %q", i, res.Links[i], want)
		}
	}
}

func TestExtractNoTitle(t *testing.T) {
	html := `<html><body><p>No title here</p></body></html>`
	res := Extract(strings.NewReader(html), "https://example.com/")
	if res.Title != "" {
		t.Errorf("Title = %q, want empty", res.Title)
	}
	if !strings.Contains(res.Content, "No title here") {
		t.Error("Content should contain body text")
	}
}

func TestExtractSkipsNonHTMLLinks(t *testing.T) {
	html := `<html><body>
		<a href="mailto:test@example.com">Email</a>
		<a href="javascript:void(0)">Click</a>
		<a href="/valid">Valid</a>
		<a href="https://example.com/image.jpg">Image</a>
	</body></html>`

	res := Extract(strings.NewReader(html), "https://example.com/")
	if len(res.Links) != 1 {
		t.Errorf("Links = %v, want 1 valid link", res.Links)
	}
	if len(res.Links) > 0 && res.Links[0] != "https://example.com/valid" {
		t.Errorf("Links[0] = %q, want %q", res.Links[0], "https://example.com/valid")
	}
}

func TestCollapseWhitespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello   world  ", "hello world"},
		{"a\n\n\nb", "a b"},
		{"\t\tindented\t\t", "indented"},
		{"no change", "no change"},
	}

	for _, tt := range tests {
		got := collapseWhitespace(tt.input)
		if got != tt.want {
			t.Errorf("collapseWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
