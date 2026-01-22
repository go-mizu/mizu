package sqlite

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestDataSourceStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid input", func(t *testing.T) {
		ds := &store.DataSource{
			Name:     "Test Database",
			Engine:   "sqlite",
			Database: "test.db",
		}
		err := s.DataSources().Create(ctx, ds)
		require.NoError(t, err)

		assertIDGenerated(t, ds.ID)
		assertTimestampsSet(t, ds.CreatedAt, ds.UpdatedAt)
	})

	t.Run("minimal fields", func(t *testing.T) {
		ds := &store.DataSource{
			Name:     "Minimal DB",
			Engine:   "postgres",
			Database: "db",
		}
		err := s.DataSources().Create(ctx, ds)
		require.NoError(t, err)
		assert.NotEmpty(t, ds.ID)
	})

	t.Run("with all optional fields", func(t *testing.T) {
		ds := &store.DataSource{
			Name:     "Full Database",
			Engine:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "mydb",
			Username: "admin",
			Password: "secret",
			SSL:      true,
			Options:  map[string]string{"sslmode": "require"},
		}
		err := s.DataSources().Create(ctx, ds)
		require.NoError(t, err)

		// Retrieve and verify all fields
		retrieved, err := s.DataSources().GetByID(ctx, ds.ID)
		require.NoError(t, err)
		assert.Equal(t, "localhost", retrieved.Host)
		assert.Equal(t, 5432, retrieved.Port)
		assert.Equal(t, "admin", retrieved.Username)
		assert.Equal(t, "secret", retrieved.Password)
		assert.True(t, retrieved.SSL)
		assert.Equal(t, "require", retrieved.Options["sslmode"])
	})
}

func TestDataSourceStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		retrieved, err := s.DataSources().GetByID(ctx, ds.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, ds.ID, retrieved.ID)
		assert.Equal(t, ds.Name, retrieved.Name)
		assert.Equal(t, ds.Engine, retrieved.Engine)
	})

	t.Run("not found returns nil without error", func(t *testing.T) {
		retrieved, err := s.DataSources().GetByID(ctx, "nonexistent")
		require.NoError(t, err) // Should NOT return error
		assert.Nil(t, retrieved) // Should return nil
	})
}

func TestDataSourceStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("empty when no datasources", func(t *testing.T) {
		list, err := s.DataSources().List(ctx)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("returns all datasources", func(t *testing.T) {
		createTestDataSource(t, s)
		createTestDataSource(t, s)
		createTestDataSource(t, s)

		list, err := s.DataSources().List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 3)
	})

	t.Run("ordered by name", func(t *testing.T) {
		s := testStore(t) // Fresh store

		// Create in reverse order
		ds1 := &store.DataSource{Name: "Zebra DB", Engine: "sqlite", Database: "z.db"}
		ds2 := &store.DataSource{Name: "Alpha DB", Engine: "sqlite", Database: "a.db"}
		ds3 := &store.DataSource{Name: "Middle DB", Engine: "sqlite", Database: "m.db"}

		s.DataSources().Create(ctx, ds1)
		s.DataSources().Create(ctx, ds2)
		s.DataSources().Create(ctx, ds3)

		list, err := s.DataSources().List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 3)

		assert.Equal(t, "Alpha DB", list[0].Name)
		assert.Equal(t, "Middle DB", list[1].Name)
		assert.Equal(t, "Zebra DB", list[2].Name)
	})
}

func TestDataSourceStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("updates all fields", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		originalUpdatedAt := ds.UpdatedAt

		ds.Name = "Updated Name"
		ds.Host = "newhost"
		ds.Port = 3306
		ds.Username = "newuser"

		err := s.DataSources().Update(ctx, ds)
		require.NoError(t, err)

		retrieved, err := s.DataSources().GetByID(ctx, ds.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, "newhost", retrieved.Host)
		assert.Equal(t, 3306, retrieved.Port)
		assert.Equal(t, "newuser", retrieved.Username)
		assert.True(t, retrieved.UpdatedAt.After(originalUpdatedAt))
	})

	t.Run("updates partial fields", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		originalName := ds.Name

		ds.Host = "partial-host"

		err := s.DataSources().Update(ctx, ds)
		require.NoError(t, err)

		retrieved, err := s.DataSources().GetByID(ctx, ds.ID)
		require.NoError(t, err)
		assert.Equal(t, originalName, retrieved.Name) // Unchanged
		assert.Equal(t, "partial-host", retrieved.Host)
	})
}

func TestDataSourceStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes existing datasource", func(t *testing.T) {
		ds := createTestDataSource(t, s)

		err := s.DataSources().Delete(ctx, ds.ID)
		require.NoError(t, err)

		retrieved, err := s.DataSources().GetByID(ctx, ds.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("no error when deleting non-existent", func(t *testing.T) {
		err := s.DataSources().Delete(ctx, "nonexistent")
		require.NoError(t, err)
	})

	t.Run("cascades to tables", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		tbl := createTestTable(t, s, ds.ID)

		// Verify table exists
		tables, err := s.Tables().ListByDataSource(ctx, ds.ID)
		require.NoError(t, err)
		require.Len(t, tables, 1)
		assert.Equal(t, tbl.ID, tables[0].ID)

		// Delete datasource
		err = s.DataSources().Delete(ctx, ds.ID)
		require.NoError(t, err)

		// Verify table is gone
		tables, err = s.Tables().ListByDataSource(ctx, ds.ID)
		require.NoError(t, err)
		assert.Empty(t, tables)
	})
}

func TestDataSourceStore_Concurrent(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("concurrent creates", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				ds := &store.DataSource{
					Name:     "Concurrent DB",
					Engine:   "sqlite",
					Database: "concurrent.db",
				}
				if err := s.DataSources().Create(ctx, ds); err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("concurrent create failed: %v", err)
		}

		list, err := s.DataSources().List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 10)
	})
}
