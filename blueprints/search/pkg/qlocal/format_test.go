package qlocal

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatSearchResults_AllFormats(t *testing.T) {
	results := []SearchResult{
		{
			Filepath:    "qmd://notes/readme.md",
			DisplayPath: "notes/readme.md",
			Title:       "README",
			DocID:       "abc123",
			Context:     "notes",
			Score:       0.91,
			Body:        "Compiler parser notes.\nRecursive descent parser.\n",
		},
	}
	formats := []OutputFormat{OutputCLI, OutputJSON, OutputCSV, OutputMD, OutputXML, OutputFiles}
	for _, f := range formats {
		out, err := FormatSearchResults(results, OutputOptions{Format: f, Query: "parser"})
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if strings.TrimSpace(out) == "" {
			t.Fatalf("%s produced empty output", f)
		}
		if f == OutputJSON {
			var v any
			if err := json.Unmarshal([]byte(out), &v); err != nil {
				t.Fatalf("json parse: %v", err)
			}
		}
	}
}

func TestFormatMultiGet_AllFormats(t *testing.T) {
	results := []MultiGetResult{
		{
			Doc: Document{
				DisplayPath: "notes/readme.md",
				Title:       "README",
				Context:     "notes",
				Body:        "hello\nworld\n",
			},
		},
		{
			Doc:        Document{DisplayPath: "notes/big.md", Title: "big"},
			Skipped:    true,
			SkipReason: "File too large",
		},
	}
	formats := []OutputFormat{OutputCLI, OutputJSON, OutputCSV, OutputMD, OutputXML, OutputFiles}
	for _, f := range formats {
		out, err := FormatMultiGet(results, f, false)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if strings.TrimSpace(out) == "" {
			t.Fatalf("%s produced empty output", f)
		}
	}
}
