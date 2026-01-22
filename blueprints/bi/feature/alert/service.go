package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Alert API.
type Service struct {
	alerts       Store
	executions   ExecutionStore
	notifications NotificationStore
	questions    QuestionExecutor
	notifier     Notifier
}

// NewService creates a new Alert service.
func NewService(
	alerts Store,
	executions ExecutionStore,
	notifications NotificationStore,
	questions QuestionExecutor,
	notifier Notifier,
) *Service {
	return &Service{
		alerts:       alerts,
		executions:   executions,
		notifications: notifications,
		questions:    questions,
		notifier:     notifier,
	}
}

// Create creates a new alert.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Alert, error) {
	if in.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if in.QuestionID == "" {
		return nil, ErrQuestionRequired
	}
	if in.Condition == nil {
		return nil, ErrInvalidCondition
	}
	if len(in.Channels) == 0 {
		return nil, ErrChannelRequired
	}

	// Validate condition
	if err := validateCondition(in.Condition); err != nil {
		return nil, err
	}

	// Validate channels
	for _, ch := range in.Channels {
		if err := validateChannel(ch); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	alert := &Alert{
		ID:         ulid.Make().String(),
		Name:       in.Name,
		QuestionID: in.QuestionID,
		Condition:  in.Condition,
		Channels:   in.Channels,
		Schedule:   in.Schedule,
		CreatorID:  in.CreatorID,
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.alerts.Create(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// Get returns an alert by ID.
func (s *Service) Get(ctx context.Context, id string) (*Alert, error) {
	alert, err := s.alerts.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return alert, nil
}

// List returns alerts matching the options.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Alert, error) {
	if opts.QuestionID != "" {
		return s.alerts.ListByQuestion(ctx, opts.QuestionID)
	}
	return s.alerts.List(ctx)
}

// Update updates an alert.
func (s *Service) Update(ctx context.Context, in *UpdateIn) (*Alert, error) {
	alert, err := s.alerts.GetByID(ctx, in.ID)
	if err != nil {
		return nil, ErrNotFound
	}

	if in.Name != "" {
		alert.Name = in.Name
	}
	if in.Condition != nil {
		if err := validateCondition(in.Condition); err != nil {
			return nil, err
		}
		alert.Condition = in.Condition
	}
	if in.Channels != nil {
		for _, ch := range in.Channels {
			if err := validateChannel(ch); err != nil {
				return nil, err
			}
		}
		alert.Channels = in.Channels
	}
	if in.Schedule != nil {
		alert.Schedule = in.Schedule
	}
	if in.Active != nil {
		alert.Active = *in.Active
	}

	alert.UpdatedAt = time.Now()

	if err := s.alerts.Update(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// Delete deletes an alert.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.alerts.Delete(ctx, id)
}

// Execute manually triggers an alert check.
func (s *Service) Execute(ctx context.Context, id string) (*AlertExecution, error) {
	alert, err := s.alerts.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	return s.executeAlert(ctx, alert)
}

// CheckPending checks all pending alerts that should run now.
func (s *Service) CheckPending(ctx context.Context) ([]*AlertExecution, error) {
	alerts, err := s.alerts.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	var executions []*AlertExecution
	now := time.Now()

	for _, alert := range alerts {
		if !shouldRunAlert(alert, now) {
			continue
		}

		exec, err := s.executeAlert(ctx, alert)
		if err != nil {
			// Log error but continue with other alerts
			exec = &AlertExecution{
				ID:         ulid.Make().String(),
				AlertID:    alert.ID,
				Triggered:  false,
				Error:      err.Error(),
				ExecutedAt: now,
			}
			if s.executions != nil {
				s.executions.Create(ctx, exec)
			}
		}

		executions = append(executions, exec)
	}

	return executions, nil
}

// ListExecutions returns execution history for an alert.
func (s *Service) ListExecutions(ctx context.Context, alertID string, limit int) ([]*AlertExecution, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.executions.ListByAlert(ctx, alertID, limit)
}

// SendNotification sends a notification for an alert.
func (s *Service) SendNotification(ctx context.Context, alertID string, executionID string, channel Channel) error {
	alert, err := s.alerts.GetByID(ctx, alertID)
	if err != nil {
		return ErrNotFound
	}

	// Create notification record
	notification := &Notification{
		ID:          ulid.Make().String(),
		AlertID:     alertID,
		ExecutionID: executionID,
		Channel:     channel.Type,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if s.notifications != nil {
		s.notifications.Create(ctx, notification)
	}

	// Send based on channel type
	var sendErr error
	switch channel.Type {
	case "email":
		recipients := []string{channel.Config["to"]}
		subject := fmt.Sprintf("Alert: %s", alert.Name)
		body := buildEmailBody(alert, channel.Config)
		sendErr = s.notifier.SendEmail(ctx, recipients, subject, body)

	case "slack":
		message := buildSlackMessage(alert)
		sendErr = s.notifier.SendSlack(ctx, channel.Config["webhook_url"], message)

	case "webhook":
		payload := buildWebhookPayload(alert)
		sendErr = s.notifier.SendWebhook(ctx, channel.Config["url"], payload)

	default:
		sendErr = fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	// Update notification status
	if sendErr != nil {
		notification.Status = "failed"
		notification.Error = sendErr.Error()
	} else {
		notification.Status = "sent"
		notification.SentAt = time.Now()
	}

	if s.notifications != nil {
		s.notifications.Update(ctx, notification)
	}

	return sendErr
}

// executeAlert runs a single alert check.
func (s *Service) executeAlert(ctx context.Context, alert *Alert) (*AlertExecution, error) {
	start := time.Now()

	exec := &AlertExecution{
		ID:         ulid.Make().String(),
		AlertID:    alert.ID,
		ExecutedAt: start,
	}

	// Execute the question
	result, err := s.questions.Execute(ctx, alert.QuestionID)
	if err != nil {
		exec.Error = err.Error()
		exec.Duration = float64(time.Since(start).Milliseconds())
		if s.executions != nil {
			s.executions.Create(ctx, exec)
		}
		return exec, err
	}

	exec.Result = result

	// Check condition
	triggered, message := checkCondition(alert.Condition, result)
	exec.Triggered = triggered
	exec.Message = message
	exec.Duration = float64(time.Since(start).Milliseconds())

	// Save execution
	if s.executions != nil {
		s.executions.Create(ctx, exec)
	}

	// Update alert timestamps
	s.alerts.UpdateLastChecked(ctx, alert.ID, start)
	if triggered {
		s.alerts.UpdateLastTriggered(ctx, alert.ID, start)

		// Send notifications
		for _, channel := range alert.Channels {
			if channel.Enabled {
				go s.SendNotification(ctx, alert.ID, exec.ID, channel)
			}
		}
	}

	return exec, nil
}

// checkCondition evaluates the alert condition against the result.
func checkCondition(condition *Condition, result any) (bool, string) {
	if condition == nil {
		return false, "no condition specified"
	}

	// Try to extract the value from result
	var value any
	switch r := result.(type) {
	case map[string]any:
		if rows, ok := r["rows"].([]map[string]any); ok && len(rows) > 0 {
			if condition.Column != "" {
				value = rows[0][condition.Column]
			} else {
				// Get first column value
				for _, v := range rows[0] {
					value = v
					break
				}
			}
		}
	case []map[string]any:
		if len(r) > 0 {
			if condition.Column != "" {
				value = r[0][condition.Column]
			} else {
				for _, v := range r[0] {
					value = v
					break
				}
			}
		}
	}

	// Convert to float for numeric comparisons
	numValue, numOk := toFloat64(value)
	threshold, thresholdOk := toFloat64(condition.Value)

	switch condition.Type {
	case "rows_present":
		if rows, ok := result.(map[string]any); ok {
			if r, ok := rows["rows"].([]map[string]any); ok {
				if len(r) > 0 {
					return true, fmt.Sprintf("Query returned %d rows", len(r))
				}
			}
		}
		return false, "No rows returned"

	case "above":
		if numOk && thresholdOk && numValue > threshold {
			return true, fmt.Sprintf("Value %.2f is above threshold %.2f", numValue, threshold)
		}
		return false, fmt.Sprintf("Value %.2f is not above threshold %.2f", numValue, threshold)

	case "below":
		if numOk && thresholdOk && numValue < threshold {
			return true, fmt.Sprintf("Value %.2f is below threshold %.2f", numValue, threshold)
		}
		return false, fmt.Sprintf("Value %.2f is not below threshold %.2f", numValue, threshold)

	case "reaches":
		if numOk && thresholdOk && numValue == threshold {
			return true, fmt.Sprintf("Value reached %.2f", threshold)
		}
		return false, fmt.Sprintf("Value %.2f has not reached %.2f", numValue, threshold)

	default:
		return false, fmt.Sprintf("unknown condition type: %s", condition.Type)
	}
}

// shouldRunAlert determines if an alert should run based on its schedule.
func shouldRunAlert(alert *Alert, now time.Time) bool {
	if alert.Schedule == nil {
		// No schedule means run on every check (manual or cron-based)
		return true
	}

	// Check if enough time has passed since last check
	if alert.LastChecked != nil {
		var minInterval time.Duration
		switch alert.Schedule.Type {
		case "hourly":
			minInterval = time.Hour
		case "daily":
			minInterval = 24 * time.Hour
		case "weekly":
			minInterval = 7 * 24 * time.Hour
		default:
			minInterval = time.Hour
		}

		if now.Sub(*alert.LastChecked) < minInterval {
			return false
		}
	}

	return true
}

// Validation helpers

func validateCondition(c *Condition) error {
	validTypes := map[string]bool{
		"above":        true,
		"below":        true,
		"reaches":      true,
		"changes":      true,
		"rows_present": true,
	}
	if !validTypes[c.Type] {
		return fmt.Errorf("%w: unknown type %s", ErrInvalidCondition, c.Type)
	}
	return nil
}

func validateChannel(ch Channel) error {
	validTypes := map[string]bool{
		"email":   true,
		"slack":   true,
		"webhook": true,
	}
	if !validTypes[ch.Type] {
		return fmt.Errorf("unknown channel type: %s", ch.Type)
	}

	switch ch.Type {
	case "email":
		if ch.Config["to"] == "" {
			return fmt.Errorf("email channel requires 'to' config")
		}
	case "slack":
		if ch.Config["webhook_url"] == "" {
			return fmt.Errorf("slack channel requires 'webhook_url' config")
		}
	case "webhook":
		if ch.Config["url"] == "" {
			return fmt.Errorf("webhook channel requires 'url' config")
		}
	}

	return nil
}

// Helper functions

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func buildEmailBody(alert *Alert, config map[string]string) string {
	return fmt.Sprintf(`
Alert: %s

Your alert has been triggered.

Condition: %s %v
`, alert.Name, alert.Condition.Type, alert.Condition.Value)
}

func buildSlackMessage(alert *Alert) string {
	return fmt.Sprintf(":bell: *Alert Triggered: %s*\n\nCondition: %s %v",
		alert.Name, alert.Condition.Type, alert.Condition.Value)
}

func buildWebhookPayload(alert *Alert) map[string]any {
	return map[string]any{
		"alert_id":   alert.ID,
		"alert_name": alert.Name,
		"condition":  alert.Condition,
		"triggered":  true,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}
