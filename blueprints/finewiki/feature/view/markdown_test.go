package view

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wikiname string
		contains []string
	}{
		{
			name:     "simple paragraph",
			input:    "Hello world",
			wikiname: "enwiki",
			contains: []string{"<p>Hello world</p>"},
		},
		{
			name:     "heading",
			input:    "# Title\n\nSome text",
			wikiname: "enwiki",
			contains: []string{"<h1", "Title", "<p>Some text</p>"},
		},
		{
			name:     "bold and italic",
			input:    "This is **bold** and *italic*",
			wikiname: "enwiki",
			contains: []string{"<strong>bold</strong>", "<em>italic</em>"},
		},
		{
			name:     "list",
			input:    "- Item 1\n- Item 2",
			wikiname: "enwiki",
			contains: []string{"<ul>", "<li>Item 1</li>", "<li>Item 2</li>"},
		},
		{
			name:     "wiki link simple",
			input:    "See [[Alan Turing]] for more",
			wikiname: "enwiki",
			contains: []string{`href="/page?wiki=enwiki&amp;title=Alan+Turing"`, ">Alan Turing</a>"},
		},
		{
			name:     "wiki link with display text",
			input:    "See [[Alan Turing|the famous mathematician]] for more",
			wikiname: "enwiki",
			contains: []string{`href="/page?wiki=enwiki&amp;title=Alan+Turing"`, ">the famous mathematician</a>"},
		},
		{
			name:     "multiple wiki links",
			input:    "[[Apple]] and [[Orange]] are fruits",
			wikiname: "enwiki",
			contains: []string{`title=Apple"`, `title=Orange"`},
		},
		{
			name:     "table",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			wikiname: "enwiki",
			contains: []string{"<table>", "<th>A</th>", "<td>1</td>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderMarkdown(tt.input, tt.wikiname)
			if err != nil {
				t.Fatalf("RenderMarkdown() error = %v", err)
			}

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("RenderMarkdown() result missing %q\nGot: %s", s, result)
				}
			}
		})
	}
}

func TestConvertWikiLinks(t *testing.T) {
	tests := []struct {
		input    string
		wikiname string
		want     string
	}{
		{
			input:    "[[Page Name]]",
			wikiname: "enwiki",
			want:     "[Page Name](/page?wiki=enwiki&title=Page+Name)",
		},
		{
			input:    "[[Page Name|Display Text]]",
			wikiname: "enwiki",
			want:     "[Display Text](/page?wiki=enwiki&title=Page+Name)",
		},
		{
			input:    "No links here",
			wikiname: "enwiki",
			want:     "No links here",
		},
		{
			input:    "[[First]] and [[Second]]",
			wikiname: "viwiki",
			want:     "[First](/page?wiki=viwiki&title=First) and [Second](/page?wiki=viwiki&title=Second)",
		},
		{
			input:    "[[Page with spaces]]",
			wikiname: "enwiki",
			want:     "[Page with spaces](/page?wiki=enwiki&title=Page+with+spaces)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := convertWikiLinks(tt.input, tt.wikiname)
			if got != tt.want {
				t.Errorf("convertWikiLinks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wikiname string
		contains []string
	}{
		{
			name:     "simple text",
			input:    "Hello world",
			wikiname: "enwiki",
			contains: []string{"<p>Hello world</p>"},
		},
		{
			name:     "html escaping",
			input:    "<script>alert('xss')</script>",
			wikiname: "enwiki",
			contains: []string{"&lt;script&gt;"},
		},
		{
			name:     "paragraphs",
			input:    "First paragraph\n\nSecond paragraph",
			wikiname: "enwiki",
			contains: []string{"<p>First paragraph</p>", "<p>Second paragraph</p>"},
		},
		{
			name:     "wiki links in text",
			input:    "See [[Alan Turing]] for more",
			wikiname: "enwiki",
			contains: []string{"[Alan Turing](/page?wiki=enwiki&amp;title=Alan+Turing)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderText(tt.input, tt.wikiname)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("RenderText() result missing %q\nGot: %s", s, result)
				}
			}
		})
	}
}
