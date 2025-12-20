package frontend

import (
	"os"
	"strings"
	"testing"
)

func TestInjectEnv(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html>
<head>
    <title>Test</title>
</head>
<body>
    <div id="app"></div>
</body>
</html>`)

	os.Setenv("TEST_API_URL", "https://api.example.com")
	os.Setenv("TEST_FEATURE_FLAG", "true")
	defer os.Unsetenv("TEST_API_URL")
	defer os.Unsetenv("TEST_FEATURE_FLAG")

	result := InjectEnv(html, []string{"TEST_API_URL", "TEST_FEATURE_FLAG", "NONEXISTENT"})

	if !strings.Contains(string(result), "window.__ENV__") {
		t.Error("expected window.__ENV__ injection")
	}
	if !strings.Contains(string(result), "TEST_API_URL") {
		t.Error("expected TEST_API_URL in env")
	}
	if !strings.Contains(string(result), "https://api.example.com") {
		t.Error("expected API URL value")
	}
	if !strings.Contains(string(result), "TEST_FEATURE_FLAG") {
		t.Error("expected TEST_FEATURE_FLAG in env")
	}
	if strings.Contains(string(result), "NONEXISTENT") {
		t.Error("should not include nonexistent env var")
	}
}

func TestInjectEnvEmpty(t *testing.T) {
	html := []byte("<html><head></head></html>")

	// Empty vars list
	result := InjectEnv(html, []string{})
	if string(result) != string(html) {
		t.Error("empty vars should return unchanged html")
	}

	// All nonexistent vars
	result = InjectEnv(html, []string{"DOES_NOT_EXIST"})
	if string(result) != string(html) {
		t.Error("nonexistent vars should return unchanged html")
	}
}

func TestInjectMeta(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html>
<head>
    <title>Test</title>
</head>
<body></body>
</html>`)

	result := InjectMeta(html, map[string]string{
		"description": "Test description",
		"og:title":    "Test Title",
	})

	if !strings.Contains(string(result), `<meta name="description"`) {
		t.Error("expected description meta tag")
	}
	if !strings.Contains(string(result), `content="Test description"`) {
		t.Error("expected description content")
	}
	if !strings.Contains(string(result), `<meta name="og:title"`) {
		t.Error("expected og:title meta tag")
	}
}

func TestInjectMetaEscaping(t *testing.T) {
	html := []byte("<html><head></head></html>")

	result := InjectMeta(html, map[string]string{
		"test": `<script>alert("xss")</script>`,
	})

	if strings.Contains(string(result), "<script>") {
		t.Error("should escape HTML in meta content")
	}
	if !strings.Contains(string(result), "&lt;script&gt;") {
		t.Error("expected escaped script tag")
	}
}

func TestInjectScript(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html>
<head></head>
<body></body>
</html>`)

	t.Run("inject before body", func(t *testing.T) {
		result := InjectScript(html, "console.log('test')", true)
		if !strings.Contains(string(result), "<script>console.log('test')</script></body>") {
			t.Error("expected script before </body>")
		}
	})

	t.Run("inject before head", func(t *testing.T) {
		result := InjectScript(html, "console.log('test')", false)
		if !strings.Contains(string(result), "<script>console.log('test')</script></head>") {
			t.Error("expected script before </head>")
		}
	})
}

func TestInjectPreload(t *testing.T) {
	html := []byte("<html><head></head></html>")

	assets := []PreloadAsset{
		{Href: "/app.js", Rel: "modulepreload"},
		{Href: "/styles.css", Rel: "preload", As: "style"},
		{Href: "/font.woff2", Rel: "preload", As: "font", Type: "font/woff2", CrossOrigin: "anonymous"},
	}

	result := InjectPreload(html, assets)

	if !strings.Contains(string(result), `rel="modulepreload" href="/app.js"`) {
		t.Error("expected modulepreload for app.js")
	}
	if !strings.Contains(string(result), `rel="preload" href="/styles.css" as="style"`) {
		t.Error("expected preload for styles.css")
	}
	if !strings.Contains(string(result), `crossorigin="anonymous"`) {
		t.Error("expected crossorigin for font")
	}
}

func TestInsertBeforeTag(t *testing.T) {
	tests := []struct {
		html     string
		tag      string
		content  string
		expected string
	}{
		{
			html:     "<html><head></head></html>",
			tag:      "</head>",
			content:  "<script>test</script>",
			expected: "<html><head><script>test</script></head></html>",
		},
		{
			html:     "<html><HEAD></HEAD></html>",
			tag:      "</head>",
			content:  "<script>test</script>",
			expected: "<html><HEAD><script>test</script></HEAD></html>",
		},
		{
			html:     "<html><body></body></html>",
			tag:      "</notfound>",
			content:  "test",
			expected: "<html><body></body></html>",
		},
	}

	for _, tt := range tests {
		result := insertBeforeTag([]byte(tt.html), tt.tag, tt.content)
		if string(result) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(result))
		}
	}
}
