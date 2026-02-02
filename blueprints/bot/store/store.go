package store

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Store is the top-level storage interface for the bot gateway.
type Store interface {
	AgentStore
	ChannelStore
	SessionStore
	MessageStore
	BindingStore
	CronStore
	ConfigStore

	// Ensure creates all tables and runs migrations.
	Ensure(ctx context.Context) error

	// SeedData populates the database with sample data.
	SeedData(ctx context.Context) error

	// Close releases database resources.
	Close() error

	// Stats returns aggregate counts for the dashboard.
	Stats(ctx context.Context) (*Stats, error)
}

// Stats holds aggregate counts.
type Stats struct {
	Agents   int `json:"agents"`
	Channels int `json:"channels"`
	Sessions int `json:"sessions"`
	Messages int `json:"messages"`
	Bindings int `json:"bindings"`
}

// AgentStore manages AI agents.
type AgentStore interface {
	ListAgents(ctx context.Context) ([]types.Agent, error)
	GetAgent(ctx context.Context, id string) (*types.Agent, error)
	CreateAgent(ctx context.Context, a *types.Agent) error
	UpdateAgent(ctx context.Context, a *types.Agent) error
	DeleteAgent(ctx context.Context, id string) error
}

// ChannelStore manages messaging channels.
type ChannelStore interface {
	ListChannels(ctx context.Context) ([]types.Channel, error)
	GetChannel(ctx context.Context, id string) (*types.Channel, error)
	CreateChannel(ctx context.Context, c *types.Channel) error
	UpdateChannel(ctx context.Context, c *types.Channel) error
	DeleteChannel(ctx context.Context, id string) error
}

// SessionStore manages conversation sessions.
type SessionStore interface {
	ListSessions(ctx context.Context) ([]types.Session, error)
	GetSession(ctx context.Context, id string) (*types.Session, error)
	GetOrCreateSession(ctx context.Context, agentID, channelID, channelType, peerID, displayName, origin string) (*types.Session, error)
	UpdateSession(ctx context.Context, s *types.Session) error
	DeleteSession(ctx context.Context, id string) error
	PatchSession(ctx context.Context, id string, updates map[string]any) error
	CreateSession(ctx context.Context, s *types.Session) error
	ExpireSessions(ctx context.Context, mode string, idleMinutes int) (int, error)
}

// MessageStore manages messages.
type MessageStore interface {
	ListMessages(ctx context.Context, sessionID string, limit int) ([]types.Message, error)
	CreateMessage(ctx context.Context, m *types.Message) error
	CountMessages(ctx context.Context) (int, error)
}

// BindingStore manages agent-channel bindings for message routing.
type BindingStore interface {
	ListBindings(ctx context.Context) ([]types.Binding, error)
	CreateBinding(ctx context.Context, b *types.Binding) error
	DeleteBinding(ctx context.Context, id string) error

	// ResolveAgent finds the best-matching agent for an inbound message
	// using most-specific-wins priority: peer > channel > channelType > default.
	ResolveAgent(ctx context.Context, channelType, channelID, peerID string) (*types.Agent, error)
}

// CronStore manages cron jobs and their run history.
type CronStore interface {
	ListCronJobs(ctx context.Context) ([]types.CronJob, error)
	GetCronJob(ctx context.Context, id string) (*types.CronJob, error)
	CreateCronJob(ctx context.Context, job *types.CronJob) error
	UpdateCronJob(ctx context.Context, job *types.CronJob) error
	DeleteCronJob(ctx context.Context, id string) error
	CreateCronRun(ctx context.Context, run *types.CronRun) error
	UpdateCronRun(ctx context.Context, run *types.CronRun) error
	ListCronRuns(ctx context.Context, jobID string, limit int) ([]types.CronRun, error)
}

// ConfigStore manages key-value configuration.
type ConfigStore interface {
	GetConfigVal(ctx context.Context, key string) (string, error)
	SetConfigVal(ctx context.Context, key, value string) error
	DeleteConfigVal(ctx context.Context, key string) error
	ListConfigVals(ctx context.Context) (map[string]string, error)
}
