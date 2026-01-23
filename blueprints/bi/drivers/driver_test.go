package drivers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapColumnType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Integer types
		{"INTEGER uppercase", "INTEGER", "number"},
		{"INT uppercase", "INT", "number"},
		{"BIGINT uppercase", "BIGINT", "number"},
		{"SMALLINT uppercase", "SMALLINT", "number"},
		{"TINYINT uppercase", "TINYINT", "number"},
		{"SERIAL uppercase", "SERIAL", "number"},

		// PostgreSQL specific
		{"int2", "int2", "number"},
		{"int4", "int4", "number"},
		{"int8", "int8", "number"},
		{"serial4", "serial4", "number"},
		{"serial8", "serial8", "number"},
		{"bigserial", "bigserial", "number"},

		// Float types
		{"REAL", "REAL", "number"},
		{"FLOAT", "FLOAT", "number"},
		{"DOUBLE", "DOUBLE", "number"},
		{"DOUBLE PRECISION", "DOUBLE PRECISION", "number"},
		{"NUMERIC", "NUMERIC", "number"},
		{"DECIMAL", "DECIMAL", "number"},
		{"float4", "float4", "number"},
		{"float8", "float8", "number"},
		{"money", "MONEY", "number"},

		// Boolean types
		{"BOOLEAN uppercase", "BOOLEAN", "boolean"},
		{"BOOL uppercase", "BOOL", "boolean"},
		{"boolean lowercase", "boolean", "boolean"},
		{"bool lowercase", "bool", "boolean"},
		{"BIT", "BIT", "boolean"},

		// Date/time types
		{"DATE", "DATE", "datetime"},
		{"TIME", "TIME", "datetime"},
		{"DATETIME", "DATETIME", "datetime"},
		{"TIMESTAMP", "TIMESTAMP", "datetime"},
		{"TIMESTAMPTZ", "TIMESTAMPTZ", "datetime"},
		{"timestamp lowercase", "timestamp", "datetime"},
		{"timestamptz lowercase", "timestamptz", "datetime"},
		{"INTERVAL", "INTERVAL", "datetime"},
		{"YEAR", "YEAR", "datetime"},

		// Text types
		{"TEXT", "TEXT", "string"},
		{"VARCHAR", "VARCHAR", "string"},
		{"CHAR", "CHAR", "string"},
		{"CHARACTER VARYING", "CHARACTER VARYING", "string"},
		{"text lowercase", "text", "string"},
		{"varchar lowercase", "varchar", "string"},
		{"UUID", "UUID", "string"},
		{"CITEXT", "CITEXT", "string"},
		{"NAME", "NAME", "string"},

		// MySQL specific text
		{"TINYTEXT", "TINYTEXT", "string"},
		{"MEDIUMTEXT", "MEDIUMTEXT", "string"},
		{"LONGTEXT", "LONGTEXT", "string"},
		{"ENUM", "ENUM", "string"},
		{"SET", "SET", "string"},

		// JSON types
		{"JSON", "JSON", "string"},
		{"JSONB", "JSONB", "string"},
		{"json lowercase", "json", "string"},
		{"jsonb lowercase", "jsonb", "string"},

		// Binary types
		{"BLOB", "BLOB", "string"},
		{"BYTEA", "BYTEA", "string"},
		{"bytea lowercase", "bytea", "string"},
		{"BINARY", "BINARY", "string"},
		{"VARBINARY", "VARBINARY", "string"},
		{"TINYBLOB", "TINYBLOB", "string"},
		{"MEDIUMBLOB", "MEDIUMBLOB", "string"},
		{"LONGBLOB", "LONGBLOB", "string"},

		// Types with parameters
		{"VARCHAR(255)", "VARCHAR(255)", "string"},
		{"CHAR(10)", "CHAR(10)", "string"},
		{"INT(11)", "INT(11)", "number"},
		{"DECIMAL(10,2)", "DECIMAL(10,2)", "number"},
		{"NUMERIC(15,4)", "NUMERIC(15,4)", "number"},
		{"FLOAT(8)", "FLOAT(8)", "number"},
		{"DOUBLE(16,8)", "DOUBLE(16,8)", "number"},

		// Unknown types default to string
		{"CUSTOM_TYPE", "CUSTOM_TYPE", "string"},
		{"some_unknown", "some_unknown", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapColumnType(tt.input)
			assert.Equal(t, tt.expected, result, "MapColumnType(%q)", tt.input)
		})
	}
}

func TestDriverCapabilities(t *testing.T) {
	caps := DriverCapabilities{
		SupportsSSH:          true,
		SupportsSSL:          true,
		SupportsSchemas:      true,
		SupportsCTEs:         true,
		SupportsJSON:         true,
		SupportsArrays:       false,
		SupportsWindowFuncs:  true,
		SupportsTransactions: true,
		DefaultPort:          5432,
	}

	assert.True(t, caps.SupportsSSH)
	assert.True(t, caps.SupportsSSL)
	assert.True(t, caps.SupportsSchemas)
	assert.True(t, caps.SupportsCTEs)
	assert.True(t, caps.SupportsJSON)
	assert.False(t, caps.SupportsArrays)
	assert.True(t, caps.SupportsWindowFuncs)
	assert.True(t, caps.SupportsTransactions)
	assert.Equal(t, 5432, caps.DefaultPort)
}

func TestTunnelConfig(t *testing.T) {
	tunnel := TunnelConfig{
		Enabled:    true,
		Host:       "bastion.example.com",
		Port:       22,
		User:       "admin",
		AuthMethod: "ssh-key",
		PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\n...",
	}

	assert.True(t, tunnel.Enabled)
	assert.Equal(t, "bastion.example.com", tunnel.Host)
	assert.Equal(t, 22, tunnel.Port)
	assert.Equal(t, "admin", tunnel.User)
	assert.Equal(t, "ssh-key", tunnel.AuthMethod)
}

func TestConfigSSL(t *testing.T) {
	config := Config{
		Engine:        "postgres",
		Host:          "db.example.com",
		Port:          5432,
		Database:      "app",
		Username:      "user",
		Password:      "pass",
		SSL:           true,
		SSLMode:       "verify-full",
		SSLRootCert:   "/path/to/ca.pem",
		SSLClientCert: "/path/to/client.pem",
		SSLClientKey:  "/path/to/client.key",
	}

	assert.Equal(t, "postgres", config.Engine)
	assert.True(t, config.SSL)
	assert.Equal(t, "verify-full", config.SSLMode)
	assert.Equal(t, "/path/to/ca.pem", config.SSLRootCert)
}

func TestListDrivers(t *testing.T) {
	// Register a test driver if not already registered
	Register("test_driver", func() Driver { return nil })

	drivers := ListDrivers()
	assert.Contains(t, drivers, "test_driver")
}

func TestGetUnknownDriver(t *testing.T) {
	_, err := Get("nonexistent_driver_12345")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown driver")
}
