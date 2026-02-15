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
	Source      string      `json:"source"`           // "sse" or "labs"
	SearchedAt  time.Time   `json:"searched_at"`
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
