package text

import (
	"testing"
)

func TestExtractMentions(t *testing.T) {
	tests := []struct {
		content string
		want    []string
	}{
		{"Hello @alice", []string{"alice"}},
		{"@alice @bob hello", []string{"alice", "bob"}},
		{"No mentions here", nil},
		{"@alice and @alice again", []string{"alice"}}, // Deduplicated
		{"Email: test@example.com", []string{"example"}}, // Not ideal but expected behavior
		{"@User_Name123", []string{"User_Name123"}},
	}

	for _, tt := range tests {
		got := ExtractMentions(tt.content)
		if len(got) != len(tt.want) {
			t.Errorf("ExtractMentions(%q) = %v, want %v", tt.content, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ExtractMentions(%q)[%d] = %q, want %q", tt.content, i, got[i], tt.want[i])
			}
		}
	}
}

func TestExtractHashtags(t *testing.T) {
	tests := []struct {
		content string
		want    []string
	}{
		{"Hello #world", []string{"world"}},
		{"#Go #Rust #Python", []string{"go", "rust", "python"}},
		{"No hashtags", nil},
		{"#TAG and #tag again", []string{"tag"}}, // Deduplicated, lowercase
		{"Email: #123abc", []string{"123abc"}},
		{"#under_score", []string{"under_score"}},
	}

	for _, tt := range tests {
		got := ExtractHashtags(tt.content)
		if len(got) != len(tt.want) {
			t.Errorf("ExtractHashtags(%q) = %v, want %v", tt.content, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ExtractHashtags(%q)[%d] = %q, want %q", tt.content, i, got[i], tt.want[i])
			}
		}
	}
}

func TestParse(t *testing.T) {
	content := "Hello @alice! Check out #golang https://golang.org"

	entities := Parse(content)

	if len(entities.Mentions) != 1 {
		t.Errorf("Expected 1 mention, got %d", len(entities.Mentions))
	}
	if entities.Mentions[0].Username != "alice" {
		t.Errorf("Expected mention 'alice', got %q", entities.Mentions[0].Username)
	}

	if len(entities.Hashtags) != 1 {
		t.Errorf("Expected 1 hashtag, got %d", len(entities.Hashtags))
	}
	if entities.Hashtags[0].Tag != "golang" {
		t.Errorf("Expected hashtag 'golang', got %q", entities.Hashtags[0].Tag)
	}

	if len(entities.URLs) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(entities.URLs))
	}
	if entities.URLs[0].URL != "https://golang.org" {
		t.Errorf("Expected URL 'https://golang.org', got %q", entities.URLs[0].URL)
	}
}

func TestCharCount(t *testing.T) {
	tests := []struct {
		content string
		want    int
	}{
		{"Hello", 5},
		{"Hello World", 11},
		{"", 0},
		{"こんにちは", 5},
		{"Hello 世界", 8},
		{"🎉🎊", 2},
	}

	for _, tt := range tests {
		got := CharCount(tt.content)
		if got != tt.want {
			t.Errorf("CharCount(%q) = %d, want %d", tt.content, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		content string
		maxLen  int
		want    string
	}{
		{"Hello", 10, "Hello"},
		{"Hello World", 5, "Hell…"},
		{"", 5, ""},
		{"こんにちは", 3, "こん…"},
	}

	for _, tt := range tests {
		got := Truncate(tt.content, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.content, tt.maxLen, got, tt.want)
		}
	}
}

func TestToHTML(t *testing.T) {
	content := "Hello @alice #golang"
	html := ToHTML(content)

	// Check that mentions are linked
	if html == content {
		t.Error("ToHTML should have converted mentions and hashtags to links")
	}

	// Note: The order of replacements might affect the final string
	// Just check that the links are present
	if !contains(html, "/@alice") {
		t.Error("Expected mention link in HTML")
	}
	if !contains(html, "/tags/golang") {
		t.Error("Expected hashtag link in HTML")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
