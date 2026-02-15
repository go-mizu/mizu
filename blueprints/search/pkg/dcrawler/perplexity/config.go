// Package perplexity provides search scraping for Perplexity AI
// via SSE (anonymous), Socket.IO Labs (anonymous), and account registration.
package perplexity

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	baseURL    = "https://www.perplexity.ai"
	apiVersion = "2.18"

	// Scraping endpoints
	endpointAuthSession = baseURL + "/api/auth/session"
	endpointAuthSignin  = baseURL + "/api/auth/signin/email"
	endpointSSEAsk      = baseURL + "/rest/sse/perplexity_ask"
	endpointSocketIO    = baseURL + "/socket.io/"

	// Official API endpoints
	apiBaseURL             = "https://api.perplexity.ai"
	apiChatCompletions     = apiBaseURL + "/chat/completions"
	apiSearch              = apiBaseURL + "/search"

	// Timeouts
	defaultTimeout    = 30 * time.Second
	sseReadTimeout    = 120 * time.Second // SSE streams can be slow
	apiTimeout        = 60 * time.Second  // API can be slow for deep research
	accountTimeout    = 120 * time.Second
	emailRetryDelay   = 5 * time.Second
	rateLimitMinDelay = 1 * time.Second
	rateLimitMaxDelay = 3 * time.Second

	// Account limits
	defaultCopilotQueries = 5
	defaultFileUploads    = 10

	// SSE parsing
	sseEventPrefix  = "event: message\r\n"
	sseDataPrefix   = "event: message\r\ndata: "
	sseEndOfStream  = "event: end_of_stream\r\n"
	sseChunkDelim   = "\r\n\r\n"
	signinURLRegex  = `"(https://www\.perplexity\.ai/api/auth/callback/email\?callbackUrl=.*?)"`
	signinSubject   = "Sign in to Perplexity"

	// User agent
	chromeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"

	// Response body truncation for error logging
	maxErrorBodyLen = 4096
)

// Search modes
const (
	ModeAuto         = "auto"
	ModePro          = "pro"
	ModeReasoning    = "reasoning"
	ModeDeepResearch = "deep research"
)

// Search sources
const (
	SourceWeb     = "web"
	SourceScholar = "scholar"
	SourceSocial  = "social"
)

// Labs models
const (
	ModelR1          = "r1-1776"
	ModelSonarPro    = "sonar-pro"
	ModelSonar       = "sonar"
	ModelSonarReason = "sonar-reasoning-pro"
	ModelSonarReasonBase = "sonar-reasoning"
)

// LabsModels lists all available Labs models.
var LabsModels = []string{ModelR1, ModelSonarPro, ModelSonar, ModelSonarReason, ModelSonarReasonBase}

// Official API models
const (
	APISonar         = "sonar"
	APISonarPro      = "sonar-pro"
	APISonarReasoning = "sonar-reasoning-pro"
	APISonarDeep     = "sonar-deep-research"
)

// APIModels lists all available official API models.
var APIModels = []string{APISonar, APISonarPro, APISonarReasoning, APISonarDeep}

// modePayload maps user mode to the API mode field.
var modePayload = map[string]string{
	ModeAuto:         "concise",
	ModePro:          "copilot",
	ModeReasoning:    "copilot",
	ModeDeepResearch: "copilot",
}

// modelPreference maps (mode, model) to the API model_preference field.
var modelPreference = map[string]map[string]string{
	ModeAuto: {"": "turbo"},
	ModePro: {
		"":                  "pplx_pro",
		"sonar":             "experimental",
		"gpt-5.2":           "gpt52",
		"claude-4.5-sonnet": "claude45sonnet",
		"grok-4.1":          "grok41nonreasoning",
	},
	ModeReasoning: {
		"":                          "pplx_reasoning",
		"gpt-5.2-thinking":          "gpt52_thinking",
		"claude-4.5-sonnet-thinking": "claude45sonnetthinking",
		"gemini-3.0-pro":            "gemini30pro",
		"kimi-k2-thinking":          "kimik2thinking",
		"grok-4.1-reasoning":        "grok41reasoning",
	},
	ModeDeepResearch: {"": "pplx_alpha"},
}

// defaultHeaders returns the Chrome 128 headers needed to bypass Cloudflare.
func defaultHeaders() http.Header {
	return http.Header{
		"Accept":                      {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"Accept-Language":             {"en-US,en;q=0.9"},
		"Cache-Control":               {"max-age=0"},
		"Dnt":                         {"1"},
		"Priority":                    {"u=0, i"},
		"Sec-Ch-Ua":                   {`"Not;A=Brand";v="24", "Chromium";v="128"`},
		"Sec-Ch-Ua-Arch":              {`"x86"`},
		"Sec-Ch-Ua-Bitness":           {`"64"`},
		"Sec-Ch-Ua-Full-Version":      {`"128.0.6613.120"`},
		"Sec-Ch-Ua-Full-Version-List": {`"Not;A=Brand";v="24.0.0.0", "Chromium";v="128.0.6613.120"`},
		"Sec-Ch-Ua-Mobile":            {"?0"},
		"Sec-Ch-Ua-Model":             {`""`},
		"Sec-Ch-Ua-Platform":          {`"Windows"`},
		"Sec-Ch-Ua-Platform-Version":  {`"19.0.0"`},
		"Sec-Fetch-Dest":              {"document"},
		"Sec-Fetch-Mode":              {"navigate"},
		"Sec-Fetch-Site":              {"same-origin"},
		"Sec-Fetch-User":              {"?1"},
		"Upgrade-Insecure-Requests":   {"1"},
		"User-Agent":                  {chromeUA},
	}
}

// Config holds configuration for the perplexity package.
type Config struct {
	DataDir  string
	Timeout  time.Duration
	Language string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DataDir:  defaultDataDir(),
		Timeout:  defaultTimeout,
		Language: "en-US",
	}
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "perplexity")
}

// DBPath returns the DuckDB path.
func (c Config) DBPath() string {
	return filepath.Join(c.DataDir, "perplexity.duckdb")
}

// SessionPath returns the session file path.
func (c Config) SessionPath() string {
	return filepath.Join(c.DataDir, ".session.json")
}
