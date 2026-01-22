package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestMetricStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid input", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		m := &store.Metric{
			Name:        "Total Revenue",
			Description: "Sum of all order totals",
			TableID:     tbl.ID,
			Definition: map[string]any{
				"aggregation": "sum",
				"column":      "total",
			},
		}
		err := s.Metrics().Create(ctx, m)
		require.NoError(t, err)

		assertIDGenerated(t, m.ID)
		assertTimestampsSet(t, m.CreatedAt, m.UpdatedAt)
	})

	t.Run("invalid table fails", func(t *testing.T) {
		m := &store.Metric{
			Name:       "Bad Metric",
			TableID:    "nonexistent",
			Definition: map[string]any{"aggregation": "count"},
		}
		err := s.Metrics().Create(ctx, m)
		require.Error(t, err)
	})

	t.Run("complex definition", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		m := &store.Metric{
			Name:    "Average Order Value",
			TableID: tbl.ID,
			Definition: map[string]any{
				"aggregation": "avg",
				"column":      "total",
				"filters": []any{
					map[string]any{
						"column":   "status",
						"operator": "=",
						"value":    "completed",
					},
				},
			},
		}
		err := s.Metrics().Create(ctx, m)
		require.NoError(t, err)

		// Verify JSON round-trip
		retrieved, _ := s.Metrics().GetByID(ctx, m.ID)
		assert.Equal(t, "avg", retrieved.Definition["aggregation"])
	})
}

func TestMetricStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists with definition", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)
		m := createTestMetric(t, s, tbl.ID)

		retrieved, err := s.Metrics().GetByID(ctx, m.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, m.ID, retrieved.ID)
		assert.Equal(t, "sum", retrieved.Definition["aggregation"])
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Metrics().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestMetricStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all metrics", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)
		createTestMetric(t, s, tbl.ID)
		createTestMetric(t, s, tbl.ID)

		metrics, err := s.Metrics().List(ctx)
		require.NoError(t, err)
		assert.Len(t, metrics, 2)
	})
}

func TestMetricStore_ListByTable(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("filters by table", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl1 := createTestTable(t, s, ds.ID)
		tbl2 := createTestTable(t, s, ds.ID)

		createTestMetric(t, s, tbl1.ID)
		createTestMetric(t, s, tbl1.ID)
		createTestMetric(t, s, tbl2.ID)

		metrics1, err := s.Metrics().ListByTable(ctx, tbl1.ID)
		require.NoError(t, err)
		assert.Len(t, metrics1, 2)

		metrics2, err := s.Metrics().ListByTable(ctx, tbl2.ID)
		require.NoError(t, err)
		assert.Len(t, metrics2, 1)
	})
}

func TestMetricStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update definition", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)
		m := createTestMetric(t, s, tbl.ID)

		m.Definition = map[string]any{
			"aggregation": "count",
			"distinct":    true,
		}
		err := s.Metrics().Update(ctx, m)
		require.NoError(t, err)

		retrieved, _ := s.Metrics().GetByID(ctx, m.ID)
		assert.Equal(t, "count", retrieved.Definition["aggregation"])
		assert.Equal(t, true, retrieved.Definition["distinct"])
	})

	t.Run("update name and description", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)
		m := createTestMetric(t, s, tbl.ID)

		m.Name = "Updated Metric"
		m.Description = "Updated description"
		err := s.Metrics().Update(ctx, m)
		require.NoError(t, err)

		retrieved, _ := s.Metrics().GetByID(ctx, m.ID)
		assert.Equal(t, "Updated Metric", retrieved.Name)
		assert.Equal(t, "Updated description", retrieved.Description)
	})
}

func TestMetricStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes metric", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)
		m := createTestMetric(t, s, tbl.ID)

		err := s.Metrics().Delete(ctx, m.ID)
		require.NoError(t, err)

		retrieved, _ := s.Metrics().GetByID(ctx, m.ID)
		assert.Nil(t, retrieved)
	})
}
