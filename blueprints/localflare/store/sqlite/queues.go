package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

// QueueStoreImpl implements store.QueueStore.
type QueueStoreImpl struct {
	db *sql.DB
}

func (s *QueueStoreImpl) CreateQueue(ctx context.Context, queue *store.Queue) error {
	settings, _ := json.Marshal(queue.Settings)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO queues (id, name, settings, created_at)
		VALUES (?, ?, ?, ?)`,
		queue.ID, queue.Name, string(settings), queue.CreatedAt)
	return err
}

func (s *QueueStoreImpl) GetQueue(ctx context.Context, id string) (*store.Queue, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, settings, created_at FROM queues WHERE id = ?`, id)
	return s.scanQueue(row)
}

func (s *QueueStoreImpl) GetQueueByName(ctx context.Context, name string) (*store.Queue, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, settings, created_at FROM queues WHERE name = ?`, name)
	return s.scanQueue(row)
}

func (s *QueueStoreImpl) scanQueue(row *sql.Row) (*store.Queue, error) {
	var queue store.Queue
	var settings string
	if err := row.Scan(&queue.ID, &queue.Name, &settings, &queue.CreatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(settings), &queue.Settings)
	return &queue, nil
}

func (s *QueueStoreImpl) ListQueues(ctx context.Context) ([]*store.Queue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, settings, created_at FROM queues ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queues []*store.Queue
	for rows.Next() {
		var queue store.Queue
		var settings string
		if err := rows.Scan(&queue.ID, &queue.Name, &settings, &queue.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(settings), &queue.Settings)
		queues = append(queues, &queue)
	}
	return queues, rows.Err()
}

func (s *QueueStoreImpl) DeleteQueue(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete messages
	_, err = tx.ExecContext(ctx, `DELETE FROM queue_messages WHERE queue_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete consumers
	_, err = tx.ExecContext(ctx, `DELETE FROM queue_consumers WHERE queue_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete queue
	_, err = tx.ExecContext(ctx, `DELETE FROM queues WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *QueueStoreImpl) SendMessage(ctx context.Context, queueID string, msg *store.QueueMessage) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO queue_messages (id, queue_id, body, content_type, attempts, created_at, visible_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, queueID, msg.Body, msg.ContentType, msg.Attempts, msg.CreatedAt, msg.VisibleAt, msg.ExpiresAt)
	return err
}

