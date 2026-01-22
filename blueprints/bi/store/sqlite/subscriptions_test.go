package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestSubscriptionStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid input", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		sub := &store.Subscription{
			DashboardID: d.ID,
			Schedule:    "0 9 * * 1",
			Format:      "pdf",
			Recipients:  []string{"team@example.com"},
			Enabled:     true,
		}
		err := s.Subscriptions().Create(ctx, sub)
		require.NoError(t, err)

		assertIDGenerated(t, sub.ID)
		assert.False(t, sub.CreatedAt.IsZero())
	})

	t.Run("invalid dashboard fails", func(t *testing.T) {
		sub := &store.Subscription{
			DashboardID: "nonexistent",
			Schedule:    "0 9 * * *",
			Format:      "pdf",
			Recipients:  []string{"test@example.com"},
		}
		err := s.Subscriptions().Create(ctx, sub)
		require.Error(t, err)
	})

	t.Run("multiple recipients", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		sub := &store.Subscription{
			DashboardID: d.ID,
			Schedule:    "0 8 * * 1-5",
			Format:      "csv",
			Recipients: []string{
				"user1@example.com",
				"user2@example.com",
				"user3@example.com",
			},
			Enabled: true,
		}
		err := s.Subscriptions().Create(ctx, sub)
		require.NoError(t, err)

		retrieved, _ := s.Subscriptions().GetByID(ctx, sub.ID)
		assert.Len(t, retrieved.Recipients, 3)
	})

	t.Run("different formats", func(t *testing.T) {
		formats := []string{"pdf", "png", "csv"}

		for _, format := range formats {
			d := createTestDashboard(t, s, "")
			sub := &store.Subscription{
				DashboardID: d.ID,
				Schedule:    "0 0 * * *",
				Format:      format,
				Recipients:  []string{"test@example.com"},
			}
			err := s.Subscriptions().Create(ctx, sub)
			require.NoError(t, err, "format %s should work", format)

			retrieved, _ := s.Subscriptions().GetByID(ctx, sub.ID)
			assert.Equal(t, format, retrieved.Format)
		}
	})
}

func TestSubscriptionStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists with recipients", func(t *testing.T) {
		d := createTestDashboard(t, s, "")
		sub := createTestSubscription(t, s, d.ID)

		retrieved, err := s.Subscriptions().GetByID(ctx, sub.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, sub.ID, retrieved.ID)
		assert.Equal(t, d.ID, retrieved.DashboardID)
		assert.Len(t, retrieved.Recipients, 1)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Subscriptions().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestSubscriptionStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all subscriptions", func(t *testing.T) {
		d := createTestDashboard(t, s, "")
		createTestSubscription(t, s, d.ID)
		createTestSubscription(t, s, d.ID)

		subs, err := s.Subscriptions().List(ctx)
		require.NoError(t, err)
		assert.Len(t, subs, 2)
	})
}

func TestSubscriptionStore_ListByDashboard(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("filters by dashboard", func(t *testing.T) {
		d1 := createTestDashboard(t, s, "")
		d2 := createTestDashboard(t, s, "")

		createTestSubscription(t, s, d1.ID)
		createTestSubscription(t, s, d1.ID)
		createTestSubscription(t, s, d2.ID)

		subs1, err := s.Subscriptions().ListByDashboard(ctx, d1.ID)
		require.NoError(t, err)
		assert.Len(t, subs1, 2)

		subs2, err := s.Subscriptions().ListByDashboard(ctx, d2.ID)
		require.NoError(t, err)
		assert.Len(t, subs2, 1)
	})
}

func TestSubscriptionStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update schedule", func(t *testing.T) {
		d := createTestDashboard(t, s, "")
		sub := createTestSubscription(t, s, d.ID)

		sub.Schedule = "0 0 1 * *" // Monthly
		err := s.Subscriptions().Update(ctx, sub)
		require.NoError(t, err)

		retrieved, _ := s.Subscriptions().GetByID(ctx, sub.ID)
		assert.Equal(t, "0 0 1 * *", retrieved.Schedule)
	})

	t.Run("update recipients", func(t *testing.T) {
		d := createTestDashboard(t, s, "")
		sub := createTestSubscription(t, s, d.ID)

		sub.Recipients = []string{"new@example.com", "another@example.com"}
		err := s.Subscriptions().Update(ctx, sub)
		require.NoError(t, err)

		retrieved, _ := s.Subscriptions().GetByID(ctx, sub.ID)
		assert.Len(t, retrieved.Recipients, 2)
		assert.Contains(t, retrieved.Recipients, "new@example.com")
	})

	t.Run("toggle enabled", func(t *testing.T) {
		d := createTestDashboard(t, s, "")
		sub := createTestSubscription(t, s, d.ID)
		assert.True(t, sub.Enabled)

		sub.Enabled = false
		err := s.Subscriptions().Update(ctx, sub)
		require.NoError(t, err)

		retrieved, _ := s.Subscriptions().GetByID(ctx, sub.ID)
		assert.False(t, retrieved.Enabled)
	})

	t.Run("update format", func(t *testing.T) {
		d := createTestDashboard(t, s, "")
		sub := createTestSubscription(t, s, d.ID)

		sub.Format = "png"
		err := s.Subscriptions().Update(ctx, sub)
		require.NoError(t, err)

		retrieved, _ := s.Subscriptions().GetByID(ctx, sub.ID)
		assert.Equal(t, "png", retrieved.Format)
	})
}

func TestSubscriptionStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes subscription", func(t *testing.T) {
		d := createTestDashboard(t, s, "")
		sub := createTestSubscription(t, s, d.ID)

		err := s.Subscriptions().Delete(ctx, sub.ID)
		require.NoError(t, err)

		retrieved, _ := s.Subscriptions().GetByID(ctx, sub.ID)
		assert.Nil(t, retrieved)
	})
}
