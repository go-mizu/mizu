package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestSettingsStore_Set(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("set new key", func(t *testing.T) {
		err := s.Settings().Set(ctx, "site_name", "My BI Tool")
		require.NoError(t, err)

		value, err := s.Settings().Get(ctx, "site_name")
		require.NoError(t, err)
		assert.Equal(t, "My BI Tool", value)
	})

	t.Run("update existing key (upsert)", func(t *testing.T) {
		err := s.Settings().Set(ctx, "theme", "light")
		require.NoError(t, err)

		err = s.Settings().Set(ctx, "theme", "dark")
		require.NoError(t, err)

		value, _ := s.Settings().Get(ctx, "theme")
		assert.Equal(t, "dark", value)
	})
}

func TestSettingsStore_Get(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists", func(t *testing.T) {
		s.Settings().Set(ctx, "test_key", "test_value")

		value, err := s.Settings().Get(ctx, "test_key")
		require.NoError(t, err)
		assert.Equal(t, "test_value", value)
	})

	t.Run("not exists returns empty", func(t *testing.T) {
		value, err := s.Settings().Get(ctx, "nonexistent_key")
		require.NoError(t, err)
		assert.Empty(t, value)
	})
}

func TestSettingsStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all settings", func(t *testing.T) {
		s.Settings().Set(ctx, "key1", "value1")
		s.Settings().Set(ctx, "key2", "value2")
		s.Settings().Set(ctx, "key3", "value3")

		settings, err := s.Settings().List(ctx)
		require.NoError(t, err)
		assert.Len(t, settings, 3)
	})

	t.Run("ordered by key", func(t *testing.T) {
		s := testStore(t) // Fresh store

		s.Settings().Set(ctx, "zebra", "z")
		s.Settings().Set(ctx, "alpha", "a")
		s.Settings().Set(ctx, "middle", "m")

		settings, err := s.Settings().List(ctx)
		require.NoError(t, err)
		require.Len(t, settings, 3)

		assert.Equal(t, "alpha", settings[0].Key)
		assert.Equal(t, "middle", settings[1].Key)
		assert.Equal(t, "zebra", settings[2].Key)
	})
}

// Audit Log tests

func TestAuditLogStore_Write(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid log entry", func(t *testing.T) {
		log := &store.AuditLog{
			ActorID:      "user123",
			ActorEmail:   "user@example.com",
			Action:       "question.created",
			ResourceType: "question",
			ResourceID:   "q123",
			IPAddress:    "192.168.1.1",
		}
		err := s.Settings().WriteAuditLog(ctx, log)
		require.NoError(t, err)

		assertIDGenerated(t, log.ID)
		assert.False(t, log.Timestamp.IsZero())
	})

	t.Run("with metadata", func(t *testing.T) {
		log := &store.AuditLog{
			ActorID:      "user456",
			ActorEmail:   "admin@example.com",
			Action:       "datasource.updated",
			ResourceType: "datasource",
			ResourceID:   "ds123",
			Metadata: map[string]string{
				"old_name": "Old DB",
				"new_name": "New DB",
				"field":    "name",
			},
			IPAddress: "10.0.0.1",
		}
		err := s.Settings().WriteAuditLog(ctx, log)
		require.NoError(t, err)

		// Retrieve and verify metadata
		logs, _ := s.Settings().ListAuditLogs(ctx, 10, 0)
		require.NotEmpty(t, logs)
		assert.Equal(t, "Old DB", logs[0].Metadata["old_name"])
	})
}

func TestAuditLogStore_ListAuditLogs(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("ordered by timestamp DESC", func(t *testing.T) {
		// Create logs in order
		for i := 0; i < 3; i++ {
			log := &store.AuditLog{
				ActorID:      "user",
				ActorEmail:   "user@example.com",
				Action:       "test.action",
				ResourceType: "test",
			}
			s.Settings().WriteAuditLog(ctx, log)
		}

		logs, err := s.Settings().ListAuditLogs(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 3)

		// Most recent should be first (DESC order)
		for i := 0; i < len(logs)-1; i++ {
			assert.True(t, logs[i].Timestamp.After(logs[i+1].Timestamp) ||
				logs[i].Timestamp.Equal(logs[i+1].Timestamp))
		}
	})

	t.Run("pagination with limit", func(t *testing.T) {
		s := testStore(t) // Fresh store

		for i := 0; i < 10; i++ {
			log := &store.AuditLog{
				ActorID:      "user",
				ActorEmail:   "user@example.com",
				Action:       "test.action",
				ResourceType: "test",
			}
			s.Settings().WriteAuditLog(ctx, log)
		}

		logs, err := s.Settings().ListAuditLogs(ctx, 5, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 5)
	})

	t.Run("pagination with offset", func(t *testing.T) {
		s := testStore(t) // Fresh store

		for i := 0; i < 10; i++ {
			log := &store.AuditLog{
				ActorID:      "user",
				ActorEmail:   "user@example.com",
				Action:       "test.action",
				ResourceType: "test",
			}
			s.Settings().WriteAuditLog(ctx, log)
		}

		// Get first 5
		logs1, _ := s.Settings().ListAuditLogs(ctx, 5, 0)

		// Get next 5
		logs2, _ := s.Settings().ListAuditLogs(ctx, 5, 5)

		// Should have different IDs
		assert.NotEqual(t, logs1[0].ID, logs2[0].ID)
	})
}
