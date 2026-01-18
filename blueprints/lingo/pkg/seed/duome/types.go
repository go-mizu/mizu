package duome

import "time"

// VocabularyEntry represents a single vocabulary item from Duome
type VocabularyEntry struct {
	Word         string   `json:"word"`                    // Word in target language
	Romanization string   `json:"romanization,omitempty"`  // Romanized form (ja, ko, zh, ar)
	Translations []string `json:"translations"`            // English translations
	POS          string   `json:"pos,omitempty"`           // Part of speech
	SkillName    string   `json:"skill_name"`              // Parent skill name
}

// Skill represents a skill/category with its vocabulary and tips
type Skill struct {
	Name       string            `json:"name"`
	Position   int               `json:"position"`
	Vocabulary []VocabularyEntry `json:"vocabulary"`
	Tips       *SkillTips        `json:"tips,omitempty"`
}

// SkillTips contains grammar tips for a skill
type SkillTips struct {
	SkillName string    `json:"skill_name"`
	Content   string    `json:"content"`           // Markdown content
	Tables    []Table   `json:"tables,omitempty"`  // Grammar tables
	Examples  []Example `json:"examples,omitempty"`
}

// Table represents a grammar/vocabulary table
type Table struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// Example represents an example sentence
type Example struct {
	Source      string `json:"source"`      // Target language
	Translation string `json:"translation"` // English
}

