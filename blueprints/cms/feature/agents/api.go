// Package agents provides the AI agent framework for the CMS.
package agents

import (
	"context"
	"time"
)

// AgentType identifies the type of agent.
type AgentType string

const (
	AgentTypeContent       AgentType = "content"
	AgentTypeSEO           AgentType = "seo"
	AgentTypeWorkflow      AgentType = "workflow"
	AgentTypeTranslation   AgentType = "translation"
	AgentTypeAnalytics     AgentType = "analytics"
	AgentTypeBulk          AgentType = "bulk"
	AgentTypePersonalize   AgentType = "personalize"
)

// Agent defines the interface for all AI agents.
type Agent interface {
	// Type returns the agent type.
	Type() AgentType

	// Name returns the agent name.
	Name() string

	// Description returns what this agent does.
	Description() string

	// Capabilities returns the list of actions this agent can perform.
	Capabilities() []Capability

	// Execute performs an action with the given input.
	Execute(ctx context.Context, action string, input *ActionInput) (*ActionOutput, error)

	// CanHandle returns true if this agent can handle the given action.
	CanHandle(action string) bool
}

// Capability describes an action an agent can perform.
type Capability struct {
	Action      string         `json:"action"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
	Examples    []Example      `json:"examples,omitempty"`
}

// Example shows how to use a capability.
type Example struct {
	Description string         `json:"description"`
	Input       map[string]any `json:"input"`
	Output      map[string]any `json:"output,omitempty"`
}

// ActionInput provides input for agent actions.
type ActionInput struct {
	Action     string         `json:"action"`
	Collection string         `json:"collection,omitempty"`
	DocumentID string         `json:"documentId,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
	Options    map[string]any `json:"options,omitempty"`
	Context    *AgentContext  `json:"context,omitempty"`
}

// ActionOutput holds the result of an agent action.
type ActionOutput struct {
	Success     bool              `json:"success"`
	Data        map[string]any    `json:"data,omitempty"`
	Messages    []Message         `json:"messages,omitempty"`
	Artifacts   []Artifact        `json:"artifacts,omitempty"`
	NextActions []SuggestedAction `json:"nextActions,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

// AgentContext provides context for agent execution.
type AgentContext struct {
	User         map[string]any        `json:"user,omitempty"`
	Workspace    string                `json:"workspace,omitempty"`
	Locale       string                `json:"locale,omitempty"`
	Conversation []ConversationMessage `json:"conversation,omitempty"`
	Memory       map[string]any        `json:"memory,omitempty"`
}

// Message is a human-readable message.
type Message struct {
	Role    string `json:"role"` // "assistant", "system", "error"
	Content string `json:"content"`
}

// Artifact is something produced by an agent.
type Artifact struct {
	Type string         `json:"type"` // "document", "file", "analysis", etc.
	ID   string         `json:"id"`
	Name string         `json:"name,omitempty"`
	Data map[string]any `json:"data,omitempty"`
}

// SuggestedAction is a follow-up action suggestion.
type SuggestedAction struct {
	Agent  AgentType      `json:"agent"`
	Action string         `json:"action"`
	Reason string         `json:"reason"`
	Input  map[string]any `json:"input,omitempty"`
}

// ConversationMessage is a message in the conversation history.
type ConversationMessage struct {
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// AgentInfo provides information about a registered agent.
type AgentInfo struct {
	Type         AgentType    `json:"type"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Capabilities []Capability `json:"capabilities"`
	Status       string       `json:"status"` // "active", "disabled", "error"
}

// Plan represents a multi-step execution plan.
type Plan struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Steps       []PlanStep `json:"steps"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// PlanStep is a single step in a plan.
type PlanStep struct {
	ID        string         `json:"id"`
	Agent     AgentType      `json:"agent"`
	Action    string         `json:"action"`
	Input     map[string]any `json:"input,omitempty"`
	DependsOn []string       `json:"dependsOn,omitempty"`
	Condition string         `json:"condition,omitempty"`
}

// PlanResult holds the result of plan execution.
type PlanResult struct {
	PlanID    string        `json:"planId"`
	Success   bool          `json:"success"`
	Steps     []StepResult  `json:"steps"`
	TotalTime time.Duration `json:"totalTime"`
}

// StepResult holds the result of a single step.
type StepResult struct {
	StepID   string        `json:"stepId"`
	Success  bool          `json:"success"`
	Output   *ActionOutput `json:"output,omitempty"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// TaskStatus represents the status of an agent task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// Task represents an agent task.
type Task struct {
	ID          string         `json:"id"`
	AgentType   AgentType      `json:"agentType"`
	Action      string         `json:"action"`
	Status      TaskStatus     `json:"status"`
	Input       map[string]any `json:"input,omitempty"`
	Output      map[string]any `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	StartedAt   *time.Time     `json:"startedAt,omitempty"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}

// OrchestratorAPI defines the orchestrator interface.
type OrchestratorAPI interface {
	// Route determines which agent(s) should handle a request.
	Route(ctx context.Context, input string) ([]AgentType, error)

	// Execute runs an action with the appropriate agent.
	Execute(ctx context.Context, agentType AgentType, input *ActionInput) (*ActionOutput, error)

	// ExecuteNL processes a natural language command.
	ExecuteNL(ctx context.Context, command string, agentCtx *AgentContext) (*ActionOutput, error)

	// ExecutePlan executes a multi-step plan.
	ExecutePlan(ctx context.Context, plan *Plan) (*PlanResult, error)

	// GetAgent returns an agent by type.
	GetAgent(agentType AgentType) (Agent, bool)

	// ListAgents returns all registered agents.
	ListAgents() []AgentInfo
}

// RegistryAPI defines the registry interface.
type RegistryAPI interface {
	// Register adds an agent to the registry.
	Register(agent Agent) error

	// Unregister removes an agent from the registry.
	Unregister(agentType AgentType) error

	// Get retrieves an agent by type.
	Get(agentType AgentType) (Agent, bool)

	// List returns all registered agents.
	List() []AgentInfo

	// FindByCapability finds agents that can handle an action.
	FindByCapability(action string) []Agent
}

// Store defines the store interface for agent data.
type Store interface {
	// CreateTask creates a new task.
	CreateTask(ctx context.Context, task *Task) error

	// UpdateTask updates a task.
	UpdateTask(ctx context.Context, task *Task) error

	// GetTask retrieves a task by ID.
	GetTask(ctx context.Context, id string) (*Task, error)

	// ListTasks lists tasks with optional filters.
	ListTasks(ctx context.Context, filters map[string]any, limit, offset int) ([]*Task, int, error)

	// SaveMemory saves agent memory.
	SaveMemory(ctx context.Context, agentType AgentType, userID, key string, value any, expiresAt *time.Time) error

	// GetMemory retrieves agent memory.
	GetMemory(ctx context.Context, agentType AgentType, userID, key string) (any, error)

	// DeleteMemory deletes agent memory.
	DeleteMemory(ctx context.Context, agentType AgentType, userID, key string) error
}
