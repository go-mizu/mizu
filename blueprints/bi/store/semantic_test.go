package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSemanticTypeConstants(t *testing.T) {
	// Verify key semantic types are defined correctly
	assert.Equal(t, "type/PK", SemanticPK)
	assert.Equal(t, "type/FK", SemanticFK)
	assert.Equal(t, "type/Price", SemanticPrice)
	assert.Equal(t, "type/Currency", SemanticCurrency)
	assert.Equal(t, "type/Percentage", SemanticPercent)
	assert.Equal(t, "type/Email", SemanticEmail)
	assert.Equal(t, "type/URL", SemanticURL)
	assert.Equal(t, "type/CreationDate", SemanticCreated)
	assert.Equal(t, "type/Latitude", SemanticLatitude)
	assert.Equal(t, "type/Longitude", SemanticLongitude)
}

func TestValidSemanticTypes(t *testing.T) {
	types := ValidSemanticTypes()

	// Check that all expected types are present
	expectedTypes := []string{
		SemanticPK, SemanticFK,
		SemanticPrice, SemanticCurrency, SemanticScore, SemanticPercent, SemanticQuantity,
		SemanticName, SemanticTitle, SemanticDescription, SemanticCategory, SemanticURL, SemanticEmail, SemanticPhone,
		SemanticCreated, SemanticUpdated, SemanticJoined, SemanticBirthday,
		SemanticLatitude, SemanticLongitude, SemanticZipCode, SemanticCity, SemanticState, SemanticCountry, SemanticAddress,
	}

	for _, expected := range expectedTypes {
		assert.Contains(t, types, expected, "ValidSemanticTypes should contain %s", expected)
	}

	// Check total count
	assert.Equal(t, len(expectedTypes), len(types))
}

func TestDataSourceModel(t *testing.T) {
	ds := DataSource{
		ID:       "ds-123",
		Name:     "Production DB",
		Engine:   "postgres",
		Host:     "db.example.com",
		Port:     5432,
		Database: "production",
		Username: "app_user",
		Password: "secret",

		// SSL Configuration
		SSL:           true,
		SSLMode:       "verify-full",
		SSLRootCert:   "-----BEGIN CERTIFICATE-----",
		SSLClientCert: "-----BEGIN CERTIFICATE-----",
		SSLClientKey:  "-----BEGIN PRIVATE KEY-----",

		// SSH Tunnel
		TunnelEnabled:    true,
		TunnelHost:       "bastion.example.com",
		TunnelPort:       22,
		TunnelUser:       "tunnel_user",
		TunnelAuthMethod: "ssh-key",

		// Schema Filter
		SchemaFilterType:     "inclusion",
		SchemaFilterPatterns: []string{"public", "analytics"},

		// Sync
		AutoSync:     true,
		SyncSchedule: "0 * * * *",

		// Cache
		CacheTTL: 3600,

		// Pool
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
		ConnMaxIdleTime: 60,
	}

	assert.Equal(t, "ds-123", ds.ID)
	assert.Equal(t, "Production DB", ds.Name)
	assert.Equal(t, "postgres", ds.Engine)
	assert.True(t, ds.SSL)
	assert.Equal(t, "verify-full", ds.SSLMode)
	assert.True(t, ds.TunnelEnabled)
	assert.Equal(t, "bastion.example.com", ds.TunnelHost)
	assert.Equal(t, "inclusion", ds.SchemaFilterType)
	assert.Len(t, ds.SchemaFilterPatterns, 2)
	assert.True(t, ds.AutoSync)
	assert.Equal(t, 3600, ds.CacheTTL)
	assert.Equal(t, 25, ds.MaxOpenConns)
}

func TestColumnModel(t *testing.T) {
	col := Column{
		ID:          "col-123",
		TableID:     "tbl-456",
		Name:        "email",
		DisplayName: "Email Address",
		Type:        "varchar(255)",
		MappedType:  "string",
		Semantic:    SemanticEmail,
		Description: "User's email address",
		Position:    2,
		Visibility:  "everywhere",
		Nullable:    false,
		PrimaryKey:  false,
		ForeignKey:  false,

		// Fingerprint data
		DistinctCount: 1000,
		NullCount:     5,
		MinValue:      "a@example.com",
		MaxValue:      "z@example.com",
		AvgLength:     25.5,

		// Cached values
		CachedValues: []string{"user1@example.com", "user2@example.com"},
	}

	assert.Equal(t, "col-123", col.ID)
	assert.Equal(t, "email", col.Name)
	assert.Equal(t, "Email Address", col.DisplayName)
	assert.Equal(t, "varchar(255)", col.Type)
	assert.Equal(t, "string", col.MappedType)
	assert.Equal(t, SemanticEmail, col.Semantic)
	assert.Equal(t, "everywhere", col.Visibility)
	assert.False(t, col.Nullable)
	assert.Equal(t, int64(1000), col.DistinctCount)
	assert.Equal(t, int64(5), col.NullCount)
	assert.Equal(t, 25.5, col.AvgLength)
	assert.Len(t, col.CachedValues, 2)
}

func TestColumnVisibility(t *testing.T) {
	// Test default visibility
	col := Column{Name: "test"}
	assert.Equal(t, "", col.Visibility) // Empty by default in Go

	// Test different visibility options
	visibilities := []string{"everywhere", "detail_only", "hidden"}
	for _, v := range visibilities {
		col.Visibility = v
		assert.Equal(t, v, col.Visibility)
	}
}
