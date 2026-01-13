package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

// DurableObjectStoreImpl implements store.DurableObjectStore.
type DurableObjectStoreImpl struct {
	db      *sql.DB
	dataDir string
}

func (s *DurableObjectStoreImpl) CreateNamespace(ctx context.Context, ns *store.DurableObjectNamespace) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO durable_object_namespaces (id, name, script, class_name, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		ns.ID, ns.Name, ns.Script, ns.ClassName, ns.CreatedAt)
	return err
}

func (s *DurableObjectStoreImpl) GetNamespace(ctx context.Context, id string) (*store.DurableObjectNamespace, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, script, class_name, created_at FROM durable_object_namespaces WHERE id = ?`, id)
	var ns store.DurableObjectNamespace
	if err := row.Scan(&ns.ID, &ns.Name, &ns.Script, &ns.ClassName, &ns.CreatedAt); err != nil {
		return nil, err
	}
	return &ns, nil
}

func (s *DurableObjectStoreImpl) ListNamespaces(ctx context.Context) ([]*store.DurableObjectNamespace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, script, class_name, created_at FROM durable_object_namespaces ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var namespaces []*store.DurableObjectNamespace
	for rows.Next() {
		var ns store.DurableObjectNamespace
		if err := rows.Scan(&ns.ID, &ns.Name, &ns.Script, &ns.ClassName, &ns.CreatedAt); err != nil {
			return nil, err
		}
		namespaces = append(namespaces, &ns)
	}
	return namespaces, rows.Err()
}

func (s *DurableObjectStoreImpl) DeleteNamespace(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM durable_object_namespaces WHERE id = ?`, id)
	return err
}

func (s *DurableObjectStoreImpl) GetOrCreateInstance(ctx context.Context, nsID, objectID string, name string) (*store.DurableObjectInstance, error) {
	now := time.Now()

	// Try to get existing instance
	instance, err := s.GetInstance(ctx, objectID)
	if err == nil {
		// Update last access
		s.db.ExecContext(ctx, `UPDATE durable_object_instances SET last_access = ? WHERE id = ?`, now, objectID)
		return instance, nil
	}

	// Create new instance
	instance = &store.DurableObjectInstance{
		ID:          objectID,
		NamespaceID: nsID,
		Name:        name,
		HasStorage:  false,
		CreatedAt:   now,
		LastAccess:  now,
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO durable_object_instances (id, namespace_id, name, has_storage, created_at, last_access)
		VALUES (?, ?, ?, ?, ?, ?)`,
		instance.ID, instance.NamespaceID, instance.Name, instance.HasStorage, instance.CreatedAt, instance.LastAccess)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *DurableObjectStoreImpl) GetInstance(ctx context.Context, id string) (*store.DurableObjectInstance, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, namespace_id, name, has_storage, created_at, last_access
		FROM durable_object_instances WHERE id = ?`, id)
	var inst store.DurableObjectInstance
	var name sql.NullString
	if err := row.Scan(&inst.ID, &inst.NamespaceID, &name, &inst.HasStorage, &inst.CreatedAt, &inst.LastAccess); err != nil {
		return nil, err
	}
	inst.Name = name.String
	return &inst, nil
}

