package view

import (
	"strings"
	"testing"
)

func TestConvertWikiTextToMarkdown_Links(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wikiname string
		want     string // substring to check
	}{
		{
			name:     "simple link",
			input:    "See [[Alan Turing]] for more",
			wikiname: "viwiki",
			want:     "[Alan Turing](/page?wiki=viwiki&title=Alan+Turing)",
		},
		{
			name:     "piped link",
			input:    "the [[Bletchley Park|secret base]]",
			wikiname: "viwiki",
			want:     "[secret base](/page?wiki=viwiki&title=Bletchley+Park)",
		},
		{
			name:     "vietnamese link",
			input:    "Trong [[Chiến tranh thế giới thứ hai]]",
			wikiname: "viwiki",
			want:     "[Chiến tranh thế giới thứ hai](/page?wiki=viwiki&title=Chi%E1%BA%BFn+tranh+th%E1%BA%BF+gi%E1%BB%9Bi+th%E1%BB%A9+hai)",
		},
		{
			name:     "link with section",
			input:    "See [[Page#Section]]",
			wikiname: "enwiki",
			want:     "[Page#Section](/page?wiki=enwiki&title=Page)",
		},
		{
			name:     "multiple links",
			input:    "From [[A]] to [[B]]",
			wikiname: "enwiki",
			want:     "[A](/page?wiki=enwiki&title=A)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWikiTextToMarkdown(tt.input, tt.wikiname)
			if !strings.Contains(got, tt.want) {
				t.Errorf("ConvertWikiTextToMarkdown() = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestConvertWikiTextToMarkdown_Formatting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bold",
			input: "This is '''bold''' text",
			want:  "This is **bold** text",
		},
		{
			name:  "italic",
			input: "This is ''italic'' text",
			want:  "This is *italic* text",
		},
		{
			name:  "bold and italic",
			input: "'''bold''' and ''italic''",
			want:  "**bold** and *italic*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWikiTextToMarkdown(tt.input, "enwiki")
			if got != tt.want {
				t.Errorf("ConvertWikiTextToMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertWikiTextToMarkdown_Headings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "h2",
			input: "== Heading ==",
			want:  "## Heading",
		},
		{
			name:  "h3",
			input: "=== Subheading ===",
			want:  "### Subheading",
		},
		{
			name:  "h4",
			input: "==== Deep ====",
			want:  "#### Deep",
		},
		{
			name:  "h2 with content",
			input: "Text before\n== Section ==\nText after",
			want:  "## Section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWikiTextToMarkdown(tt.input, "enwiki")
			if !strings.Contains(got, tt.want) {
				t.Errorf("ConvertWikiTextToMarkdown() = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestConvertWikiTextToMarkdown_StripTemplates(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		notWant  string
	}{
		{
			name:    "simple template",
			input:   "Text {{cite web|url=...}} more",
			notWant: "{{",
		},
		{
			name:    "infobox",
			input:   "{{Infobox person|name=Alan}}\nBio text",
			notWant: "Infobox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWikiTextToMarkdown(tt.input, "enwiki")
			if strings.Contains(got, tt.notWant) {
				t.Errorf("ConvertWikiTextToMarkdown() = %q, should not contain %q", got, tt.notWant)
			}
		})
	}
}

func TestConvertWikiTextToMarkdown_StripRefs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		notWant string
	}{
		{
			name:    "ref block",
			input:   "Some text<ref>Citation</ref> more text",
			notWant: "<ref",
		},
		{
			name:    "ref with name",
			input:   `Text<ref name="foo">Citation</ref>`,
			notWant: "<ref",
		},
		{
			name:    "self-closing ref",
			input:   `Text<ref name="bar" /> more`,
			notWant: "<ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWikiTextToMarkdown(tt.input, "enwiki")
			if strings.Contains(got, tt.notWant) {
				t.Errorf("ConvertWikiTextToMarkdown() = %q, should not contain %q", got, tt.notWant)
			}
		})
	}
}

func TestConvertWikiTextToMarkdown_StripSpecialLinks(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		notWant string
	}{
		{
			name:    "file link",
			input:   "[[File:Photo.jpg|thumb|Caption]]",
			notWant: "File:",
		},
		{
			name:    "image link",
			input:   "[[Image:Photo.png]]",
			notWant: "Image:",
		},
		{
			name:    "category link",
			input:   "[[Category:Scientists]]",
			notWant: "Category:",
		},
		{
			name:    "vietnamese file",
			input:   "[[Tập tin:Anh.jpg]]",
			notWant: "Tập tin:",
		},
		{
			name:    "vietnamese category",
			input:   "[[Thể loại:Nhà khoa học]]",
			notWant: "Thể loại:",
		},
		{
			name:    "interwiki",
			input:   "[[en:Alan Turing]]",
			notWant: "en:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWikiTextToMarkdown(tt.input, "viwiki")
			if strings.Contains(got, tt.notWant) {
				t.Errorf("ConvertWikiTextToMarkdown() = %q, should not contain %q", got, tt.notWant)
			}
		})
	}
}

func TestConvertWikiTextToMarkdown_ExternalLinks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "external with text",
			input: "[https://example.com Example Site]",
			want:  "[Example Site](https://example.com)",
		},
		{
			name:  "bare external",
			input: "[https://example.com]",
			want:  "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWikiTextToMarkdown(tt.input, "enwiki")
			if !strings.Contains(got, tt.want) {
				t.Errorf("ConvertWikiTextToMarkdown() = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestConvertWikiTextToMarkdown_AlanTuring(t *testing.T) {
	// Test with actual snippet from Alan Turing page
	input := `'''Alan Mathison Turing''' [[Huân chương Đế quốc Anh|OBE]] [[Thành viên Hội Hoàng gia|FRS]] ([[23 tháng 6]] năm [[1912]] – [[7 tháng 6]] năm [[1954]]) là một [[danh sách nhà toán học|nhà toán học]], [[logic|logic học]] và [[mật mã học]] người [[Anh]], được xem là một trong những nhà tiên phong của ngành [[khoa học máy tính]].`

	got := ConvertWikiTextToMarkdown(input, "viwiki")

	// Should convert bold
	if !strings.Contains(got, "**Alan Mathison Turing**") {
		t.Error("Should convert '''bold''' to **bold**")
	}

	// Should convert piped link
	if !strings.Contains(got, "[OBE](/page?wiki=viwiki") {
		t.Error("Should convert [[X|Y]] to [Y](link)")
	}

	// Should have internal links
	if !strings.Contains(got, "[khoa học máy tính](/page?wiki=viwiki") {
		t.Error("Should convert [[Page]] to [Page](link)")
	}
}

func TestConvertWikiTextToMarkdown_EmptyInput(t *testing.T) {
	got := ConvertWikiTextToMarkdown("", "enwiki")
	if got != "" {
		t.Errorf("Empty input should return empty string, got %q", got)
	}
}
