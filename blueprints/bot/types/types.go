package types

import "time"

// Agent represents an AI agent with its own workspace and configuration.
type Agent struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"systemPrompt"`
	Workspace    string    `json:"workspace"`
	MaxTokens    int       `json:"maxTokens"`
	Temperature  float64   `json:"temperature"`
	Status       string    `json:"status"` // active, paused, disabled
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ChannelType identifies a messaging platform.
type ChannelType string

const (
	ChannelTelegram   ChannelType = "telegram"
	ChannelDiscord    ChannelType = "discord"
	ChannelMattermost ChannelType = "mattermost"
	ChannelWebhook    ChannelType = "webhook"
)

// Channel represents a connected messaging platform.
type Channel struct {
	ID        string      `json:"id"`
	Type      ChannelType `json:"type"`
	Name      string      `json:"name"`
	Config    string      `json:"config"` // JSON config (tokens, webhookURL, etc.)
	Status    string      `json:"status"` // connected, disconnected, error
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

// DMScope controls how direct-message sessions are keyed.
type DMScope string

const (
	DMScopeMain               DMScope = "main"                 // unified across channels
	DMScopePerPeer            DMScope = "per-peer"             // per sender
	DMScopePerChannelPeer     DMScope = "per-channel-peer"     // per channel + sender
	DMScopePerAccountChanPeer DMScope = "per-account-chan-peer" // per account + channel + sender
)

// Session represents a conversation session between a peer and an agent.
type Session struct {
	ID              string    `json:"id"`
	AgentID         string    `json:"agentId"`
	ChannelID       string    `json:"channelId"`
	ChannelType     string    `json:"channelType"`
	PeerID          string    `json:"peerId"`
	DisplayName     string    `json:"displayName"`
	Origin          string    `json:"origin"` // dm, group, webhook, cron
	Status          string    `json:"status"` // active, expired, closed
	Metadata        string    `json:"metadata"`
	Model           string    `json:"model,omitempty"`
	CompactionCount int       `json:"compactionCount,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Message role constants.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// Message represents a single message within a session.
type Message struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	AgentID   string    `json:"agentId"`
	ChannelID string    `json:"channelId"`
	PeerID    string    `json:"peerId"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"createdAt"`
}

// Binding routes inbound messages to an agent based on channel/peer matching.
type Binding struct {
	ID          string `json:"id"`
	AgentID     string `json:"agentId"`
	ChannelType string `json:"channelType"` // telegram, discord, etc. or "*" for all
	ChannelID   string `json:"channelId"`   // specific channel ID or "*" for all
	PeerID      string `json:"peerId"`      // specific peer or "*" for all
	Priority    int    `json:"priority"`    // higher = more specific, wins ties
}

// Command represents an in-chat slash command.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
}

// GatewayStatus holds the gateway runtime status.
type GatewayStatus struct {
	Status       string   `json:"status"` // running, stopped
	Port         int      `json:"port"`
	Uptime       string   `json:"uptime"`
	ActiveAgents int      `json:"activeAgents"`
	Channels     []string `json:"channels"`
	Sessions     int      `json:"sessions"`
	Messages     int      `json:"messages"`
}

// InboundMessage is a message received from a channel driver.
type InboundMessage struct {
	ChannelType  ChannelType `json:"channelType"`
	ChannelID    string      `json:"channelId"`
	PeerID       string      `json:"peerId"`
	PeerName     string      `json:"peerName"`
	Content      string      `json:"content"`
	Origin       string      `json:"origin"` // dm, group
	GroupID      string      `json:"groupId,omitempty"`
	ReplyTo      string      `json:"replyTo,omitempty"`
	Metadata     string      `json:"metadata,omitempty"`
	SkillContext string      `json:"-"` // injected skill content for skill commands (internal)
	SkillName    string      `json:"-"` // matched skill name (internal)
}

// OutboundMessage is a message to send via a channel driver.
type OutboundMessage struct {
	ChannelType ChannelType `json:"channelType"`
	ChannelID   string      `json:"channelId"`
	PeerID      string      `json:"peerId"`
	Content     string      `json:"content"`
	ReplyTo     string      `json:"replyTo,omitempty"`
	Metadata    string      `json:"metadata,omitempty"`
}

// SessionConfig holds session-related configuration.
type SessionConfig struct {
	DMScope     DMScope `json:"dmScope"`
	ResetMode   string  `json:"resetMode"`   // daily, idle, manual
	ResetHour   int     `json:"resetHour"`   // hour of day for daily reset
	IdleMinutes int     `json:"idleMinutes"` // inactivity timeout
}

