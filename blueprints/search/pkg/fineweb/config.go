package fineweb

import (
	"os"
	"path/filepath"
	"time"
)

// Config holds downloader configuration.
type Config struct {
	DataDir     string        // Base directory for downloads (default: $HOME/data/fineweb-2)
	Concurrency int           // Concurrent downloads (default: 3)
	Timeout     time.Duration // Per-file timeout (default: 30 minutes)
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DataDir:     filepath.Join(home, "data", "fineweb-2"),
		Concurrency: 3,
		Timeout:     30 * time.Minute,
	}
}

// SupportedLanguages lists the initially supported languages.
var SupportedLanguages = []Language{
	{Code: "vie_Latn", Name: "Vietnamese", Script: "Latin"},
	{Code: "eng_Latn", Name: "English", Script: "Latin"},
	{Code: "fra_Latn", Name: "French", Script: "Latin"},
	{Code: "deu_Latn", Name: "German", Script: "Latin"},
	{Code: "spa_Latn", Name: "Spanish", Script: "Latin"},
	{Code: "ita_Latn", Name: "Italian", Script: "Latin"},
	{Code: "por_Latn", Name: "Portuguese", Script: "Latin"},
	{Code: "nld_Latn", Name: "Dutch", Script: "Latin"},
	{Code: "pol_Latn", Name: "Polish", Script: "Latin"},
	{Code: "ron_Latn", Name: "Romanian", Script: "Latin"},
	{Code: "ces_Latn", Name: "Czech", Script: "Latin"},
	{Code: "hun_Latn", Name: "Hungarian", Script: "Latin"},
}

// GetLanguage returns language info by code.
func GetLanguage(code string) (Language, bool) {
	for _, lang := range SupportedLanguages {
		if lang.Code == code {
			return lang, true
		}
	}
	return Language{}, false
}

// IsValidLanguage checks if a language code is valid.
func IsValidLanguage(code string) bool {
	_, ok := GetLanguage(code)
	return ok
}
