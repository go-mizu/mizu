package perplexity

import "time"

// SearchResult is the structured output from a Perplexity search.
type SearchResult struct {
	Query       string      `json:"query"`
	Answer      string      `json:"answer"`          // LLM markdown answer with [N] citations
	Citations   []Citation  `json:"citations"`        // Source URLs with metadata
	Chunks      []Chunk     `json:"chunks"`           // Answer segments with source refs
	WebResults  []WebResult `json:"web_results"`      // Raw web search results
	MediaItems  []MediaItem `json:"media_items"`      // Images/videos
	RelatedQ    []string    `json:"related_queries"`  // Related questions
	BackendUUID string      `json:"backend_uuid"`     // For follow-up queries
	Mode        string      `json:"mode"`
	Model       string      `json:"model"`
	Source      string      `json:"source"`           // "sse", "labs", or "api"
	SearchedAt  time.Time   `json:"searched_at"`
	AccountID   int         `json:"account_id,omitempty"`
	APIKeyID    int         `json:"api_key_id,omitempty"`
	TokensUsed  int         `json:"tokens_used,omitempty"`
	DurationMs  int64       `json:"duration_ms,omitempty"`
}

// Citation is a source referenced in the answer.
type Citation struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Date    string `json:"date,omitempty"`
	Domain  string `json:"domain,omitempty"`
}

// Chunk is a segment of the answer with source references.
type Chunk struct {
	Text          string `json:"text"`
	SourceIndices []int  `json:"source_indices"`
}

// WebResult is a raw web search result.
type WebResult struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Date    string `json:"date,omitempty"`
}

// MediaItem is an image or video in the answer.
type MediaItem struct {
	URL  string `json:"url"`
	Type string `json:"type"` // image, video
	Alt  string `json:"alt,omitempty"`
}

// LabsResult is the output from a Labs query.
type LabsResult struct {
	Output string `json:"output"` // Text answer
	Final  bool   `json:"final"`
	Model  string `json:"model"`
}

// SearchOptions configures a search query.
type SearchOptions struct {
	Mode        string   // auto, pro, reasoning, deep research
	Model       string   // model name (for pro/reasoning/labs)
	Sources     []string // web, scholar, social
	Language    string   // en-US, etc.
	Incognito   bool
	Stream      bool     // stream results to callback
	FollowUpUUID string  // backend_uuid for follow-up queries
}

// DefaultSearchOptions returns sensible defaults.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Mode:     ModeAuto,
		Sources:  []string{SourceWeb},
		Language: "en-US",
	}
}

// ssePayload is the JSON payload for the SSE search endpoint.
type ssePayload struct {
	QueryStr string    `json:"query_str"`
	Params   sseParams `json:"params"`
}

type sseParams struct {
	Attachments        []string `json:"attachments"`
	FrontendContextUUID string  `json:"frontend_context_uuid"`
	FrontendUUID       string   `json:"frontend_uuid"`
	IsIncognito        bool     `json:"is_incognito"`
	Language           string   `json:"language"`
	LastBackendUUID    *string  `json:"last_backend_uuid"`
	Mode               string   `json:"mode"`
	ModelPreference    string   `json:"model_preference"`
	Source             string   `json:"source"`
	Sources            []string `json:"sources"`
	Version            string   `json:"version"`
}

// sseStep represents a step in the SSE response text field.
type sseStep struct {
	StepType string         `json:"step_type"`
	Content  map[string]any `json:"content"`
}

// answerData is the nested JSON inside FINAL step's answer field.
type answerData struct {
	Answer string  `json:"answer"`
	Chunks []Chunk `json:"chunks"`
}

// socketIOHandshake is the Engine.IO v4 handshake response.
type socketIOHandshake struct {
	SID          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
}

// labsPayload is sent via Socket.IO for Labs queries.
type labsPayload struct {
	Messages []labsMessage `json:"messages"`
	Model    string        `json:"model"`
	Source   string        `json:"source"`
	Version  string        `json:"version"`
}

type labsMessage struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	Priority int    `json:"priority,omitempty"`
}

// emailnatorResp is the response from emailnator generate-email.
type emailnatorResp struct {
	Email []string `json:"email"`
}

// emailnatorMessageList is the response from emailnator message-list.
type emailnatorMessageList struct {
	MessageData []emailnatorMessage `json:"messageData"`
}

type emailnatorMessage struct {
	MessageID string `json:"messageID"`
	From      string `json:"from"`
	Subject   string `json:"subject"`
	Time      string `json:"time"`
}

// sessionData is persisted to disk for authenticated sessions.
type sessionData struct {
	Cookies        []*cookieData `json:"cookies"`
	CopilotQueries int          `json:"copilot_queries"`
	FileUploads    int          `json:"file_uploads"`
	CreatedAt      time.Time    `json:"created_at"`
}

type cookieData struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

// --- Account Management Types ---

