package stories

import "time"

// StoryData represents a parsed story
type StoryData struct {
	ID               string           `json:"id"`
	ExternalID       string           `json:"external_id"`
	Title            string           `json:"title"`
	TitleTranslation string           `json:"title_translation,omitempty"`
	FromLanguage     string           `json:"from_language"`
	ToLanguage       string           `json:"to_language"`
	SetID            int              `json:"set_id"`
	SetPosition      int              `json:"set_position"`
	Difficulty       int              `json:"difficulty"`
	IconHash         string           `json:"icon_hash,omitempty"`
	Characters       []CharacterData  `json:"characters"`
	Elements         []ElementData    `json:"elements"`
	DurationSeconds  int              `json:"duration_seconds"`
	CEFRLevel        string           `json:"cefr_level,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

// CharacterData represents a story character
type CharacterData struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	VoiceID     string `json:"voice_id,omitempty"`
}

// ElementData represents a story element (line, challenge, etc.)
type ElementData struct {
	Type          ElementType       `json:"type"`
	Position      int               `json:"position"`
	SpeakerName   string            `json:"speaker_name,omitempty"`
	Text          string            `json:"text,omitempty"`
	Translation   string            `json:"translation,omitempty"`
	AudioURL      string            `json:"audio_url,omitempty"`
	AudioTiming   []AudioTimingData `json:"audio_timing,omitempty"`
	ChallengeData *ChallengeInfo    `json:"challenge_data,omitempty"`
}

// ElementType defines the type of story element
type ElementType string

const (
	ElementTypeHeader       ElementType = "header"
	ElementTypeLine         ElementType = "line"
	ElementTypeMultiChoice  ElementType = "multiple_choice"
	ElementTypeSelectPhrase ElementType = "select_phrase"
	ElementTypeArrange      ElementType = "arrange"
	ElementTypeMatch        ElementType = "match"
	ElementTypePointPhrase  ElementType = "point_to_phrase"
	ElementTypeTapComplete  ElementType = "tap_complete"
)

// AudioTimingData represents timing for word-level audio
type AudioTimingData struct {
	Word     string `json:"word"`
	Start    int    `json:"start"`    // milliseconds
	Duration int    `json:"duration"` // milliseconds
}

// ChallengeInfo contains challenge-specific data
type ChallengeInfo struct {
	Question       string   `json:"question,omitempty"`
	CorrectAnswers []string `json:"correct_answers"`
	WrongAnswers   []string `json:"wrong_answers,omitempty"`
	CorrectIndex   int      `json:"correct_index,omitempty"`
	Pairs          []Pair   `json:"pairs,omitempty"`
	Tokens         []Token  `json:"tokens,omitempty"`
}

// Pair represents a matching pair
type Pair struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// Token represents an arrangeable token
type Token struct {
	Text  string `json:"text"`
	Hint  string `json:"hint,omitempty"`
	Order int    `json:"order"`
}

// StorySet represents a collection of stories
type StorySet struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

// LanguagePair represents a from->to language pair
type LanguagePair struct {
	From string
	To   string
}

// StoriesIndex represents the index of available stories
type StoriesIndex struct {
	Pairs []PairIndex `json:"pairs"`
}

// PairIndex represents story availability for a language pair
type PairIndex struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Stories []string `json:"stories"` // Story file paths
}

// AudioBaseURL is the base URL for story audio files
const AudioBaseURL = "https://stories-cdn.duolingo.com/audio/"

// ImageBaseURL is the base URL for story images
const ImageBaseURL = "https://stories-cdn.duolingo.com/image/"

// GitHubRepoURL is the URL of the unofficial duolingo stories content repository
const GitHubRepoURL = "https://raw.githubusercontent.com/rgerum/unofficial-duolingo-stories-content/main/"

// SupportedStoryPairs contains language pairs that have stories available
var SupportedStoryPairs = []LanguagePair{
	{From: "en", To: "es"},
	{From: "en", To: "fr"},
	{From: "en", To: "de"},
	{From: "en", To: "pt"},
	{From: "en", To: "it"},
	{From: "en", To: "ja"},
	{From: "en", To: "ko"},
	{From: "en", To: "zh"},
	{From: "en", To: "ru"},
	{From: "en", To: "hi"},
	{From: "en", To: "ar"},
	{From: "en", To: "tr"},
	{From: "en", To: "nl"},
	{From: "en", To: "sv"},
	{From: "en", To: "pl"},
	{From: "en", To: "vi"},
	{From: "en", To: "el"},
	{From: "en", To: "he"},
	{From: "en", To: "id"},
	{From: "en", To: "uk"},
	{From: "en", To: "cs"},
	{From: "en", To: "da"},
	{From: "en", To: "fi"},
	{From: "en", To: "hu"},
	{From: "en", To: "nb"},
	{From: "en", To: "ro"},
	{From: "en", To: "th"},
	{From: "es", To: "en"},
	{From: "pt", To: "en"},
	{From: "fr", To: "en"},
	{From: "de", To: "en"},
}
