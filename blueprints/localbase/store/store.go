package store

import (
	"context"
	"time"
)

// Store defines the interface for all storage operations.
type Store interface {
	// Schema management
	Ensure(ctx context.Context) error
	CreateExtensions(ctx context.Context) error
	Close() error

	// Seeding
	SeedUsers(ctx context.Context) error
	SeedStorage(ctx context.Context) error
	SeedTables(ctx context.Context) error

	// Feature stores
	Auth() AuthStore
	Storage() StorageStore
	Database() DatabaseStore
	Functions() FunctionsStore
	Realtime() RealtimeStore
}

// ========== Auth Types ==========

// User represents an auth user.
type User struct {
	ID                 string            `json:"id"`
	Email              string            `json:"email,omitempty"`
	Phone              string            `json:"phone,omitempty"`
	EncryptedPassword  string            `json:"-"`
	EmailConfirmedAt   *time.Time        `json:"email_confirmed_at,omitempty"`
	PhoneConfirmedAt   *time.Time        `json:"phone_confirmed_at,omitempty"`
	AppMetadata        map[string]any    `json:"app_metadata"`
	UserMetadata       map[string]any    `json:"user_metadata"`
	IsSuperAdmin       bool              `json:"is_super_admin"`
	Role               string            `json:"role"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	LastSignInAt       *time.Time        `json:"last_sign_in_at,omitempty"`
	BannedUntil        *time.Time        `json:"banned_until,omitempty"`
	ConfirmationToken  string            `json:"-"`
	RecoveryToken      string            `json:"-"`
	Identities         []*Identity       `json:"identities,omitempty"`
}

// Session represents a user session.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	FactorID     string    `json:"factor_id,omitempty"`
	AAL          string    `json:"aal"`
	NotAfter     time.Time `json:"not_after"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    int64     `json:"expires_at,omitempty"`
}

// RefreshToken represents a refresh token.
type RefreshToken struct {
	ID        int64     `json:"id"`
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Parent    string    `json:"parent,omitempty"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MFAFactor represents an MFA factor.
type MFAFactor struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	FriendlyName string    `json:"friendly_name,omitempty"`
	FactorType   string    `json:"factor_type"` // totp, webauthn
	Status       string    `json:"status"`      // unverified, verified
	Secret       string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Identity represents an OAuth identity.
type Identity struct {
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	Provider     string         `json:"provider"`
	ProviderID   string         `json:"provider_id"`
	IdentityData map[string]any `json:"identity_data"`
	LastSignInAt *time.Time     `json:"last_sign_in_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// AuthStore defines auth storage operations.
type AuthStore interface {
	// Users
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByPhone(ctx context.Context, phone string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, page, perPage int) ([]*User, int, error)

	// Sessions
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, userID string) error

	// Refresh Tokens
	CreateRefreshToken(ctx context.Context, token *RefreshToken) error
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RotateRefreshToken(ctx context.Context, oldToken, newToken string) error

	// MFA
	CreateMFAFactor(ctx context.Context, factor *MFAFactor) error
	GetMFAFactor(ctx context.Context, id string) (*MFAFactor, error)
	GetUserMFAFactors(ctx context.Context, userID string) ([]*MFAFactor, error)
	UpdateMFAFactor(ctx context.Context, factor *MFAFactor) error
	DeleteMFAFactor(ctx context.Context, id string) error

	// Identities
	CreateIdentity(ctx context.Context, identity *Identity) error
	GetIdentity(ctx context.Context, provider, providerID string) (*Identity, error)
	GetUserIdentities(ctx context.Context, userID string) ([]*Identity, error)
	DeleteIdentity(ctx context.Context, id string) error
}

// ========== Storage Types ==========