// Account represents a registered Perplexity account (scraped access).
type Account struct {
	ID          int       `json:"id"`
	Email       string    `json:"email"`
	Source      string    `json:"source"`       // emailnator, manual
	SessionData string    `json:"session_data"` // JSON blob
	ProQueries  int       `json:"pro_queries"`
	FileUploads int       `json:"file_uploads"`
	Status      string    `json:"status"`       // active, exhausted, failed, banned
	ErrorMsg    string    `json:"error_msg,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	UseCount    int       `json:"use_count"`
}

// Account status constants.
const (
	AccountActive    = "active"
	AccountExhausted = "exhausted"
	AccountFailed    = "failed"
	AccountBanned    = "banned"
)

// APIKey represents a Perplexity API key.
type APIKey struct {
	ID          int       `json:"id"`
	Key         string    `json:"api_key"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // active, exhausted, invalid, rate_limited
	ErrorMsg    string    `json:"error_msg,omitempty"`
	Tier        string    `json:"tier"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	UseCount    int       `json:"use_count"`
	TotalTokens int       `json:"total_tokens"`
}

// API key status constants.
const (
	KeyActive      = "active"
	KeyExhausted   = "exhausted"
	KeyInvalid     = "invalid"
	KeyRateLimited = "rate_limited"
)

// ErrorLog represents a logged error.
type ErrorLog struct {
	ID           int       `json:"id"`
	AccountID    int       `json:"account_id,omitempty"`
	APIKeyID     int       `json:"api_key_id,omitempty"`
	Source       string    `json:"source"`    // sse, labs, api, register
	Operation    string    `json:"operation"` // search, labs_query, register, api_chat, api_search
	Query        string    `json:"query,omitempty"`
	ErrorType    string    `json:"error_type"` // http_error, parse_error, auth_error, rate_limit, timeout, cloudflare_block
	ErrorMsg     string    `json:"error_msg"`
	HTTPStatus   int       `json:"http_status,omitempty"`
	ResponseBody string    `json:"response_body,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Error type constants.
const (
	ErrHTTP       = "http_error"
	ErrParse      = "parse_error"
	ErrAuth       = "auth_error"
	ErrRateLimit  = "rate_limit"
	ErrTimeout    = "timeout"
	ErrCloudflare = "cloudflare_block"
)

// --- Official API Types ---

// ChatRequest is the request body for the Chat Completions API.
type ChatRequest struct {
	Model                string            `json:"model"`
	Messages             []ChatMessage     `json:"messages"`
	MaxTokens            int               `json:"max_tokens,omitempty"`
	Stream               bool              `json:"stream,omitempty"`
	SearchMode           string            `json:"search_mode,omitempty"`           // web, academic, sec
	SearchDomainFilter   []string          `json:"search_domain_filter,omitempty"`
	SearchRecencyFilter  string            `json:"search_recency_filter,omitempty"` // hour, day, week, month, year
	ReturnImages         bool              `json:"return_images,omitempty"`
	ReturnRelated        bool              `json:"return_related_questions,omitempty"`
	DisableSearch        bool              `json:"disable_search,omitempty"`
	WebSearchOptions     *WebSearchOptions `json:"web_search_options,omitempty"`
}

// ChatMessage is a message in the chat history.
type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// WebSearchOptions configures web search behavior.
type WebSearchOptions struct {
	SearchContextSize string `json:"search_context_size,omitempty"` // low, medium, high
}

// ChatResponse is the response from the Chat Completions API.
type ChatResponse struct {
	ID       string       `json:"id"`
	Model    string       `json:"model"`
	Created  int64        `json:"created"`
	Choices  []ChatChoice `json:"choices"`
	Citations []string    `json:"citations,omitempty"`
	SearchResults []APISearchResult `json:"search_results,omitempty"`
	Usage    *ChatUsage   `json:"usage,omitempty"`
}

// ChatChoice is a completion choice.
type ChatChoice struct {
	Index        int          `json:"index"`
	FinishReason string       `json:"finish_reason"`
	Message      *ChatMessage `json:"message,omitempty"`
	Delta        *ChatMessage `json:"delta,omitempty"`
}

// APISearchResult is a search result from the API.
type APISearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Date    string `json:"date,omitempty"`
}

// ChatUsage contains token usage info.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatStreamChunk is a single chunk in a streaming response.
type ChatStreamChunk struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
}

// SearchAPIRequest is the request body for the Search API.
type SearchAPIRequest struct {
	Query               string   `json:"query"`
	SearchDomainFilter  []string `json:"search_domain_filter,omitempty"`
	SearchRecencyFilter string   `json:"search_recency_filter,omitempty"`
	SearchMode          string   `json:"search_mode,omitempty"`
}

// SearchAPIResponse is the response from the Search API.
type SearchAPIResponse struct {
	ID      string            `json:"id"`
	Results []APISearchResult `json:"results"`
}

// --- Thread/Conversation Types ---

// Thread represents a multi-turn conversation.
type Thread struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Mode         string    `json:"mode"`
	Model        string    `json:"model"`
	Source       string    `json:"source"`
	AccountID    int       `json:"account_id,omitempty"`
	APIKeyID     int       `json:"api_key_id,omitempty"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ThreadMessage is a single message in a conversation thread.
type ThreadMessage struct {
	ID          int         `json:"id"`
	ThreadID    int         `json:"thread_id"`
	Role        string      `json:"role"` // "user" or "assistant"
	Content     string      `json:"content"`
	BackendUUID string      `json:"backend_uuid,omitempty"`
	Citations   []Citation  `json:"citations,omitempty"`
	WebResults  []WebResult `json:"web_results,omitempty"`
	RelatedQ    []string    `json:"related_queries,omitempty"`
	TokensUsed  int         `json:"tokens_used,omitempty"`
	DurationMs  int64       `json:"duration_ms,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}
