package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestQuestionStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("query type", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		q := &store.Question{
			Name:         "Sales by Region",
			DataSourceID: ds.ID,
			QueryType:    "query",
			Query: map[string]any{
				"table":   "sales",
				"columns": []string{"region", "total"},
				"filters": []map[string]any{
					{"column": "year", "operator": "=", "value": 2024},
				},
			},
		}
		err := s.Questions().Create(ctx, q)
		require.NoError(t, err)

		assertIDGenerated(t, q.ID)
		assertTimestampsSet(t, q.CreatedAt, q.UpdatedAt)
	})

	t.Run("native type", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		q := &store.Question{
			Name:         "Complex Query",
			DataSourceID: ds.ID,
			QueryType:    "native",
			Query: map[string]any{
				"sql": "SELECT * FROM orders WHERE total > 1000",
			},
		}
		err := s.Questions().Create(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, "native", q.QueryType)
	})

	t.Run("with collection", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		coll := createTestCollection(t, s, "")

		q := &store.Question{
			Name:         "Collection Question",
			DataSourceID: ds.ID,
			CollectionID: coll.ID,
			QueryType:    "query",
			Query:        map[string]any{"table": "test"},
		}
		err := s.Questions().Create(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, coll.ID, q.CollectionID)
	})

	t.Run("without collection", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		q := &store.Question{
			Name:         "Root Question",
			DataSourceID: ds.ID,
			QueryType:    "query",
			Query:        map[string]any{"table": "test"},
		}
		err := s.Questions().Create(ctx, q)
		require.NoError(t, err)
		assert.Empty(t, q.CollectionID)
	})

	t.Run("invalid datasource fails", func(t *testing.T) {
		q := &store.Question{
			Name:         "Bad Question",
			DataSourceID: "nonexistent",
			QueryType:    "query",
			Query:        map[string]any{"table": "test"},
		}
		err := s.Questions().Create(ctx, q)
		require.Error(t, err)
	})

	t.Run("with visualization", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		q := &store.Question{
			Name:         "Chart Question",
			DataSourceID: ds.ID,
			QueryType:    "query",
			Query:        map[string]any{"table": "sales"},
			Visualization: map[string]any{
				"type": "line",
				"settings": map[string]any{
					"x_axis":   "date",
					"y_axis":   []string{"revenue"},
					"stacking": "none",
				},
			},
		}
		err := s.Questions().Create(ctx, q)
		require.NoError(t, err)

		// Retrieve and verify JSON round-trip
		retrieved, err := s.Questions().GetByID(ctx, q.ID)
		require.NoError(t, err)
		assert.Equal(t, "line", retrieved.Visualization["type"])
	})
}

func TestQuestionStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists with full query", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")

		retrieved, err := s.Questions().GetByID(ctx, q.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, q.ID, retrieved.ID)
		assert.Equal(t, q.Name, retrieved.Name)
		assert.Equal(t, q.Query, retrieved.Query)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Questions().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestQuestionStore_List(t *testing.T) {
	ctx := testContext()

	t.Run("empty when no questions", func(t *testing.T) {
		s := testStore(t) // Fresh store
		questions, err := s.Questions().List(ctx)
		require.NoError(t, err)
		assert.Empty(t, questions)
	})

	t.Run("returns all questions", func(t *testing.T) {
		s := testStore(t) // Fresh store
		ds := createTestDataSource(t, s)
		createTestQuestion(t, s, ds.ID, "")
		createTestQuestion(t, s, ds.ID, "")

		questions, err := s.Questions().List(ctx)
		require.NoError(t, err)
		assert.Len(t, questions, 2)
	})
}

