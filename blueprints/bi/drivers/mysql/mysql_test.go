package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-mizu/blueprints/bi/drivers"
)

func TestDriverName(t *testing.T) {
	d := &Driver{}
	assert.Equal(t, "mysql", d.Name())
}

func TestDriverRegistration(t *testing.T) {
	// Test that the driver is registered
	driver, err := drivers.Get("mysql")
	require.NoError(t, err)
	assert.NotNil(t, driver)
	assert.Equal(t, "mysql", driver.Name())

	// Test mariadb alias
	driver, err = drivers.Get("mariadb")
	require.NoError(t, err)
	assert.NotNil(t, driver)
}

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name     string
		config   drivers.Config
		contains []string
	}{
		{
			name: "basic connection",
			config: drivers.Config{
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				Username: "root",
				Password: "secret",
			},
			contains: []string{
				"root:secret@",
				"tcp(localhost:3306)",
				"/testdb",
				"charset=utf8mb4",
				"parseTime=true",
			},
		},
		{
			name: "with SSL",
			config: drivers.Config{
				Host:     "db.example.com",
				Port:     3306,
				Database: "production",
				Username: "app",
				Password: "pass",
				SSL:      true,
			},
			contains: []string{
				"tls=true",
			},
		},
		{
			name: "SSL verify-full",
			config: drivers.Config{
				Host:     "db.example.com",
				Port:     3306,
				Database: "production",
				Username: "app",
				Password: "pass",
				SSL:      true,
				SSLMode:  "verify-full",
			},
			contains: []string{
				"tls=skip-verify",
			},
		},
		{
			name: "SSL disabled",
			config: drivers.Config{
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				Username: "root",
				SSL:      false,
			},
			contains: []string{
				"tls=false",
			},
		},
		{
			name: "default port",
			config: drivers.Config{
				Host:     "localhost",
				Database: "testdb",
				Username: "root",
			},
			contains: []string{
				"tcp(localhost:3306)",
			},
		},
		{
			name: "default host",
			config: drivers.Config{
				Port:     3306,
				Database: "testdb",
				Username: "root",
			},
			contains: []string{
				"tcp(localhost:3306)",
			},
		},
		{
			name: "with custom options",
			config: drivers.Config{
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				Username: "root",
				Options: map[string]string{
					"timeout":    "10s",
					"readTimeout": "30s",
				},
			},
			contains: []string{
				"timeout=10s",
				"readTimeout=30s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Driver{}
			dsn := d.buildDSN(tt.config)

			for _, s := range tt.contains {
				assert.Contains(t, dsn, s, "DSN should contain %s", s)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	d := &Driver{}

	tests := []struct {
		input    string
		expected string
	}{
		{"table", "`table`"},
		{"my_table", "`my_table`"},
		{"table`name", "`table``name`"},
		{"user", "`user`"},
		{"select", "`select`"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := d.QuoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportsSchemas(t *testing.T) {
	d := &Driver{}
	assert.True(t, d.SupportsSchemas())
}

func TestCapabilities(t *testing.T) {
	d := &Driver{}
	caps := d.Capabilities()

	assert.True(t, caps.SupportsSSH)
	assert.True(t, caps.SupportsSSL)
	assert.True(t, caps.SupportsSchemas)
	assert.True(t, caps.SupportsCTEs)
	assert.True(t, caps.SupportsJSON)
	assert.False(t, caps.SupportsArrays)
	assert.True(t, caps.SupportsWindowFuncs)
	assert.True(t, caps.SupportsTransactions)
	assert.Equal(t, time.Hour, caps.MaxQueryTimeout)
	assert.Equal(t, 3306, caps.DefaultPort)
}

func TestDriverNotOpened(t *testing.T) {
	d := &Driver{}
	ctx := context.Background()

	// Test operations on unopened driver
	err := d.Ping(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not opened")

	_, err = d.ListSchemas(ctx)
	assert.Error(t, err)

	_, err = d.ListTables(ctx, "public")
	assert.Error(t, err)

	_, err = d.ListColumns(ctx, "public", "users")
	assert.Error(t, err)

	_, err = d.Execute(ctx, "SELECT 1")
	assert.Error(t, err)

	_, err = d.Version(ctx)
	assert.Error(t, err)

	_, err = d.CurrentDatabase(ctx)
	assert.Error(t, err)

	// Close should not error
	err = d.Close()
	assert.NoError(t, err)

	// DB should return nil
	assert.Nil(t, d.DB())
}

func TestConvertValue(t *testing.T) {
	d := &Driver{}

	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "byte slice",
			input:    []byte("hello"),
			expected: "hello",
		},
		{
			name:     "time value",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expected: "2024-01-15T10:30:00Z",
		},
		{
			name:     "integer",
			input:    42,
			expected: 42,
		},
		{
			name:     "string",
			input:    "test",
			expected: "test",
		},
		{
			name:     "float",
			input:    3.14,
			expected: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.convertValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapColumnType(t *testing.T) {
	tests := []struct {
		dbType   string
		expected string
	}{
		// Integer types
		{"INT", "number"},
		{"INTEGER", "number"},
		{"BIGINT", "number"},
		{"SMALLINT", "number"},
		{"TINYINT", "number"},
		{"MEDIUMINT", "number"},

		// Float types
		{"FLOAT", "number"},
		{"DOUBLE", "number"},
		{"DECIMAL", "number"},
		{"NUMERIC", "number"},

		// Boolean
		{"BOOLEAN", "boolean"},
		{"BOOL", "boolean"},
		{"BIT", "boolean"},

		// Date/time
		{"DATE", "datetime"},
		{"TIME", "datetime"},
		{"DATETIME", "datetime"},
		{"TIMESTAMP", "datetime"},
		{"YEAR", "datetime"},

		// String types
		{"VARCHAR", "string"},
		{"CHAR", "string"},
		{"TEXT", "string"},
		{"TINYTEXT", "string"},
		{"MEDIUMTEXT", "string"},
		{"LONGTEXT", "string"},
		{"ENUM", "string"},
		{"SET", "string"},

		// JSON
		{"JSON", "string"},

		// Binary
		{"BLOB", "string"},
		{"BINARY", "string"},
		{"VARBINARY", "string"},
		{"TINYBLOB", "string"},
		{"MEDIUMBLOB", "string"},
		{"LONGBLOB", "string"},

		// Unknown
		{"CUSTOM_TYPE", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			result := drivers.MapColumnType(tt.dbType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration tests - require a real MySQL database
// These tests are skipped by default and can be enabled with:
// MYSQL_TEST_DSN=user:pass@tcp(localhost:3306)/testdb go test -v

func TestIntegrationConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires a running MySQL server
	// Skip if not available
	t.Skip("Skipping MySQL integration test - no database available")

	config := drivers.Config{
		Engine:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "test",
		Username: "root",
		Password: "",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	driver, err := drivers.Open(ctx, config)
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}
	defer driver.Close()

	// Test ping
	err = driver.Ping(ctx)
	require.NoError(t, err)

	// Test version
	if versioner, ok := driver.(interface{ Version(context.Context) (string, error) }); ok {
		version, err := versioner.Version(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, version)
		t.Logf("MySQL version: %s", version)
	}

	// Test list schemas
	schemas, err := driver.ListSchemas(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, schemas)
	t.Logf("Schemas: %v", schemas)

	// Test list tables
	tables, err := driver.ListTables(ctx, "test")
	require.NoError(t, err)
	t.Logf("Tables in 'test': %v", tables)

	// Test execute query
	result, err := driver.Execute(ctx, "SELECT 1 as num, 'hello' as msg")
	require.NoError(t, err)
	assert.Len(t, result.Columns, 2)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, int64(1), result.RowCount)
}