func (s *QueueStoreImpl) SendBatch(ctx context.Context, queueID string, msgs []*store.QueueMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO queue_messages (id, queue_id, body, content_type, attempts, created_at, visible_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, msg := range msgs {
		if _, err := stmt.ExecContext(ctx, msg.ID, queueID, msg.Body, msg.ContentType,
			msg.Attempts, msg.CreatedAt, msg.VisibleAt, msg.ExpiresAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *QueueStoreImpl) PullMessages(ctx context.Context, queueID string, batchSize int, visibilityTimeout int) ([]*store.QueueMessage, error) {
	now := time.Now()
	newVisibleAt := now.Add(time.Duration(visibilityTimeout) * time.Second)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Select messages that are visible
	rows, err := tx.QueryContext(ctx,
		`SELECT id, queue_id, body, content_type, attempts, created_at, visible_at, expires_at
		FROM queue_messages
		WHERE queue_id = ? AND visible_at <= ? AND expires_at > ?
		ORDER BY created_at
		LIMIT ?`,
		queueID, now, now, batchSize)
	if err != nil {
		return nil, err
	}

	var msgs []*store.QueueMessage
	var msgIDs []interface{}
	for rows.Next() {
		var msg store.QueueMessage
		if err := rows.Scan(&msg.ID, &msg.QueueID, &msg.Body, &msg.ContentType,
			&msg.Attempts, &msg.CreatedAt, &msg.VisibleAt, &msg.ExpiresAt); err != nil {
			rows.Close()
			return nil, err
		}
		msgs = append(msgs, &msg)
		msgIDs = append(msgIDs, msg.ID)
	}
	rows.Close()

	if len(msgs) == 0 {
		return msgs, nil
	}

	// Update visibility timeout and increment attempts
	query := `UPDATE queue_messages SET visible_at = ?, attempts = attempts + 1 WHERE id IN (`
	args := []interface{}{newVisibleAt}
	for i, id := range msgIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Update attempts in returned messages
	for _, msg := range msgs {
		msg.Attempts++
		msg.VisibleAt = newVisibleAt
	}

	return msgs, nil
}

func (s *QueueStoreImpl) AckMessage(ctx context.Context, queueID, msgID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM queue_messages WHERE queue_id = ? AND id = ?`, queueID, msgID)
	return err
}

func (s *QueueStoreImpl) AckBatch(ctx context.Context, queueID string, msgIDs []string) error {
	if len(msgIDs) == 0 {
		return nil
	}

	query := `DELETE FROM queue_messages WHERE queue_id = ? AND id IN (`
	args := []interface{}{queueID}
	for i, id := range msgIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *QueueStoreImpl) RetryMessage(ctx context.Context, queueID, msgID string, delaySeconds int) error {
	newVisibleAt := time.Now().Add(time.Duration(delaySeconds) * time.Second)
	_, err := s.db.ExecContext(ctx,
		`UPDATE queue_messages SET visible_at = ? WHERE queue_id = ? AND id = ?`,
		newVisibleAt, queueID, msgID)
	return err
}

func (s *QueueStoreImpl) GetQueueStats(ctx context.Context, queueID string) (*store.QueueStats, error) {
	now := time.Now()
	var stats store.QueueStats

	row := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM queue_messages WHERE queue_id = ? AND expires_at > ?`, queueID, now)
	if err := row.Scan(&stats.Messages); err != nil {
		return nil, err
	}

	row = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM queue_messages WHERE queue_id = ? AND visible_at <= ? AND expires_at > ?`,
		queueID, now, now)
	if err := row.Scan(&stats.MessagesReady); err != nil {
		return nil, err
	}

	row = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM queue_messages WHERE queue_id = ? AND visible_at > ? AND expires_at > ?`,
		queueID, now, now)
	if err := row.Scan(&stats.MessagesDelayed); err != nil {
		return nil, err
	}

	return &stats, nil
}

func (s *QueueStoreImpl) CreateConsumer(ctx context.Context, consumer *store.QueueConsumer) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO queue_consumers (id, queue_id, script_name, type, max_batch_size, max_batch_timeout, max_retries, dead_letter_queue, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		consumer.ID, consumer.QueueID, consumer.ScriptName, consumer.Type,
		consumer.MaxBatchSize, consumer.MaxBatchTimeout, consumer.MaxRetries,
		consumer.DeadLetterQueue, consumer.CreatedAt)
	return err
}

func (s *QueueStoreImpl) GetConsumer(ctx context.Context, id string) (*store.QueueConsumer, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, queue_id, script_name, type, max_batch_size, max_batch_timeout, max_retries, dead_letter_queue, created_at
		FROM queue_consumers WHERE id = ?`, id)
	var c store.QueueConsumer
	var dlq sql.NullString
	if err := row.Scan(&c.ID, &c.QueueID, &c.ScriptName, &c.Type,
		&c.MaxBatchSize, &c.MaxBatchTimeout, &c.MaxRetries, &dlq, &c.CreatedAt); err != nil {
		return nil, err
	}
	c.DeadLetterQueue = dlq.String
	return &c, nil
}

func (s *QueueStoreImpl) ListConsumers(ctx context.Context, queueID string) ([]*store.QueueConsumer, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, queue_id, script_name, type, max_batch_size, max_batch_timeout, max_retries, dead_letter_queue, created_at
		FROM queue_consumers WHERE queue_id = ?`, queueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consumers []*store.QueueConsumer
	for rows.Next() {
		var c store.QueueConsumer
		var dlq sql.NullString
		if err := rows.Scan(&c.ID, &c.QueueID, &c.ScriptName, &c.Type,
			&c.MaxBatchSize, &c.MaxBatchTimeout, &c.MaxRetries, &dlq, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.DeadLetterQueue = dlq.String
		consumers = append(consumers, &c)
	}
	return consumers, rows.Err()
}

func (s *QueueStoreImpl) DeleteConsumer(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM queue_consumers WHERE id = ?`, id)
	return err
}

func (s *QueueStoreImpl) MoveToDeadLetter(ctx context.Context, queueID, msgID string) error {
	// Get the consumer's dead letter queue
	row := s.db.QueryRowContext(ctx,
		`SELECT dead_letter_queue FROM queue_consumers WHERE queue_id = ? AND dead_letter_queue IS NOT NULL LIMIT 1`, queueID)
	var dlqID sql.NullString
	if err := row.Scan(&dlqID); err != nil {
		if err == sql.ErrNoRows {
			// No DLQ configured, just delete the message
			return s.AckMessage(ctx, queueID, msgID)
		}
		return err
	}

	if !dlqID.Valid || dlqID.String == "" {
		return s.AckMessage(ctx, queueID, msgID)
	}

	// Move message to DLQ
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the message
	row = tx.QueryRowContext(ctx,
		`SELECT body, content_type, created_at FROM queue_messages WHERE queue_id = ? AND id = ?`, queueID, msgID)
	var body []byte
	var contentType string
	var createdAt time.Time
	if err := row.Scan(&body, &contentType, &createdAt); err != nil {
		return err
	}

	// Delete from original queue
	_, err = tx.ExecContext(ctx, `DELETE FROM queue_messages WHERE queue_id = ? AND id = ?`, queueID, msgID)
	if err != nil {
		return err
	}

	// Insert into DLQ
	now := time.Now()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO queue_messages (id, queue_id, body, content_type, attempts, created_at, visible_at, expires_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?)`,
		msgID+"-dlq", dlqID.String, body, contentType, now, now, now.Add(30*24*time.Hour))
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Schema for Queues
const queuesSchema = `
	-- Queues
	CREATE TABLE IF NOT EXISTS queues (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		settings TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Queue Messages
	CREATE TABLE IF NOT EXISTS queue_messages (
		id TEXT PRIMARY KEY,
		queue_id TEXT NOT NULL,
		body BLOB NOT NULL,
		content_type TEXT DEFAULT 'json',
		attempts INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		visible_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (queue_id) REFERENCES queues(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_queue_messages_queue ON queue_messages(queue_id);
	CREATE INDEX IF NOT EXISTS idx_queue_messages_visible ON queue_messages(queue_id, visible_at, expires_at);

	-- Queue Consumers
	CREATE TABLE IF NOT EXISTS queue_consumers (
		id TEXT PRIMARY KEY,
		queue_id TEXT NOT NULL,
		script_name TEXT NOT NULL,
		type TEXT DEFAULT 'worker',
		max_batch_size INTEGER DEFAULT 10,
		max_batch_timeout INTEGER DEFAULT 30,
		max_retries INTEGER DEFAULT 3,
		dead_letter_queue TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (queue_id) REFERENCES queues(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_queue_consumers_queue ON queue_consumers(queue_id);
`
