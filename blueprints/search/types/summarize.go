// Package types contains shared data types for the search blueprint.
package types

import "time"

// SummaryEngine represents a summarization engine type.
type SummaryEngine string

const (
	EngineCecil  SummaryEngine = "cecil"  // Friendly, descriptive, fast
	EngineAgnes  SummaryEngine = "agnes"  // Formal, technical, analytical
	EngineMuriel SummaryEngine = "muriel" // Enterprise-grade, highest quality
)

// SummaryType represents the output format type.
type SummaryType string

const (
	SummaryTypeSummary  SummaryType = "summary"  // Paragraph prose
	SummaryTypeTakeaway SummaryType = "takeaway" // Bulleted key points
)

// SupportedLanguages for summarization translation.
var SupportedLanguages = map[string]string{
	"EN":      "English",
	"ES":      "Spanish",
	"FR":      "French",
	"DE":      "German",
	"IT":      "Italian",
	"PT":      "Portuguese",
	"NL":      "Dutch",
	"PL":      "Polish",
	"RU":      "Russian",
	"JA":      "Japanese",
	"ZH":      "Chinese (Simplified)",
	"ZH-HANT": "Chinese (Traditional)",
	"KO":      "Korean",
	"AR":      "Arabic",
	"TR":      "Turkish",
	"VI":      "Vietnamese",
	"TH":      "Thai",
	"ID":      "Indonesian",
	"CS":      "Czech",
	"DA":      "Danish",
	"FI":      "Finnish",
	"EL":      "Greek",
	"HU":      "Hungarian",
	"RO":      "Romanian",
	"SV":      "Swedish",
	"UK":      "Ukrainian",
	"BG":      "Bulgarian",
}

// SummarizeRequest represents a summarization request.
type SummarizeRequest struct {
	URL            string        `json:"url,omitempty"`
	Text           string        `json:"text,omitempty"`
	Engine         SummaryEngine `json:"engine,omitempty"`         // Default: cecil
	SummaryType    SummaryType   `json:"summary_type,omitempty"`   // Default: summary
	TargetLanguage string        `json:"target_language,omitempty"` // Language code
	Cache          *bool         `json:"cache,omitempty"`          // Default: true
}

// SummarizeResponse represents a summarization response.
type SummarizeResponse struct {
	Meta SummaryMeta `json:"meta"`
	Data SummaryData `json:"data"`
}

// SummaryMeta contains request metadata.
type SummaryMeta struct {
	ID   string `json:"id"`
	Node string `json:"node"`
	Ms   int64  `json:"ms"`
}

// SummaryData contains the summary output.
type SummaryData struct {
	Output string `json:"output"`
	Tokens int    `json:"tokens"`
}

// SummaryCache represents a cached summary.
type SummaryCache struct {
	ID             int64         `json:"id"`
	URLHash        string        `json:"url_hash"`
	URL            string        `json:"url"`
	Engine         SummaryEngine `json:"engine"`
	SummaryType    SummaryType   `json:"summary_type"`
	TargetLanguage string        `json:"target_language,omitempty"`
	Output         string        `json:"output"`
	Tokens         int           `json:"tokens"`
	CreatedAt      time.Time     `json:"created_at"`
	ExpiresAt      time.Time     `json:"expires_at"`
}
