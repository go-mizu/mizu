package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// SettingsHandler handles project settings API endpoints.
type SettingsHandler struct {
	store    *postgres.Store
	settings *ProjectSettings
}

// ProjectSettings holds all project configuration.
type ProjectSettings struct {
	Project  ProjectConfig  `json:"project"`
	API      APIConfig      `json:"api"`
	Auth     AuthConfig     `json:"auth"`
	Database DatabaseConfig `json:"database"`
	Storage  StorageConfig  `json:"storage"`
}

// ProjectConfig represents basic project configuration.
type ProjectConfig struct {
	ID        string    `json:"project_id"`
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// APIConfig represents API configuration.
type APIConfig struct {
	MaxRows              int      `json:"max_rows"`
	ExposeSchemas        []string `json:"expose_schemas"`
	DBExtraSearchPath    string   `json:"db_extra_search_path"`
	JWTSecret            string   `json:"jwt_secret"`
	JWTExp               int      `json:"jwt_exp"`
	EnableSchemaDetect   bool     `json:"enable_schema_detect"`
	EnableOpenAPI        bool     `json:"enable_openapi"`
	EnableGraphQL        bool     `json:"enable_graphql"`
	RateLimitEnabled     bool     `json:"rate_limit_enabled"`
	RateLimitPerSecond   int      `json:"rate_limit_per_second"`
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	SiteURL                 string   `json:"site_url"`
	DisableSignup           bool     `json:"disable_signup"`
	ExternalEmailEnabled    bool     `json:"external_email_enabled"`
	ExternalPhoneEnabled    bool     `json:"external_phone_enabled"`
	MFAEnabled              bool     `json:"mfa_enabled"`
	DoubleConfirmChanges    bool     `json:"double_confirm_changes"`
	EnableAutoConfirm       bool     `json:"enable_auto_confirm"`
	RefreshTokenRotation    bool     `json:"refresh_token_rotation"`
	PasswordMinLength       int      `json:"password_min_length"`
	AllowedRedirectURLs     []string `json:"allowed_redirect_urls"`
	JWTExp                  int      `json:"jwt_exp"`
	SessionTimeoutSeconds   int      `json:"session_timeout_seconds"`
	MaxEnrolledFactors      int      `json:"max_enrolled_factors"`
}

// DatabaseConfig represents database configuration.
type DatabaseConfig struct {
	PoolMode             string `json:"pool_mode"`
	MaxConnections       int    `json:"max_connections"`
	StatementTimeout     int    `json:"statement_timeout_ms"`
	DefaultTransactionIso string `json:"default_transaction_isolation"`
	LogMinDurationMs     int    `json:"log_min_duration_ms"`
	LogStatement         string `json:"log_statement"`
	SharedBuffers        string `json:"shared_buffers"`
	WorkMem              string `json:"work_mem"`
	MaintenanceWorkMem   string `json:"maintenance_work_mem"`
}

// StorageConfig represents storage configuration.
type StorageConfig struct {
	FileSizeLimit        int64    `json:"file_size_limit"`
	ImageTransformations bool     `json:"image_transformations"`
	AllowedMimeTypes     []string `json:"allowed_mime_types"`
	S3Enabled            bool     `json:"s3_enabled"`
	S3Bucket             string   `json:"s3_bucket"`
	S3Region             string   `json:"s3_region"`
}

// NewSettingsHandler creates a new settings handler.
func NewSettingsHandler(store *postgres.Store) *SettingsHandler {
	return &SettingsHandler{
		store: store,
		settings: &ProjectSettings{
			Project: ProjectConfig{
				ID:        "localbase",
				Name:      "LocalBase",
				Region:    "local",
				Status:    "active",
				CreatedAt: time.Now(),
			},
			API: APIConfig{
				MaxRows:            1000,
				ExposeSchemas:      []string{"public"},
				DBExtraSearchPath:  "public,extensions",
				JWTSecret:          "your-super-secret-jwt-token-with-at-least-32-characters-long",
				JWTExp:             3600,
				EnableSchemaDetect: true,
				EnableOpenAPI:      true,
				EnableGraphQL:      false,
				RateLimitEnabled:   true,
				RateLimitPerSecond: 100,
			},
			Auth: AuthConfig{
				SiteURL:               "http://localhost:3000",
				DisableSignup:         false,
				ExternalEmailEnabled:  true,
				ExternalPhoneEnabled:  false,
				MFAEnabled:            true,
				DoubleConfirmChanges:  true,
				EnableAutoConfirm:     false,
				RefreshTokenRotation:  true,
				PasswordMinLength:     6,
				AllowedRedirectURLs:   []string{"http://localhost:3000"},
				JWTExp:                3600,
				SessionTimeoutSeconds: 86400,
				MaxEnrolledFactors:    10,
			},
			Database: DatabaseConfig{
				PoolMode:             "transaction",
				MaxConnections:       100,
				StatementTimeout:     120000,
				DefaultTransactionIso: "read committed",
				LogMinDurationMs:     1000,
				LogStatement:         "ddl",
				SharedBuffers:        "128MB",
				WorkMem:              "4MB",
				MaintenanceWorkMem:   "64MB",
			},
			Storage: StorageConfig{
				FileSizeLimit:        52428800,
				ImageTransformations: true,
				AllowedMimeTypes:     []string{"image/*", "application/pdf"},
				S3Enabled:            false,
				S3Bucket:             "",
				S3Region:             "",
			},
		},
	}
}

// GetProjectSettings returns the project configuration.
func (h *SettingsHandler) GetProjectSettings(c *mizu.Ctx) error {
	return c.JSON(200, h.settings.Project)
}

// UpdateProjectSettings updates project configuration.
func (h *SettingsHandler) UpdateProjectSettings(c *mizu.Ctx) error {
	var req ProjectConfig
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name != "" {
		h.settings.Project.Name = req.Name
	}

	return c.JSON(200, h.settings.Project)
}

// GetAPISettings returns API configuration.
func (h *SettingsHandler) GetAPISettings(c *mizu.Ctx) error {
	// Mask JWT secret
	config := h.settings.API
	if len(config.JWTSecret) > 8 {
		config.JWTSecret = config.JWTSecret[:4] + "****" + config.JWTSecret[len(config.JWTSecret)-4:]
	}
	return c.JSON(200, config)
}

// UpdateAPISettings updates API configuration.
func (h *SettingsHandler) UpdateAPISettings(c *mizu.Ctx) error {
	var req APIConfig
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.MaxRows > 0 {
		h.settings.API.MaxRows = req.MaxRows
	}
	if len(req.ExposeSchemas) > 0 {
		h.settings.API.ExposeSchemas = req.ExposeSchemas
	}
	if req.JWTExp > 0 {
		h.settings.API.JWTExp = req.JWTExp
	}
	h.settings.API.EnableSchemaDetect = req.EnableSchemaDetect
	h.settings.API.EnableOpenAPI = req.EnableOpenAPI
	h.settings.API.EnableGraphQL = req.EnableGraphQL
	h.settings.API.RateLimitEnabled = req.RateLimitEnabled
	if req.RateLimitPerSecond > 0 {
		h.settings.API.RateLimitPerSecond = req.RateLimitPerSecond
	}

	return c.JSON(200, h.settings.API)
}

// GetAuthSettings returns auth configuration.
func (h *SettingsHandler) GetAuthSettings(c *mizu.Ctx) error {
	return c.JSON(200, h.settings.Auth)
}

// UpdateAuthSettings updates auth configuration.
func (h *SettingsHandler) UpdateAuthSettings(c *mizu.Ctx) error {
	var req AuthConfig
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.SiteURL != "" {
		h.settings.Auth.SiteURL = req.SiteURL
	}
	h.settings.Auth.DisableSignup = req.DisableSignup
	h.settings.Auth.ExternalEmailEnabled = req.ExternalEmailEnabled
	h.settings.Auth.ExternalPhoneEnabled = req.ExternalPhoneEnabled
	h.settings.Auth.MFAEnabled = req.MFAEnabled
	h.settings.Auth.DoubleConfirmChanges = req.DoubleConfirmChanges
	h.settings.Auth.EnableAutoConfirm = req.EnableAutoConfirm
	h.settings.Auth.RefreshTokenRotation = req.RefreshTokenRotation
	if req.PasswordMinLength > 0 {
		h.settings.Auth.PasswordMinLength = req.PasswordMinLength
	}
	if len(req.AllowedRedirectURLs) > 0 {
		h.settings.Auth.AllowedRedirectURLs = req.AllowedRedirectURLs
	}
	if req.JWTExp > 0 {
		h.settings.Auth.JWTExp = req.JWTExp
	}
	if req.SessionTimeoutSeconds > 0 {
		h.settings.Auth.SessionTimeoutSeconds = req.SessionTimeoutSeconds
	}
	if req.MaxEnrolledFactors > 0 {
		h.settings.Auth.MaxEnrolledFactors = req.MaxEnrolledFactors
	}

	return c.JSON(200, h.settings.Auth)
}

// GetDatabaseSettings returns database configuration.
func (h *SettingsHandler) GetDatabaseSettings(c *mizu.Ctx) error {
	return c.JSON(200, h.settings.Database)
}

// UpdateDatabaseSettings updates database configuration.
func (h *SettingsHandler) UpdateDatabaseSettings(c *mizu.Ctx) error {
	var req DatabaseConfig
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.PoolMode != "" {
		h.settings.Database.PoolMode = req.PoolMode
	}
	if req.MaxConnections > 0 {
		h.settings.Database.MaxConnections = req.MaxConnections
	}
	if req.StatementTimeout > 0 {
		h.settings.Database.StatementTimeout = req.StatementTimeout
	}
	if req.LogMinDurationMs >= 0 {
		h.settings.Database.LogMinDurationMs = req.LogMinDurationMs
	}
	if req.LogStatement != "" {
		h.settings.Database.LogStatement = req.LogStatement
	}

	return c.JSON(200, h.settings.Database)
}

// GetStorageSettings returns storage configuration.
func (h *SettingsHandler) GetStorageSettings(c *mizu.Ctx) error {
	return c.JSON(200, h.settings.Storage)
}

// UpdateStorageSettings updates storage configuration.
func (h *SettingsHandler) UpdateStorageSettings(c *mizu.Ctx) error {
	var req StorageConfig
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.FileSizeLimit > 0 {
		h.settings.Storage.FileSizeLimit = req.FileSizeLimit
	}
	h.settings.Storage.ImageTransformations = req.ImageTransformations
	if len(req.AllowedMimeTypes) > 0 {
		h.settings.Storage.AllowedMimeTypes = req.AllowedMimeTypes
	}
	h.settings.Storage.S3Enabled = req.S3Enabled
	if req.S3Bucket != "" {
		h.settings.Storage.S3Bucket = req.S3Bucket
	}
	if req.S3Region != "" {
		h.settings.Storage.S3Region = req.S3Region
	}

	return c.JSON(200, h.settings.Storage)
}

// GetAllSettings returns all settings.
func (h *SettingsHandler) GetAllSettings(c *mizu.Ctx) error {
	// Mask sensitive data
	settings := *h.settings
	if len(settings.API.JWTSecret) > 8 {
		settings.API.JWTSecret = settings.API.JWTSecret[:4] + "****" + settings.API.JWTSecret[len(settings.API.JWTSecret)-4:]
	}
	return c.JSON(200, settings)
}
