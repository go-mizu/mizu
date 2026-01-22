package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/store"
)

// testStore creates a new store for testing with a temporary database.
func testStore(t *testing.T) *Store {
	t.Helper()

	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := s.Ensure(ctx); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	t.Cleanup(func() {
		s.Close()
	})

	return s
}

// testContext returns a context for tests.
func testContext() context.Context {
	return context.Background()
}

// assertTimestampsSet verifies CreatedAt/UpdatedAt are set.
func assertTimestampsSet(t *testing.T, createdAt, updatedAt time.Time) {
	t.Helper()
	assert.False(t, createdAt.IsZero(), "CreatedAt should be set")
	assert.False(t, updatedAt.IsZero(), "UpdatedAt should be set")
}

// assertIDGenerated verifies ID was auto-generated.
func assertIDGenerated(t *testing.T, id string) {
	t.Helper()
	assert.NotEmpty(t, id, "ID should be generated")
	assert.Len(t, id, 26, "ID should be ULID format (26 chars)")
}

// Test data factories

func createTestDataSource(t *testing.T, s *Store) *store.DataSource {
	t.Helper()
	ds := &store.DataSource{
		Name:     "Test DB " + ulid.Make().String()[:8],
		Engine:   "sqlite",
		Database: "test.db",
	}
	err := s.DataSources().Create(testContext(), ds)
	require.NoError(t, err)
	return ds
}

func createTestTable(t *testing.T, s *Store, dsID string) *store.Table {
	t.Helper()
	tbl := &store.Table{
		DataSourceID: dsID,
		Schema:       "public",
		Name:         "test_table_" + ulid.Make().String()[:8],
		DisplayName:  "Test Table",
	}
	err := s.Tables().Create(testContext(), tbl)
	require.NoError(t, err)
	return tbl
}

func createTestColumn(t *testing.T, s *Store, tableID string, pos int) *store.Column {
	t.Helper()
	col := &store.Column{
		TableID:     tableID,
		Name:        fmt.Sprintf("col_%d", pos),
		DisplayName: fmt.Sprintf("Column %d", pos),
		Type:        "string",
		Position:    pos,
	}
	err := s.Tables().CreateColumn(testContext(), col)
	require.NoError(t, err)
	return col
}

func createTestCollection(t *testing.T, s *Store, parentID string) *store.Collection {
	t.Helper()
	c := &store.Collection{
		Name:     "Test Collection " + ulid.Make().String()[:8],
		ParentID: parentID,
		Color:    "#509EE3",
	}
	err := s.Collections().Create(testContext(), c)
	require.NoError(t, err)
	return c
}

func createTestQuestion(t *testing.T, s *Store, dsID string, collID string) *store.Question {
	t.Helper()
	q := &store.Question{
		Name:          "Test Question " + ulid.Make().String()[:8],
		DataSourceID:  dsID,
		CollectionID:  collID,
		QueryType:     "query",
		Query:         map[string]any{"table": "test"},
		Visualization: map[string]any{"type": "table"},
	}
	err := s.Questions().Create(testContext(), q)
	require.NoError(t, err)
	return q
}

func createTestDashboard(t *testing.T, s *Store, collID string) *store.Dashboard {
	t.Helper()
	d := &store.Dashboard{
		Name:         "Test Dashboard " + ulid.Make().String()[:8],
		CollectionID: collID,
		AutoRefresh:  0,
	}
	err := s.Dashboards().Create(testContext(), d)
	require.NoError(t, err)
	return d
}

func createTestCard(t *testing.T, s *Store, dashID, qID string, row, col int) *store.DashboardCard {
	t.Helper()
	card := &store.DashboardCard{
		DashboardID: dashID,
		QuestionID:  qID,
		CardType:    "question",
		Row:         row,
		Col:         col,
		Width:       6,
		Height:      4,
	}
	err := s.Dashboards().CreateCard(testContext(), card)
	require.NoError(t, err)
	return card
}

func createTestModel(t *testing.T, s *Store, dsID string) *store.Model {
	t.Helper()
	m := &store.Model{
		Name:         "Test Model " + ulid.Make().String()[:8],
		DataSourceID: dsID,
		Query:        map[string]any{"table": "customers"},
	}
	err := s.Models().Create(testContext(), m)
	require.NoError(t, err)
	return m
}