// CourseData represents all data for a language pair
type CourseData struct {
	FromLanguage string    `json:"from_language"`
	ToLanguage   string    `json:"to_language"`
	Skills       []Skill   `json:"skills"`
	TotalWords   int       `json:"total_words"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// TipsData represents all tips for a language pair
type TipsData struct {
	FromLanguage string                `json:"from_language"`
	ToLanguage   string                `json:"to_language"`
	Skills       map[string]*SkillTips `json:"skills"`
	FetchedAt    time.Time             `json:"fetched_at"`
}

// Metadata tracks download state for all files
type Metadata struct {
	Downloads map[string]DownloadInfo `json:"downloads"`
	UpdatedAt time.Time               `json:"updated_at"`
}

// DownloadInfo contains information about a downloaded file
type DownloadInfo struct {
	URL         string    `json:"url"`
	FetchedAt   time.Time `json:"fetched_at"`
	ContentHash string    `json:"content_hash"`
	Size        int64     `json:"size"`
}

// LanguagePair represents a source and target language combination
type LanguagePair struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// LanguageInfo contains metadata about a language
type LanguageInfo struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
	FlagEmoji  string `json:"flag_emoji"`
	RTL        bool   `json:"rtl"`
	HasRoman   bool   `json:"has_roman"` // Has romanization in Duome
}

// SupportedLanguages contains all languages supported by Duome (from English)
var SupportedLanguages = map[string]LanguageInfo{
	"es": {Code: "es", Name: "Spanish", NativeName: "Español", FlagEmoji: "🇪🇸", RTL: false, HasRoman: false},
	"fr": {Code: "fr", Name: "French", NativeName: "Français", FlagEmoji: "🇫🇷", RTL: false, HasRoman: false},
	"de": {Code: "de", Name: "German", NativeName: "Deutsch", FlagEmoji: "🇩🇪", RTL: false, HasRoman: false},
	"it": {Code: "it", Name: "Italian", NativeName: "Italiano", FlagEmoji: "🇮🇹", RTL: false, HasRoman: false},
	"pt": {Code: "pt", Name: "Portuguese", NativeName: "Português", FlagEmoji: "🇧🇷", RTL: false, HasRoman: false},
	"nl": {Code: "nl", Name: "Dutch", NativeName: "Nederlands", FlagEmoji: "🇳🇱", RTL: false, HasRoman: false},
	"sv": {Code: "sv", Name: "Swedish", NativeName: "Svenska", FlagEmoji: "🇸🇪", RTL: false, HasRoman: false},
	"no": {Code: "no", Name: "Norwegian", NativeName: "Norsk", FlagEmoji: "🇳🇴", RTL: false, HasRoman: false},
	"da": {Code: "da", Name: "Danish", NativeName: "Dansk", FlagEmoji: "🇩🇰", RTL: false, HasRoman: false},
	"fi": {Code: "fi", Name: "Finnish", NativeName: "Suomi", FlagEmoji: "🇫🇮", RTL: false, HasRoman: false},
	"ru": {Code: "ru", Name: "Russian", NativeName: "Русский", FlagEmoji: "🇷🇺", RTL: false, HasRoman: true},
	"tr": {Code: "tr", Name: "Turkish", NativeName: "Türkçe", FlagEmoji: "🇹🇷", RTL: false, HasRoman: false},
	"ar": {Code: "ar", Name: "Arabic", NativeName: "العربية", FlagEmoji: "🇸🇦", RTL: true, HasRoman: true},
	"ja": {Code: "ja", Name: "Japanese", NativeName: "日本語", FlagEmoji: "🇯🇵", RTL: false, HasRoman: true},
	"ko": {Code: "ko", Name: "Korean", NativeName: "한국어", FlagEmoji: "🇰🇷", RTL: false, HasRoman: true},
	"zh": {Code: "zh", Name: "Chinese", NativeName: "中文", FlagEmoji: "🇨🇳", RTL: false, HasRoman: true},
	"hu": {Code: "hu", Name: "Hungarian", NativeName: "Magyar", FlagEmoji: "🇭🇺", RTL: false, HasRoman: false},
	"ro": {Code: "ro", Name: "Romanian", NativeName: "Română", FlagEmoji: "🇷🇴", RTL: false, HasRoman: false},
	"ga": {Code: "ga", Name: "Irish", NativeName: "Gaeilge", FlagEmoji: "🇮🇪", RTL: false, HasRoman: false},
	"ca": {Code: "ca", Name: "Catalan", NativeName: "Català", FlagEmoji: "🏴󠁥󠁳󠁣󠁴󠁿", RTL: false, HasRoman: false},
	"pl": {Code: "pl", Name: "Polish", NativeName: "Polski", FlagEmoji: "🇵🇱", RTL: false, HasRoman: false},
	"uk": {Code: "uk", Name: "Ukrainian", NativeName: "Українська", FlagEmoji: "🇺🇦", RTL: false, HasRoman: true},
	"cs": {Code: "cs", Name: "Czech", NativeName: "Čeština", FlagEmoji: "🇨🇿", RTL: false, HasRoman: false},
	"el": {Code: "el", Name: "Greek", NativeName: "Ελληνικά", FlagEmoji: "🇬🇷", RTL: false, HasRoman: true},
	"he": {Code: "he", Name: "Hebrew", NativeName: "עברית", FlagEmoji: "🇮🇱", RTL: true, HasRoman: true},
	"hi": {Code: "hi", Name: "Hindi", NativeName: "हिन्दी", FlagEmoji: "🇮🇳", RTL: false, HasRoman: true},
	"vi": {Code: "vi", Name: "Vietnamese", NativeName: "Tiếng Việt", FlagEmoji: "🇻🇳", RTL: false, HasRoman: false},
	"id": {Code: "id", Name: "Indonesian", NativeName: "Bahasa Indonesia", FlagEmoji: "🇮🇩", RTL: false, HasRoman: false},
	"th": {Code: "th", Name: "Thai", NativeName: "ภาษาไทย", FlagEmoji: "🇹🇭", RTL: false, HasRoman: true},
}

// GetSupportedPairs returns all supported language pairs (from English)
func GetSupportedPairs() []LanguagePair {
	pairs := make([]LanguagePair, 0, len(SupportedLanguages))
	for code := range SupportedLanguages {
		pairs = append(pairs, LanguagePair{From: "en", To: code})
	}
	return pairs
}

// GetPrimaryPairs returns the most commonly used language pairs
func GetPrimaryPairs() []LanguagePair {
	primaryLangs := []string{"es", "fr", "de", "it", "pt", "ja", "ko", "zh", "ru", "ar"}
	pairs := make([]LanguagePair, 0, len(primaryLangs))
	for _, code := range primaryLangs {
		pairs = append(pairs, LanguagePair{From: "en", To: code})
	}
	return pairs
}

// String returns a string representation of the language pair
func (lp LanguagePair) String() string {
	return lp.From + "/" + lp.To
}

// VocabularyURL returns the Duome vocabulary URL for this pair
func (lp LanguagePair) VocabularyURL() string {
	return "https://duome.eu/vocabulary/" + lp.From + "/" + lp.To + "/skills"
}

// TipsURL returns the Duome tips URL for this pair
func (lp LanguagePair) TipsURL() string {
	return "https://duome.eu/tips/" + lp.From + "/" + lp.To
}

// ProgressCallback is called during long operations to report progress
type ProgressCallback func(current, total int, message string)
