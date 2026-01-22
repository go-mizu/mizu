package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestDashboardStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid input", func(t *testing.T) {
		d := &store.Dashboard{
			Name:        "Sales Dashboard",
			Description: "Overview of sales metrics",
		}
		err := s.Dashboards().Create(ctx, d)
		require.NoError(t, err)

		assertIDGenerated(t, d.ID)
		assertTimestampsSet(t, d.CreatedAt, d.UpdatedAt)
	})

	t.Run("with auto refresh", func(t *testing.T) {
		d := &store.Dashboard{
			Name:        "Real-time Dashboard",
			AutoRefresh: 60, // 60 seconds
		}
		err := s.Dashboards().Create(ctx, d)
		require.NoError(t, err)

		retrieved, err := s.Dashboards().GetByID(ctx, d.ID)
		require.NoError(t, err)
		assert.Equal(t, 60, retrieved.AutoRefresh)
	})

	t.Run("with collection", func(t *testing.T) {
		coll := createTestCollection(t, s, "")
		d := &store.Dashboard{
			Name:         "Collection Dashboard",
			CollectionID: coll.ID,
		}
		err := s.Dashboards().Create(ctx, d)
		require.NoError(t, err)
		assert.Equal(t, coll.ID, d.CollectionID)
	})
}

func TestDashboardStore_GetByID(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		retrieved, err := s.Dashboards().GetByID(ctx, d.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, d.ID, retrieved.ID)
		assert.Equal(t, d.Name, retrieved.Name)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := s.Dashboards().GetByID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestDashboardStore_List(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all dashboards", func(t *testing.T) {
		createTestDashboard(t, s, "")
		createTestDashboard(t, s, "")

		dashboards, err := s.Dashboards().List(ctx)
		require.NoError(t, err)
		assert.Len(t, dashboards, 2)
	})

	t.Run("ordered by name", func(t *testing.T) {
		s := testStore(t) // Fresh store

		d1 := &store.Dashboard{Name: "Zebra Dashboard"}
		d2 := &store.Dashboard{Name: "Alpha Dashboard"}
		s.Dashboards().Create(ctx, d1)
		s.Dashboards().Create(ctx, d2)

		dashboards, err := s.Dashboards().List(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Alpha Dashboard", dashboards[0].Name)
		assert.Equal(t, "Zebra Dashboard", dashboards[1].Name)
	})
}

func TestDashboardStore_ListByCollection(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("filters by collection", func(t *testing.T) {
		coll1 := createTestCollection(t, s, "")
		coll2 := createTestCollection(t, s, "")

		createTestDashboard(t, s, coll1.ID)
		createTestDashboard(t, s, coll1.ID)
		createTestDashboard(t, s, coll2.ID)

		dashboards1, err := s.Dashboards().ListByCollection(ctx, coll1.ID)
		require.NoError(t, err)
		assert.Len(t, dashboards1, 2)

		dashboards2, err := s.Dashboards().ListByCollection(ctx, coll2.ID)
		require.NoError(t, err)
		assert.Len(t, dashboards2, 1)
	})
}

func TestDashboardStore_Update(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update name and description", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		d.Name = "Updated Dashboard"
		d.Description = "Updated description"
		err := s.Dashboards().Update(ctx, d)
		require.NoError(t, err)

		retrieved, err := s.Dashboards().GetByID(ctx, d.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Dashboard", retrieved.Name)
		assert.Equal(t, "Updated description", retrieved.Description)
	})

	t.Run("update auto refresh", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		d.AutoRefresh = 300
		err := s.Dashboards().Update(ctx, d)
		require.NoError(t, err)

		retrieved, err := s.Dashboards().GetByID(ctx, d.ID)
		require.NoError(t, err)
		assert.Equal(t, 300, retrieved.AutoRefresh)
	})
}

