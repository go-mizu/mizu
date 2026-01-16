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

// SeedStorage creates sample storage buckets.
func (s *Store) SeedStorage(ctx context.Context) error {
	sql := `
	INSERT INTO storage.buckets (id, name, public, file_size_limit)
	VALUES
		($1, 'avatars', true, 5242880),
		($2, 'documents', false, 52428800),
		($3, 'public', true, NULL)
	ON CONFLICT (name) DO NOTHING
	`

	_, err := s.pool.Exec(ctx, sql,
		newULID(),
		newULID(),
		newULID(),
	)
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

// newULID generates a new ULID string.
func newULID() string {
	return ulid.Make().String()
}
