package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func (s *Store) ListAgents(ctx context.Context) ([]types.Agent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, model, system_prompt, workspace, max_tokens, temperature, status, created_at, updated_at FROM agents ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []types.Agent
	for rows.Next() {
		var a types.Agent
		if err := rows.Scan(&a.ID, &a.Name, &a.Model, &a.SystemPrompt, &a.Workspace, &a.MaxTokens, &a.Temperature, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func (s *Store) GetAgent(ctx context.Context, id string) (*types.Agent, error) {
	var a types.Agent
	err := s.db.QueryRowContext(ctx, `SELECT id, name, model, system_prompt, workspace, max_tokens, temperature, status, created_at, updated_at FROM agents WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.Model, &a.SystemPrompt, &a.Workspace, &a.MaxTokens, &a.Temperature, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) CreateAgent(ctx context.Context, a *types.Agent) error {
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Status == "" {
		a.Status = "active"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO agents (id, name, model, system_prompt, workspace, max_tokens, temperature, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Name, a.Model, a.SystemPrompt, a.Workspace, a.MaxTokens, a.Temperature, a.Status, a.CreatedAt, a.UpdatedAt)
	return err
}

func (s *Store) UpdateAgent(ctx context.Context, a *types.Agent) error {
	a.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE agents SET name=?, model=?, system_prompt=?, workspace=?, max_tokens=?, temperature=?, status=?, updated_at=? WHERE id=?`,
		a.Name, a.Model, a.SystemPrompt, a.Workspace, a.MaxTokens, a.Temperature, a.Status, a.UpdatedAt, a.ID)
	return err
}

func (s *Store) DeleteAgent(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM agents WHERE id = ?`, id)
	return err
}