func TestDashboardStore_Delete(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes dashboard", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		err := s.Dashboards().Delete(ctx, d.ID)
		require.NoError(t, err)

		retrieved, err := s.Dashboards().GetByID(ctx, d.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("cascades to cards", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		d := createTestDashboard(t, s, "")
		createTestCard(t, s, d.ID, q.ID, 0, 0)
		createTestCard(t, s, d.ID, q.ID, 0, 6)

		// Verify cards exist
		cards, err := s.Dashboards().ListCards(ctx, d.ID)
		require.NoError(t, err)
		assert.Len(t, cards, 2)

		// Delete dashboard
		err = s.Dashboards().Delete(ctx, d.ID)
		require.NoError(t, err)

		// Verify cards are gone
		cards, err = s.Dashboards().ListCards(ctx, d.ID)
		require.NoError(t, err)
		assert.Empty(t, cards)
	})
}

// Card tests

func TestDashboardCardStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("question card", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		d := createTestDashboard(t, s, "")

		card := &store.DashboardCard{
			DashboardID: d.ID,
			QuestionID:  q.ID,
			CardType:    "question",
			Row:         0,
			Col:         0,
			Width:       6,
			Height:      4,
		}
		err := s.Dashboards().CreateCard(ctx, card)
		require.NoError(t, err)
		assertIDGenerated(t, card.ID)
	})

	t.Run("text card without question", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		card := &store.DashboardCard{
			DashboardID: d.ID,
			CardType:    "text",
			Row:         0,
			Col:         0,
			Width:       12,
			Height:      2,
			Settings: map[string]any{
				"text": "# Welcome\nThis is a dashboard",
			},
		}
		err := s.Dashboards().CreateCard(ctx, card)
		require.NoError(t, err)
		assert.Empty(t, card.QuestionID)
	})

	t.Run("invalid dashboard fails", func(t *testing.T) {
		card := &store.DashboardCard{
			DashboardID: "nonexistent",
			CardType:    "text",
			Row:         0,
			Col:         0,
			Width:       6,
			Height:      4,
		}
		err := s.Dashboards().CreateCard(ctx, card)
		require.Error(t, err)
	})
}

func TestDashboardCardStore_GetCard(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("exists", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		d := createTestDashboard(t, s, "")
		card := createTestCard(t, s, d.ID, q.ID, 0, 0)

		retrieved, err := s.Dashboards().GetCard(ctx, card.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, card.ID, retrieved.ID)
		assert.Equal(t, d.ID, retrieved.DashboardID)
	})
}

func TestDashboardCardStore_ListCards(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("ordered by row then col", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		d := createTestDashboard(t, s, "")

		// Create in different order
		createTestCard(t, s, d.ID, q.ID, 1, 0)
		createTestCard(t, s, d.ID, q.ID, 0, 6)
		createTestCard(t, s, d.ID, q.ID, 0, 0)

		cards, err := s.Dashboards().ListCards(ctx, d.ID)
		require.NoError(t, err)
		require.Len(t, cards, 3)

		assert.Equal(t, 0, cards[0].Row)
		assert.Equal(t, 0, cards[0].Col)
		assert.Equal(t, 0, cards[1].Row)
		assert.Equal(t, 6, cards[1].Col)
		assert.Equal(t, 1, cards[2].Row)
	})
}

func TestDashboardCardStore_UpdateCard(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("update position", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		d := createTestDashboard(t, s, "")
		card := createTestCard(t, s, d.ID, q.ID, 0, 0)

		card.Row = 2
		card.Col = 6
		card.Width = 4
		card.Height = 3
		err := s.Dashboards().UpdateCard(ctx, card)
		require.NoError(t, err)

		retrieved, _ := s.Dashboards().GetCard(ctx, card.ID)
		assert.Equal(t, 2, retrieved.Row)
		assert.Equal(t, 6, retrieved.Col)
		assert.Equal(t, 4, retrieved.Width)
		assert.Equal(t, 3, retrieved.Height)
	})

	t.Run("update settings", func(t *testing.T) {
		d := createTestDashboard(t, s, "")
		card := &store.DashboardCard{
			DashboardID: d.ID,
			CardType:    "text",
			Row:         0,
			Col:         0,
			Width:       12,
			Height:      2,
		}
		s.Dashboards().CreateCard(ctx, card)

		card.Settings = map[string]any{
			"text":  "Updated content",
			"color": "#509EE3",
		}
		err := s.Dashboards().UpdateCard(ctx, card)
		require.NoError(t, err)

		retrieved, _ := s.Dashboards().GetCard(ctx, card.ID)
		assert.Equal(t, "Updated content", retrieved.Settings["text"])
	})
}

