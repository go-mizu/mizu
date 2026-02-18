package qq

import (
	"testing"
	"time"
)

func TestExtractArticleID(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://news.qq.com/rain/a/20260217A02A7D00", "20260217A02A7D00"},
		{"https://news.qq.com/rain/a/20251029A0581J00", "20251029A0581J00"},
		{"https://view.inews.qq.com/a/20260117V05XV500", "20260117V05XV500"},
		{"https://news.qq.com/rain/a/20260217A02A7D00?from=share", "20260217A02A7D00"},
		{"https://news.qq.com/rain/a/20260217A02A7D00/", "20260217A02A7D00"},
		{"https://news.qq.com/ch/tech", ""},
		{"https://example.com/foo", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := ExtractArticleID(tt.url)
		if got != tt.want {
			t.Errorf("ExtractArticleID(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestParseQQTime(t *testing.T) {
	got := parseQQTime("2025-10-29 17:53:02")
	if got.IsZero() {
		t.Fatal("expected non-zero time")
	}
	if got.Year() != 2025 || got.Month() != time.October || got.Day() != 29 {
		t.Errorf("unexpected date: %v", got)
	}
	if got.Hour() != 17 || got.Minute() != 53 || got.Second() != 2 {
		t.Errorf("unexpected time: %v", got)
	}
}

func TestParseQQTimeInvalid(t *testing.T) {
	got := parseQQTime("invalid")
	if !got.IsZero() {
		t.Errorf("expected zero time for invalid input, got %v", got)
	}
}

func TestDecodeUnicodeEscapes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`\u003cdiv\u003ehello\u003c/div\u003e`, "<div>hello</div>"},
		{"plain text", "plain text"},
		{"", ""},
	}

	for _, tt := range tests {
		got := decodeUnicodeEscapes(tt.input)
		if got != tt.want {
			t.Errorf("decodeUnicodeEscapes(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseArticlePage(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
<script>
window.DATA = {"url":"https://view.inews.qq.com/a/20260217A02A7D00","article_id":"20260217A02A7D00","title":"Test Article Title","desc":"Test description","catalog1":"tech","media":"Tech Daily","media_id":"12345","pubtime":"2026-02-17 10:30:00","comment_id":"1234567890","cmsId":"20260217A02A7D00","shareImg":"https://example.com/img.jpg","article_type":"0","atype":"0","originContent":{"text":"\u003cp\u003eHello world\u003c/p\u003e"}};
window.something = {};
</script>
</body>
</html>`

	article, err := ParseArticlePage(html, "20260217A02A7D00")
	if err != nil {
		t.Fatalf("ParseArticlePage failed: %v", err)
	}

	if article.ArticleID != "20260217A02A7D00" {
		t.Errorf("ArticleID = %q, want %q", article.ArticleID, "20260217A02A7D00")
	}
	if article.Title != "Test Article Title" {
		t.Errorf("Title = %q, want %q", article.Title, "Test Article Title")
	}
	if article.Abstract != "Test description" {
		t.Errorf("Abstract = %q, want %q", article.Abstract, "Test description")
	}
	if article.Channel != "tech" {
		t.Errorf("Channel = %q, want %q", article.Channel, "tech")
	}
	if article.Source != "Tech Daily" {
		t.Errorf("Source = %q, want %q", article.Source, "Tech Daily")
	}
	if article.SourceID != "12345" {
		t.Errorf("SourceID = %q, want %q", article.SourceID, "12345")
	}
	if article.ArticleType != 0 {
		t.Errorf("ArticleType = %d, want 0", article.ArticleType)
	}
	if article.Content != "<p>Hello world</p>" {
		t.Errorf("Content = %q, want %q", article.Content, "<p>Hello world</p>")
	}
	if article.ImageURL != "https://example.com/img.jpg" {
		t.Errorf("ImageURL = %q, want %q", article.ImageURL, "https://example.com/img.jpg")
	}
	if article.PublishTime.IsZero() {
		t.Error("PublishTime should not be zero")
	}
}

func TestParseArticlePageNoData(t *testing.T) {
	html := `<!DOCTYPE html><html><body>No data here</body></html>`
	_, err := ParseArticlePage(html, "test")
	if err == nil {
		t.Error("expected error for page without window.DATA")
	}
}
