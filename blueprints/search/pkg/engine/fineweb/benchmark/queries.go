// Package benchmark provides benchmarking utilities for fineweb search drivers.
package benchmark

// Query represents a test query with metadata.
type Query struct {
	Text      string    `json:"text"`
	Type      QueryType `json:"type"`
	Frequency Frequency `json:"frequency"`
}

// QueryType categorizes the query structure.
type QueryType string

const (
	QuerySingleWord QueryType = "single_word"
	QueryPhrase     QueryType = "phrase"
	QueryNumeric    QueryType = "numeric"
	QueryMixed      QueryType = "mixed"
)

// Frequency indicates how common the terms are in the corpus.
type Frequency string

const (
	FreqCommon Frequency = "common"
	FreqMedium Frequency = "medium"
	FreqRare   Frequency = "rare"
)

// VietnameseQueries contains test queries for Vietnamese data.
var VietnameseQueries = []Query{
	// Single word - common
	{Text: "Việt Nam", Type: QuerySingleWord, Frequency: FreqCommon},
	{Text: "thành phố", Type: QuerySingleWord, Frequency: FreqCommon},
	{Text: "người", Type: QuerySingleWord, Frequency: FreqCommon},
	{Text: "năm", Type: QuerySingleWord, Frequency: FreqCommon},

	// Single word - medium
	{Text: "công nghệ", Type: QuerySingleWord, Frequency: FreqMedium},
	{Text: "kinh tế", Type: QuerySingleWord, Frequency: FreqMedium},
	{Text: "giáo dục", Type: QuerySingleWord, Frequency: FreqMedium},

	// Single word - rare
	{Text: "blockchain", Type: QuerySingleWord, Frequency: FreqRare},
	{Text: "cryptocurrency", Type: QuerySingleWord, Frequency: FreqRare},
	{Text: "metaverse", Type: QuerySingleWord, Frequency: FreqRare},

	// Multi-word phrase
	{Text: "công nghệ thông tin", Type: QueryPhrase, Frequency: FreqMedium},
	{Text: "trí tuệ nhân tạo", Type: QueryPhrase, Frequency: FreqMedium},
	{Text: "Hồ Chí Minh", Type: QueryPhrase, Frequency: FreqCommon},

	// Numbers/dates
	{Text: "2024", Type: QueryNumeric, Frequency: FreqCommon},
	{Text: "2023", Type: QueryNumeric, Frequency: FreqCommon},

	// Mixed content
	{Text: "COVID-19", Type: QueryMixed, Frequency: FreqMedium},
	{Text: "internet", Type: QueryMixed, Frequency: FreqCommon},
	{Text: "AI", Type: QueryMixed, Frequency: FreqMedium},
}

// EnglishQueries contains test queries for English data.
var EnglishQueries = []Query{
	// Single word - common
	{Text: "the", Type: QuerySingleWord, Frequency: FreqCommon},
	{Text: "technology", Type: QuerySingleWord, Frequency: FreqCommon},
	{Text: "data", Type: QuerySingleWord, Frequency: FreqCommon},

	// Single word - medium
	{Text: "artificial intelligence", Type: QueryPhrase, Frequency: FreqMedium},
	{Text: "machine learning", Type: QueryPhrase, Frequency: FreqMedium},

	// Single word - rare
	{Text: "cryptocurrency", Type: QuerySingleWord, Frequency: FreqRare},
	{Text: "decentralized", Type: QuerySingleWord, Frequency: FreqRare},

	// Numbers
	{Text: "2024", Type: QueryNumeric, Frequency: FreqCommon},
}

// GetQueries returns queries based on language.
func GetQueries(language string) []Query {
	switch language {
	case "vie_Latn", "vi", "vietnamese":
		return VietnameseQueries
	case "eng_Latn", "en", "english":
		return EnglishQueries
	default:
		return VietnameseQueries // Default to Vietnamese
	}
}
