package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/email/store"
	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// ListEmails returns a paginated, filtered list of emails.
func (s *Store) ListEmails(ctx context.Context, filter store.EmailFilter) (*types.EmailListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 25
	}

	var conditions []string
	var args []any

	if filter.LabelID != "" {
		conditions = append(conditions, "e.id IN (SELECT email_id FROM email_labels WHERE label_id = ?)")
		args = append(args, filter.LabelID)
	}
	if filter.IsRead != nil {
		conditions = append(conditions, "e.is_read = ?")
		if *filter.IsRead {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if filter.IsStarred != nil {
		conditions = append(conditions, "e.is_starred = ?")
		if *filter.IsStarred {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if filter.IsDraft != nil {
		conditions = append(conditions, "e.is_draft = ?")
		if *filter.IsDraft {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if filter.Query != "" {
		conditions = append(conditions, "e.id IN (SELECT e2.id FROM emails e2 WHERE e2.subject LIKE ? OR e2.from_name LIKE ? OR e2.from_address LIKE ?)")
		q := "%" + filter.Query + "%"
		args = append(args, q, q, q)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM emails e %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count emails: %w", err)
	}

	totalPages := (total + filter.PerPage - 1) / filter.PerPage
	offset := (filter.Page - 1) * filter.PerPage

	// Get emails
	query := fmt.Sprintf(`
		SELECT e.id, e.thread_id, e.message_id, e.in_reply_to, e.reference_ids,
			e.from_address, e.from_name, e.to_addresses, e.cc_addresses, e.bcc_addresses,
			e.subject, e.body_text, e.body_html, e.snippet,
			e.is_read, e.is_starred, e.is_important, e.is_draft, e.is_sent,
			e.has_attachments, e.size_bytes, e.sent_at, e.received_at, e.created_at, e.updated_at,
			e.snoozed_until, e.scheduled_at, e.is_muted
		FROM emails e
		%s
		ORDER BY e.received_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	listArgs := append(args, filter.PerPage, offset)
	rows, err := s.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()

	var emails []types.Email
	for rows.Next() {
		email, err := scanEmail(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, *email)
	}
	rows.Close()

	// Load labels after closing the rows cursor to avoid SQLite connection contention
	for i := range emails {
		labels, err := s.getEmailLabels(ctx, emails[i].ID)
		if err != nil {
			return nil, err
		}
		emails[i].Labels = labels
	}

	if emails == nil {
		emails = []types.Email{}
	}

	return &types.EmailListResponse{
		Emails:     emails,
		Total:      total,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalPages: totalPages,
	}, nil
}

// GetEmail returns a single email by ID.
func (s *Store) GetEmail(ctx context.Context, id string) (*types.Email, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, thread_id, message_id, in_reply_to, reference_ids,
			from_address, from_name, to_addresses, cc_addresses, bcc_addresses,
			subject, body_text, body_html, snippet,
			is_read, is_starred, is_important, is_draft, is_sent,
			has_attachments, size_bytes, sent_at, received_at, created_at, updated_at,
			snoozed_until, scheduled_at, is_muted
		FROM emails
		WHERE id = ?
	`, id)

	email, err := scanEmailRow(row)
	if err != nil {
		return nil, fmt.Errorf("email not found: %w", err)
	}

	// Load labels
	labels, err := s.getEmailLabels(ctx, email.ID)
	if err != nil {
		return nil, err
	}
	email.Labels = labels

	return email, nil
}

// CreateEmail creates a new email in the database.
func (s *Store) CreateEmail(ctx context.Context, email *types.Email) error {
	if email.ID == "" {
		email.ID = uuid.New().String()
	}
	if email.ThreadID == "" {
		email.ThreadID = uuid.New().String()
	}
	if email.MessageID == "" {
		email.MessageID = fmt.Sprintf("<%s@email.local>", uuid.New().String())
	}

	now := time.Now()
	if email.CreatedAt.IsZero() {
		email.CreatedAt = now
	}
	if email.UpdatedAt.IsZero() {
		email.UpdatedAt = now
	}
	if email.ReceivedAt.IsZero() {
		email.ReceivedAt = now
	}

	refsJSON, _ := json.Marshal(email.References)
	toJSON, _ := json.Marshal(email.ToAddresses)
	ccJSON, _ := json.Marshal(email.CCAddresses)
	bccJSON, _ := json.Marshal(email.BCCAddresses)

	// Generate snippet from body text if not provided
	if email.Snippet == "" && email.BodyText != "" {
		snippet := email.BodyText
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		email.Snippet = snippet
	}

	// Calculate size if not set
	if email.SizeBytes == 0 {
		email.SizeBytes = int64(len(email.BodyText) + len(email.BodyHTML) + len(email.Subject))
	}

	var sentAt *string
	if email.SentAt != nil {
		s := email.SentAt.Format(time.RFC3339)
		sentAt = &s
	}

	var snoozedUntil *string
	if email.SnoozedUntil != nil {
		s := email.SnoozedUntil.Format(time.RFC3339)
		snoozedUntil = &s
	}

	var scheduledAt *string
	if email.ScheduledAt != nil {
		s := email.ScheduledAt.Format(time.RFC3339)
		scheduledAt = &s
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO emails (id, thread_id, message_id, in_reply_to, reference_ids,
			from_address, from_name, to_addresses, cc_addresses, bcc_addresses,
			subject, body_text, body_html, snippet,
			is_read, is_starred, is_important, is_draft, is_sent,
			has_attachments, size_bytes, sent_at, received_at, created_at, updated_at,
			snoozed_until, scheduled_at, is_muted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`,
		email.ID, email.ThreadID, email.MessageID, email.InReplyTo, string(refsJSON),
		email.FromAddress, email.FromName, string(toJSON), string(ccJSON), string(bccJSON),
		email.Subject, email.BodyText, email.BodyHTML, email.Snippet,
		boolToInt(email.IsRead), boolToInt(email.IsStarred), boolToInt(email.IsImportant),
		boolToInt(email.IsDraft), boolToInt(email.IsSent),
		boolToInt(email.HasAttachments), email.SizeBytes,
		sentAt, email.ReceivedAt.Format(time.RFC3339),
		email.CreatedAt.Format(time.RFC3339), email.UpdatedAt.Format(time.RFC3339),
		snoozedUntil, scheduledAt, boolToInt(email.IsMuted),
	)
	if err != nil {
		return fmt.Errorf("failed to create email: %w", err)
	}

	// Add labels
	for _, labelID := range email.Labels {
		s.AddEmailLabel(ctx, email.ID, labelID)
	}

	// Auto-create or update contact for the sender
	s.upsertContactFromEmail(ctx, email.FromAddress, email.FromName)

	return nil
}

// isJSONField returns true if the field should be JSON-encoded before storage.
func isJSONField(field string) bool {
	return field == "to_addresses" || field == "cc_addresses" || field == "bcc_addresses"
}

// UpdateEmail updates specific fields of an email.
func (s *Store) UpdateEmail(ctx context.Context, id string, updates map[string]any) error {
	var setClauses []string
	var args []any

	allowedFields := map[string]string{
		"is_read":         "is_read",
		"is_starred":      "is_starred",
		"is_important":    "is_important",
		"is_draft":        "is_draft",
		"is_sent":         "is_sent",
		"is_muted":        "is_muted",
		"subject":         "subject",
		"body_text":       "body_text",
		"body_html":       "body_html",
		"snippet":         "snippet",
		"sent_at":         "sent_at",
		"to_addresses":    "to_addresses",
		"cc_addresses":    "cc_addresses",
		"bcc_addresses":   "bcc_addresses",
		"has_attachments":  "has_attachments",
		"snoozed_until":   "snoozed_until",
		"scheduled_at":    "scheduled_at",
	}

	for key, col := range allowedFields {
		if val, ok := updates[key]; ok {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
			switch v := val.(type) {
			case bool:
				args = append(args, boolToInt(v))
			case time.Time:
				args = append(args, v.Format(time.RFC3339))
			case *time.Time:
				if v != nil {
					args = append(args, v.Format(time.RFC3339))
				} else {
					args = append(args, nil)
				}
			default:
				if isJSONField(key) {
					jsonBytes, _ := json.Marshal(v)
					args = append(args, string(jsonBytes))
				} else {
					args = append(args, v)
				}
			}
		}
	}

	if len(setClauses) == 0 {
		// Still handle labels even if no column updates
		if labels, ok := updates["labels"]; ok {
			s.db.ExecContext(ctx, "DELETE FROM email_labels WHERE email_id = ?", id)
			if labelList, ok := labels.([]string); ok {
				for _, labelID := range labelList {
					s.AddEmailLabel(ctx, id, labelID)
				}
			}
			if labelList, ok := labels.([]any); ok {
				for _, l := range labelList {
					if labelID, ok := l.(string); ok {
						s.AddEmailLabel(ctx, id, labelID)
					}
				}
			}
		}
		return nil
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now().Format(time.RFC3339))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE emails SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	// Handle label updates
	if labels, ok := updates["labels"]; ok {
		s.db.ExecContext(ctx, "DELETE FROM email_labels WHERE email_id = ?", id)
		if labelList, ok := labels.([]string); ok {
			for _, labelID := range labelList {
				s.AddEmailLabel(ctx, id, labelID)
			}
		}
		if labelList, ok := labels.([]any); ok {
			for _, l := range labelList {
				if labelID, ok := l.(string); ok {
					s.AddEmailLabel(ctx, id, labelID)
				}
			}
		}
	}

	return nil
}

// DeleteEmail moves an email to trash or permanently deletes it.
func (s *Store) DeleteEmail(ctx context.Context, id string, permanent bool) error {
	if permanent {
		_, err := s.db.ExecContext(ctx, "DELETE FROM emails WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete email: %w", err)
		}
		return nil
	}

	// Move to trash: remove from inbox, add trash label
	s.RemoveEmailLabel(ctx, id, "inbox")
	s.AddEmailLabel(ctx, id, "trash")

	return nil
}

// BatchUpdateEmails performs batch operations on multiple emails.
func (s *Store) BatchUpdateEmails(ctx context.Context, action *types.BatchAction) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, id := range action.IDs {
		switch action.Action {
		case "archive":
			// Remove inbox label
			tx.ExecContext(ctx, "DELETE FROM email_labels WHERE email_id = ? AND label_id = 'inbox'", id)
		case "unarchive":
			// Re-add inbox label
			tx.ExecContext(ctx, "INSERT OR IGNORE INTO email_labels (email_id, label_id) VALUES (?, 'inbox')", id)
		case "trash":
			// Remove inbox label, add trash label
			tx.ExecContext(ctx, "DELETE FROM email_labels WHERE email_id = ? AND label_id = 'inbox'", id)
			tx.ExecContext(ctx, "INSERT OR IGNORE INTO email_labels (email_id, label_id) VALUES (?, 'trash')", id)
		case "untrash":
			// Remove trash label, re-add inbox label
			tx.ExecContext(ctx, "DELETE FROM email_labels WHERE email_id = ? AND label_id = 'trash'", id)
			tx.ExecContext(ctx, "INSERT OR IGNORE INTO email_labels (email_id, label_id) VALUES (?, 'inbox')", id)
		case "delete":
			tx.ExecContext(ctx, "DELETE FROM email_labels WHERE email_id = ?", id)
			tx.ExecContext(ctx, "DELETE FROM emails WHERE id = ?", id)
		case "read":
			tx.ExecContext(ctx, "UPDATE emails SET is_read = 1, updated_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), id)
		case "unread":
			tx.ExecContext(ctx, "UPDATE emails SET is_read = 0, updated_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), id)
		case "star":
			tx.ExecContext(ctx, "UPDATE emails SET is_starred = 1, updated_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), id)
		case "unstar":
			tx.ExecContext(ctx, "UPDATE emails SET is_starred = 0, updated_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), id)
		case "important":
			tx.ExecContext(ctx, "UPDATE emails SET is_important = 1, updated_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), id)
		case "unimportant":
			tx.ExecContext(ctx, "UPDATE emails SET is_important = 0, updated_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), id)
		case "label":
			if action.LabelID != "" {
				tx.ExecContext(ctx, "INSERT OR IGNORE INTO email_labels (email_id, label_id) VALUES (?, ?)", id, action.LabelID)
			}
		case "unlabel":
			if action.LabelID != "" {
				tx.ExecContext(ctx, "DELETE FROM email_labels WHERE email_id = ? AND label_id = ?", id, action.LabelID)
			}
		case "add_label":
			if action.LabelID != "" {
				tx.ExecContext(ctx, "INSERT OR IGNORE INTO email_labels (email_id, label_id) VALUES (?, ?)", id, action.LabelID)
			}
		case "remove_label":
			if action.LabelID != "" {
				tx.ExecContext(ctx, "DELETE FROM email_labels WHERE email_id = ? AND label_id = ?", id, action.LabelID)
			}
		case "mute":
			tx.ExecContext(ctx, "UPDATE emails SET is_muted = 1, updated_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), id)
		case "unmute":
			tx.ExecContext(ctx, "UPDATE emails SET is_muted = 0, updated_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), id)
		}
	}

	return tx.Commit()
}

// searchFilter holds parsed components of a Gmail-style search query.
type searchFilter struct {
	from          string
	to            string
	subject       string
	hasAttachment bool
	isUnread      *bool
	isStarred     *bool
	isImportant   *bool
	before        string // RFC3339
	after         string // RFC3339
	label         string
	freeText      string
}

// parseSearchQuery extracts Gmail-style operators from a query string.
// Supported: from:, to:, subject:, has:attachment, is:unread, is:starred,
// is:important, before:YYYY/MM/DD, after:YYYY/MM/DD, label:
func parseSearchQuery(query string) searchFilter {
	var sf searchFilter
	var freeWords []string

	tokens := tokenizeSearch(query)
	for _, tok := range tokens {
		colonIdx := strings.Index(tok, ":")
		if colonIdx <= 0 {
			freeWords = append(freeWords, tok)
			continue
		}
		op := strings.ToLower(tok[:colonIdx])
		val := tok[colonIdx+1:]
		switch op {
		case "from":
			sf.from = val
		case "to":
			sf.to = val
		case "subject":
			sf.subject = val
		case "has":
			if strings.EqualFold(val, "attachment") {
				sf.hasAttachment = true
			}
		case "is":
			switch strings.ToLower(val) {
			case "unread":
				b := true
				sf.isUnread = &b
			case "read":
				b := false
				sf.isUnread = &b
			case "starred":
				b := true
				sf.isStarred = &b
			case "important":
				b := true
				sf.isImportant = &b
			}
		case "before":
			sf.before = parseDateOperator(val)
		case "after":
			sf.after = parseDateOperator(val)
		case "label":
			sf.label = val
		default:
			freeWords = append(freeWords, tok)
		}
	}
	sf.freeText = strings.TrimSpace(strings.Join(freeWords, " "))
	return sf
}

// tokenizeSearch splits a search query respecting quoted strings.
func tokenizeSearch(query string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	for _, ch := range query {
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == ' ' && !inQuote:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// parseDateOperator converts YYYY/MM/DD or YYYY-MM-DD to RFC3339.
func parseDateOperator(val string) string {
	val = strings.ReplaceAll(val, "/", "-")
	for _, layout := range []string{"2006-01-02", "2006-1-2"} {
		if t, err := time.Parse(layout, val); err == nil {
			return t.Format(time.RFC3339)
		}
	}
	return ""
}

// SearchEmails searches emails using Gmail-style operators and FTS5 full-text search.
func (s *Store) SearchEmails(ctx context.Context, query string, page, perPage int) (*types.EmailListResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}

	sf := parseSearchQuery(query)

	var conditions []string
	var args []any

	if sf.from != "" {
		conditions = append(conditions, "(e.from_address LIKE ? OR e.from_name LIKE ?)")
		q := "%" + sf.from + "%"
		args = append(args, q, q)
	}
	if sf.to != "" {
		conditions = append(conditions, "e.to_addresses LIKE ?")
		args = append(args, "%"+sf.to+"%")
	}
	if sf.subject != "" {
		conditions = append(conditions, "e.subject LIKE ?")
		args = append(args, "%"+sf.subject+"%")
	}
	if sf.hasAttachment {
		conditions = append(conditions, "e.has_attachments = 1")
	}
	if sf.isUnread != nil {
		if *sf.isUnread {
			conditions = append(conditions, "e.is_read = 0")
		} else {
			conditions = append(conditions, "e.is_read = 1")
		}
	}
	if sf.isStarred != nil {
		if *sf.isStarred {
			conditions = append(conditions, "e.is_starred = 1")
		}
	}
	if sf.isImportant != nil {
		if *sf.isImportant {
			conditions = append(conditions, "e.is_important = 1")
		}
	}
	if sf.before != "" {
		conditions = append(conditions, "e.received_at < ?")
		args = append(args, sf.before)
	}
	if sf.after != "" {
		conditions = append(conditions, "e.received_at > ?")
		args = append(args, sf.after)
	}
	if sf.label != "" {
		conditions = append(conditions, "e.id IN (SELECT email_id FROM email_labels WHERE label_id = ?)")
		args = append(args, sf.label)
	}

	// Use FTS5 for remaining free-text
	useFTS := sf.freeText != ""
	joinClause := ""
	if useFTS {
		joinClause = "JOIN emails_fts ON emails_fts.rowid = e.rowid"
		conditions = append(conditions, "emails_fts MATCH ?")
		args = append(args, sf.freeText)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total matches
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM emails e %s %s", joinClause, whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	totalPages := (total + perPage - 1) / perPage
	offset := (page - 1) * perPage

	orderClause := "ORDER BY e.received_at DESC"
	if useFTS {
		orderClause = "ORDER BY rank"
	}

	selectQuery := fmt.Sprintf(`
		SELECT e.id, e.thread_id, e.message_id, e.in_reply_to, e.reference_ids,
			e.from_address, e.from_name, e.to_addresses, e.cc_addresses, e.bcc_addresses,
			e.subject, e.body_text, e.body_html, e.snippet,
			e.is_read, e.is_starred, e.is_important, e.is_draft, e.is_sent,
			e.has_attachments, e.size_bytes, e.sent_at, e.received_at, e.created_at, e.updated_at,
			e.snoozed_until, e.scheduled_at, e.is_muted
		FROM emails e
		%s %s %s
		LIMIT ? OFFSET ?
	`, joinClause, whereClause, orderClause)

	listArgs := append(args, perPage, offset)
	rows, err := s.db.QueryContext(ctx, selectQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}
	defer rows.Close()

	var emails []types.Email
	for rows.Next() {
		email, err := scanEmail(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, *email)
	}
	rows.Close()

	for i := range emails {
		labels, err := s.getEmailLabels(ctx, emails[i].ID)
		if err != nil {
			return nil, err
		}
		emails[i].Labels = labels
	}

	if emails == nil {
		emails = []types.Email{}
	}

	return &types.EmailListResponse{
		Emails:     emails,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// ListThreads returns a paginated list of email threads.
func (s *Store) ListThreads(ctx context.Context, filter store.EmailFilter) (*types.ThreadListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 25
	}

	var conditions []string
	var args []any

	if filter.LabelID != "" {
		conditions = append(conditions, "e.id IN (SELECT email_id FROM email_labels WHERE label_id = ?)")
		args = append(args, filter.LabelID)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count distinct threads
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT e.thread_id) FROM emails e %s
	`, whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count threads: %w", err)
	}

	totalPages := (total + filter.PerPage - 1) / filter.PerPage
	offset := (filter.Page - 1) * filter.PerPage

	// Get distinct thread IDs ordered by most recent email
	threadQuery := fmt.Sprintf(`
		SELECT e.thread_id, MAX(e.received_at) as last_email_at
		FROM emails e
		%s
		GROUP BY e.thread_id
		ORDER BY last_email_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	threadArgs := append(args, filter.PerPage, offset)
	rows, err := s.db.QueryContext(ctx, threadQuery, threadArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}
	defer rows.Close()

	var threadIDs []string
	for rows.Next() {
		var threadID string
		var lastEmailAt string
		if err := rows.Scan(&threadID, &lastEmailAt); err != nil {
			return nil, fmt.Errorf("failed to scan thread row: %w", err)
		}
		threadIDs = append(threadIDs, threadID)
	}
	rows.Close()

	var threads []types.Thread
	for _, threadID := range threadIDs {
		thread, err := s.GetThread(ctx, threadID)
		if err != nil {
			continue
		}
		threads = append(threads, *thread)
	}

	if threads == nil {
		threads = []types.Thread{}
	}

	return &types.ThreadListResponse{
		Threads:    threads,
		Total:      total,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalPages: totalPages,
	}, nil
}

// GetThread returns a thread by ID with all its emails.
func (s *Store) GetThread(ctx context.Context, id string) (*types.Thread, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, thread_id, message_id, in_reply_to, reference_ids,
			from_address, from_name, to_addresses, cc_addresses, bcc_addresses,
			subject, body_text, body_html, snippet,
			is_read, is_starred, is_important, is_draft, is_sent,
			has_attachments, size_bytes, sent_at, received_at, created_at, updated_at,
			snoozed_until, scheduled_at, is_muted
		FROM emails
		WHERE thread_id = ?
		ORDER BY received_at ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}
	defer rows.Close()

	var emails []types.Email
	for rows.Next() {
		email, err := scanEmail(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, *email)
	}
	rows.Close()

	if len(emails) == 0 {
		return nil, fmt.Errorf("thread not found")
	}

	for i := range emails {
		labels, err := s.getEmailLabels(ctx, emails[i].ID)
		if err != nil {
			return nil, err
		}
		emails[i].Labels = labels
	}

	// Build thread from emails
	thread := &types.Thread{
		ID:         id,
		Subject:    emails[0].Subject,
		Snippet:    emails[len(emails)-1].Snippet,
		Emails:     emails,
		EmailCount: len(emails),
		LastEmailAt: emails[len(emails)-1].ReceivedAt,
	}

	// Aggregate thread-level properties
	labelsMap := make(map[string]bool)
	for _, email := range emails {
		if !email.IsRead {
			thread.UnreadCount++
		}
		if email.IsStarred {
			thread.IsStarred = true
		}
		if email.IsImportant {
			thread.IsImportant = true
		}
		for _, label := range email.Labels {
			labelsMap[label] = true
		}
	}

	for label := range labelsMap {
		thread.Labels = append(thread.Labels, label)
	}

	return thread, nil
}

// getEmailLabels returns the label IDs for a given email.
func (s *Store) getEmailLabels(ctx context.Context, emailID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT label_id FROM email_labels WHERE email_id = ?
	`, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email labels: %w", err)
	}
	defer rows.Close()

	var labels []string
	for rows.Next() {
		var labelID string
		if err := rows.Scan(&labelID); err != nil {
			return nil, err
		}
		labels = append(labels, labelID)
	}

	return labels, nil
}

// upsertContactFromEmail creates or updates a contact from an email sender.
func (s *Store) upsertContactFromEmail(ctx context.Context, address, name string) {
	if address == "" {
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO contacts (id, email, name, last_contacted, contact_count, created_at)
		VALUES (?, ?, ?, ?, 1, ?)
		ON CONFLICT(email) DO UPDATE SET
			name = CASE WHEN excluded.name != '' THEN excluded.name ELSE contacts.name END,
			last_contacted = excluded.last_contacted,
			contact_count = contacts.contact_count + 1
	`, uuid.New().String(), address, name, now, now)
}

// scanEmail scans an email from a sql.Rows.
func scanEmail(rows *sql.Rows) (*types.Email, error) {
	var e types.Email
	var refsJSON, toJSON, ccJSON, bccJSON string
	var sentAt, snoozedUntil, scheduledAt sql.NullString
	var receivedAt, createdAt, updatedAt string
	var isRead, isStarred, isImportant, isDraft, isSent, hasAttachments, isMuted int

	if err := rows.Scan(
		&e.ID, &e.ThreadID, &e.MessageID, &e.InReplyTo, &refsJSON,
		&e.FromAddress, &e.FromName, &toJSON, &ccJSON, &bccJSON,
		&e.Subject, &e.BodyText, &e.BodyHTML, &e.Snippet,
		&isRead, &isStarred, &isImportant, &isDraft, &isSent,
		&hasAttachments, &e.SizeBytes, &sentAt, &receivedAt, &createdAt, &updatedAt,
		&snoozedUntil, &scheduledAt, &isMuted,
	); err != nil {
		return nil, fmt.Errorf("failed to scan email: %w", err)
	}

	e.IsRead = isRead == 1
	e.IsStarred = isStarred == 1
	e.IsImportant = isImportant == 1
	e.IsDraft = isDraft == 1
	e.IsSent = isSent == 1
	e.HasAttachments = hasAttachments == 1
	e.IsMuted = isMuted == 1

	json.Unmarshal([]byte(refsJSON), &e.References)
	json.Unmarshal([]byte(toJSON), &e.ToAddresses)
	json.Unmarshal([]byte(ccJSON), &e.CCAddresses)
	json.Unmarshal([]byte(bccJSON), &e.BCCAddresses)

	if sentAt.Valid {
		t, _ := time.Parse(time.RFC3339, sentAt.String)
		e.SentAt = &t
	}
	if snoozedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, snoozedUntil.String)
		e.SnoozedUntil = &t
	}
	if scheduledAt.Valid {
		t, _ := time.Parse(time.RFC3339, scheduledAt.String)
		e.ScheduledAt = &t
	}
	e.ReceivedAt, _ = time.Parse(time.RFC3339, receivedAt)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &e, nil
}

// scanEmailRow scans an email from a sql.Row.
func scanEmailRow(row *sql.Row) (*types.Email, error) {
	var e types.Email
	var refsJSON, toJSON, ccJSON, bccJSON string
	var sentAt, snoozedUntil, scheduledAt sql.NullString
	var receivedAt, createdAt, updatedAt string
	var isRead, isStarred, isImportant, isDraft, isSent, hasAttachments, isMuted int

	if err := row.Scan(
		&e.ID, &e.ThreadID, &e.MessageID, &e.InReplyTo, &refsJSON,
		&e.FromAddress, &e.FromName, &toJSON, &ccJSON, &bccJSON,
		&e.Subject, &e.BodyText, &e.BodyHTML, &e.Snippet,
		&isRead, &isStarred, &isImportant, &isDraft, &isSent,
		&hasAttachments, &e.SizeBytes, &sentAt, &receivedAt, &createdAt, &updatedAt,
		&snoozedUntil, &scheduledAt, &isMuted,
	); err != nil {
		return nil, fmt.Errorf("failed to scan email: %w", err)
	}

	e.IsRead = isRead == 1
	e.IsStarred = isStarred == 1
	e.IsImportant = isImportant == 1
	e.IsDraft = isDraft == 1
	e.IsSent = isSent == 1
	e.HasAttachments = hasAttachments == 1
	e.IsMuted = isMuted == 1

	json.Unmarshal([]byte(refsJSON), &e.References)
	json.Unmarshal([]byte(toJSON), &e.ToAddresses)
	json.Unmarshal([]byte(ccJSON), &e.CCAddresses)
	json.Unmarshal([]byte(bccJSON), &e.BCCAddresses)

	if sentAt.Valid {
		t, _ := time.Parse(time.RFC3339, sentAt.String)
		e.SentAt = &t
	}
	if snoozedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, snoozedUntil.String)
		e.SnoozedUntil = &t
	}
	if scheduledAt.Valid {
		t, _ := time.Parse(time.RFC3339, scheduledAt.String)
		e.ScheduledAt = &t
	}
	e.ReceivedAt, _ = time.Parse(time.RFC3339, receivedAt)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &e, nil
}

// boolToInt converts a bool to an int for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
