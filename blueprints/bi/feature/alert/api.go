// Package alert provides alert management and execution.
package alert

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound         = errors.New("alert not found")
	ErrInvalidCondition = errors.New("invalid alert condition")
	ErrInvalidSchedule  = errors.New("invalid schedule")
	ErrQuestionRequired = errors.New("question is required")
	ErrChannelRequired  = errors.New("at least one channel is required")
	ErrExecutionFailed  = errors.New("alert execution failed")
)

// Alert represents an alert configuration.
type Alert struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	QuestionID  string     `json:"question_id"`
	Condition   *Condition `json:"condition"`
	Channels    []Channel  `json:"channels"`
	Schedule    *Schedule  `json:"schedule,omitempty"`
	CreatorID   string     `json:"creator_id"`
	Active      bool       `json:"active"`
	LastChecked *time.Time `json:"last_checked,omitempty"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Condition defines when an alert should trigger.
type Condition struct {
	Type       string `json:"type"`       // above, below, reaches, changes, rows_present
	Column     string `json:"column,omitempty"`
	Value      any    `json:"value,omitempty"`
	Comparison string `json:"comparison,omitempty"` // gt, gte, lt, lte, eq
}

// Channel represents a notification channel.
type Channel struct {
	Type    string            `json:"type"`    // email, slack, webhook
	Config  map[string]string `json:"config"`  // type-specific configuration
	Enabled bool              `json:"enabled"`
}

// Schedule defines when to check the alert.
type Schedule struct {
	Type     string `json:"type"`      // hourly, daily, weekly
	Hour     int    `json:"hour,omitempty"`
	Minute   int    `json:"minute,omitempty"`
	DayOfWeek int   `json:"day_of_week,omitempty"` // 0=Sunday
	Timezone string `json:"timezone,omitempty"`
}

// AlertExecution represents a single alert check execution.
type AlertExecution struct {
	ID          string    `json:"id"`
	AlertID     string    `json:"alert_id"`
	Triggered   bool      `json:"triggered"`
	Result      any       `json:"result,omitempty"`
	Message     string    `json:"message,omitempty"`
	Error       string    `json:"error,omitempty"`
	ExecutedAt  time.Time `json:"executed_at"`
	Duration    float64   `json:"duration_ms"`
}

// Notification represents a notification sent.
type Notification struct {
	ID          string    `json:"id"`
	AlertID     string    `json:"alert_id"`
	ExecutionID string    `json:"execution_id"`
	Channel     string    `json:"channel"`
	Status      string    `json:"status"` // pending, sent, failed
	Error       string    `json:"error,omitempty"`
	SentAt      time.Time `json:"sent_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateIn contains input for creating an alert.
type CreateIn struct {
	Name       string     `json:"name"`
	QuestionID string     `json:"question_id"`
	Condition  *Condition `json:"condition"`
	Channels   []Channel  `json:"channels"`
	Schedule   *Schedule  `json:"schedule,omitempty"`
	CreatorID  string     `json:"-"`
}

// UpdateIn contains input for updating an alert.
type UpdateIn struct {
	ID        string     `json:"-"`
	Name      string     `json:"name,omitempty"`
	Condition *Condition `json:"condition,omitempty"`
	Channels  []Channel  `json:"channels,omitempty"`
	Schedule  *Schedule  `json:"schedule,omitempty"`
	Active    *bool      `json:"active,omitempty"`
}

// ListOpts specifies options for listing alerts.
type ListOpts struct {
	QuestionID string
	CreatorID  string
	Active     *bool
	Limit      int
	Offset     int
}

// API defines the Alert service contract.
type API interface {
	// Create creates a new alert.
	Create(ctx context.Context, in *CreateIn) (*Alert, error)

	// Get returns an alert by ID.
	Get(ctx context.Context, id string) (*Alert, error)

	// List returns alerts matching the options.
	List(ctx context.Context, opts ListOpts) ([]*Alert, error)

	// Update updates an alert.
	Update(ctx context.Context, in *UpdateIn) (*Alert, error)

	// Delete deletes an alert.
	Delete(ctx context.Context, id string) error

	// Execute manually triggers an alert check.
	Execute(ctx context.Context, id string) (*AlertExecution, error)

	// CheckPending checks all pending alerts that should run now.
	CheckPending(ctx context.Context) ([]*AlertExecution, error)

	// ListExecutions returns execution history for an alert.
	ListExecutions(ctx context.Context, alertID string, limit int) ([]*AlertExecution, error)

	// SendNotification sends a notification for an alert.
	SendNotification(ctx context.Context, alertID string, executionID string, channel Channel) error
}

// Store defines data access for alerts.
type Store interface {
	Create(ctx context.Context, alert *Alert) error
	GetByID(ctx context.Context, id string) (*Alert, error)
	List(ctx context.Context) ([]*Alert, error)
	ListByQuestion(ctx context.Context, questionID string) ([]*Alert, error)
	ListActive(ctx context.Context) ([]*Alert, error)
	Update(ctx context.Context, alert *Alert) error
	Delete(ctx context.Context, id string) error
	UpdateLastChecked(ctx context.Context, id string, t time.Time) error
	UpdateLastTriggered(ctx context.Context, id string, t time.Time) error
}

// ExecutionStore defines data access for alert executions.
type ExecutionStore interface {
	Create(ctx context.Context, exec *AlertExecution) error
	ListByAlert(ctx context.Context, alertID string, limit int) ([]*AlertExecution, error)
}

// NotificationStore defines data access for notifications.
type NotificationStore interface {
	Create(ctx context.Context, n *Notification) error
	Update(ctx context.Context, n *Notification) error
	ListByExecution(ctx context.Context, executionID string) ([]*Notification, error)
}

// QuestionExecutor executes a question and returns results.
type QuestionExecutor interface {
	Execute(ctx context.Context, questionID string) (any, error)
}

// Notifier sends notifications through various channels.
type Notifier interface {
	SendEmail(ctx context.Context, to []string, subject, body string) error
	SendSlack(ctx context.Context, webhookURL, message string) error
	SendWebhook(ctx context.Context, url string, payload any) error
}
