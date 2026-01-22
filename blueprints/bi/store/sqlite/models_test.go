package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestModelStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid input", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		m := &store.Model{
			Name:         "Customer Model",
			Description:  "A model for customer data",
			DataSourceID: ds.ID,
			Query: map[string]any{
				"table":   "customers",
				"columns": []string{"id", "name", "email"},
			},
		}
		err := s.Models().Create(ctx, m)
		require.NoError(t, err)

		assertIDGenerated(t, m.ID)
		assertTimestampsSet(t, m.CreatedAt, m.UpdatedAt)
	})

	t.Run("invalid datasource fails", func(t *testing.T) {
		m := &store.Model{
			Name:         "Bad Model",
			DataSourceID: "nonexistent",
			Query:        map[string]any{"table": "test"},
		}
		err := s.Models().Create(ctx, m)
		require.Error(t, err)
	})

	t.Run("with collection", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		coll := createTestCollection(t, s, "")

		m := &store.Model{
			Name:         "Collection Model",
			DataSourceID: ds.ID,
			CollectionID: coll.ID,
			Query:        map[string]any{"table": "test"},
		}
		err := s.Models().Create(ctx, m)
		require.NoError(t, err)
		assert.Equal(t, coll.ID, m.CollectionID)
	})
}

func TestModelStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists with query", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		m := createTestModel(t, s, ds.ID)

		retrieved, err := s.Models().GetByID(ctx, m.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, m.ID, retrieved.ID)
		assert.Equal(t, "customers", retrieved.Query["table"])
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Models().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestModelStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all models", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		createTestModel(t, s, ds.ID)
		createTestModel(t, s, ds.ID)

		models, err := s.Models().List(ctx)
		require.NoError(t, err)
		assert.Len(t, models, 2)
	})
}

func TestModelStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update query", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		m := createTestModel(t, s, ds.ID)

		m.Query = map[string]any{
			"table":   "updated_table",
			"columns": []string{"col1", "col2", "col3"},
		}
		err := s.Models().Update(ctx, m)
		require.NoError(t, err)

		retrieved, _ := s.Models().GetByID(ctx, m.ID)
		assert.Equal(t, "updated_table", retrieved.Query["table"])
	})

	t.Run("update name and description", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		m := createTestModel(t, s, ds.ID)

		m.Name = "Updated Model Name"
		m.Description = "Updated description"
		err := s.Models().Update(ctx, m)
		require.NoError(t, err)

		retrieved, _ := s.Models().GetByID(ctx, m.ID)
		assert.Equal(t, "Updated Model Name", retrieved.Name)
		assert.Equal(t, "Updated description", retrieved.Description)
	})
}

func TestModelStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes model", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		m := createTestModel(t, s, ds.ID)

		err := s.Models().Delete(ctx, m.ID)
		require.NoError(t, err)

		retrieved, _ := s.Models().GetByID(ctx, m.ID)
		assert.Nil(t, retrieved)
	})

	t.Run("cascades to model columns", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		m := createTestModel(t, s, ds.ID)

		// Create model columns
		col := &store.ModelColumn{
			ModelID:     m.ID,
			Name:        "test_col",
			DisplayName: "Test Column",
		}
		s.Models().CreateColumn(ctx, col)

		// Verify column exists
		cols, _ := s.Models().ListColumns(ctx, m.ID)
		require.Len(t, cols, 1)

		// Delete model
		err := s.Models().Delete(ctx, m.ID)
		require.NoError(t, err)

		// Verify columns are gone
		cols, _ = s.Models().ListColumns(ctx, m.ID)
		assert.Empty(t, cols)
	})
}

// Model Column tests

func TestModelColumnStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid column", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		m := createTestModel(t, s, ds.ID)

		col := &store.ModelColumn{
			ModelID:     m.ID,
			Name:        "customer_name",
			DisplayName: "Customer Name",
			Description: "The customer's full name",
			Semantic:    "name",
			Visible:     true,
		}
		err := s.Models().CreateColumn(ctx, col)
		require.NoError(t, err)
		assertIDGenerated(t, col.ID)
	})

	t.Run("invalid model fails", func(t *testing.T) {
		col := &store.ModelColumn{
			ModelID: "nonexistent",
			Name:    "test",
		}
		err := s.Models().CreateColumn(ctx, col)
		require.Error(t, err)
	})
}

func TestModelColumnStore_ListColumns(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all columns for model", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		m := createTestModel(t, s, ds.ID)

		s.Models().CreateColumn(ctx, &store.ModelColumn{ModelID: m.ID, Name: "col1"})
		s.Models().CreateColumn(ctx, &store.ModelColumn{ModelID: m.ID, Name: "col2"})

		cols, err := s.Models().ListColumns(ctx, m.ID)
		require.NoError(t, err)
		assert.Len(t, cols, 2)
	})
}

func TestModelColumnStore_UpdateColumn(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update display name and visible", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		m := createTestModel(t, s, ds.ID)

		col := &store.ModelColumn{
			ModelID: m.ID,
			Name:    "test_col",
			Visible: true,
		}
		s.Models().CreateColumn(ctx, col)

		col.DisplayName = "Updated Display"
		col.Visible = false
		err := s.Models().UpdateColumn(ctx, col)
		require.NoError(t, err)

		cols, _ := s.Models().ListColumns(ctx, m.ID)
		require.Len(t, cols, 1)
		assert.Equal(t, "Updated Display", cols[0].DisplayName)
		assert.False(t, cols[0].Visible)
	})
}
