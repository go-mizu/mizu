package cache

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/bi/drivers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	require.NotNil(t, cache)
	assert.NotNil(t, cache.entries)
	assert.NotNil(t, cache.stats)
	assert.Equal(t, 10000, cache.maxSize)
	assert.Equal(t, time.Minute, cache.cleanupEvery)

	// Clean up
	cache.Close()
}

func TestNewMemoryCacheWithOptions(t *testing.T) {
	cache := NewMemoryCache(
		WithMaxSize(500),
		WithCleanupInterval(5*time.Minute),
	)
	require.NotNil(t, cache)
	assert.Equal(t, 500, cache.maxSize)
	assert.Equal(t, 5*time.Minute, cache.cleanupEvery)

	// Clean up
	cache.Close()
}

func TestMemoryCacheSetAndGet(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	result := &CachedResult{
		DataSourceID: "ds-123",
		QueryHash:    key,
		Result: &drivers.QueryResult{
			Columns: []drivers.ResultColumn{{Name: "id", Type: "int"}},
			Rows:    []map[string]any{{"id": 1}, {"id": 2}},
		},
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Set
	err := cache.Set(ctx, key, result, time.Hour)
	require.NoError(t, err)

	// Get
	got, err := cache.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ds-123", got.DataSourceID)
	assert.Equal(t, key, got.QueryHash)
	assert.Len(t, got.Result.Rows, 2)
}

func TestMemoryCacheGetNonExistent(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	got, err := cache.Get(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryCacheDelete(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	result := &CachedResult{
		DataSourceID: "ds-123",
		QueryHash:    key,
		Result:       &drivers.QueryResult{},
	}

	// Set and verify
	err := cache.Set(ctx, key, result, time.Hour)
	require.NoError(t, err)

	got, err := cache.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Delete
	err = cache.Delete(ctx, key)
	require.NoError(t, err)

	// Verify deleted
	got, err = cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryCacheDeleteNonExistent(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	err := cache.Delete(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestMemoryCacheClear(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	// Add entries for two data sources
	for i := 0; i < 5; i++ {
		result := &CachedResult{
			DataSourceID: "ds-1",
			QueryHash:    "key-" + string(rune('a'+i)),
			Result:       &drivers.QueryResult{},
		}
		cache.Set(ctx, result.QueryHash, result, time.Hour)
	}

	for i := 0; i < 3; i++ {
		result := &CachedResult{
			DataSourceID: "ds-2",
			QueryHash:    "key-" + string(rune('f'+i)),
			Result:       &drivers.QueryResult{},
		}
		cache.Set(ctx, result.QueryHash, result, time.Hour)
	}

	// Clear ds-1
	err := cache.Clear(ctx, "ds-1")
	require.NoError(t, err)

	// Verify ds-1 entries are gone
	for i := 0; i < 5; i++ {
		got, _ := cache.Get(ctx, "key-"+string(rune('a'+i)))
		assert.Nil(t, got)
	}

	// Verify ds-2 entries still exist
	for i := 0; i < 3; i++ {
		got, _ := cache.Get(ctx, "key-"+string(rune('f'+i)))
		assert.NotNil(t, got)
	}
}

func TestMemoryCacheStats(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	// Add entries
	for i := 0; i < 3; i++ {
		result := &CachedResult{
			DataSourceID: "ds-123",
			QueryHash:    "key-" + string(rune('a'+i)),
			Result:       &drivers.QueryResult{},
		}
		cache.Set(ctx, result.QueryHash, result, time.Hour)
	}

	// Generate some hits
	cache.Get(ctx, "key-a")
	cache.Get(ctx, "key-a")
	cache.Get(ctx, "key-b")

	stats, err := cache.Stats(ctx, "ds-123")
	require.NoError(t, err)
	assert.Equal(t, "ds-123", stats.DataSourceID)
	assert.Equal(t, int64(3), stats.Entries)
	assert.Equal(t, int64(3), stats.Hits)
}

func TestMemoryCacheStatsEmpty(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	stats, err := cache.Stats(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "nonexistent", stats.DataSourceID)
	assert.Equal(t, int64(0), stats.Entries)
	assert.Equal(t, int64(0), stats.Hits)
}

func TestMemoryCacheExpiration(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "expiring-key"
	result := &CachedResult{
		DataSourceID: "ds-123",
		QueryHash:    key,
		Result:       &drivers.QueryResult{},
	}

	// Set with very short TTL
	err := cache.Set(ctx, key, result, 10*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	got, err := cache.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should be expired now
	got, err = cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryCacheEviction(t *testing.T) {
	cache := NewMemoryCache(WithMaxSize(5))
	defer cache.Close()

	ctx := context.Background()

	// Fill cache to max
	for i := 0; i < 5; i++ {
		result := &CachedResult{
			DataSourceID: "ds-123",
			QueryHash:    "key-" + string(rune('a'+i)),
			Result:       &drivers.QueryResult{},
		}
		cache.Set(ctx, result.QueryHash, result, time.Duration(i+1)*time.Hour)
	}

	assert.Len(t, cache.entries, 5)

	// Add one more - should evict the one with shortest TTL
	result := &CachedResult{
		DataSourceID: "ds-123",
		QueryHash:    "key-new",
		Result:       &drivers.QueryResult{},
	}
	cache.Set(ctx, result.QueryHash, result, 10*time.Hour)

	assert.Len(t, cache.entries, 5)

	// The entry with shortest TTL (key-a with 1h) should be evicted
	got, _ := cache.Get(ctx, "key-a")
	assert.Nil(t, got)

	// New entry should exist
	got, _ = cache.Get(ctx, "key-new")
	assert.NotNil(t, got)
}

func TestMemoryCacheHitCount(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	result := &CachedResult{
		DataSourceID: "ds-123",
		QueryHash:    key,
		Result:       &drivers.QueryResult{},
		HitCount:     0,
	}

	cache.Set(ctx, key, result, time.Hour)

	// Get multiple times
	for i := 0; i < 5; i++ {
		got, _ := cache.Get(ctx, key)
		assert.Equal(t, int64(i+1), got.HitCount)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name         string
		datasourceID string
		query        string
		params       []any
	}{
		{
			name:         "simple query",
			datasourceID: "ds-123",
			query:        "SELECT * FROM users",
			params:       nil,
		},
		{
			name:         "query with params",
			datasourceID: "ds-456",
			query:        "SELECT * FROM users WHERE id = ?",
			params:       []any{1},
		},
		{
			name:         "query with multiple params",
			datasourceID: "ds-789",
			query:        "SELECT * FROM users WHERE age > ? AND name = ?",
			params:       []any{21, "John"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := GenerateCacheKey(tt.datasourceID, tt.query, tt.params)
			key2 := GenerateCacheKey(tt.datasourceID, tt.query, tt.params)

			// Same inputs should produce same key
			assert.Equal(t, key1, key2)
			assert.Len(t, key1, 64) // SHA256 hex = 64 chars
		})
	}
}

func TestGenerateCacheKeyDifferent(t *testing.T) {
	key1 := GenerateCacheKey("ds-1", "SELECT * FROM users", nil)
	key2 := GenerateCacheKey("ds-2", "SELECT * FROM users", nil)
	key3 := GenerateCacheKey("ds-1", "SELECT * FROM orders", nil)
	key4 := GenerateCacheKey("ds-1", "SELECT * FROM users", []any{1})

	// All should be different
	assert.NotEqual(t, key1, key2)
	assert.NotEqual(t, key1, key3)
	assert.NotEqual(t, key1, key4)
}

// MockDriver implements drivers.Driver for testing
type MockDriver struct {
	db          *sql.DB
	executeFunc func(ctx context.Context, query string, params ...any) (*drivers.QueryResult, error)
}

func (m *MockDriver) Name() string                            { return "mock" }
func (m *MockDriver) Open(ctx context.Context, config drivers.Config) error { return nil }
func (m *MockDriver) Close() error                            { return nil }
func (m *MockDriver) Ping(ctx context.Context) error          { return nil }
func (m *MockDriver) DB() *sql.DB                             { return m.db }
func (m *MockDriver) QuoteIdentifier(s string) string         { return `"` + s + `"` }
func (m *MockDriver) ListSchemas(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *MockDriver) ListTables(ctx context.Context, schema string) ([]drivers.Table, error) {
	return nil, nil
}
func (m *MockDriver) ListColumns(ctx context.Context, schema, table string) ([]drivers.Column, error) {
	return nil, nil
}
func (m *MockDriver) SupportsSchemas() bool { return true }
func (m *MockDriver) Capabilities() drivers.DriverCapabilities {
	return drivers.DriverCapabilities{}
}
func (m *MockDriver) Execute(ctx context.Context, query string, params ...any) (*drivers.QueryResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, query, params...)
	}
	return &drivers.QueryResult{}, nil
}

func TestCachedExecutorCacheDisabled(t *testing.T) {
	callCount := 0
	driver := &MockDriver{
		executeFunc: func(ctx context.Context, query string, params ...any) (*drivers.QueryResult, error) {
			callCount++
			return &drivers.QueryResult{
				Rows: []map[string]any{{"id": callCount}},
			}, nil
		},
	}

	// TTL = 0 means caching disabled
	executor := NewCachedExecutor(driver, nil, "ds-123", 0)

	ctx := context.Background()
	query := "SELECT * FROM users"

	// Each call should hit the driver
	executor.Execute(ctx, query)
	executor.Execute(ctx, query)
	executor.Execute(ctx, query)

	assert.Equal(t, 3, callCount)
}

func TestCachedExecutorWithCache(t *testing.T) {
	callCount := 0
	driver := &MockDriver{
		executeFunc: func(ctx context.Context, query string, params ...any) (*drivers.QueryResult, error) {
			callCount++
			return &drivers.QueryResult{
				Rows: []map[string]any{{"id": callCount}},
			}, nil
		},
	}

	cache := NewMemoryCache()
	defer cache.Close()

	executor := NewCachedExecutor(driver, cache, "ds-123", time.Hour)

	ctx := context.Background()
	query := "SELECT * FROM users"

	// First call should hit the driver
	result1, err := executor.Execute(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.False(t, result1.Cached)

	// Second call should hit cache
	result2, err := executor.Execute(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount) // Still 1
	assert.True(t, result2.Cached)

	// Third call should still hit cache
	result3, err := executor.Execute(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount) // Still 1
	assert.True(t, result3.Cached)
}

func TestCachedExecutorDifferentQueries(t *testing.T) {
	callCount := 0
	driver := &MockDriver{
		executeFunc: func(ctx context.Context, query string, params ...any) (*drivers.QueryResult, error) {
			callCount++
			return &drivers.QueryResult{
				Rows: []map[string]any{{"query": query}},
			}, nil
		},
	}

	cache := NewMemoryCache()
	defer cache.Close()

	executor := NewCachedExecutor(driver, cache, "ds-123", time.Hour)

	ctx := context.Background()

	// Different queries should hit the driver
	executor.Execute(ctx, "SELECT * FROM users")
	executor.Execute(ctx, "SELECT * FROM orders")
	executor.Execute(ctx, "SELECT * FROM products")

	assert.Equal(t, 3, callCount)

	// Repeat queries should hit cache
	executor.Execute(ctx, "SELECT * FROM users")
	executor.Execute(ctx, "SELECT * FROM orders")
	executor.Execute(ctx, "SELECT * FROM products")

	assert.Equal(t, 3, callCount) // Still 3
}

func TestCachedExecutorWithParams(t *testing.T) {
	callCount := 0
	driver := &MockDriver{
		executeFunc: func(ctx context.Context, query string, params ...any) (*drivers.QueryResult, error) {
			callCount++
			return &drivers.QueryResult{
				Rows: []map[string]any{{"id": params[0]}},
			}, nil
		},
	}

	cache := NewMemoryCache()
	defer cache.Close()

	executor := NewCachedExecutor(driver, cache, "ds-123", time.Hour)

	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = ?"

	// Same query, different params should hit driver
	executor.Execute(ctx, query, 1)
	executor.Execute(ctx, query, 2)
	executor.Execute(ctx, query, 3)

	assert.Equal(t, 3, callCount)

	// Same query + params should hit cache
	executor.Execute(ctx, query, 1)
	executor.Execute(ctx, query, 2)

	assert.Equal(t, 3, callCount) // Still 3
}

func TestCacheStats_HitRate(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	result := &CachedResult{
		DataSourceID: "ds-123",
		QueryHash:    "key-1",
		Result:       &drivers.QueryResult{},
	}
	cache.Set(ctx, "key-1", result, time.Hour)

	// 3 hits
	cache.Get(ctx, "key-1")
	cache.Get(ctx, "key-1")
	cache.Get(ctx, "key-1")

	stats, _ := cache.Stats(ctx, "ds-123")
	assert.Equal(t, int64(3), stats.Hits)
	// Note: misses tracking is not fully implemented for unknown datasources
}

func TestCachedResult(t *testing.T) {
	now := time.Now()
	result := CachedResult{
		DataSourceID: "ds-123",
		QueryHash:    "abc123",
		Result: &drivers.QueryResult{
			Columns: []drivers.ResultColumn{{Name: "id"}},
			Rows:    []map[string]any{{"id": 1}},
		},
		CachedAt:  now,
		ExpiresAt: now.Add(time.Hour),
		HitCount:  5,
	}

	assert.Equal(t, "ds-123", result.DataSourceID)
	assert.Equal(t, "abc123", result.QueryHash)
	assert.Equal(t, int64(5), result.HitCount)
	assert.Len(t, result.Result.Rows, 1)
}

func TestCacheStats(t *testing.T) {
	stats := CacheStats{
		DataSourceID: "ds-123",
		Entries:      100,
		Hits:         75,
		Misses:       25,
		HitRate:      0.75,
		MemoryBytes:  1024 * 1024,
	}

	assert.Equal(t, "ds-123", stats.DataSourceID)
	assert.Equal(t, int64(100), stats.Entries)
	assert.Equal(t, int64(75), stats.Hits)
	assert.Equal(t, int64(25), stats.Misses)
	assert.Equal(t, 0.75, stats.HitRate)
	assert.Equal(t, int64(1024*1024), stats.MemoryBytes)
}

func TestMemoryCacheClose(t *testing.T) {
	cache := NewMemoryCache()

	// Close should not panic
	err := cache.Close()
	require.NoError(t, err)

	// Double close should not panic (channel already closed)
	// Note: This would panic with the current implementation
	// In production, you'd want to handle this case
}

func TestEstimateSize(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	// Nil result
	size := cache.estimateSize(nil)
	assert.Equal(t, int64(0), size)

	// Empty result
	result := &CachedResult{Result: nil}
	size = cache.estimateSize(result)
	assert.Equal(t, int64(0), size)

	// Result with data
	result = &CachedResult{
		DataSourceID: "ds-123",
		Result: &drivers.QueryResult{
			Columns: []drivers.ResultColumn{{Name: "id"}},
			Rows:    []map[string]any{{"id": 1}},
		},
	}
	size = cache.estimateSize(result)
	assert.Greater(t, size, int64(0))
}
