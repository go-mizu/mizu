package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// ListContacts returns contacts, optionally filtered by a search query.
func (s *Store) ListContacts(ctx context.Context, query string) ([]types.Contact, error) {
	var rows *sql.Rows
	var err error

	if query != "" {
		// Try FTS first for better search
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.email, c.name, c.avatar_url, c.is_frequent,
				c.last_contacted, c.contact_count, c.created_at
			FROM contacts c
			JOIN contacts_fts ON contacts_fts.rowid = c.rowid
			WHERE contacts_fts MATCH ?
			ORDER BY c.contact_count DESC
			LIMIT 50
		`, query)
		if err != nil {
			// Fallback to LIKE search if FTS fails
			q := "%" + query + "%"
			rows, err = s.db.QueryContext(ctx, `
				SELECT id, email, name, avatar_url, is_frequent,
					last_contacted, contact_count, created_at
				FROM contacts
				WHERE name LIKE ? OR email LIKE ?
				ORDER BY contact_count DESC
				LIMIT 50
			`, q, q)
			if err != nil {
				return nil, fmt.Errorf("failed to search contacts: %w", err)
			}
		}
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, email, name, avatar_url, is_frequent,
				last_contacted, contact_count, created_at
			FROM contacts
			ORDER BY contact_count DESC, name ASC
			LIMIT 100
		`)
		if err != nil {
			return nil, fmt.Errorf("failed to list contacts: %w", err)
		}
	}
	defer rows.Close()

	var contacts []types.Contact
	for rows.Next() {
		var c types.Contact
		var isFrequent int
		var lastContacted sql.NullString
		var createdAt string

		if err := rows.Scan(
			&c.ID, &c.Email, &c.Name, &c.AvatarURL,
			&isFrequent, &lastContacted, &c.ContactCount, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}

		c.IsFrequent = isFrequent == 1
		if lastContacted.Valid {
			t, _ := time.Parse(time.RFC3339, lastContacted.String)
			c.LastContacted = &t
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		contacts = append(contacts, c)
	}

	if contacts == nil {
		contacts = []types.Contact{}
	}

	return contacts, nil
}

// CreateContact creates a new contact.
func (s *Store) CreateContact(ctx context.Context, contact *types.Contact) error {
	if contact.ID == "" {
		contact.ID = uuid.New().String()
	}

	now := time.Now()
	if contact.CreatedAt.IsZero() {
		contact.CreatedAt = now
	}

	var lastContacted *string
	if contact.LastContacted != nil {
		s := contact.LastContacted.Format(time.RFC3339)
		lastContacted = &s
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO contacts (id, email, name, avatar_url, is_frequent, last_contacted, contact_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			name = CASE WHEN excluded.name != '' THEN excluded.name ELSE contacts.name END,
			contact_count = contacts.contact_count + 1
	`,
		contact.ID, contact.Email, contact.Name, contact.AvatarURL,
		boolToInt(contact.IsFrequent), lastContacted, contact.ContactCount,
		contact.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}

	return nil
}

// UpdateContact updates specific fields of a contact.
func (s *Store) UpdateContact(ctx context.Context, id string, updates map[string]any) error {
	var setClauses []string
	var args []any

	allowedFields := map[string]string{
		"name":       "name",
		"email":      "email",
		"avatar_url": "avatar_url",
	}

	for key, col := range allowedFields {
		if val, ok := updates[key]; ok {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
			args = append(args, val)
		}
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE contacts SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	return nil
}

// DeleteContact deletes a contact.
func (s *Store) DeleteContact(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM contacts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}
	return nil
}
