package duome

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// AudioProvider defines the TTS provider to use
type AudioProvider string

const (
	// GoogleTTS uses Google Translate TTS (free, works for most languages)
	GoogleTTS AudioProvider = "google"
	// ForvoTTS uses Forvo pronunciation database (requires API key)
	ForvoTTS AudioProvider = "forvo"
	// LocalTTS uses locally generated audio files
	LocalTTS AudioProvider = "local"
)

// DefaultAudioProvider is the default TTS provider
var DefaultAudioProvider = GoogleTTS

// GenerateAudioURL creates a TTS audio URL for a word
func GenerateAudioURL(word, languageCode string) string {
	return GenerateAudioURLWithProvider(word, languageCode, DefaultAudioProvider)
}

// GenerateAudioURLWithProvider creates a TTS audio URL using the specified provider
func GenerateAudioURLWithProvider(word, languageCode string, provider AudioProvider) string {
	switch provider {
	case GoogleTTS:
		return generateGoogleTTSURL(word, languageCode)
	case ForvoTTS:
		return generateForvoURL(word, languageCode)
	case LocalTTS:
		return generateLocalAudioPath(word, languageCode)
	default:
		return generateGoogleTTSURL(word, languageCode)
	}
}

// generateGoogleTTSURL creates a Google Translate TTS URL
// This is free and works for most languages Duolingo supports
func generateGoogleTTSURL(word, languageCode string) string {
	// Map Duolingo language codes to Google TTS language codes
	langMap := map[string]string{
		"zs": "zh-CN", // Chinese Simplified
		"zh": "zh-CN", // Chinese
		"zc": "zh-HK", // Cantonese
		"nb": "no",    // Norwegian Bokmål
		"hv": "",      // High Valyrian (not supported)
		"kl": "",      // Klingon (not supported)
		"gn": "",      // Guarani (not supported)
		"nv": "",      // Navajo (not supported)
		"hw": "",      // Hawaiian (limited support)
		"ga": "ga",    // Irish
		"gd": "gd",    // Scottish Gaelic
		"cy": "cy",    // Welsh
		"yi": "yi",    // Yiddish
	}

	// Use mapped code or original
	ttsLang := languageCode
	if mapped, ok := langMap[languageCode]; ok {
		if mapped == "" {
			// Language not supported by Google TTS
			return ""
		}
		ttsLang = mapped
	}

	// URL encode the word
	encodedWord := url.QueryEscape(word)

	// Google Translate TTS URL pattern
	// Using multiple TK values for rate limiting avoidance
	return fmt.Sprintf(
		"https://translate.google.com/translate_tts?ie=UTF-8&q=%s&tl=%s&client=tw-ob",
		encodedWord,
		ttsLang,
	)
}

// generateForvoURL creates a Forvo pronunciation URL placeholder
// Forvo requires API key for actual audio URLs
func generateForvoURL(word, languageCode string) string {
	encodedWord := url.QueryEscape(word)
	return fmt.Sprintf("/api/audio/forvo/%s/%s", languageCode, encodedWord)
}

// generateLocalAudioPath generates a path for locally stored audio
func generateLocalAudioPath(word, languageCode string) string {
	// Create a hash of the word for file naming
	hash := md5.Sum([]byte(word))
	hashStr := hex.EncodeToString(hash[:8])

	// Sanitize word for filename (basic)
	safeWord := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, word)

	if len(safeWord) > 20 {
		safeWord = safeWord[:20]
	}

	return fmt.Sprintf("/audio/%s/%s_%s.mp3", languageCode, safeWord, hashStr)
}

// GenerateAudioURLsForVocabulary adds audio URLs to vocabulary entries
func GenerateAudioURLsForVocabulary(vocab []VocabularyEntry, languageCode string) []VocabularyEntry {
	result := make([]VocabularyEntry, len(vocab))
	for i, v := range vocab {
		result[i] = v
		if result[i].AudioURL == "" {
			result[i].AudioURL = GenerateAudioURL(v.Word, languageCode)
		}
	}
	return result
}

// LanguageHasAudio returns whether a language has TTS support
func LanguageHasAudio(languageCode string) bool {
	// Languages without Google TTS support
	unsupportedLanguages := map[string]bool{
		"hv": true, // High Valyrian
		"kl": true, // Klingon
		"gn": true, // Guarani
		"nv": true, // Navajo
	}
	return !unsupportedLanguages[languageCode]
}

// SlowAudioURL returns a slower version of the audio URL (for learning)
func SlowAudioURL(audioURL string) string {
	if strings.Contains(audioURL, "translate.google.com") {
		// Google TTS supports slow mode with ttsspeed parameter
		if strings.Contains(audioURL, "?") {
			return audioURL + "&ttsspeed=0.3"
		}
		return audioURL + "?ttsspeed=0.3"
	}
	return audioURL
}