// Bucket represents a storage bucket.
type Bucket struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Public           bool      `json:"public"`
	FileSizeLimit    *int64    `json:"file_size_limit,omitempty"`
	AllowedMimeTypes []string  `json:"allowed_mime_types,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Object represents a storage object.
type Object struct {
	ID             string            `json:"id"`
	BucketID       string            `json:"bucket_id"`
	Name           string            `json:"name"`
	Owner          string            `json:"owner,omitempty"`
	PathTokens     []string          `json:"path_tokens,omitempty"`
	Version        string            `json:"version,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ContentType    string            `json:"content_type,omitempty"`
	Size           int64             `json:"size"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	LastAccessedAt *time.Time        `json:"last_accessed_at,omitempty"`
}

// StorageStore defines storage operations.
type StorageStore interface {
	// Buckets
	CreateBucket(ctx context.Context, bucket *Bucket) error
	GetBucket(ctx context.Context, id string) (*Bucket, error)
	GetBucketByName(ctx context.Context, name string) (*Bucket, error)
	ListBuckets(ctx context.Context) ([]*Bucket, error)
	UpdateBucket(ctx context.Context, bucket *Bucket) error
	DeleteBucket(ctx context.Context, id string) error

	// Objects
	CreateObject(ctx context.Context, obj *Object) error
	GetObject(ctx context.Context, bucketID, name string) (*Object, error)
	ListObjects(ctx context.Context, bucketID, prefix string, limit, offset int) ([]*Object, error)
	UpdateObject(ctx context.Context, obj *Object) error
	DeleteObject(ctx context.Context, bucketID, name string) error
	MoveObject(ctx context.Context, bucketID, srcName, dstName string) error
	CopyObject(ctx context.Context, srcBucketID, srcName, dstBucketID, dstName string) error
}

// ========== Database Types ==========

// Table represents a database table.
type Table struct {
	ID         int64     `json:"id"`
	Schema     string    `json:"schema"`
	Name       string    `json:"name"`
	RowCount   int64     `json:"row_count"`
	SizeBytes  int64     `json:"size_bytes"`
	Comment    string    `json:"comment,omitempty"`
	RLSEnabled bool      `json:"rls_enabled"`
	Columns    []*Column `json:"columns,omitempty"`
}

// Column represents a table column.
type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"default_value,omitempty"`
	IsNullable   bool   `json:"is_nullable"`
	IsPrimaryKey bool   `json:"is_primary_key"`
	IsUnique     bool   `json:"is_unique"`
	Comment      string `json:"comment,omitempty"`
}

// Index represents a database index.
type Index struct {
	Name      string   `json:"name"`
	Schema    string   `json:"schema"`
	Table     string   `json:"table"`
	Columns   []string `json:"columns"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
	Type      string   `json:"type"` // btree, hash, gin, gist, etc.
}

// ForeignKey represents a foreign key relationship.
type ForeignKey struct {
	Name           string `json:"name"`
	Schema         string `json:"schema"`
	Table          string `json:"table"`
	Column         string `json:"column"`
	TargetSchema   string `json:"target_schema"`
	TargetTable    string `json:"target_table"`
	TargetColumn   string `json:"target_column"`
	OnDelete       string `json:"on_delete"`
	OnUpdate       string `json:"on_update"`
}

// Policy represents a Row Level Security policy.
type Policy struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	Command    string `json:"command"` // ALL, SELECT, INSERT, UPDATE, DELETE
	Definition string `json:"definition"`
	CheckExpr  string `json:"check_expression,omitempty"`
	Roles      []string `json:"roles"`
}

// Extension represents a PostgreSQL extension.
type Extension struct {
	Name           string `json:"name"`
	InstalledVersion string `json:"installed_version,omitempty"`
	DefaultVersion string `json:"default_version"`
	Comment        string `json:"comment,omitempty"`
}

// QueryResult represents query execution results.
type QueryResult struct {
	Columns   []string                 `json:"columns"`
	Rows      []map[string]interface{} `json:"rows"`
	RowCount  int                      `json:"row_count"`
	Duration  float64                  `json:"duration_ms"`
}

// DatabaseStore defines database operations.
type DatabaseStore interface {
	// Tables
	ListTables(ctx context.Context, schema string) ([]*Table, error)
	GetTable(ctx context.Context, schema, name string) (*Table, error)
	CreateTable(ctx context.Context, schema, name string, columns []*Column) error
	DropTable(ctx context.Context, schema, name string) error

	// Columns
	ListColumns(ctx context.Context, schema, table string) ([]*Column, error)
	AddColumn(ctx context.Context, schema, table string, column *Column) error
	AlterColumn(ctx context.Context, schema, table string, column *Column) error
	DropColumn(ctx context.Context, schema, table, column string) error

	// Indexes
	ListIndexes(ctx context.Context, schema, table string) ([]*Index, error)
	CreateIndex(ctx context.Context, index *Index) error
	DropIndex(ctx context.Context, schema, name string) error

	// Foreign Keys
	ListForeignKeys(ctx context.Context, schema, table string) ([]*ForeignKey, error)
	CreateForeignKey(ctx context.Context, fk *ForeignKey) error
	DropForeignKey(ctx context.Context, schema, table, name string) error

	// RLS Policies
	ListPolicies(ctx context.Context, schema, table string) ([]*Policy, error)
	CreatePolicy(ctx context.Context, policy *Policy) error
	DropPolicy(ctx context.Context, schema, table, name string) error
	EnableRLS(ctx context.Context, schema, table string) error
	DisableRLS(ctx context.Context, schema, table string) error

	// Extensions
	ListExtensions(ctx context.Context) ([]*Extension, error)
	EnableExtension(ctx context.Context, name string) error
	DisableExtension(ctx context.Context, name string) error

	// Query execution
	Query(ctx context.Context, sql string, params ...interface{}) (*QueryResult, error)
	Exec(ctx context.Context, sql string, params ...interface{}) (int64, error)

	// Schemas
	ListSchemas(ctx context.Context) ([]string, error)
	CreateSchema(ctx context.Context, name string) error
	DropSchema(ctx context.Context, name string) error
}

// ========== Functions Types ==========

// Function represents an edge function.
type Function struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	Version    int       `json:"version"`
	Status     string    `json:"status"` // active, inactive
	Entrypoint string    `json:"entrypoint"`
	ImportMap  string    `json:"import_map,omitempty"`
	VerifyJWT  bool      `json:"verify_jwt"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Deployment represents a function deployment.
type Deployment struct {
	ID         string    `json:"id"`
	FunctionID string    `json:"function_id"`
	Version    int       `json:"version"`
	SourceCode string    `json:"source_code"`
	BundlePath string    `json:"bundle_path,omitempty"`
	Status     string    `json:"status"` // pending, deploying, deployed, failed
	DeployedAt time.Time `json:"deployed_at"`
}

// Secret represents a function secret.
type Secret struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     string    `json:"-"` // never expose
	CreatedAt time.Time `json:"created_at"`
}

