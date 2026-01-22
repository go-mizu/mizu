package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestCollectionStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("root level collection", func(t *testing.T) {
		c := &store.Collection{
			Name:  "Root Collection",
			Color: "#509EE3",
		}
		err := s.Collections().Create(ctx, c)
		require.NoError(t, err)

		assertIDGenerated(t, c.ID)
		assert.False(t, c.CreatedAt.IsZero())
		assert.Empty(t, c.ParentID)
	})

	t.Run("nested collection", func(t *testing.T) {
		parent := createTestCollection(t, s, "")

		child := &store.Collection{
			Name:     "Child Collection",
			ParentID: parent.ID,
		}
		err := s.Collections().Create(ctx, child)
		require.NoError(t, err)
		assert.Equal(t, parent.ID, child.ParentID)
	})

	t.Run("with all fields", func(t *testing.T) {
		c := &store.Collection{
			Name:        "Full Collection",
			Description: "A collection with all fields",
			Color:       "#88BF4D",
			CreatedBy:   "user123",
		}
		err := s.Collections().Create(ctx, c)
		require.NoError(t, err)

		retrieved, _ := s.Collections().GetByID(ctx, c.ID)
		assert.Equal(t, "Full Collection", retrieved.Name)
		assert.Equal(t, "A collection with all fields", retrieved.Description)
		assert.Equal(t, "#88BF4D", retrieved.Color)
	})
}

func TestCollectionStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists", func(t *testing.T) {
		c := createTestCollection(t, s, "")

		retrieved, err := s.Collections().GetByID(ctx, c.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, c.ID, retrieved.ID)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Collections().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestCollectionStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all collections", func(t *testing.T) {
		createTestCollection(t, s, "")
		createTestCollection(t, s, "")

		collections, err := s.Collections().List(ctx)
		require.NoError(t, err)
		assert.Len(t, collections, 2)
	})
}

func TestCollectionStore_ListByParent(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("root level collections", func(t *testing.T) {
		createTestCollection(t, s, "")
		createTestCollection(t, s, "")
		parent := createTestCollection(t, s, "")
		createTestCollection(t, s, parent.ID) // Child

		// List root-level (empty parent)
		roots, err := s.Collections().ListByParent(ctx, "")
		require.NoError(t, err)
		assert.Len(t, roots, 3)
	})

	t.Run("children of specific parent", func(t *testing.T) {
		parent := createTestCollection(t, s, "")
		createTestCollection(t, s, parent.ID)
		createTestCollection(t, s, parent.ID)

		children, err := s.Collections().ListByParent(ctx, parent.ID)
		require.NoError(t, err)
		assert.Len(t, children, 2)
	})
}

func TestCollectionStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update name and description", func(t *testing.T) {
		c := createTestCollection(t, s, "")

		c.Name = "Updated Name"
		c.Description = "Updated description"
		err := s.Collections().Update(ctx, c)
		require.NoError(t, err)

		retrieved, _ := s.Collections().GetByID(ctx, c.ID)
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, "Updated description", retrieved.Description)
	})

	t.Run("move to different parent", func(t *testing.T) {
		parent1 := createTestCollection(t, s, "")
		parent2 := createTestCollection(t, s, "")
		child := createTestCollection(t, s, parent1.ID)

		child.ParentID = parent2.ID
		err := s.Collections().Update(ctx, child)
		require.NoError(t, err)

		retrieved, _ := s.Collections().GetByID(ctx, child.ID)
		assert.Equal(t, parent2.ID, retrieved.ParentID)
	})
}

func TestCollectionStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes leaf collection", func(t *testing.T) {
		c := createTestCollection(t, s, "")

		err := s.Collections().Delete(ctx, c.ID)
		require.NoError(t, err)

		retrieved, _ := s.Collections().GetByID(ctx, c.ID)
		assert.Nil(t, retrieved)
	})

	t.Run("cascades to children", func(t *testing.T) {
		parent := createTestCollection(t, s, "")
		child := createTestCollection(t, s, parent.ID)
		grandchild := createTestCollection(t, s, child.ID)

		// Delete parent
		err := s.Collections().Delete(ctx, parent.ID)
		require.NoError(t, err)

		// Verify all descendants are gone
		retrieved, _ := s.Collections().GetByID(ctx, child.ID)
		assert.Nil(t, retrieved)
		retrieved, _ = s.Collections().GetByID(ctx, grandchild.ID)
		assert.Nil(t, retrieved)
	})
}
