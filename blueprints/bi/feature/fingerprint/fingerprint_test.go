package fingerprint

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFingerprint(t *testing.T) {
	fp := Fingerprint{
		ColumnID:      "col-123",
		DistinctCount: 1000,
		NullCount:     50,
		TotalCount:    10000,
		MinValue:      "0",
		MaxValue:      "999",
		AvgLength:     3.5,
		SampleSize:    10000,
		ComputedAt:    time.Now(),
	}

	assert.Equal(t, "col-123", fp.ColumnID)
	assert.Equal(t, int64(1000), fp.DistinctCount)
	assert.Equal(t, int64(50), fp.NullCount)
	assert.Equal(t, int64(10000), fp.TotalCount)
	assert.Equal(t, "0", fp.MinValue)
	assert.Equal(t, "999", fp.MaxValue)
	assert.Equal(t, 3.5, fp.AvgLength)
	assert.Equal(t, int64(10000), fp.SampleSize)
}

func TestFingerprintResult(t *testing.T) {
	now := time.Now()
	result := FingerprintResult{
		DataSourceID:         "ds-123",
		Status:               "success",
		StartedAt:            now,
		CompletedAt:          now.Add(30 * time.Second),
		DurationMs:           30000,
		TablesProcessed:      45,
		ColumnsFingerprinted: 380,
		ValuesScanned:        120,
		Errors:               nil,
	}

	assert.Equal(t, "ds-123", result.DataSourceID)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, int64(30000), result.DurationMs)
	assert.Equal(t, 45, result.TablesProcessed)
	assert.Equal(t, 380, result.ColumnsFingerprinted)
	assert.Equal(t, 120, result.ValuesScanned)
	assert.Empty(t, result.Errors)
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int64
	}{
		{"int64", int64(100), 100},
		{"int", int(50), 50},
		{"int32", int32(25), 25},
		{"float64", float64(42.5), 42},
		{"float32", float32(21.9), 21},
		{"string", "invalid", 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toInt64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"float64", float64(3.14), 3.14},
		{"float32", float32(2.5), 2.5},
		{"int64", int64(100), 100.0},
		{"int", int(50), 50.0},
		{"string", "invalid", 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"bytes", []byte("world"), "world"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"nil", nil, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildFingerprintQueryNumber(t *testing.T) {
	query := buildFingerprintQuery("postgres", "public.users", "\"age\"", "number")

	assert.Contains(t, query, "COUNT(DISTINCT")
	assert.Contains(t, query, "MIN(")
	assert.Contains(t, query, "MAX(")
	assert.Contains(t, query, "null_count")
	assert.Contains(t, query, "total_count")
}

func TestBuildFingerprintQueryDatetime(t *testing.T) {
	query := buildFingerprintQuery("mysql", "orders", "`created_at`", "datetime")

	assert.Contains(t, query, "COUNT(DISTINCT")
	assert.Contains(t, query, "MIN(")
	assert.Contains(t, query, "MAX(")
	assert.Contains(t, query, "CAST(")
}

func TestBuildFingerprintQueryStringPostgres(t *testing.T) {
	query := buildFingerprintQuery("postgres", "public.users", "\"email\"", "string")

	assert.Contains(t, query, "COUNT(DISTINCT")
	assert.Contains(t, query, "AVG(LENGTH(")
	assert.Contains(t, query, "min_value")
	assert.Contains(t, query, "max_value")
	assert.Contains(t, query, "avg_length")
}

func TestBuildFingerprintQueryStringMySQL(t *testing.T) {
	query := buildFingerprintQuery("mysql", "users", "`name`", "string")

	assert.Contains(t, query, "COUNT(DISTINCT")
	assert.Contains(t, query, "AVG(CHAR_LENGTH(")
	assert.Contains(t, query, "avg_length")
}

func TestBuildFingerprintQueryStringSQLite(t *testing.T) {
	query := buildFingerprintQuery("sqlite", "users", "\"name\"", "string")

	assert.Contains(t, query, "COUNT(DISTINCT")
	assert.Contains(t, query, "AVG(LENGTH(")
	assert.Contains(t, query, "avg_length")
}

func TestBuildFingerprintQueryBoolean(t *testing.T) {
	query := buildFingerprintQuery("postgres", "public.users", "\"active\"", "boolean")

	// Boolean uses base query (no min/max/avg_length)
	assert.Contains(t, query, "COUNT(DISTINCT")
	assert.Contains(t, query, "null_count")
	assert.Contains(t, query, "total_count")
	assert.Contains(t, query, "sample_size")
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 10000, MaxSampleSize)
	assert.Equal(t, 1000, MaxCachedValues)
	assert.Equal(t, 100, MaxValueLength)
}

func TestNewService(t *testing.T) {
	service := NewService(nil)
	assert.NotNil(t, service)
}
