package sanitizer

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var capturedName string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedName = c.Query("name")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?name=<script>alert('xss')</script>", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(capturedName, "<script>") {
		t.Errorf("expected HTML to be escaped, got %q", capturedName)
	}
}

func TestWithOptions_TrimSpaces(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		TrimSpaces: true,
		HTMLEscape: false,
	}))

	var capturedName string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedName = c.Query("name")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?name="+url.QueryEscape("  John  "), nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedName != "John" {
		t.Errorf("expected 'John', got %q", capturedName)
	}
}

func TestWithOptions_StripTags(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		StripTags:  true,
		HTMLEscape: false,
	}))

	var capturedContent string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedContent = c.Query("content")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?content="+url.QueryEscape("<p>Hello</p><script>bad</script>World"), nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedContent != "HelloWorld" {
		t.Errorf("expected 'HelloWorld', got %q", capturedContent)
	}
}

func TestWithOptions_MaxLength(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		MaxLength:  5,
		HTMLEscape: false,
	}))

	var capturedName string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedName = c.Query("name")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?name=VeryLongName", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedName != "VeryL" {
		t.Errorf("expected 'VeryL', got %q", capturedName)
	}
}

func TestWithOptions_Fields(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		HTMLEscape: true,
		Fields:     []string{"name"},
	}))

	var capturedName, capturedEmail string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedName = c.Query("name")
		capturedEmail = c.Query("email")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?name=<b>John</b>&email=<b>john@example.com</b>", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// name should be escaped
	if strings.Contains(capturedName, "<b>") {
		t.Error("expected name to be escaped")
	}

	// email should NOT be escaped (not in Fields list)
	if !strings.Contains(capturedEmail, "<b>") {
		t.Error("expected email to not be escaped")
	}
}

func TestWithOptions_Exclude(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		HTMLEscape: true,
		Exclude:    []string{"html_content"},
	}))

	var capturedName, capturedContent string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedName = c.Query("name")
		capturedContent = c.Query("html_content")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?name=<b>John</b>&html_content=<p>Hello</p>", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// name should be escaped
	if strings.Contains(capturedName, "<b>") {
		t.Error("expected name to be escaped")
	}

	// html_content should NOT be escaped (excluded)
	if !strings.Contains(capturedContent, "<p>") {
		t.Error("expected html_content to not be escaped")
	}
}

func TestXSS(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(XSS())

	var capturedInput string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedInput = c.Query("input")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?input=<script>alert(1)</script>", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(capturedInput, "<script>") {
		t.Error("expected XSS to be prevented")
	}
}

func TestStripHTML(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(StripHTML())

	var capturedInput string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedInput = c.Query("input")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?input="+url.QueryEscape("<div><p>Hello</p></div>"), nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedInput != "Hello" {
		t.Errorf("expected 'Hello', got %q", capturedInput)
	}
}

func TestTrim(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Trim())

	var capturedInput string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedInput = c.Query("input")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?input="+url.QueryEscape("  hello  "), nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedInput != "hello" {
		t.Errorf("expected 'hello', got %q", capturedInput)
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     Options
		expected string
	}{
		{
			name:     "HTML escape",
			input:    "<script>alert(1)</script>",
			opts:     Options{HTMLEscape: true},
			expected: "&lt;script&gt;alert(1)&lt;/script&gt;",
		},
		{
			name:     "trim spaces",
			input:    "  hello  ",
			opts:     Options{TrimSpaces: true},
			expected: "hello",
		},
		{
			name:     "strip tags",
			input:    "<p>hello</p>",
			opts:     Options{StripTags: true},
			expected: "hello",
		},
		{
			name:     "max length",
			input:    "hello world",
			opts:     Options{MaxLength: 5},
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sanitize(tt.input, tt.opts)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	result := SanitizeHTML("  <script>bad</script>  ")
	expected := "&lt;script&gt;bad&lt;/script&gt;"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStripTagsString(t *testing.T) {
	result := StripTagsString("<div><p>Hello</p></div>")
	if result != "Hello" {
		t.Errorf("expected 'Hello', got %q", result)
	}
}

func TestTrimString(t *testing.T) {
	result := TrimString("  hello  ")
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestClean(t *testing.T) {
	result := Clean("  <script>alert(1)</script>  ")
	if strings.Contains(result, "<") || strings.Contains(result, ">") {
		t.Errorf("expected no HTML tags, got %q", result)
	}
}

func TestWithOptions_StripNonPrintable(t *testing.T) {
	result := sanitizeValue("hello\x00world\x1f", Options{StripNonPrintable: true})
	if result != "helloworld" {
		t.Errorf("expected 'helloworld', got %q", result)
	}
}
