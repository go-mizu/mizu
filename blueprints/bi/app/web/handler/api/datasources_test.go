package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-mizu/blueprints/bi/store"
)

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name         string
		errorMsg     string
		expectedCode string
	}{
		{
			name:         "connection refused",
			errorMsg:     "dial tcp: connection refused",
			expectedCode: "CONNECTION_REFUSED",
		},
		{
			name:         "password auth failed",
			errorMsg:     "password authentication failed for user",
			expectedCode: "AUTH_FAILED",
		},
		{
			name:         "access denied",
			errorMsg:     "Access denied for user 'root'@'localhost'",
			expectedCode: "AUTH_FAILED",
		},
		{
			name:         "database not found",
			errorMsg:     "database \"mydb\" does not exist",
			expectedCode: "DATABASE_NOT_FOUND",
		},
		{
			name:         "unknown database mysql",
			errorMsg:     "Unknown database 'testdb'",
			expectedCode: "DATABASE_NOT_FOUND",
		},
		{
			name:         "ssl error",
			errorMsg:     "SSL connection error: certificate verify failed",
			expectedCode: "SSL_ERROR",
		},
		{
			name:         "tls error",
			errorMsg:     "tls: handshake failure",
			expectedCode: "SSL_ERROR",
		},
		{
			name:         "timeout",
			errorMsg:     "i/o timeout",
			expectedCode: "TIMEOUT",
		},
		{
			name:         "permission denied",
			errorMsg:     "permission denied for table users",
			expectedCode: "PERMISSION_DENIED",
		},
		{
			name:         "unknown error",
			errorMsg:     "some other error occurred",
			expectedCode: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMsg)
			code := categorizeError(err)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestGetSuggestions(t *testing.T) {
	tests := []struct {
		name            string
		errorMsg        string
		expectNonEmpty  bool
		containsPartial string
	}{
		{
			name:            "connection refused",
			errorMsg:        "connection refused",
			expectNonEmpty:  true,
			containsPartial: "database server is running",
		},
		{
			name:            "auth failed",
			errorMsg:        "password authentication failed",
			expectNonEmpty:  true,
			containsPartial: "username and password",
		},
		{
			name:            "database not found",
			errorMsg:        "database does not exist",
			expectNonEmpty:  true,
			containsPartial: "database name is spelled",
		},
		{
			name:            "ssl error",
			errorMsg:        "ssl certificate error",
			expectNonEmpty:  true,
			containsPartial: "SSL configuration",
		},
		{
			name:            "timeout",
			errorMsg:        "connection timeout",
			expectNonEmpty:  true,
			containsPartial: "network connectivity",
		},
		{
			name:           "unknown error",
			errorMsg:       "some random error",
			expectNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMsg)
			suggestions := getSuggestions(err)

			if tt.expectNonEmpty {
				assert.NotEmpty(t, suggestions)
				found := false
				for _, s := range suggestions {
					if contains(s, tt.containsPartial) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected suggestion containing %q", tt.containsPartial)
			} else {
				assert.Empty(t, suggestions)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFilterSchemas(t *testing.T) {
	schemas := []string{"public", "analytics", "internal", "pg_catalog"}

	tests := []struct {
		name     string
		patterns []string
		include  bool
		expected []string
	}{
		{
			name:     "include public and analytics",
			patterns: []string{"public", "analytics"},
			include:  true,
			expected: []string{"public", "analytics"},
		},
		{
			name:     "exclude internal",
			patterns: []string{"internal", "pg_catalog"},
			include:  false,
			expected: []string{"public", "analytics"},
		},
		{
			name:     "case insensitive include",
			patterns: []string{"PUBLIC", "ANALYTICS"},
			include:  true,
			expected: []string{"public", "analytics"},
		},
		{
			name:     "include non-existent",
			patterns: []string{"nonexistent"},
			include:  true,
			expected: nil,
		},
		{
			name:     "exclude non-existent",
			patterns: []string{"nonexistent"},
			include:  false,
			expected: []string{"public", "analytics", "internal", "pg_catalog"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterSchemas(schemas, tt.patterns, tt.include)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferSemanticType(t *testing.T) {
	tests := []struct {
		name       string
		colName    string
		mappedType string
		isPK       bool
		isFK       bool
		expected   string
	}{
		// Primary and Foreign Keys
		{"primary key", "id", "number", true, false, store.SemanticPK},
		{"foreign key", "user_id", "number", false, true, store.SemanticFK},

		// Date patterns
		{"created_at datetime", "created_at", "datetime", false, false, store.SemanticCreated},
		{"updated_at datetime", "updated_at", "datetime", false, false, store.SemanticUpdated},
		{"joined_at datetime", "joined_at", "datetime", false, false, store.SemanticJoined},
		{"birth_date datetime", "birth_date", "datetime", false, false, store.SemanticBirthday},

		// Number patterns
		{"price column", "price", "number", false, false, store.SemanticPrice},
		{"unit_price", "unit_price", "number", false, false, store.SemanticPrice},
		{"total_cost", "total_cost", "number", false, false, store.SemanticPrice},
		{"total_amount", "total_amount", "number", false, false, store.SemanticPrice},
		{"tax_percent", "tax_percent", "number", false, false, store.SemanticPercent},
		{"discount_rate", "discount_rate", "number", false, false, store.SemanticPercent},
		{"quantity", "quantity", "number", false, false, store.SemanticQuantity},
		{"item_qty", "item_qty", "number", false, false, store.SemanticQuantity},
		{"order_count", "order_count", "number", false, false, store.SemanticQuantity},
		{"score", "score", "number", false, false, store.SemanticScore},
		{"rating", "rating", "number", false, false, store.SemanticScore},
		{"latitude", "lat", "number", false, false, store.SemanticLatitude},
		{"longitude", "lng", "number", false, false, store.SemanticLongitude},
		{"longitude alt", "lon", "number", false, false, store.SemanticLongitude},

		// String patterns
		{"name column", "name", "string", false, false, store.SemanticName},
		{"first_name", "first_name", "string", false, false, store.SemanticName},
		{"title column", "title", "string", false, false, store.SemanticTitle},
		{"description", "description", "string", false, false, store.SemanticDescription},
		{"short_desc", "short_desc", "string", false, false, store.SemanticDescription},
		{"email column", "email", "string", false, false, store.SemanticEmail},
		{"email_address", "email_address", "string", false, false, store.SemanticEmail},
		{"phone", "phone", "string", false, false, store.SemanticPhone},
		{"telephone", "telephone", "string", false, false, store.SemanticPhone},
		{"url column", "url", "string", false, false, store.SemanticURL},
		{"website_link", "website_link", "string", false, false, store.SemanticURL},
		{"category", "category", "string", false, false, store.SemanticCategory},
		{"product_type", "product_type", "string", false, false, store.SemanticCategory},
		{"status", "status", "string", false, false, store.SemanticCategory},
		{"zip_code", "zip_code", "string", false, false, store.SemanticZipCode},
		{"postal_code", "postal_code", "string", false, false, store.SemanticZipCode},
		{"city", "city", "string", false, false, store.SemanticCity},
		{"state", "state", "string", false, false, store.SemanticState},
		{"province", "province", "string", false, false, store.SemanticState},
		{"country", "country", "string", false, false, store.SemanticCountry},
		{"address", "address", "string", false, false, store.SemanticAddress},

		// No match
		{"unknown column", "foobar", "string", false, false, ""},
		{"unknown number", "some_value", "number", false, false, ""},
		{"unknown datetime", "some_date", "datetime", false, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferSemanticType(tt.colName, tt.mappedType, tt.isPK, tt.isFK)
			assert.Equal(t, tt.expected, result, "inferSemanticType(%q, %q, %v, %v)", tt.colName, tt.mappedType, tt.isPK, tt.isFK)
		})
	}
}

func TestDataSourceToConfig(t *testing.T) {
	ds := &store.DataSource{
		Engine:          "postgres",
		Host:            "db.example.com",
		Port:            5432,
		Database:        "mydb",
		Username:        "user",
		Password:        "pass",
		SSL:             true,
		SSLMode:         "verify-full",
		SSLRootCert:     "/path/to/ca.pem",
		SSLClientCert:   "/path/to/client.pem",
		SSLClientKey:    "/path/to/client.key",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
		ConnMaxIdleTime: 60,
		Options: map[string]string{
			"application_name": "bi_tool",
		},
	}

	config := dataSourceToConfig(ds)

	assert.Equal(t, "postgres", config.Engine)
	assert.Equal(t, "db.example.com", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "mydb", config.Database)
	assert.Equal(t, "user", config.Username)
	assert.Equal(t, "pass", config.Password)
	assert.True(t, config.SSL)
	assert.Equal(t, "verify-full", config.SSLMode)
	assert.Equal(t, "/path/to/ca.pem", config.SSLRootCert)
	assert.Equal(t, 25, config.MaxOpenConns)
	assert.Equal(t, 5, config.MaxIdleConns)
	assert.Equal(t, "bi_tool", config.Options["application_name"])
}