func TestDashboardCardStore_DeleteCard(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes specific card", func(t *testing.T) {
		ds := createTestDataSource(t, s)
		q := createTestQuestion(t, s, ds.ID, "")
		d := createTestDashboard(t, s, "")
		card1 := createTestCard(t, s, d.ID, q.ID, 0, 0)
		card2 := createTestCard(t, s, d.ID, q.ID, 0, 6)

		err := s.Dashboards().DeleteCard(ctx, card1.ID)
		require.NoError(t, err)

		// card1 should be gone
		retrieved, _ := s.Dashboards().GetCard(ctx, card1.ID)
		assert.Nil(t, retrieved)

		// card2 should still exist
		retrieved, _ = s.Dashboards().GetCard(ctx, card2.ID)
		assert.NotNil(t, retrieved)
	})
}

// Filter tests

func TestDashboardFilterStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid filter", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		filter := &store.DashboardFilter{
			DashboardID: d.ID,
			Name:        "Date Range",
			Type:        "date",
			Default:     "last_30_days",
			Required:    true,
			Targets: []store.FilterTarget{
				{CardID: "card1", ColumnID: "created_at"},
				{CardID: "card2", ColumnID: "date"},
			},
		}
		err := s.Dashboards().CreateFilter(ctx, filter)
		require.NoError(t, err)
		assertIDGenerated(t, filter.ID)
	})
}

func TestDashboardFilterStore_ListFilters(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("returns all filters for dashboard", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		f1 := &store.DashboardFilter{DashboardID: d.ID, Name: "Filter 1", Type: "text"}
		f2 := &store.DashboardFilter{DashboardID: d.ID, Name: "Filter 2", Type: "number"}
		s.Dashboards().CreateFilter(ctx, f1)
		s.Dashboards().CreateFilter(ctx, f2)

		filters, err := s.Dashboards().ListFilters(ctx, d.ID)
		require.NoError(t, err)
		assert.Len(t, filters, 2)
	})
}

func TestDashboardFilterStore_DeleteFilter(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes filter", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		filter := &store.DashboardFilter{DashboardID: d.ID, Name: "Test", Type: "text"}
		s.Dashboards().CreateFilter(ctx, filter)

		err := s.Dashboards().DeleteFilter(ctx, filter.ID)
		require.NoError(t, err)

		filters, _ := s.Dashboards().ListFilters(ctx, d.ID)
		assert.Empty(t, filters)
	})
}

// Tab tests

func TestDashboardTabStore_Create(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("valid tab", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		tab := &store.DashboardTab{
			DashboardID: d.ID,
			Name:        "Overview",
			Position:    0,
		}
		err := s.Dashboards().CreateTab(ctx, tab)
		require.NoError(t, err)
		assertIDGenerated(t, tab.ID)
	})
}

func TestDashboardTabStore_ListTabs(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("ordered by position", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		// Create in reverse order
		s.Dashboards().CreateTab(ctx, &store.DashboardTab{DashboardID: d.ID, Name: "Third", Position: 2})
		s.Dashboards().CreateTab(ctx, &store.DashboardTab{DashboardID: d.ID, Name: "First", Position: 0})
		s.Dashboards().CreateTab(ctx, &store.DashboardTab{DashboardID: d.ID, Name: "Second", Position: 1})

		tabs, err := s.Dashboards().ListTabs(ctx, d.ID)
		require.NoError(t, err)
		require.Len(t, tabs, 3)

		assert.Equal(t, "First", tabs[0].Name)
		assert.Equal(t, "Second", tabs[1].Name)
		assert.Equal(t, "Third", tabs[2].Name)
	})
}

func TestDashboardTabStore_DeleteTab(t *testing.T) {
	s := testStore(t)
	ctx := testContext()

	t.Run("deletes tab", func(t *testing.T) {
		d := createTestDashboard(t, s, "")

		tab := &store.DashboardTab{DashboardID: d.ID, Name: "Test", Position: 0}
		s.Dashboards().CreateTab(ctx, tab)

		err := s.Dashboards().DeleteTab(ctx, tab.ID)
		require.NoError(t, err)

		tabs, _ := s.Dashboards().ListTabs(ctx, d.ID)
		assert.Empty(t, tabs)
	})
}
