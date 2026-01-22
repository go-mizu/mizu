package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestQueryHistoryStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid entry", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		qh := &store.QueryHistory{
			UserID:       "user123",
			DataSourceID: ds.ID,
			Query:        "SELECT * FROM orders LIMIT 100",
			Duration:     45.5,
			RowCount:     100,
		}
		err := s.QueryHistory().Create(ctx, qh)
		require.NoError(t, err)

		assertIDGenerated(t, qh.ID)
		assert.False(t, qh.CreatedAt.IsZero())
	})

	t.Run("with error", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		qh := &store.QueryHistory{
			UserID:       "user123",
			DataSourceID: ds.ID,
			Query:        "SELECT * FROM nonexistent",
			Duration:     10.2,
			RowCount:     0,
			Error:        "table nonexistent does not exist",
		}
		err := s.QueryHistory().Create(ctx, qh)
		require.NoError(t, err)
		assert.NotEmpty(t, qh.Error)
	})

	t.Run("complex query", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		u := createTestUser(t, s)

		qh := &store.QueryHistory{
			UserID:       u.ID,
			DataSourceID: ds.ID,
			Query: `
				SELECT
					c.name as customer,
					COUNT(o.id) as order_count,
					SUM(o.total) as total_revenue
				FROM customers c
				JOIN orders o ON c.id = o.customer_id
				WHERE o.created_at > '2024-01-01'
				GROUP BY c.id
				ORDER BY total_revenue DESC
				LIMIT 100
			`,
			Duration: 234.5,
			RowCount: 50,
		}
		err := s.QueryHistory().Create(ctx, qh)
		require.NoError(t, err)
	})
}

func TestQueryHistoryStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("filters by user", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		u1 := createTestUser(t, s)
		u2 := createTestUser(t, s)

		createTestQueryHistory(t, s, u1.ID, ds.ID)
		createTestQueryHistory(t, s, u1.ID, ds.ID)
		createTestQueryHistory(t, s, u2.ID, ds.ID)

		history1, err := s.QueryHistory().List(ctx, u1.ID, 50)
		require.NoError(t, err)
		assert.Len(t, history1, 2)

		history2, err := s.QueryHistory().List(ctx, u2.ID, 50)
		require.NoError(t, err)
		assert.Len(t, history2, 1)
	})

	t.Run("ordered by created_at DESC", func(t *testing.T) {
		s := testStore(t) // Fresh store
		ds := createTestDataSource(t, s)

		for i := 0; i < 3; i++ {
			createTestQueryHistory(t, s, "user123", ds.ID)
		}

		history, err := s.QueryHistory().List(ctx, "user123", 50)
		require.NoError(t, err)
		require.Len(t, history, 3)

		// Most recent should be first
		for i := 0; i < len(history)-1; i++ {
			assert.True(t, history[i].CreatedAt.After(history[i+1].CreatedAt) ||
				history[i].CreatedAt.Equal(history[i+1].CreatedAt))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		s := testStore(t) // Fresh store
		ds := createTestDataSource(t, s)

		for i := 0; i < 10; i++ {
			createTestQueryHistory(t, s, "user123", ds.ID)
		}

		history, err := s.QueryHistory().List(ctx, "user123", 5)
		require.NoError(t, err)
		assert.Len(t, history, 5)
	})

	t.Run("empty for user with no history", func(t *testing.T) {
		history, err := s.QueryHistory().List(ctx, "nobody", 50)
		require.NoError(t, err)
		assert.Empty(t, history)
	})
}