func (s *DurableObjectStoreImpl) ListInstances(ctx context.Context, nsID string) ([]*store.DurableObjectInstance, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, namespace_id, name, has_storage, created_at, last_access
		FROM durable_object_instances WHERE namespace_id = ? ORDER BY created_at`, nsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []*store.DurableObjectInstance
	for rows.Next() {
		var inst store.DurableObjectInstance
		var name sql.NullString
		if err := rows.Scan(&inst.ID, &inst.NamespaceID, &name, &inst.HasStorage, &inst.CreatedAt, &inst.LastAccess); err != nil {
			return nil, err
		}
		inst.Name = name.String
		instances = append(instances, &inst)
	}
	return instances, rows.Err()
}

func (s *DurableObjectStoreImpl) DeleteInstance(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete storage entries
	_, err = tx.ExecContext(ctx, `DELETE FROM durable_object_storage WHERE object_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete alarms
	_, err = tx.ExecContext(ctx, `DELETE FROM durable_object_alarms WHERE object_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete instance
	_, err = tx.ExecContext(ctx, `DELETE FROM durable_object_instances WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *DurableObjectStoreImpl) Get(ctx context.Context, objectID, key string) ([]byte, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT value FROM durable_object_storage WHERE object_id = ? AND key = ?`, objectID, key)
	var value []byte
	if err := row.Scan(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *DurableObjectStoreImpl) GetMultiple(ctx context.Context, objectID string, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Build query with placeholders
	query := `SELECT key, value FROM durable_object_storage WHERE object_id = ? AND key IN (`
	args := []interface{}{objectID}
	for i, key := range keys {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, key)
	}
	query += ")"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]byte)
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, rows.Err()
}

func (s *DurableObjectStoreImpl) Put(ctx context.Context, objectID, key string, value []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO durable_object_storage (object_id, key, value, updated_at)
		VALUES (?, ?, ?, ?)`,
		objectID, key, value, time.Now())
	if err != nil {
		return err
	}

	// Mark instance as having storage
	_, err = s.db.ExecContext(ctx,
		`UPDATE durable_object_instances SET has_storage = 1 WHERE id = ?`, objectID)
	return err
}

func (s *DurableObjectStoreImpl) PutMultiple(ctx context.Context, objectID string, entries map[string][]byte) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO durable_object_storage (object_id, key, value, updated_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for key, value := range entries {
		if _, err := stmt.ExecContext(ctx, objectID, key, value, now); err != nil {
			return err
		}
	}

	// Mark instance as having storage
	_, err = tx.ExecContext(ctx,
		`UPDATE durable_object_instances SET has_storage = 1 WHERE id = ?`, objectID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *DurableObjectStoreImpl) Delete(ctx context.Context, objectID, key string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM durable_object_storage WHERE object_id = ? AND key = ?`, objectID, key)
	return err
}

func (s *DurableObjectStoreImpl) DeleteMultiple(ctx context.Context, objectID string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	query := `DELETE FROM durable_object_storage WHERE object_id = ? AND key IN (`
	args := []interface{}{objectID}
	for i, key := range keys {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, key)
	}
	query += ")"

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *DurableObjectStoreImpl) DeleteAll(ctx context.Context, objectID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM durable_object_storage WHERE object_id = ?`, objectID)
	return err
}

func (s *DurableObjectStoreImpl) List(ctx context.Context, objectID string, opts *store.DOListOptions) (map[string][]byte, error) {
	query := `SELECT key, value FROM durable_object_storage WHERE object_id = ?`
	args := []interface{}{objectID}

	if opts != nil {
		if opts.Prefix != "" {
			query += ` AND key LIKE ?`
			args = append(args, opts.Prefix+"%")
		}
		if opts.Start != "" {
			query += ` AND key >= ?`
			args = append(args, opts.Start)
		}
		if opts.End != "" {
			query += ` AND key < ?`
			args = append(args, opts.End)
		}
		if opts.Reverse {
			query += ` ORDER BY key DESC`
		} else {
			query += ` ORDER BY key ASC`
		}
		if opts.Limit > 0 {
			query += ` LIMIT ?`
			args = append(args, opts.Limit)
		}
	} else {
		query += ` ORDER BY key ASC`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]byte)
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, rows.Err()
}

func (s *DurableObjectStoreImpl) GetAlarm(ctx context.Context, objectID string) (*time.Time, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT scheduled_time FROM durable_object_alarms WHERE object_id = ?`, objectID)
	var scheduledTime time.Time
	if err := row.Scan(&scheduledTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &scheduledTime, nil
}

func (s *DurableObjectStoreImpl) SetAlarm(ctx context.Context, objectID string, scheduledTime time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO durable_object_alarms (object_id, scheduled_time)
		VALUES (?, ?)`,
		objectID, scheduledTime)
	return err
}

func (s *DurableObjectStoreImpl) DeleteAlarm(ctx context.Context, objectID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM durable_object_alarms WHERE object_id = ?`, objectID)
	return err
}

func (s *DurableObjectStoreImpl) GetDueAlarms(ctx context.Context, before time.Time) ([]*store.DurableObjectAlarm, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT object_id, scheduled_time FROM durable_object_alarms
		WHERE scheduled_time <= ? ORDER BY scheduled_time`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alarms []*store.DurableObjectAlarm
	for rows.Next() {
		var alarm store.DurableObjectAlarm
		if err := rows.Scan(&alarm.ObjectID, &alarm.ScheduledTime); err != nil {
			return nil, err
		}
		alarms = append(alarms, &alarm)
	}
	return alarms, rows.Err()
}

func (s *DurableObjectStoreImpl) ExecSQL(ctx context.Context, objectID, query string, args []interface{}) ([]map[string]interface{}, error) {
	// For SQLite-backed Durable Objects, we use a separate database per object
	// This is a simplified implementation - in production, each DO would have its own SQLite file
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		// If it's not a SELECT, try Exec
		_, execErr := s.db.ExecContext(ctx, query, args...)
		if execErr != nil {
			return nil, err
		}
		return nil, nil
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// Schema for Durable Objects
const durableObjectsSchema = `
	-- Durable Object Namespaces
	CREATE TABLE IF NOT EXISTS durable_object_namespaces (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		script TEXT NOT NULL,
		class_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Durable Object Instances
	CREATE TABLE IF NOT EXISTS durable_object_instances (
		id TEXT PRIMARY KEY,
		namespace_id TEXT NOT NULL,
		name TEXT,
		has_storage INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_access DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (namespace_id) REFERENCES durable_object_namespaces(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_do_instances_namespace ON durable_object_instances(namespace_id);
	CREATE INDEX IF NOT EXISTS idx_do_instances_name ON durable_object_instances(name) WHERE name IS NOT NULL;

	-- Durable Object Storage
	CREATE TABLE IF NOT EXISTS durable_object_storage (
		object_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value BLOB,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (object_id, key),
		FOREIGN KEY (object_id) REFERENCES durable_object_instances(id) ON DELETE CASCADE
	);

	-- Durable Object Alarms
	CREATE TABLE IF NOT EXISTS durable_object_alarms (
		object_id TEXT PRIMARY KEY,
		scheduled_time DATETIME NOT NULL,
		FOREIGN KEY (object_id) REFERENCES durable_object_instances(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_do_alarms_time ON durable_object_alarms(scheduled_time);
`

// Helper for JSON serialization
func jsonMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
