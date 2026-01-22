package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestAlertStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("goal type", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")

		a := &store.Alert{
			Name:       "Revenue Alert",
			QuestionID: q.ID,
			AlertType:  "goal",
			Condition: store.AlertCondition{
				Operator: "below",
				Value:    10000,
			},
			Channels: []store.AlertChannel{
				{Type: "email", Targets: []string{"admin@example.com"}},
			},
			Enabled: true,
		}
		err := s.Alerts().Create(ctx, a)
		require.NoError(t, err)

		assertIDGenerated(t, a.ID)
		assert.False(t, a.CreatedAt.IsZero())
	})

	t.Run("rows type", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")

		a := &store.Alert{
			Name:       "Rows Alert",
			QuestionID: q.ID,
			AlertType:  "rows",
			Condition: store.AlertCondition{
				Operator: "above",
				Value:    0,
			},
			Channels: []store.AlertChannel{
				{Type: "slack", Targets: []string{"#alerts"}},
			},
			Enabled: true,
		}
		err := s.Alerts().Create(ctx, a)
		require.NoError(t, err)
		assert.Equal(t, "rows", a.AlertType)
	})

	t.Run("invalid question fails", func(t *testing.T) {
		a := &store.Alert{
			Name:       "Bad Alert",
			QuestionID: "nonexistent",
			AlertType:  "goal",
			Condition:  store.AlertCondition{Operator: "below", Value: 0},
			Channels:   []store.AlertChannel{{Type: "email", Targets: []string{"test@example.com"}}},
		}
		err := s.Alerts().Create(ctx, a)
		require.Error(t, err)
	})

	t.Run("multiple channels", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")

		a := &store.Alert{
			Name:       "Multi-channel Alert",
			QuestionID: q.ID,
			AlertType:  "goal",
			Condition:  store.AlertCondition{Operator: "above", Value: 1000},
			Channels: []store.AlertChannel{
				{Type: "email", Targets: []string{"admin@example.com", "manager@example.com"}},
				{Type: "slack", Targets: []string{"#general", "#alerts"}},
			},
			Enabled: true,
		}
		err := s.Alerts().Create(ctx, a)
		require.NoError(t, err)

		retrieved, _ := s.Alerts().GetByID(ctx, a.ID)
		assert.Len(t, retrieved.Channels, 2)
	})
}

func TestAlertStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists with condition", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		a := createTestAlert(t, s, q.ID)

		retrieved, err := s.Alerts().GetByID(ctx, a.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, a.ID, retrieved.ID)
		assert.Equal(t, "below", retrieved.Condition.Operator)
		assert.Equal(t, float64(10000), retrieved.Condition.Value)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Alerts().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestAlertStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all alerts", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		createTestAlert(t, s, q.ID)
		createTestAlert(t, s, q.ID)

		alerts, err := s.Alerts().List(ctx)
		require.NoError(t, err)
		assert.Len(t, alerts, 2)
	})
}

func TestAlertStore_ListByQuestion(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("filters by question", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q1 := createTestQuestion(t, s, ds.ID, "")
		q2 := createTestQuestion(t, s, ds.ID, "")

		createTestAlert(t, s, q1.ID)
		createTestAlert(t, s, q1.ID)
		createTestAlert(t, s, q2.ID)

		alerts1, err := s.Alerts().ListByQuestion(ctx, q1.ID)
		require.NoError(t, err)
		assert.Len(t, alerts1, 2)

		alerts2, err := s.Alerts().ListByQuestion(ctx, q2.ID)
		require.NoError(t, err)
		assert.Len(t, alerts2, 1)
	})
}

func TestAlertStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update condition", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		a := createTestAlert(t, s, q.ID)

		a.Condition = store.AlertCondition{
			Operator: "above",
			Value:    50000,
		}
		err := s.Alerts().Update(ctx, a)
		require.NoError(t, err)

		retrieved, _ := s.Alerts().GetByID(ctx, a.ID)
		assert.Equal(t, "above", retrieved.Condition.Operator)
		assert.Equal(t, float64(50000), retrieved.Condition.Value)
	})

	t.Run("update channels", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		a := createTestAlert(t, s, q.ID)

		a.Channels = []store.AlertChannel{
			{Type: "slack", Targets: []string{"#new-channel"}},
		}
		err := s.Alerts().Update(ctx, a)
		require.NoError(t, err)

		retrieved, _ := s.Alerts().GetByID(ctx, a.ID)
		require.Len(t, retrieved.Channels, 1)
		assert.Equal(t, "slack", retrieved.Channels[0].Type)
	})

	t.Run("toggle enabled", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		a := createTestAlert(t, s, q.ID)
		assert.True(t, a.Enabled)

		a.Enabled = false
		err := s.Alerts().Update(ctx, a)
		require.NoError(t, err)

		retrieved, _ := s.Alerts().GetByID(ctx, a.ID)
		assert.False(t, retrieved.Enabled)
	})
}

func TestAlertStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes alert", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		a := createTestAlert(t, s, q.ID)

		err := s.Alerts().Delete(ctx, a.ID)
		require.NoError(t, err)

		retrieved, _ := s.Alerts().GetByID(ctx, a.ID)
		assert.Nil(t, retrieved)
	})
}