func createTestMetric(t *testing.T, s *Store, tableID string) *store.Metric {
	t.Helper()
	m := &store.Metric{
		Name:    "Test Metric " + ulid.Make().String()[:8],
		TableID: tableID,
		Definition: map[string]any{
			"aggregation": "sum",
			"column":      "total",
		},
	}
	err := s.Metrics().Create(testContext(), m)
	require.NoError(t, err)
	return m
}

func createTestAlert(t *testing.T, s *Store, questionID string) *store.Alert {
	t.Helper()
	a := &store.Alert{
		Name:       "Test Alert " + ulid.Make().String()[:8],
		QuestionID: questionID,
		AlertType:  "goal",
		Condition: store.AlertCondition{
			Operator: "below",
			Value:    10000,
		},
		Channels: []store.AlertChannel{
			{Type: "email", Targets: []string{"test@example.com"}},
		},
		Enabled: true,
	}
	err := s.Alerts().Create(testContext(), a)
	require.NoError(t, err)
	return a
}

func createTestSubscription(t *testing.T, s *Store, dashID string) *store.Subscription {
	t.Helper()
	sub := &store.Subscription{
		DashboardID: dashID,
		Schedule:    "0 9 * * 1",
		Format:      "pdf",
		Recipients:  []string{"team@example.com"},
		Enabled:     true,
	}
	err := s.Subscriptions().Create(testContext(), sub)
	require.NoError(t, err)
	return sub
}

func createTestUser(t *testing.T, s *Store) *store.User {
	t.Helper()
	u := &store.User{
		Email:        fmt.Sprintf("test_%s@example.com", ulid.Make().String()),
		Name:         "Test User",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$abcdefghijklmnop$abcdefghijklmnopqrstuvwxyz012345",
		Role:         "user",
	}
	err := s.Users().Create(testContext(), u)
	require.NoError(t, err)
	return u
}

func createTestSession(t *testing.T, s *Store, userID string) *store.Session {
	t.Helper()
	sess := &store.Session{
		UserID:    userID,
		Token:     ulid.Make().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := s.Users().CreateSession(testContext(), sess)
	require.NoError(t, err)
	return sess
}

func createTestAuditLog(t *testing.T, s *Store) *store.AuditLog {
	t.Helper()
	log := &store.AuditLog{
		ActorID:      "user123",
		ActorEmail:   "user@example.com",
		Action:       "question.created",
		ResourceType: "question",
		ResourceID:   "question123",
		Metadata:     map[string]string{"name": "Test Question"},
		IPAddress:    "192.168.1.1",
	}
	err := s.Settings().WriteAuditLog(testContext(), log)
	require.NoError(t, err)
	return log
}

func createTestQueryHistory(t *testing.T, s *Store, userID, dsID string) *store.QueryHistory {
	t.Helper()
	qh := &store.QueryHistory{
		UserID:       userID,
		DataSourceID: dsID,
		Query:        "SELECT * FROM orders LIMIT 100",
		Duration:     45.5,
		RowCount:     100,
	}
	err := s.QueryHistory().Create(testContext(), qh)
	require.NoError(t, err)
	return qh
}

// TestStoreNew tests Store creation.
func TestStoreNew(t *testing.T) {
	t.Run("creates store with valid dir", func(t *testing.T) {
		dir := t.TempDir()
		s, err := New(dir)
		require.NoError(t, err)
		require.NotNil(t, s)
		s.Close()
	})
}

// TestStoreEnsure tests schema creation.
func TestStoreEnsure(t *testing.T) {
	t.Run("creates all tables", func(t *testing.T) {
		s := testStore(t)
		ctx := testContext()

		// Verify tables exist by attempting to query them
		tables := []string{
			"datasources", "tables", "columns", "collections", "questions",
			"dashboards", "dashboard_cards", "dashboard_filters", "dashboard_tabs",
			"models", "model_columns", "metrics", "alerts", "subscriptions",
			"users", "sessions", "settings", "audit_logs", "query_history",
		}

		for _, table := range tables {
			var count int
			err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count)
			assert.NoError(t, err, "table %s should exist", table)
		}
	})
}
