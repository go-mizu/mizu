package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestTableStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid input", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		tbl := &store.Table{
			DataSourceID: ds.ID,
			Schema:       "public",
			Name:         "users",
			DisplayName:  "Users",
			Description:  "User accounts table",
		}
		err := s.Tables().Create(ctx, tbl)
		require.NoError(t, err)

		assertIDGenerated(t, tbl.ID)
		assertTimestampsSet(t, tbl.CreatedAt, tbl.UpdatedAt)
	})

	t.Run("invalid foreign key fails", func(t *testing.T) {
		tbl := &store.Table{
			DataSourceID: "nonexistent",
			Name:         "test",
		}
		err := s.Tables().Create(ctx, tbl)
		require.Error(t, err) // Should fail with FK constraint
	})

	t.Run("multiple tables for same datasource", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		tbl1 := &store.Table{DataSourceID: ds.ID, Name: "table1"}
		tbl2 := &store.Table{DataSourceID: ds.ID, Name: "table2"}

		require.NoError(t, s.Tables().Create(ctx, tbl1))
		require.NoError(t, s.Tables().Create(ctx, tbl2))

		tables, err := s.Tables().ListByDataSource(ctx, ds.ID)
		require.NoError(t, err)
		assert.Len(t, tables, 2)
	})
}

func TestTableStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		retrieved, err := s.Tables().GetByID(ctx, tbl.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, tbl.ID, retrieved.ID)
		assert.Equal(t, tbl.Name, retrieved.Name)
		assert.Equal(t, ds.ID, retrieved.DataSourceID)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Tables().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestTableStore_ListByDataSource(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("empty when no tables", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		tables, err := s.Tables().ListByDataSource(ctx, ds.ID)
		require.NoError(t, err)
		assert.Empty(t, tables)
	})

	t.Run("returns only tables for specified datasource", func(t *testing.T) {
		ds1 := createTestDataSource(t, s)
		ds2 := createTestDataSource(t, s)

		createTestTable(t, s, ds1.ID)
		createTestTable(t, s, ds1.ID)
		createTestTable(t, s, ds2.ID)

		tables1, err := s.Tables().ListByDataSource(ctx, ds1.ID)
		require.NoError(t, err)
		assert.Len(t, tables1, 2)

		tables2, err := s.Tables().ListByDataSource(ctx, ds2.ID)
		require.NoError(t, err)
		assert.Len(t, tables2, 1)
	})
}

func TestTableStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("updates fields", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		tbl.DisplayName = "Updated Display Name"
		tbl.Description = "Updated description"
		tbl.RowCount = 1000

		err := s.Tables().Update(ctx, tbl)
		require.NoError(t, err)

		retrieved, err := s.Tables().GetByID(ctx, tbl.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Display Name", retrieved.DisplayName)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, int64(1000), retrieved.RowCount)
	})
}

func TestTableStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes table", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		err := s.Tables().Delete(ctx, tbl.ID)
		require.NoError(t, err)

		retrieved, err := s.Tables().GetByID(ctx, tbl.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("cascades to columns", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)
		createTestColumn(t, s, tbl.ID, 0)
		createTestColumn(t, s, tbl.ID, 1)

		// Verify columns exist
		cols, err := s.Tables().ListColumns(ctx, tbl.ID)
		require.NoError(t, err)
		assert.Len(t, cols, 2)

		// Delete table
		err = s.Tables().Delete(ctx, tbl.ID)
		require.NoError(t, err)

		// Verify columns are gone
		cols, err = s.Tables().ListColumns(ctx, tbl.ID)
		require.NoError(t, err)
		assert.Empty(t, cols)
	})
}

// Column tests

func TestColumnStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid input", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		col := &store.Column{
			TableID:     tbl.ID,
			Name:        "email",
			DisplayName: "Email Address",
			Type:        "string",
			Semantic:    "email",
			Description: "User email",
			Position:    0,
		}
		err := s.Tables().CreateColumn(ctx, col)
		require.NoError(t, err)
		assertIDGenerated(t, col.ID)
	})

	t.Run("invalid table FK fails", func(t *testing.T) {
		col := &store.Column{
			TableID: "nonexistent",
			Name:    "test",
			Type:    "string",
		}
		err := s.Tables().CreateColumn(ctx, col)
		require.Error(t, err)
	})
}

func TestColumnStore_ListColumns(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("ordered by position", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		// Create in reverse order
		createTestColumn(t, s, tbl.ID, 2)
		createTestColumn(t, s, tbl.ID, 0)
		createTestColumn(t, s, tbl.ID, 1)

		cols, err := s.Tables().ListColumns(ctx, tbl.ID)
		require.NoError(t, err)
		require.Len(t, cols, 3)

		assert.Equal(t, 0, cols[0].Position)
		assert.Equal(t, 1, cols[1].Position)
		assert.Equal(t, 2, cols[2].Position)
	})

	t.Run("empty when no columns", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		cols, err := s.Tables().ListColumns(ctx, tbl.ID)
		require.NoError(t, err)
		assert.Empty(t, cols)
	})
}

func TestColumnStore_DeleteByTable(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes all columns for table", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl1 := createTestTable(t, s, ds.ID)
		tbl2 := createTestTable(t, s, ds.ID)

		// Create columns for both tables
		createTestColumn(t, s, tbl1.ID, 0)
		createTestColumn(t, s, tbl1.ID, 1)
		createTestColumn(t, s, tbl2.ID, 0)

		// Delete columns for tbl1 only
		err := s.Tables().DeleteColumnsByTable(ctx, tbl1.ID)
		require.NoError(t, err)

		// Verify tbl1 columns are gone
		cols1, _ := s.Tables().ListColumns(ctx, tbl1.ID)
		assert.Empty(t, cols1)

		// Verify tbl2 columns remain
		cols2, _ := s.Tables().ListColumns(ctx, tbl2.ID)
		assert.Len(t, cols2, 1)
	})
}