func TestQuestionStore_ListByCollection(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("filters by collection", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		coll1 := createTestCollection(t, s, "")
		coll2 := createTestCollection(t, s, "")

		createTestQuestion(t, s, ds.ID, coll1.ID)
		createTestQuestion(t, s, ds.ID, coll1.ID)
		createTestQuestion(t, s, ds.ID, coll2.ID)

		questions1, err := s.Questions().ListByCollection(ctx, coll1.ID)
		require.NoError(t, err)
		assert.Len(t, questions1, 2)

		questions2, err := s.Questions().ListByCollection(ctx, coll2.ID)
		require.NoError(t, err)
		assert.Len(t, questions2, 1)
	})

	t.Run("empty for collection with no questions", func(t *testing.T) {
		coll := createTestCollection(t, s, "")

		questions, err := s.Questions().ListByCollection(ctx, coll.ID)
		require.NoError(t, err)
		assert.Empty(t, questions)
	})
}

func TestQuestionStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update query", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")

		q.Query = map[string]any{
			"table":   "new_table",
			"columns": []string{"col1", "col2"},
		}
		err := s.Questions().Update(ctx, q)
		require.NoError(t, err)

		retrieved, err := s.Questions().GetByID(ctx, q.ID)
		require.NoError(t, err)
		assert.Equal(t, "new_table", retrieved.Query["table"])
	})

	t.Run("update visualization", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")

		q.Visualization = map[string]any{
			"type": "bar",
			"settings": map[string]any{
				"stacking": "normal",
			},
		}
		err := s.Questions().Update(ctx, q)
		require.NoError(t, err)

		retrieved, err := s.Questions().GetByID(ctx, q.ID)
		require.NoError(t, err)
		assert.Equal(t, "bar", retrieved.Visualization["type"])
	})

	t.Run("move to different collection", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		coll1 := createTestCollection(t, s, "")
		coll2 := createTestCollection(t, s, "")
		q := createTestQuestion(t, s, ds.ID, coll1.ID)

		q.CollectionID = coll2.ID
		err := s.Questions().Update(ctx, q)
		require.NoError(t, err)

		retrieved, err := s.Questions().GetByID(ctx, q.ID)
		require.NoError(t, err)
		assert.Equal(t, coll2.ID, retrieved.CollectionID)
	})
}

func TestQuestionStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes question", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")

		err := s.Questions().Delete(ctx, q.ID)
		require.NoError(t, err)

		retrieved, err := s.Questions().GetByID(ctx, q.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("removes from collection list", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		coll := createTestCollection(t, s, "")
		q := createTestQuestion(t, s, ds.ID, coll.ID)

		// Verify in collection
		questions, _ := s.Questions().ListByCollection(ctx, coll.ID)
		require.Len(t, questions, 1)

		// Delete
		err := s.Questions().Delete(ctx, q.ID)
		require.NoError(t, err)

		// Verify removed from collection
		questions, _ = s.Questions().ListByCollection(ctx, coll.ID)
		assert.Empty(t, questions)
	})
}

func TestQuestionStore_JSONRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("complex query and visualization", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		q := &store.Question{
			Name:         "Complex JSON Test",
			DataSourceID: ds.ID,
			QueryType:    "query",
			Query: map[string]any{
				"table":   "orders",
				"columns": []any{"id", "customer_id", "total"},
				"filters": []any{
					map[string]any{
						"column":   "status",
						"operator": "=",
						"value":    "completed",
					},
					map[string]any{
						"column":   "total",
						"operator": ">",
						"value":    float64(100),
					},
				},
				"order_by": []any{
					map[string]any{"column": "created_at", "direction": "DESC"},
				},
				"limit": float64(1000),
			},
			Visualization: map[string]any{
				"type": "line",
				"settings": map[string]any{
					"x_axis": "date",
					"y_axis": []any{"total", "count"},
					"colors": map[string]any{
						"total": "#509EE3",
						"count": "#88BF4D",
					},
				},
			},
		}

		err := s.Questions().Create(ctx, q)
		require.NoError(t, err)

		retrieved, err := s.Questions().GetByID(ctx, q.ID)
		require.NoError(t, err)

		// Verify query structure
		assert.Equal(t, "orders", retrieved.Query["table"])
		filters := retrieved.Query["filters"].([]any)
		assert.Len(t, filters, 2)

		// Verify visualization structure
		assert.Equal(t, "line", retrieved.Visualization["type"])
		settings := retrieved.Visualization["settings"].(map[string]any)
		assert.Equal(t, "date", settings["x_axis"])
	})
}
