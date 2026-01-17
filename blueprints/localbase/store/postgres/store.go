package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

// Store implements the store.Store interface using PostgreSQL.
type Store struct {
	pool *pgxpool.Pool

	// Feature stores
	auth      *AuthStore
	storage   *StorageStore
	database  *DatabaseStore
	functions *FunctionsStore
	realtime  *RealtimeStore
	pgmeta    *PGMetaStore
	logs      *LogsStore
}

// New creates a new PostgreSQL store.
func New(ctx context.Context, connString string) (*Store, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure pool
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Store{pool: pool}
	s.auth = &AuthStore{pool: pool}
	s.storage = &StorageStore{pool: pool}
	s.database = &DatabaseStore{pool: pool}
	s.functions = &FunctionsStore{pool: pool}
	s.realtime = &RealtimeStore{pool: pool}
	s.pgmeta = &PGMetaStore{pool: pool}
	s.logs = &LogsStore{pool: pool}

	return s, nil
}

// Close closes the database connection pool.
func (s *Store) Close() error {
	s.pool.Close()
	return nil
}

// CreateExtensions creates required PostgreSQL extensions.
func (s *Store) CreateExtensions(ctx context.Context) error {
	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS pgcrypto",
		"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"",
		"CREATE EXTENSION IF NOT EXISTS pg_stat_statements",
		// Note: pgvector needs to be installed on the system first
		// "CREATE EXTENSION IF NOT EXISTS vector",
	}

	for _, ext := range extensions {
		if _, err := s.pool.Exec(ctx, ext); err != nil {
			return fmt.Errorf("failed to create extension: %w", err)
		}
	}

	return nil
}

// Ensure creates all required schemas and tables.
func (s *Store) Ensure(ctx context.Context) error {
	// Create schemas
	schemas := []string{
		"CREATE SCHEMA IF NOT EXISTS auth",
		"CREATE SCHEMA IF NOT EXISTS storage",
		"CREATE SCHEMA IF NOT EXISTS functions",
		"CREATE SCHEMA IF NOT EXISTS realtime",
		"CREATE SCHEMA IF NOT EXISTS analytics",
	}

	for _, schema := range schemas {
		if _, err := s.pool.Exec(ctx, schema); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
	}

	// Create auth tables
	if err := s.createAuthTables(ctx); err != nil {
		return fmt.Errorf("failed to create auth tables: %w", err)
	}

	// Create storage tables
	if err := s.createStorageTables(ctx); err != nil {
		return fmt.Errorf("failed to create storage tables: %w", err)
	}

	// Create functions tables
	if err := s.createFunctionsTables(ctx); err != nil {
		return fmt.Errorf("failed to create functions tables: %w", err)
	}

	// Create realtime tables
	if err := s.createRealtimeTables(ctx); err != nil {
		return fmt.Errorf("failed to create realtime tables: %w", err)
	}

	// Create analytics/logs tables
	if err := s.createLogsTables(ctx); err != nil {
		return fmt.Errorf("failed to create logs tables: %w", err)
	}

	return nil
}