// TelegramConfig holds Telegram-specific channel settings.
type TelegramConfig struct {
	BotToken     string `json:"botToken"`
	DMPolicy     string `json:"dmPolicy"` // pairing, allowlist, open
	WebhookURL   string `json:"webhookUrl,omitempty"`
	HistoryLimit int    `json:"historyLimit"`
}

// DiscordConfig holds Discord-specific channel settings.
type DiscordConfig struct {
	BotToken string `json:"botToken"`
	DMPolicy string `json:"dmPolicy"`
}

// MattermostConfig holds Mattermost-specific channel settings.
type MattermostConfig struct {
	ServerURL string `json:"serverUrl"`
	BotToken  string `json:"botToken"`
	TeamID    string `json:"teamId"`
}

// WebhookConfig holds webhook channel settings.
type WebhookConfig struct {
	Secret   string `json:"secret"`
	Endpoint string `json:"endpoint"`
}

// CronJob represents a scheduled job.
type CronJob struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	AgentID       string    `json:"agentId"`
	Enabled       bool      `json:"enabled"`
	Schedule      string    `json:"schedule"`      // JSON: {kind, interval, unit, at, cron, tz}
	SessionTarget string    `json:"sessionTarget"` // main, isolated
	WakeMode      string    `json:"wakeMode"`      // next-heartbeat, now
	Payload       string    `json:"payload"`        // JSON: {kind, text, message, ...}
	LastRunAt     time.Time `json:"lastRunAt"`
	LastStatus    string    `json:"lastStatus"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// CronRun represents a single execution of a cron job.
type CronRun struct {
	ID         string    `json:"id"`
	JobID      string    `json:"jobId"`
	Status     string    `json:"status"` // running, success, failed
	StartedAt  time.Time `json:"startedAt"`
	EndedAt    time.Time `json:"endedAt"`
	DurationMs int64     `json:"durationMs"`
	Summary    string    `json:"summary"`
	Error      string    `json:"error"`
}

// CronStatus holds scheduler status.
type CronStatus struct {
	Enabled      bool  `json:"enabled"`
	Jobs         int   `json:"jobs"`
	NextWakeAtMs int64 `json:"nextWakeAtMs,omitempty"`
}

// LogEntry represents a structured log entry.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Subsystem string `json:"subsystem"`
	Message   string `json:"message"`
	Raw       string `json:"raw,omitempty"`
}

// Instance represents a connected dashboard client.
type Instance struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	RemoteAddr  string `json:"remoteAddr"`
	ConnectedAt int64  `json:"connectedAt"`
	LastPingAt  int64  `json:"lastPingAt"`
	Role        string `json:"role"`
	UserAgent   string `json:"userAgent"`
}

// SkillEntry represents a skill's full status for the dashboard.
// Matches OpenClaw's SkillStatus schema for protocol compatibility.
type SkillEntry struct {
	Key            string            `json:"key"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Emoji          string            `json:"emoji"`
	Source         string            `json:"source"`
	FilePath       string            `json:"filePath,omitempty"`
	BaseDir        string            `json:"baseDir,omitempty"`
	SkillKey       string            `json:"skillKey"`
	PrimaryEnv     string            `json:"primaryEnv,omitempty"`
	Homepage       string            `json:"homepage,omitempty"`
	Always         bool              `json:"always"`
	Disabled       bool              `json:"disabled"`
	BlockedByAllow bool              `json:"blockedByAllowlist"`
	Eligible       bool              `json:"eligible"`
	Enabled        bool              `json:"enabled"`
	UserInvocable  bool              `json:"userInvocable"`
	Requirements   SkillRequirements `json:"requirements"`
	Missing        SkillMissing      `json:"missing"`
	Install        []SkillInstallOpt `json:"install"`
}

// SkillRequirements lists what a skill needs to be eligible.
type SkillRequirements struct {
	Bins    []string `json:"bins"`
	AnyBins []string `json:"anyBins"`
	Env     []string `json:"env"`
	Config  []string `json:"config"`
	OS      []string `json:"os"`
}

// SkillMissing lists unsatisfied requirements.
type SkillMissing struct {
	Bins    []string `json:"bins"`
	AnyBins []string `json:"anyBins"`
	Env     []string `json:"env"`
	Config  []string `json:"config"`
	OS      []string `json:"os"`
}

// SkillInstallOpt represents an install option for a missing dependency.
type SkillInstallOpt struct {
	ID    string   `json:"id"`
	Kind  string   `json:"kind"`
	Label string   `json:"label"`
	Bins  []string `json:"bins"`
}