// FunctionsStore defines edge functions storage operations.
type FunctionsStore interface {
	// Functions
	CreateFunction(ctx context.Context, fn *Function) error
	GetFunction(ctx context.Context, id string) (*Function, error)
	GetFunctionByName(ctx context.Context, name string) (*Function, error)
	ListFunctions(ctx context.Context) ([]*Function, error)
	UpdateFunction(ctx context.Context, fn *Function) error
	DeleteFunction(ctx context.Context, id string) error

	// Deployments
	CreateDeployment(ctx context.Context, deployment *Deployment) error
	GetDeployment(ctx context.Context, id string) (*Deployment, error)
	GetLatestDeployment(ctx context.Context, functionID string) (*Deployment, error)
	ListDeployments(ctx context.Context, functionID string, limit int) ([]*Deployment, error)
	UpdateDeployment(ctx context.Context, deployment *Deployment) error

	// Secrets
	CreateSecret(ctx context.Context, secret *Secret) error
	GetSecret(ctx context.Context, name string) (*Secret, error)
	ListSecrets(ctx context.Context) ([]*Secret, error)
	UpdateSecret(ctx context.Context, secret *Secret) error
	DeleteSecret(ctx context.Context, name string) error
}

// ========== Realtime Types ==========

// Channel represents a realtime channel.
type Channel struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	InsertedAt time.Time `json:"inserted_at"`
}

// Subscription represents a channel subscription.
type Subscription struct {
	ID        string         `json:"id"`
	ChannelID string         `json:"channel_id"`
	UserID    string         `json:"user_id,omitempty"`
	Filters   map[string]any `json:"filters,omitempty"`
	Claims    map[string]any `json:"claims,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// RealtimeStore defines realtime storage operations.
type RealtimeStore interface {
	// Channels
	CreateChannel(ctx context.Context, channel *Channel) error
	GetChannel(ctx context.Context, name string) (*Channel, error)
	ListChannels(ctx context.Context) ([]*Channel, error)
	DeleteChannel(ctx context.Context, name string) error

	// Subscriptions
	CreateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscription(ctx context.Context, id string) (*Subscription, error)
	ListSubscriptions(ctx context.Context, channelID string) ([]*Subscription, error)
	DeleteSubscription(ctx context.Context, id string) error
}
