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

// SupportedLanguages contains all languages supported by Duome
var SupportedLanguages = map[string]LanguageInfo{
	// Major languages
	"en": {Code: "en", Name: "English", NativeName: "English", FlagEmoji: "🇺🇸", RTL: false, HasRoman: false},
	"es": {Code: "es", Name: "Spanish", NativeName: "Español", FlagEmoji: "🇪🇸", RTL: false, HasRoman: false},
	"fr": {Code: "fr", Name: "French", NativeName: "Français", FlagEmoji: "🇫🇷", RTL: false, HasRoman: false},
	"de": {Code: "de", Name: "German", NativeName: "Deutsch", FlagEmoji: "🇩🇪", RTL: false, HasRoman: false},
	"it": {Code: "it", Name: "Italian", NativeName: "Italiano", FlagEmoji: "🇮🇹", RTL: false, HasRoman: false},
	"pt": {Code: "pt", Name: "Portuguese", NativeName: "Português", FlagEmoji: "🇧🇷", RTL: false, HasRoman: false},
	"nl": {Code: "nl", Name: "Dutch", NativeName: "Nederlands", FlagEmoji: "🇳🇱", RTL: false, HasRoman: false},
	"sv": {Code: "sv", Name: "Swedish", NativeName: "Svenska", FlagEmoji: "🇸🇪", RTL: false, HasRoman: false},
	"no": {Code: "no", Name: "Norwegian", NativeName: "Norsk", FlagEmoji: "🇳🇴", RTL: false, HasRoman: false},
	"nb": {Code: "nb", Name: "Norwegian Bokmål", NativeName: "Norsk Bokmål", FlagEmoji: "🇳🇴", RTL: false, HasRoman: false},
	"da": {Code: "da", Name: "Danish", NativeName: "Dansk", FlagEmoji: "🇩🇰", RTL: false, HasRoman: false},
	"fi": {Code: "fi", Name: "Finnish", NativeName: "Suomi", FlagEmoji: "🇫🇮", RTL: false, HasRoman: false},
	"ru": {Code: "ru", Name: "Russian", NativeName: "Русский", FlagEmoji: "🇷🇺", RTL: false, HasRoman: true},
	"tr": {Code: "tr", Name: "Turkish", NativeName: "Türkçe", FlagEmoji: "🇹🇷", RTL: false, HasRoman: false},
	"ar": {Code: "ar", Name: "Arabic", NativeName: "العربية", FlagEmoji: "🇸🇦", RTL: true, HasRoman: true},
	"ja": {Code: "ja", Name: "Japanese", NativeName: "日本語", FlagEmoji: "🇯🇵", RTL: false, HasRoman: true},
	"ko": {Code: "ko", Name: "Korean", NativeName: "한국어", FlagEmoji: "🇰🇷", RTL: false, HasRoman: true},
	"zh": {Code: "zh", Name: "Chinese", NativeName: "中文", FlagEmoji: "🇨🇳", RTL: false, HasRoman: true},
	"zs": {Code: "zs", Name: "Chinese (Simplified)", NativeName: "简体中文", FlagEmoji: "🇨🇳", RTL: false, HasRoman: true},
	"zc": {Code: "zc", Name: "Cantonese", NativeName: "廣東話", FlagEmoji: "🇭🇰", RTL: false, HasRoman: true},
	"hu": {Code: "hu", Name: "Hungarian", NativeName: "Magyar", FlagEmoji: "🇭🇺", RTL: false, HasRoman: false},
	"ro": {Code: "ro", Name: "Romanian", NativeName: "Română", FlagEmoji: "🇷🇴", RTL: false, HasRoman: false},
	"ga": {Code: "ga", Name: "Irish", NativeName: "Gaeilge", FlagEmoji: "🇮🇪", RTL: false, HasRoman: false},
	"gd": {Code: "gd", Name: "Scottish Gaelic", NativeName: "Gàidhlig", FlagEmoji: "🏴󠁧󠁢󠁳󠁣󠁴󠁿", RTL: false, HasRoman: false},
	"cy": {Code: "cy", Name: "Welsh", NativeName: "Cymraeg", FlagEmoji: "🏴󠁧󠁢󠁷󠁬󠁳󠁿", RTL: false, HasRoman: false},
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
	// Constructed and special languages
	"eo": {Code: "eo", Name: "Esperanto", NativeName: "Esperanto", FlagEmoji: "🟢", RTL: false, HasRoman: false},
	"la": {Code: "la", Name: "Latin", NativeName: "Latina", FlagEmoji: "🏛️", RTL: false, HasRoman: false},
	"kl": {Code: "kl", Name: "Klingon", NativeName: "tlhIngan Hol", FlagEmoji: "🖖", RTL: false, HasRoman: false},
	"hv": {Code: "hv", Name: "High Valyrian", NativeName: "High Valyrian", FlagEmoji: "🐉", RTL: false, HasRoman: false},
	// Indigenous and regional languages
	"gn": {Code: "gn", Name: "Guarani", NativeName: "Avañe'ẽ", FlagEmoji: "🇵🇾", RTL: false, HasRoman: false},
	"hw": {Code: "hw", Name: "Hawaiian", NativeName: "ʻŌlelo Hawaiʻi", FlagEmoji: "🌺", RTL: false, HasRoman: false},
	"nv": {Code: "nv", Name: "Navajo", NativeName: "Diné bizaad", FlagEmoji: "🏜️", RTL: false, HasRoman: false},
	"sw": {Code: "sw", Name: "Swahili", NativeName: "Kiswahili", FlagEmoji: "🇰🇪", RTL: false, HasRoman: false},
	"zu": {Code: "zu", Name: "Zulu", NativeName: "isiZulu", FlagEmoji: "🇿🇦", RTL: false, HasRoman: false},
	"yi": {Code: "yi", Name: "Yiddish", NativeName: "ייִדיש", FlagEmoji: "🕎", RTL: true, HasRoman: true},
	"ht": {Code: "ht", Name: "Haitian Creole", NativeName: "Kreyòl ayisyen", FlagEmoji: "🇭🇹", RTL: false, HasRoman: false},
	"dn": {Code: "dn", Name: "Dutch", NativeName: "Nederlands", FlagEmoji: "🇳🇱", RTL: false, HasRoman: false},
}

