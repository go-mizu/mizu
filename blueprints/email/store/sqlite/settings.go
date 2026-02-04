package sqlite

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/email/types"
)

// GetSettings returns the current user settings.
func (s *Store) GetSettings(ctx context.Context) (*types.Settings, error) {
	var settings types.Settings
	var conversationView int

	err := s.db.QueryRowContext(ctx, `
		SELECT id, display_name, email_address, signature, theme, density,
			conversation_view, auto_advance, undo_send_seconds
		FROM settings
		WHERE id = 1
	`).Scan(
		&settings.ID, &settings.DisplayName, &settings.EmailAddress,
		&settings.Signature, &settings.Theme, &settings.Density,
		&conversationView, &settings.AutoAdvance, &settings.UndoSendSeconds,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	settings.ConversationView = conversationView == 1

	return &settings, nil
}

// UpdateSettings updates the user settings.
func (s *Store) UpdateSettings(ctx context.Context, settings *types.Settings) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE settings SET
			display_name = ?,
			email_address = ?,
			signature = ?,
			theme = ?,
			density = ?,
			conversation_view = ?,
			auto_advance = ?,
			undo_send_seconds = ?
		WHERE id = 1
	`,
		settings.DisplayName, settings.EmailAddress, settings.Signature,
		settings.Theme, settings.Density, boolToInt(settings.ConversationView),
		settings.AutoAdvance, settings.UndoSendSeconds,
	)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}
