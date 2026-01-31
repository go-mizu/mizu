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
	ID          string    `json:"id"`
	AgentID     string    `json:"agentId"`
	ChannelID   string    `json:"channelId"`
	ChannelType string    `json:"channelType"`
	PeerID      string    `json:"peerId"`
	DisplayName string    `json:"displayName"`
	Origin      string    `json:"origin"` // dm, group, webhook, cron
	Status      string    `json:"status"` // active, expired, closed
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
	ChannelType ChannelType `json:"channelType"`
	ChannelID   string      `json:"channelId"`
	PeerID      string      `json:"peerId"`
	PeerName    string      `json:"peerName"`
	Content     string      `json:"content"`
	Origin      string      `json:"origin"` // dm, group
	GroupID     string      `json:"groupId,omitempty"`
	ReplyTo     string      `json:"replyTo,omitempty"`
	Metadata    string      `json:"metadata,omitempty"`
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