// AllLanguagePairs contains all known language pairs from Duome
// Discovered from https://duome.eu/vocabulary
var AllLanguagePairs = []LanguagePair{
	// From English (en) - 40 targets
	{From: "en", To: "ar"}, {From: "en", To: "cs"}, {From: "en", To: "cy"}, {From: "en", To: "da"},
	{From: "en", To: "de"}, {From: "en", To: "dn"}, {From: "en", To: "el"}, {From: "en", To: "eo"},
	{From: "en", To: "es"}, {From: "en", To: "fi"}, {From: "en", To: "fr"}, {From: "en", To: "ga"},
	{From: "en", To: "gd"}, {From: "en", To: "he"}, {From: "en", To: "hi"}, {From: "en", To: "ht"},
	{From: "en", To: "hu"}, {From: "en", To: "hv"}, {From: "en", To: "hw"}, {From: "en", To: "id"},
	{From: "en", To: "it"}, {From: "en", To: "ja"}, {From: "en", To: "kl"}, {From: "en", To: "ko"},
	{From: "en", To: "la"}, {From: "en", To: "nb"}, {From: "en", To: "nv"}, {From: "en", To: "pl"},
	{From: "en", To: "pt"}, {From: "en", To: "ro"}, {From: "en", To: "ru"}, {From: "en", To: "sv"},
	{From: "en", To: "sw"}, {From: "en", To: "tr"}, {From: "en", To: "uk"}, {From: "en", To: "vi"},
	{From: "en", To: "yi"}, {From: "en", To: "zs"}, {From: "en", To: "zu"},
	// From Spanish (es) - 13 targets
	{From: "es", To: "ca"}, {From: "es", To: "de"}, {From: "es", To: "en"}, {From: "es", To: "eo"},
	{From: "es", To: "fr"}, {From: "es", To: "gn"}, {From: "es", To: "it"}, {From: "es", To: "ja"},
	{From: "es", To: "ko"}, {From: "es", To: "pt"}, {From: "es", To: "ru"}, {From: "es", To: "sv"},
	{From: "es", To: "zs"},
	// From German (de) - 8 targets
	{From: "de", To: "en"}, {From: "de", To: "es"}, {From: "de", To: "fr"}, {From: "de", To: "it"},
	{From: "de", To: "ja"}, {From: "de", To: "ko"}, {From: "de", To: "pt"}, {From: "de", To: "zs"},
	// From French (fr) - 9 targets
	{From: "fr", To: "de"}, {From: "fr", To: "en"}, {From: "fr", To: "eo"}, {From: "fr", To: "es"},
	{From: "fr", To: "it"}, {From: "fr", To: "ja"}, {From: "fr", To: "ko"}, {From: "fr", To: "pt"},
	{From: "fr", To: "zs"},
	// From Portuguese (pt) - 9 targets
	{From: "pt", To: "de"}, {From: "pt", To: "en"}, {From: "pt", To: "eo"}, {From: "pt", To: "es"},
	{From: "pt", To: "fr"}, {From: "pt", To: "it"}, {From: "pt", To: "ja"}, {From: "pt", To: "ko"},
	{From: "pt", To: "zs"},
	// From Italian (it) - 8 targets
	{From: "it", To: "de"}, {From: "it", To: "en"}, {From: "it", To: "es"}, {From: "it", To: "fr"},
	{From: "it", To: "ja"}, {From: "it", To: "ko"}, {From: "it", To: "pt"}, {From: "it", To: "zs"},
	// From Japanese (ja) - 8 targets
	{From: "ja", To: "de"}, {From: "ja", To: "en"}, {From: "ja", To: "es"}, {From: "ja", To: "fr"},
	{From: "ja", To: "it"}, {From: "ja", To: "ko"}, {From: "ja", To: "pt"}, {From: "ja", To: "zs"},
	// From Korean (ko) - 8 targets
	{From: "ko", To: "de"}, {From: "ko", To: "en"}, {From: "ko", To: "es"}, {From: "ko", To: "fr"},
	{From: "ko", To: "it"}, {From: "ko", To: "ja"}, {From: "ko", To: "pt"}, {From: "ko", To: "zs"},
	// From Chinese Simplified (zs) - 9 targets
	{From: "zs", To: "de"}, {From: "zs", To: "en"}, {From: "zs", To: "es"}, {From: "zs", To: "fr"},
	{From: "zs", To: "it"}, {From: "zs", To: "ja"}, {From: "zs", To: "ko"}, {From: "zs", To: "pt"},
	{From: "zs", To: "zc"},
	// From Russian (ru) - 9 targets
	{From: "ru", To: "de"}, {From: "ru", To: "en"}, {From: "ru", To: "es"}, {From: "ru", To: "fr"},
	{From: "ru", To: "it"}, {From: "ru", To: "ja"}, {From: "ru", To: "ko"}, {From: "ru", To: "pt"},
	{From: "ru", To: "zs"},
	// From Arabic (ar) - 10 targets
	{From: "ar", To: "de"}, {From: "ar", To: "en"}, {From: "ar", To: "es"}, {From: "ar", To: "fr"},
	{From: "ar", To: "it"}, {From: "ar", To: "ja"}, {From: "ar", To: "ko"}, {From: "ar", To: "pt"},
	{From: "ar", To: "sv"}, {From: "ar", To: "zs"},
	// From Hindi (hi) - 9 targets
	{From: "hi", To: "de"}, {From: "hi", To: "en"}, {From: "hi", To: "es"}, {From: "hi", To: "fr"},
	{From: "hi", To: "it"}, {From: "hi", To: "ja"}, {From: "hi", To: "ko"}, {From: "hi", To: "pt"},
	{From: "hi", To: "zs"},
	// From Turkish (tr) - 10 targets
	{From: "tr", To: "de"}, {From: "tr", To: "en"}, {From: "tr", To: "es"}, {From: "tr", To: "fr"},
	{From: "tr", To: "it"}, {From: "tr", To: "ja"}, {From: "tr", To: "ko"}, {From: "tr", To: "pt"},
	{From: "tr", To: "ru"}, {From: "tr", To: "zs"},
	// From Vietnamese (vi) - 9 targets
	{From: "vi", To: "de"}, {From: "vi", To: "en"}, {From: "vi", To: "es"}, {From: "vi", To: "fr"},
	{From: "vi", To: "it"}, {From: "vi", To: "ja"}, {From: "vi", To: "ko"}, {From: "vi", To: "pt"},
	{From: "vi", To: "zs"},
	// From Indonesian (id) - 8 targets
	{From: "id", To: "de"}, {From: "id", To: "en"}, {From: "id", To: "es"}, {From: "id", To: "fr"},
	{From: "id", To: "it"}, {From: "id", To: "ja"}, {From: "id", To: "pt"}, {From: "id", To: "zs"},
	// From Ukrainian (uk) - 9 targets
	{From: "uk", To: "de"}, {From: "uk", To: "en"}, {From: "uk", To: "es"}, {From: "uk", To: "fr"},
	{From: "uk", To: "it"}, {From: "uk", To: "ja"}, {From: "uk", To: "ko"}, {From: "uk", To: "pt"},
	{From: "uk", To: "zs"},
	// From Polish (pl) - 9 targets
	{From: "pl", To: "de"}, {From: "pl", To: "en"}, {From: "pl", To: "es"}, {From: "pl", To: "fr"},
	{From: "pl", To: "it"}, {From: "pl", To: "ja"}, {From: "pl", To: "ko"}, {From: "pl", To: "pt"},
	{From: "pl", To: "zs"},
}

// GetSupportedPairs returns all known language pairs
func GetSupportedPairs() []LanguagePair {
	return AllLanguagePairs
}

// GetPairsFromEnglish returns all pairs where English is the source language
func GetPairsFromEnglish() []LanguagePair {
	var pairs []LanguagePair
	for _, p := range AllLanguagePairs {
		if p.From == "en" {
			pairs = append(pairs, p)
		}
	}
	return pairs
}

// GetPrimaryPairs returns the most commonly used language pairs (from English)
func GetPrimaryPairs() []LanguagePair {
	primaryLangs := []string{"es", "fr", "de", "it", "pt", "ja", "ko", "zs", "ru", "ar"}
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