func (s *Store) createAuthTables(ctx context.Context) error {
	sql := `
	-- Users table
	CREATE TABLE IF NOT EXISTS auth.users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE,
		phone VARCHAR(50) UNIQUE,
		encrypted_password TEXT,
		email_confirmed_at TIMESTAMPTZ,
		phone_confirmed_at TIMESTAMPTZ,
		raw_app_meta_data JSONB DEFAULT '{}',
		raw_user_meta_data JSONB DEFAULT '{}',
		is_super_admin BOOLEAN DEFAULT FALSE,
		role VARCHAR(50) DEFAULT 'authenticated',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		last_sign_in_at TIMESTAMPTZ,
		banned_until TIMESTAMPTZ,
		confirmation_token VARCHAR(255),
		recovery_token VARCHAR(255)
	);

	-- Sessions table
	CREATE TABLE IF NOT EXISTS auth.sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		factor_id UUID,
		aal VARCHAR(50) DEFAULT 'aal1',
		not_after TIMESTAMPTZ
	);

	-- Refresh tokens table
	CREATE TABLE IF NOT EXISTS auth.refresh_tokens (
		id BIGSERIAL PRIMARY KEY,
		token VARCHAR(255) UNIQUE NOT NULL,
		user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
		session_id UUID REFERENCES auth.sessions(id) ON DELETE CASCADE,
		parent VARCHAR(255),
		revoked BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- MFA factors table
	CREATE TABLE IF NOT EXISTS auth.mfa_factors (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
		friendly_name TEXT,
		factor_type VARCHAR(50) NOT NULL,
		status VARCHAR(50) DEFAULT 'unverified',
		secret TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Identities table (OAuth providers)
	CREATE TABLE IF NOT EXISTS auth.identities (
		id TEXT PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
		provider VARCHAR(50) NOT NULL,
		provider_id TEXT NOT NULL,
		identity_data JSONB DEFAULT '{}',
		last_sign_in_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(provider, provider_id)
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_users_email ON auth.users(email);
	CREATE INDEX IF NOT EXISTS idx_users_phone ON auth.users(phone);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON auth.sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON auth.refresh_tokens(token);
	CREATE INDEX IF NOT EXISTS idx_identities_user_id ON auth.identities(user_id);
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

func (s *Store) createStorageTables(ctx context.Context) error {
	sql := `
	-- Buckets table
	CREATE TABLE IF NOT EXISTS storage.buckets (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		public BOOLEAN DEFAULT FALSE,
		file_size_limit BIGINT,
		allowed_mime_types TEXT[],
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Objects table
	CREATE TABLE IF NOT EXISTS storage.objects (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		bucket_id TEXT NOT NULL REFERENCES storage.buckets(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		owner UUID REFERENCES auth.users(id),
		path_tokens TEXT[] GENERATED ALWAYS AS (string_to_array(name, '/')) STORED,
		version TEXT,
		metadata JSONB DEFAULT '{}',
		content_type TEXT,
		size BIGINT DEFAULT 0,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		last_accessed_at TIMESTAMPTZ,
		UNIQUE(bucket_id, name)
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_objects_bucket_id ON storage.objects(bucket_id);
	CREATE INDEX IF NOT EXISTS idx_objects_owner ON storage.objects(owner);
	CREATE INDEX IF NOT EXISTS idx_objects_path_tokens ON storage.objects USING GIN(path_tokens);
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

func (s *Store) createFunctionsTables(ctx context.Context) error {
	sql := `
	-- Functions table
	CREATE TABLE IF NOT EXISTS functions.functions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT UNIQUE NOT NULL,
		slug TEXT UNIQUE NOT NULL,
		version INTEGER DEFAULT 1,
		status VARCHAR(50) DEFAULT 'active',
		entrypoint TEXT DEFAULT 'index.ts',
		import_map TEXT,
		verify_jwt BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Deployments table
	CREATE TABLE IF NOT EXISTS functions.deployments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		function_id UUID NOT NULL REFERENCES functions.functions(id) ON DELETE CASCADE,
		version INTEGER NOT NULL,
		source_code TEXT,
		bundle_path TEXT,
		status VARCHAR(50) DEFAULT 'pending',
		deployed_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Secrets table
	CREATE TABLE IF NOT EXISTS functions.secrets (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT UNIQUE NOT NULL,
		value TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_deployments_function_id ON functions.deployments(function_id);
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

func (s *Store) createRealtimeTables(ctx context.Context) error {
	sql := `
	-- Channels table
	CREATE TABLE IF NOT EXISTS realtime.channels (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT UNIQUE NOT NULL,
		inserted_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Subscriptions table
	CREATE TABLE IF NOT EXISTS realtime.subscriptions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		channel_id UUID NOT NULL REFERENCES realtime.channels(id) ON DELETE CASCADE,
		user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
		filters JSONB DEFAULT '{}',
		claims JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_subscriptions_channel_id ON realtime.subscriptions(channel_id);
	CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON realtime.subscriptions(user_id);
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// SeedUsers creates sample users for development.
func (s *Store) SeedUsers(ctx context.Context) error {
	// Create a sample admin user
	// Password is "password123" hashed with bcrypt
	passwordHash := "$2a$10$uMRJEarvPUADPTbWgJ70W.0J5Clf/FcQfIUmN1NB.jpVD9oxmsxEa"

	sql := `
	INSERT INTO auth.users (id, email, encrypted_password, email_confirmed_at, role, is_super_admin, raw_user_meta_data)
	VALUES
		(gen_random_uuid(), 'admin@localbase.dev', $1, NOW(), 'admin', true, '{"name": "Admin User"}'),
		(gen_random_uuid(), 'user@localbase.dev', $1, NOW(), 'authenticated', false, '{"name": "Test User"}')
	ON CONFLICT (email) DO NOTHING
	`

	_, err := s.pool.Exec(ctx, sql, passwordHash)
	return err
}

// SeedStorage creates sample storage buckets and files.
func (s *Store) SeedStorage(ctx context.Context) error {
	// Create buckets
	bucketSQL := `
	INSERT INTO storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
	VALUES
		($1, 'avatars', true, 5242880, ARRAY['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/svg+xml']),
		($2, 'documents', false, 52428800, NULL),
		($3, 'public', true, NULL, NULL),
		($4, 'media', true, 104857600, ARRAY['image/*', 'video/*', 'audio/*']),
		($5, 'backups', false, NULL, NULL)
	ON CONFLICT (name) DO NOTHING
	`

	_, err := s.pool.Exec(ctx, bucketSQL,
		newULID(),
		newULID(),
		newULID(),
		newULID(),
		newULID(),
	)
	if err != nil {
		return err
	}

	// Create sample files in storage
	return s.seedStorageFiles(ctx)
}

// seedStorageFiles creates sample files in storage buckets.
func (s *Store) seedStorageFiles(ctx context.Context) error {
	sql := `
	-- Insert sample objects into avatars bucket
	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'default.svg', 'image/svg+xml', 1024, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'avatars'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'default.svg');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'user-1.jpg', 'image/jpeg', 45056, NULL, '{"width": 200, "height": 200}'
	FROM storage.buckets b WHERE b.name = 'avatars'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'user-1.jpg');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'user-2.png', 'image/png', 32768, NULL, '{"width": 200, "height": 200}'
	FROM storage.buckets b WHERE b.name = 'avatars'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'user-2.png');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'team/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'avatars'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'team/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'team/alice.jpg', 'image/jpeg', 28672, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'avatars'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'team/alice.jpg');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'team/bob.png', 'image/png', 35840, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'avatars'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'team/bob.png');

	-- Insert sample objects into documents bucket
	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'README.md', 'text/markdown', 2048, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'documents'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'README.md');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'reports/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'documents'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'reports/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'reports/2024/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'documents'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'reports/2024/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'reports/2024/annual-report.pdf', 'application/pdf', 1048576, NULL, '{"pages": 24}'
	FROM storage.buckets b WHERE b.name = 'documents'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'reports/2024/annual-report.pdf');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'reports/2025/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'documents'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'reports/2025/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'reports/2025/q1-summary.pdf', 'application/pdf', 524288, NULL, '{"pages": 12}'
	FROM storage.buckets b WHERE b.name = 'documents'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'reports/2025/q1-summary.pdf');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'contracts/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'documents'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'contracts/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'contracts/nda-template.docx', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 32768, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'documents'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'contracts/nda-template.docx');

	-- Insert sample objects into public bucket
	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'assets/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'assets/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'assets/logo.svg', 'image/svg+xml', 4096, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'assets/logo.svg');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'assets/favicon.ico', 'image/x-icon', 16384, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'assets/favicon.ico');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'examples/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'examples/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'examples/sample.json', 'application/json', 512, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'examples/sample.json');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'examples/config.yaml', 'application/x-yaml', 1024, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'examples/config.yaml');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'examples/script.py', 'text/x-python', 2048, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'examples/script.py');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'examples/main.go', 'text/x-go', 1536, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'examples/main.go');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'downloads/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'downloads/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'downloads/user-guide.pdf', 'application/pdf', 2097152, NULL, '{"pages": 48}'
	FROM storage.buckets b WHERE b.name = 'public'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'downloads/user-guide.pdf');

	-- Insert sample objects into media bucket
	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'images/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'images/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'images/hero.jpg', 'image/jpeg', 204800, NULL, '{"width": 1920, "height": 1080}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'images/hero.jpg');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'images/gallery/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'images/gallery/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'images/gallery/photo-001.jpg', 'image/jpeg', 153600, NULL, '{"width": 1200, "height": 800}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'images/gallery/photo-001.jpg');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'images/gallery/photo-002.png', 'image/png', 184320, NULL, '{"width": 1200, "height": 800}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'images/gallery/photo-002.png');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'videos/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'videos/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'videos/intro.mp4', 'video/mp4', 10485760, NULL, '{"duration": 60, "resolution": "1080p"}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'videos/intro.mp4');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'audio/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'audio/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'audio/notification.mp3', 'audio/mpeg', 51200, NULL, '{"duration": 2}'
	FROM storage.buckets b WHERE b.name = 'media'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'audio/notification.mp3');

	-- Insert sample objects into backups bucket
	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'database/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'backups'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'database/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'database/2025-01-15.sql.gz', 'application/gzip', 5242880, NULL, '{"tables": 12}'
	FROM storage.buckets b WHERE b.name = 'backups'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'database/2025-01-15.sql.gz');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'database/2025-01-16.sql.gz', 'application/gzip', 5373952, NULL, '{"tables": 12}'
	FROM storage.buckets b WHERE b.name = 'backups'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'database/2025-01-16.sql.gz');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'configs/.keep', 'application/octet-stream', 0, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'backups'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'configs/.keep');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'configs/nginx.conf', 'text/plain', 4096, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'backups'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'configs/nginx.conf');

	INSERT INTO storage.objects (id, bucket_id, name, content_type, size, owner, metadata)
	SELECT gen_random_uuid(), b.id, 'configs/docker-compose.yml', 'application/x-yaml', 2048, NULL, '{}'
	FROM storage.buckets b WHERE b.name = 'backups'
	AND NOT EXISTS (SELECT 1 FROM storage.objects o WHERE o.bucket_id = b.id AND o.name = 'configs/docker-compose.yml');
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// SeedTables creates sample public schema tables.
func (s *Store) SeedTables(ctx context.Context) error {
	sql := `
	-- Create a sample todos table
	CREATE TABLE IF NOT EXISTS public.todos (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT,
		completed BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create a sample profiles table
	CREATE TABLE IF NOT EXISTS public.profiles (
		id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
		username TEXT UNIQUE,
		full_name TEXT,
		avatar_url TEXT,
		website TEXT,
		bio TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create a sample posts table
	CREATE TABLE IF NOT EXISTS public.posts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		author_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		content TEXT,
		published BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create comments table with nested structure
	CREATE TABLE IF NOT EXISTS public.comments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		post_id UUID REFERENCES public.posts(id) ON DELETE CASCADE,
		author_id UUID REFERENCES auth.users(id),
		parent_id UUID REFERENCES public.comments(id),
		content TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create tags table
	CREATE TABLE IF NOT EXISTS public.tags (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) UNIQUE NOT NULL,
		slug VARCHAR(100) UNIQUE NOT NULL,
		color VARCHAR(7),
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create post_tags junction table
	CREATE TABLE IF NOT EXISTS public.post_tags (
		post_id UUID REFERENCES public.posts(id) ON DELETE CASCADE,
		tag_id UUID REFERENCES public.tags(id) ON DELETE CASCADE,
		PRIMARY KEY (post_id, tag_id)
	);

	-- Create products table with various data types
	CREATE TABLE IF NOT EXISTS public.products (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		price DECIMAL(10,2) NOT NULL,
		stock INTEGER DEFAULT 0,
		category VARCHAR(100),
		tags TEXT[],
		metadata JSONB DEFAULT '{}',
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create orders table
	CREATE TABLE IF NOT EXISTS public.orders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES auth.users(id),
		status VARCHAR(50) DEFAULT 'pending',
		total DECIMAL(10,2),
		items JSONB NOT NULL DEFAULT '[]',
		shipping_address JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create order_items junction table
	CREATE TABLE IF NOT EXISTS public.order_items (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		order_id UUID REFERENCES public.orders(id) ON DELETE CASCADE,
		product_id UUID REFERENCES public.products(id),
		quantity INTEGER NOT NULL,
		unit_price DECIMAL(10,2) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create test_users table (public, separate from auth.users)
	CREATE TABLE IF NOT EXISTS public.test_users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(100) UNIQUE,
		display_name VARCHAR(255),
		avatar_url TEXT,
		bio TEXT,
		website VARCHAR(255),
		social_links JSONB DEFAULT '{}',
		preferences JSONB DEFAULT '{}',
		is_verified BOOLEAN DEFAULT false,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create auth helper functions first (needed by policies)
	CREATE OR REPLACE FUNCTION auth.uid() RETURNS UUID AS $$
		SELECT NULLIF(current_setting('request.jwt.claims', TRUE)::json->>'sub', '')::UUID
	$$ LANGUAGE SQL STABLE;

	CREATE OR REPLACE FUNCTION auth.role() RETURNS TEXT AS $$
		SELECT NULLIF(current_setting('request.jwt.claims', TRUE)::json->>'role', '')::TEXT
	$$ LANGUAGE SQL STABLE;

	-- Enable RLS on sample tables
	ALTER TABLE public.todos ENABLE ROW LEVEL SECURITY;
	ALTER TABLE public.profiles ENABLE ROW LEVEL SECURITY;
	ALTER TABLE public.posts ENABLE ROW LEVEL SECURITY;

	-- Create policies for todos
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'todos' AND policyname = 'Users can view own todos') THEN
			CREATE POLICY "Users can view own todos" ON public.todos FOR SELECT USING (auth.uid() = user_id);
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'todos' AND policyname = 'Users can insert own todos') THEN
			CREATE POLICY "Users can insert own todos" ON public.todos FOR INSERT WITH CHECK (auth.uid() = user_id);
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'todos' AND policyname = 'Users can update own todos') THEN
			CREATE POLICY "Users can update own todos" ON public.todos FOR UPDATE USING (auth.uid() = user_id);
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'todos' AND policyname = 'Users can delete own todos') THEN
			CREATE POLICY "Users can delete own todos" ON public.todos FOR DELETE USING (auth.uid() = user_id);
		END IF;
	END $$;
	`

	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return err
	}

	// Seed sample data
	return s.seedSampleData(ctx)
}

// seedSampleData inserts realistic sample data into tables.
func (s *Store) seedSampleData(ctx context.Context) error {
	sql := `
	-- Insert tags
	INSERT INTO public.tags (name, slug, color) VALUES
		('Technology', 'technology', '#3B82F6'),
		('Design', 'design', '#EC4899'),
		('Business', 'business', '#10B981'),
		('Tutorial', 'tutorial', '#F59E0B'),
		('News', 'news', '#6366F1'),
		('Open Source', 'open-source', '#8B5CF6'),
		('Database', 'database', '#EF4444'),
		('Frontend', 'frontend', '#06B6D4'),
		('Backend', 'backend', '#84CC16'),
		('DevOps', 'devops', '#F97316')
	ON CONFLICT (slug) DO NOTHING;

	-- Insert products
	INSERT INTO public.products (name, description, price, stock, category, tags, metadata) VALUES
		('Wireless Headphones Pro', 'Premium noise-canceling wireless headphones with 40hr battery', 299.99, 50, 'Electronics',
		 ARRAY['audio', 'wireless', 'premium'], '{"brand": "AudioMax", "warranty": "2 years", "weight": "250g"}'),
		('Ergonomic Keyboard', 'Split mechanical keyboard with Cherry MX switches', 179.99, 30, 'Electronics',
		 ARRAY['keyboard', 'ergonomic', 'mechanical'], '{"switches": "Cherry MX Brown", "layout": "ANSI"}'),
		('Standing Desk Pro', 'Electric height-adjustable desk with memory presets', 599.99, 15, 'Furniture',
		 ARRAY['desk', 'standing', 'electric'], '{"max_height": "48 inches", "weight_capacity": "300 lbs"}'),
		('4K Monitor 27"', 'Professional 4K IPS monitor with USB-C', 449.99, 25, 'Electronics',
		 ARRAY['monitor', '4k', 'usb-c'], '{"resolution": "3840x2160", "refresh_rate": "60Hz"}'),
		('Webcam HD', '1080p webcam with auto-focus and noise reduction', 79.99, 100, 'Electronics',
		 ARRAY['webcam', 'streaming', 'video'], '{"resolution": "1080p", "fps": 30}')
	ON CONFLICT DO NOTHING;

	-- Insert test_users
	INSERT INTO public.test_users (email, username, display_name, bio, website, is_verified, social_links) VALUES
		('alice@example.com', 'alice', 'Alice Johnson', 'Full-stack developer passionate about open source', 'https://alice.dev', true,
		 '{"twitter": "@alice_dev", "github": "alicejohnson"}'),
		('bob@example.com', 'bob', 'Bob Smith', 'UX designer and product enthusiast', 'https://bobsmith.design', true,
		 '{"twitter": "@bobsmith", "dribbble": "bobsmith"}'),
		('charlie@example.com', 'charlie', 'Charlie Brown', 'DevOps engineer | K8s | Terraform', NULL, false,
		 '{"github": "charliebrown"}'),
		('diana@example.com', 'diana', 'Diana Prince', 'Tech lead at StartupCo', 'https://diana.io', true,
		 '{"linkedin": "dianaprince", "twitter": "@diana_tech"}'),
		('eve@example.com', 'eve', 'Eve Wilson', 'Backend developer | Go | Rust', NULL, false,
		 '{"github": "evewilson"}')
	ON CONFLICT (email) DO NOTHING;

	-- Insert posts using existing auth.users
	INSERT INTO public.posts (author_id, title, content, published)
	SELECT u.id, 'Getting Started with Supabase',
		'Supabase is an open source Firebase alternative. This guide will help you get started with authentication, database, and storage.

## Key Features

1. **Authentication** - Built-in auth with social providers
2. **Database** - PostgreSQL with real-time subscriptions
3. **Storage** - S3-compatible object storage
4. **Edge Functions** - Serverless functions at the edge

Let''s dive in!', true
	FROM auth.users u WHERE u.email = 'admin@localbase.dev'
	ON CONFLICT DO NOTHING;

	INSERT INTO public.posts (author_id, title, content, published)
	SELECT u.id, 'Building Real-time Applications',
		'Learn how to build real-time collaborative features using Supabase Realtime and PostgreSQL LISTEN/NOTIFY.

## Prerequisites

- Node.js 18+
- Supabase account
- Basic PostgreSQL knowledge

## Getting Started

First, enable realtime on your table...', true
	FROM auth.users u WHERE u.email = 'admin@localbase.dev'
	ON CONFLICT DO NOTHING;

	INSERT INTO public.posts (author_id, title, content, published)
	SELECT u.id, 'Draft: Advanced RLS Patterns',
		'This post covers advanced row-level security patterns including multi-tenancy, hierarchical permissions, and time-based access control.

TODO: Add code examples', false
	FROM auth.users u WHERE u.email = 'user@localbase.dev'
	ON CONFLICT DO NOTHING;

	-- Insert comments
	INSERT INTO public.comments (post_id, author_id, content)
	SELECT p.id, u.id, 'Great introduction! This helped me get started quickly.'
	FROM public.posts p, auth.users u
	WHERE p.title LIKE '%Getting Started%' AND u.email = 'user@localbase.dev'
	ON CONFLICT DO NOTHING;

	INSERT INTO public.comments (post_id, author_id, content)
	SELECT p.id, u.id, 'Can you add more examples about storage policies?'
	FROM public.posts p, auth.users u
	WHERE p.title LIKE '%Getting Started%' AND u.email = 'admin@localbase.dev'
	ON CONFLICT DO NOTHING;

	-- Insert orders
	INSERT INTO public.orders (user_id, status, total, items, shipping_address)
	SELECT u.id, 'completed', 479.98,
		'[{"product": "Wireless Headphones Pro", "quantity": 1, "price": 299.99}, {"product": "Ergonomic Keyboard", "quantity": 1, "price": 179.99}]'::jsonb,
		'{"street": "123 Main St", "city": "San Francisco", "state": "CA", "zip": "94102", "country": "USA"}'::jsonb
	FROM auth.users u WHERE u.email = 'admin@localbase.dev'
	ON CONFLICT DO NOTHING;

	INSERT INTO public.orders (user_id, status, total, items, shipping_address)
	SELECT u.id, 'pending', 599.99,
		'[{"product": "Standing Desk Pro", "quantity": 1, "price": 599.99}]'::jsonb,
		'{"street": "456 Oak Ave", "city": "New York", "state": "NY", "zip": "10001", "country": "USA"}'::jsonb
	FROM auth.users u WHERE u.email = 'user@localbase.dev'
	ON CONFLICT DO NOTHING;

	-- Link posts to tags
	INSERT INTO public.post_tags (post_id, tag_id)
	SELECT p.id, t.id
	FROM public.posts p, public.tags t
	WHERE p.title LIKE '%Supabase%' AND t.slug IN ('technology', 'tutorial', 'database')
	ON CONFLICT DO NOTHING;

	INSERT INTO public.post_tags (post_id, tag_id)
	SELECT p.id, t.id
	FROM public.posts p, public.tags t
	WHERE p.title LIKE '%Real-time%' AND t.slug IN ('technology', 'backend', 'database')
	ON CONFLICT DO NOTHING;

	-- Insert sample todos
	INSERT INTO public.todos (user_id, title, description, completed)
	SELECT u.id, 'Review pull requests', 'Check and approve pending PRs for the main project', true
	FROM auth.users u WHERE u.email = 'admin@localbase.dev';

	INSERT INTO public.todos (user_id, title, description, completed)
	SELECT u.id, 'Write documentation', 'Update README and API docs for v2.0 release', false
	FROM auth.users u WHERE u.email = 'admin@localbase.dev';

	INSERT INTO public.todos (user_id, title, description, completed)
	SELECT u.id, 'Setup CI/CD pipeline', 'Configure GitHub Actions for automated testing', false
	FROM auth.users u WHERE u.email = 'user@localbase.dev';

	-- Insert profiles
	INSERT INTO public.profiles (id, username, full_name, bio, website)
	SELECT u.id, 'admin', 'Admin User', 'System administrator', 'https://localbase.dev'
	FROM auth.users u WHERE u.email = 'admin@localbase.dev'
	ON CONFLICT (id) DO NOTHING;

	INSERT INTO public.profiles (id, username, full_name, bio, website)
	SELECT u.id, 'testuser', 'Test User', 'Regular test user account', NULL
	FROM auth.users u WHERE u.email = 'user@localbase.dev'
	ON CONFLICT (id) DO NOTHING;
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// Auth returns the auth store.
func (s *Store) Auth() store.AuthStore {
	return s.auth
}

// Storage returns the storage store.
func (s *Store) Storage() store.StorageStore {
	return s.storage
}

// Database returns the database store.
func (s *Store) Database() store.DatabaseStore {
	return s.database
}

// DatabaseRLS returns the database store with RLS support.
// This returns the concrete type to access RLS-aware query methods.
func (s *Store) DatabaseRLS() *DatabaseStore {
	return s.database
}

// Functions returns the functions store.
func (s *Store) Functions() store.FunctionsStore {
	return s.functions
}

// Realtime returns the realtime store.
func (s *Store) Realtime() store.RealtimeStore {
	return s.realtime
}

// PGMeta returns the postgres-meta store for dashboard compatibility.
func (s *Store) PGMeta() *PGMetaStore {
	return s.pgmeta
}

// Logs returns the logs store.
func (s *Store) Logs() store.LogsStore {
	return s.logs
}

func (s *Store) createLogsTables(ctx context.Context) error {
	sql := `
	-- Main logs table
	CREATE TABLE IF NOT EXISTS analytics.logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		event_message TEXT,
		request_id UUID,
		method VARCHAR(10),
		path TEXT,
		status_code SMALLINT,
		source VARCHAR(50) NOT NULL,
		user_id UUID,
		user_agent TEXT,
		apikey TEXT,
		request_headers JSONB DEFAULT '{}',
		response_headers JSONB DEFAULT '{}',
		duration_ms INTEGER,
		metadata JSONB DEFAULT '{}',
		search TEXT,
		CONSTRAINT logs_source_check CHECK (source IN (
			'edge', 'postgres', 'postgrest', 'pooler',
			'auth', 'storage', 'realtime', 'functions', 'cron'
		))
	);

	-- Create indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON analytics.logs (timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_logs_source ON analytics.logs (source);
	CREATE INDEX IF NOT EXISTS idx_logs_status_code ON analytics.logs (status_code) WHERE status_code IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_logs_method ON analytics.logs (method) WHERE method IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_logs_request_id ON analytics.logs (request_id) WHERE request_id IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_logs_user_id ON analytics.logs (user_id) WHERE user_id IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_logs_metadata ON analytics.logs USING GIN (metadata);

	-- Saved queries table
	CREATE TABLE IF NOT EXISTS analytics.saved_queries (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		query_params JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Query templates table
	CREATE TABLE IF NOT EXISTS analytics.query_templates (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		query_params JSONB NOT NULL,
		category VARCHAR(50)
	);

	-- Insert default templates
	INSERT INTO analytics.query_templates (id, name, description, query_params, category) VALUES
	('errors_last_hour', 'Errors in last hour', 'All requests with status >= 400 in the past hour',
	 '{"time_range": "1h", "status_min": 400}', 'debugging'),
	('slow_requests', 'Slow requests', 'Requests taking longer than 1 second',
	 '{"time_range": "24h", "duration_min_ms": 1000}', 'performance'),
	('auth_failures', 'Authentication failures', 'Failed authentication attempts',
	 '{"time_range": "24h", "source": "auth", "status_min": 400}', 'security'),
	('storage_uploads', 'Storage uploads', 'Recent file upload operations',
	 '{"time_range": "1h", "source": "storage", "method": "POST"}', 'storage'),
	('recent_errors', 'Recent 5xx errors', 'Server errors in the past hour',
	 '{"time_range": "1h", "status_min": 500}', 'debugging'),
	('api_activity', 'API Gateway activity', 'Recent API Gateway logs',
	 '{"time_range": "1h", "source": "edge"}', 'monitoring')
	ON CONFLICT (id) DO NOTHING;
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// newULID generates a new ULID string.
func newULID() string {
	return ulid.Make().String()
}