// SkillsSnapshot stores the skills state at session creation time.
type SkillsSnapshot struct {
	Prompt  string               `json:"prompt"`
	Skills  []SkillsSnapshotItem `json:"skills"`
	Version int                  `json:"version"`
}

// SkillsSnapshotItem records one skill in a snapshot.
type SkillsSnapshotItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Ready       bool   `json:"ready"`
	Source      string `json:"source"`
}

// SystemPromptReport captures diagnostic metadata about the built prompt.
type SystemPromptReport struct {
	Source            string              `json:"source"`
	GeneratedAt       int64               `json:"generatedAt"`
	SessionID         string              `json:"sessionId"`
	SessionKey        string              `json:"sessionKey"`
	Provider          string              `json:"provider"`
	Model             string              `json:"model"`
	WorkspaceDir      string              `json:"workspaceDir"`
	BootstrapMaxChars int                 `json:"bootstrapMaxChars"`
	SystemPrompt      SystemPromptStats   `json:"systemPrompt"`
	InjectedFiles     []InjectedFileStats `json:"injectedWorkspaceFiles"`
	Skills            SkillsReportStats   `json:"skills"`
	Tools             ToolsReportStats    `json:"tools"`
}

// SystemPromptStats holds character counts for the system prompt.
type SystemPromptStats struct {
	Chars               int `json:"chars"`
	ProjectContextChars int `json:"projectContextChars"`
	NonProjectChars     int `json:"nonProjectContextChars"`
}

// InjectedFileStats records a single injected workspace file.
type InjectedFileStats struct {
	Path  string `json:"path"`
	Chars int    `json:"chars"`
}

// SkillsReportStats holds skills metadata for the prompt report.
type SkillsReportStats struct {
	Count int      `json:"count"`
	Chars int      `json:"chars"`
	Names []string `json:"names"`
}

// ToolsReportStats holds tools metadata for the prompt report.
type ToolsReportStats struct {
	Count int      `json:"count"`
	Names []string `json:"names"`
}

// HealthSnapshot is a health check result.
type HealthSnapshot struct {
	Status    string         `json:"status"`
	Uptime    string         `json:"uptime"`
	Database  string         `json:"database"`
	Memory    map[string]any `json:"memory"`
	Stats     *GatewayStatus `json:"stats"`
	Timestamp string         `json:"timestamp"`
}

// LLMRequest is a request to the LLM provider.
type LLMRequest struct {
	Model        string    `json:"model"`
	SystemPrompt string    `json:"systemPrompt"`
	Messages     []LLMMsg  `json:"messages"`
	MaxTokens    int       `json:"maxTokens"`
	Temperature  float64   `json:"temperature"`
}

// LLMMsg is a single message in an LLM conversation.
type LLMMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse is a response from the LLM provider.
type LLMResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

// ToolDefinition is the Anthropic API tool definition format.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// ContentBlock is a content block in an Anthropic API response.
type ContentBlock struct {
	Type  string         `json:"type"`            // "text" or "tool_use"
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`    // tool_use block ID
	Name  string         `json:"name,omitempty"`  // tool name
	Input map[string]any `json:"input,omitempty"` // tool input parameters
}

// ToolResultBlock is a tool result sent back to the API as user message content.
type ToolResultBlock struct {
	Type      string `json:"type"`        // always "tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// LLMToolRequest extends LLMRequest with tool definitions.
type LLMToolRequest struct {
	Model        string           `json:"model"`
	SystemPrompt string           `json:"systemPrompt"`
	Messages     []any            `json:"messages"`    // mix of LLMMsg and tool result messages
	MaxTokens    int              `json:"maxTokens"`
	Temperature  float64          `json:"temperature"`
	Tools        []ToolDefinition `json:"tools,omitempty"`
}

// LLMToolResponse contains the full Anthropic response with content blocks.
type LLMToolResponse struct {
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stopReason"` // "end_turn" or "tool_use"
	InputTokens  int            `json:"inputTokens"`
	OutputTokens int            `json:"outputTokens"`
}

// TextContent extracts all text from the content blocks.
func (r *LLMToolResponse) TextContent() string {
	var text string
	for _, block := range r.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return text
}

// ToolUses extracts all tool_use blocks from the response.
func (r *LLMToolResponse) ToolUses() []ContentBlock {
	var uses []ContentBlock
	for _, block := range r.Content {
		if block.Type == "tool_use" {
			uses = append(uses, block)
		}
	}
	return uses
}
