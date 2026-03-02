package markdown

import (
	"strings"
	"testing"
)

func TestConvert_BasicArticle(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html lang="en">
<head><title>Test Article</title></head>
<body>
<nav><a href="/">Home</a> | <a href="/about">About</a></nav>
<article>
<h1>Hello World</h1>
<p>This is a test article with some <strong>bold</strong> and <em>italic</em> text.</p>
<p>Second paragraph with a <a href="https://example.com">link</a>.</p>
</article>
<footer>Copyright 2026</footer>
</body>
</html>`)

	result := Convert(html, "https://example.com/page")

	if !result.HasContent {
		t.Fatal("expected HasContent=true")
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Markdown == "" {
		t.Fatal("expected non-empty markdown")
	}
	if result.Title == "" {
		t.Error("expected non-empty title")
	}
	t.Logf("Title: %q", result.Title)
	if result.HTMLSize == 0 {
		t.Error("expected HTMLSize > 0")
	}
	if result.MarkdownSize == 0 {
		t.Error("expected MarkdownSize > 0")
	}
	// On real pages markdown is much smaller; on tiny test pages ratio may vary
	t.Logf("Ratio: md/html = %.2f", float64(result.MarkdownSize)/float64(result.HTMLSize))
	t.Logf("HTML: %d bytes → MD: %d bytes (%.1fx reduction)", result.HTMLSize, result.MarkdownSize, float64(result.HTMLSize)/float64(result.MarkdownSize))
	t.Logf("Markdown:\n%s", result.Markdown)
}

func TestConvert_EmptyHTML(t *testing.T) {
	result := Convert([]byte(""), "")
	if result.HasContent {
		t.Error("expected HasContent=false for empty input")
	}
	if result.Error == "" {
		t.Error("expected error for empty input")
	}
}

func TestConvert_NoArticleContent(t *testing.T) {
	html := []byte(`<html><body><nav><a href="/">Home</a></nav></body></html>`)
	result := Convert(html, "")
	// trafilatura may or may not find content in nav-only page; just verify no panic
	t.Logf("HasContent=%v Error=%q MD=%q", result.HasContent, result.Error, result.Markdown)
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		bytes  int
		tokens int
	}{
		{0, 0},
		{4, 1},
		{5, 2},
		{100, 25},
		{1000, 250},
	}
	for _, tt := range tests {
		got := EstimateTokens(tt.bytes)
		if got != tt.tokens {
			t.Errorf("estimateTokens(%d) = %d, want %d", tt.bytes, got, tt.tokens)
		}
	}
}

func TestConvert_RealisticArticle(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html lang="en">
<head>
<title>Understanding Go Concurrency - Tech Blog</title>
<meta name="description" content="A deep dive into Go's concurrency model">
<meta property="og:title" content="Understanding Go Concurrency">
</head>
<body>
<header>
<nav><ul><li><a href="/">Home</a></li><li><a href="/blog">Blog</a></li><li><a href="/about">About</a></li></ul></nav>
</header>
<main>
<article>
<h1>Understanding Go Concurrency</h1>
<p class="date">March 2, 2026</p>
<p>Go's concurrency model is built around <strong>goroutines</strong> and <strong>channels</strong>.
Goroutines are lightweight threads managed by the Go runtime, making it possible to
run thousands of concurrent operations with minimal overhead.</p>

<h2>Goroutines</h2>
<p>A goroutine is started with the <code>go</code> keyword. Unlike OS threads, goroutines
start with a small stack (a few KB) that grows as needed.</p>

<pre><code>func main() {
    go func() {
        fmt.Println("Hello from goroutine")
    }()
    time.Sleep(time.Second)
}</code></pre>

<h2>Channels</h2>
<p>Channels are the pipes that connect goroutines. You can send values into channels
from one goroutine and receive those values in another.</p>

<ul>
<li>Unbuffered channels block until both sender and receiver are ready</li>
<li>Buffered channels block only when the buffer is full</li>
<li>Use <code>select</code> to wait on multiple channel operations</li>
</ul>

<h2>Conclusion</h2>
<p>Go's concurrency primitives make it straightforward to write concurrent programs
that are both efficient and easy to reason about. The combination of goroutines and
channels provides a powerful abstraction for concurrent programming.</p>
</article>
</main>
<footer>
<p>Copyright 2026 Tech Blog. All rights reserved.</p>
<nav><a href="/privacy">Privacy</a> | <a href="/terms">Terms</a></nav>
</footer>
<script>console.log("analytics")</script>
</body>
</html>`)

	result := Convert(html, "https://techblog.example.com/go-concurrency")

	if !result.HasContent {
		t.Fatal("expected HasContent=true")
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Markdown == "" {
		t.Fatal("expected non-empty markdown")
	}

	// Markdown should contain key content
	md := result.Markdown
	for _, want := range []string{"goroutine", "channel", "Conclusion"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown should contain %q", want)
		}
	}

	// Should NOT contain script content
	if strings.Contains(md, "analytics") {
		t.Error("markdown should not contain script content")
	}

	t.Logf("Title: %q, Lang: %q", result.Title, result.Language)
	t.Logf("HTML: %d bytes → MD: %d bytes (%.1fx reduction)",
		result.HTMLSize, result.MarkdownSize,
		float64(result.HTMLSize)/float64(max64(result.MarkdownSize, 1)))
	t.Logf("Tokens: HTML %d → MD %d", result.HTMLTokens, result.MarkdownTokens)
	t.Logf("Convert time: %dms", result.ConvertMs)
	t.Logf("Markdown:\n%s", result.Markdown)
}

func max64(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func TestCidFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"ab/cd/ef0123456789.gz", "sha256:abcdef0123456789"},
		{"00/11/2233445566778899aabbccddeeff00112233445566778899aabbccddeeff.gz",
			"sha256:00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"},
	}
	for _, tt := range tests {
		got := cidFromPath(tt.path)
		if got != tt.want {
			t.Errorf("cidFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
