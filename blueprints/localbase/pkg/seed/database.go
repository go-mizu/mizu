// Package seed provides realistic test data seeding for Localbase and Supabase compatibility testing.
package seed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Seeder handles database seeding operations.
type Seeder struct {
	pool *pgxpool.Pool
}

// New creates a new Seeder instance.
func New(pool *pgxpool.Pool) *Seeder {
	return &Seeder{pool: pool}
}

// NewFromConnString creates a new Seeder from a connection string.
func NewFromConnString(ctx context.Context, connString string) (*Seeder, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &Seeder{pool: pool}, nil
}

// Close closes the database connection.
func (s *Seeder) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// SeedAll seeds all test data.
func (s *Seeder) SeedAll(ctx context.Context) error {
	// Create schema and tables first
	if err := s.CreateTestSchema(ctx); err != nil {
		return fmt.Errorf("failed to create test schema: %w", err)
	}

	// Seed data in dependency order
	if err := s.SeedUsers(ctx); err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}
	if err := s.SeedProfiles(ctx); err != nil {
		return fmt.Errorf("failed to seed profiles: %w", err)
	}
	if err := s.SeedTags(ctx); err != nil {
		return fmt.Errorf("failed to seed tags: %w", err)
	}
	if err := s.SeedPosts(ctx); err != nil {
		return fmt.Errorf("failed to seed posts: %w", err)
	}
	if err := s.SeedComments(ctx); err != nil {
		return fmt.Errorf("failed to seed comments: %w", err)
	}
	if err := s.SeedPostTags(ctx); err != nil {
		return fmt.Errorf("failed to seed post_tags: %w", err)
	}
	if err := s.SeedTodos(ctx); err != nil {
		return fmt.Errorf("failed to seed todos: %w", err)
	}
	if err := s.SeedProducts(ctx); err != nil {
		return fmt.Errorf("failed to seed products: %w", err)
	}
	if err := s.SeedOrders(ctx); err != nil {
		return fmt.Errorf("failed to seed orders: %w", err)
	}
	if err := s.SeedOrderItems(ctx); err != nil {
		return fmt.Errorf("failed to seed order_items: %w", err)
	}
	if err := s.CreateTestFunctions(ctx); err != nil {
		return fmt.Errorf("failed to create test functions: %w", err)
	}
	if err := s.CreateTestViews(ctx); err != nil {
		return fmt.Errorf("failed to create test views: %w", err)
	}
	if err := s.SetupRLS(ctx); err != nil {
		return fmt.Errorf("failed to setup RLS: %w", err)
	}

	return nil
}

// CreateTestSchema creates all tables required for testing.
func (s *Seeder) CreateTestSchema(ctx context.Context) error {
	// First check if auth schema exists and create helper functions if allowed
	authSQL := `
	-- Create auth schema helper functions (skip if schema doesn't exist or no permission)
	DO $$
	BEGIN
		-- Try to create auth functions, ignore errors (e.g., in Supabase Local)
		BEGIN
			CREATE OR REPLACE FUNCTION auth.uid() RETURNS UUID AS $func$
				SELECT NULLIF(current_setting('request.jwt.claims', TRUE)::json->>'sub', '')::UUID
			$func$ LANGUAGE SQL STABLE;
		EXCEPTION WHEN OTHERS THEN
			-- Function already exists or no permission, ignore
			NULL;
		END;

		BEGIN
			CREATE OR REPLACE FUNCTION auth.role() RETURNS TEXT AS $func$
				SELECT NULLIF(current_setting('request.jwt.claims', TRUE)::json->>'role', '')::TEXT
			$func$ LANGUAGE SQL STABLE;
		EXCEPTION WHEN OTHERS THEN
			-- Function already exists or no permission, ignore
			NULL;
		END;
	END $$;
	`
	// Try to create auth functions, but don't fail if we can't
	_, _ = s.pool.Exec(ctx, authSQL)

	sql := `
	-- Enable required extensions
	CREATE EXTENSION IF NOT EXISTS pgcrypto;
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	-- Drop existing test tables (for clean state)
	DROP TABLE IF EXISTS public.order_items CASCADE;
	DROP TABLE IF EXISTS public.orders CASCADE;
	DROP TABLE IF EXISTS public.products CASCADE;
	DROP TABLE IF EXISTS public.todos CASCADE;
	DROP TABLE IF EXISTS public.post_tags CASCADE;
	DROP TABLE IF EXISTS public.comments CASCADE;
	DROP TABLE IF EXISTS public.posts CASCADE;
	DROP TABLE IF EXISTS public.tags CASCADE;
	DROP TABLE IF EXISTS public.profiles CASCADE;
	DROP TABLE IF EXISTS public.test_users CASCADE;

	-- Users table (test_users to avoid conflict with auth.users)
	CREATE TABLE public.test_users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		age INTEGER,
		status VARCHAR(50) DEFAULT 'active',
		metadata JSONB DEFAULT '{}',
		tags TEXT[] DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Profiles table (one-to-one with users)
	CREATE TABLE public.profiles (
		id UUID PRIMARY KEY REFERENCES public.test_users(id) ON DELETE CASCADE,
		username VARCHAR(50) UNIQUE,
		bio TEXT,
		avatar_url TEXT,
		website TEXT,
		location VARCHAR(255),
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Tags table
	CREATE TABLE public.tags (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(50) UNIQUE NOT NULL,
		slug VARCHAR(50) UNIQUE NOT NULL,
		color VARCHAR(7) DEFAULT '#000000',
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Posts table (belongs to user)
	CREATE TABLE public.posts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		author_id UUID REFERENCES public.test_users(id) ON DELETE CASCADE,
		title VARCHAR(255) NOT NULL,
		slug VARCHAR(255) UNIQUE NOT NULL,
		content TEXT,
		excerpt TEXT,
		published BOOLEAN DEFAULT FALSE,
		view_count INTEGER DEFAULT 0,
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		published_at TIMESTAMPTZ
	);

	-- Comments table (belongs to post and user)
	CREATE TABLE public.comments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		post_id UUID REFERENCES public.posts(id) ON DELETE CASCADE,
		author_id UUID REFERENCES public.test_users(id) ON DELETE SET NULL,
		parent_id UUID REFERENCES public.comments(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		approved BOOLEAN DEFAULT TRUE,
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Post-Tags junction table (many-to-many)
	CREATE TABLE public.post_tags (
		post_id UUID REFERENCES public.posts(id) ON DELETE CASCADE,
		tag_id UUID REFERENCES public.tags(id) ON DELETE CASCADE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		PRIMARY KEY (post_id, tag_id)
	);

	-- Todos table (with RLS)
	CREATE TABLE public.todos (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES public.test_users(id) ON DELETE CASCADE,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		completed BOOLEAN DEFAULT FALSE,
		priority INTEGER CHECK (priority >= 1 AND priority <= 5),
		due_date TIMESTAMPTZ,
		tags TEXT[] DEFAULT '{}',
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Products table (for e-commerce tests)
	CREATE TABLE public.products (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sku VARCHAR(50) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
		sale_price DECIMAL(10,2),
		inventory INTEGER DEFAULT 0 CHECK (inventory >= 0),
		category VARCHAR(100),
		tags TEXT[] DEFAULT '{}',
		metadata JSONB DEFAULT '{}',
		active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Orders table
	CREATE TABLE public.orders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		customer_id UUID REFERENCES public.test_users(id) ON DELETE SET NULL,
		order_number VARCHAR(50) UNIQUE NOT NULL,
		status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'shipped', 'delivered', 'cancelled')),
		subtotal DECIMAL(10,2) DEFAULT 0,
		tax DECIMAL(10,2) DEFAULT 0,
		total DECIMAL(10,2) DEFAULT 0,
		shipping_address JSONB DEFAULT '{}',
		billing_address JSONB DEFAULT '{}',
		notes TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Order items table
	CREATE TABLE public.order_items (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		order_id UUID REFERENCES public.orders(id) ON DELETE CASCADE,
		product_id UUID REFERENCES public.products(id) ON DELETE SET NULL,
		quantity INTEGER NOT NULL CHECK (quantity > 0),
		unit_price DECIMAL(10,2) NOT NULL,
		total DECIMAL(10,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create indexes for performance
	CREATE INDEX IF NOT EXISTS idx_test_users_email ON public.test_users(email);
	CREATE INDEX IF NOT EXISTS idx_test_users_status ON public.test_users(status);
	CREATE INDEX IF NOT EXISTS idx_test_users_metadata ON public.test_users USING GIN(metadata);
	CREATE INDEX IF NOT EXISTS idx_posts_author_id ON public.posts(author_id);
	CREATE INDEX IF NOT EXISTS idx_posts_published ON public.posts(published);
	CREATE INDEX IF NOT EXISTS idx_posts_slug ON public.posts(slug);
	CREATE INDEX IF NOT EXISTS idx_comments_post_id ON public.comments(post_id);
	CREATE INDEX IF NOT EXISTS idx_comments_author_id ON public.comments(author_id);
	CREATE INDEX IF NOT EXISTS idx_todos_user_id ON public.todos(user_id);
	CREATE INDEX IF NOT EXISTS idx_todos_completed ON public.todos(completed);
	CREATE INDEX IF NOT EXISTS idx_products_category ON public.products(category);
	CREATE INDEX IF NOT EXISTS idx_products_tags ON public.products USING GIN(tags);
	CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON public.orders(customer_id);
	CREATE INDEX IF NOT EXISTS idx_orders_status ON public.orders(status);
	CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON public.order_items(order_id);
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// SeedUsers creates 100 realistic test users.
func (s *Seeder) SeedUsers(ctx context.Context) error {
	firstNames := []string{"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "William", "Elizabeth",
		"David", "Barbara", "Richard", "Susan", "Joseph", "Jessica", "Thomas", "Sarah", "Charles", "Karen",
		"Christopher", "Lisa", "Daniel", "Nancy", "Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra",
		"Emma", "Olivia", "Ava", "Isabella", "Sophia", "Mia", "Charlotte", "Amelia", "Harper", "Evelyn",
		"Liam", "Noah", "Oliver", "Elijah", "Lucas", "Mason", "Logan", "Alexander", "Ethan", "Jacob"}

	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
		"Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
		"Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
		"Walker", "Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
		"Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell", "Carter", "Roberts"}

	domains := []string{"gmail.com", "yahoo.com", "outlook.com", "company.com", "example.com", "test.io", "mail.org"}
	statuses := []string{"active", "inactive", "pending", "suspended"}

	var values []string
	var args []interface{}
	argIdx := 1

	for i := 0; i < 100; i++ {
		firstName := firstNames[i%len(firstNames)]
		lastName := lastNames[i%len(lastNames)]
		name := firstName + " " + lastName
		email := fmt.Sprintf("%s.%s%d@%s", strings.ToLower(firstName), strings.ToLower(lastName), i, domains[i%len(domains)])
		age := 18 + (i % 60)
		status := statuses[i%len(statuses)]

		// Generate random metadata
		metadata := fmt.Sprintf(`{"role": "%s", "department": "%s", "level": %d}`,
			[]string{"user", "admin", "moderator", "viewer"}[i%4],
			[]string{"engineering", "sales", "marketing", "support", "hr"}[i%5],
			(i%5)+1)

		// Generate random tags
		tags := fmt.Sprintf("{%s}", strings.Join([]string{
			[]string{"premium", "standard", "trial"}[i%3],
			[]string{"newsletter", "updates", "promotions"}[(i+1)%3],
		}, ","))

		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d::jsonb, $%d::text[])",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6))
		args = append(args, uuid.New().String(), email, name, age, status, metadata, tags)
		argIdx += 7
	}

	sql := fmt.Sprintf(`INSERT INTO public.test_users (id, email, name, age, status, metadata, tags) VALUES %s ON CONFLICT (email) DO NOTHING`,
		strings.Join(values, ", "))

	_, err := s.pool.Exec(ctx, sql, args...)
	return err
}

// SeedProfiles creates profiles for all users.
func (s *Seeder) SeedProfiles(ctx context.Context) error {
	sql := `
	INSERT INTO public.profiles (id, username, bio, avatar_url, website, location, metadata)
	SELECT
		id,
		LOWER(REPLACE(name, ' ', '_')) || '_' || SUBSTRING(id::text, 1, 4),
		CASE (ROW_NUMBER() OVER () % 5)
			WHEN 0 THEN 'Software engineer passionate about open source'
			WHEN 1 THEN 'Product designer focused on user experience'
			WHEN 2 THEN 'Data scientist exploring machine learning'
			WHEN 3 THEN 'Full-stack developer building web applications'
			ELSE 'Tech enthusiast and lifelong learner'
		END,
		'https://avatars.example.com/' || id || '.jpg',
		CASE (ROW_NUMBER() OVER () % 3)
			WHEN 0 THEN 'https://github.com/' || LOWER(REPLACE(name, ' ', ''))
			WHEN 1 THEN 'https://linkedin.com/in/' || LOWER(REPLACE(name, ' ', '-'))
			ELSE NULL
		END,
		CASE (ROW_NUMBER() OVER () % 6)
			WHEN 0 THEN 'San Francisco, CA'
			WHEN 1 THEN 'New York, NY'
			WHEN 2 THEN 'Austin, TX'
			WHEN 3 THEN 'Seattle, WA'
			WHEN 4 THEN 'London, UK'
			ELSE 'Remote'
		END,
		jsonb_build_object(
			'theme', CASE (ROW_NUMBER() OVER () % 2) WHEN 0 THEN 'dark' ELSE 'light' END,
			'notifications', (ROW_NUMBER() OVER () % 2) = 0
		)
	FROM public.test_users
	ON CONFLICT (id) DO NOTHING
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// SeedTags creates 50 unique tags.
func (s *Seeder) SeedTags(ctx context.Context) error {
	tags := []struct {
		name  string
		color string
	}{
		{"technology", "#3498db"}, {"programming", "#2ecc71"}, {"javascript", "#f1c40f"},
		{"python", "#3776ab"}, {"golang", "#00add8"}, {"rust", "#b7410e"},
		{"web-development", "#e74c3c"}, {"mobile", "#9b59b6"}, {"devops", "#1abc9c"},
		{"cloud", "#34495e"}, {"aws", "#ff9900"}, {"kubernetes", "#326ce5"},
		{"docker", "#2496ed"}, {"database", "#4a90d9"}, {"postgresql", "#336791"},
		{"mongodb", "#47a248"}, {"redis", "#dc382d"}, {"graphql", "#e10098"},
		{"rest-api", "#61dafb"}, {"microservices", "#ff6b6b"}, {"security", "#2c3e50"},
		{"testing", "#27ae60"}, {"ci-cd", "#e67e22"}, {"agile", "#8e44ad"},
		{"machine-learning", "#f39c12"}, {"data-science", "#16a085"}, {"ai", "#c0392b"},
		{"blockchain", "#f7931a"}, {"cryptocurrency", "#627eea"}, {"fintech", "#00c853"},
		{"startup", "#ff4081"}, {"design", "#673ab7"}, {"ux", "#00bcd4"},
		{"frontend", "#4caf50"}, {"backend", "#795548"}, {"fullstack", "#607d8b"},
		{"tutorial", "#ff5722"}, {"news", "#03a9f4"}, {"opinion", "#e91e63"},
		{"review", "#9c27b0"}, {"career", "#cddc39"}, {"productivity", "#ffc107"},
		{"tools", "#00bfa5"}, {"frameworks", "#6200ea"}, {"libraries", "#304ffe"},
		{"best-practices", "#64dd17"}, {"architecture", "#aa00ff"}, {"performance", "#ff1744"},
		{"open-source", "#76ff03"}, {"community", "#18ffff"},
	}

	var values []string
	var args []interface{}
	argIdx := 1

	for _, tag := range tags {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, uuid.New().String(), tag.name, strings.ToLower(strings.ReplaceAll(tag.name, " ", "-")), tag.color)
		argIdx += 4
	}

	sql := fmt.Sprintf(`INSERT INTO public.tags (id, name, slug, color) VALUES %s ON CONFLICT (name) DO NOTHING`,
		strings.Join(values, ", "))

	_, err := s.pool.Exec(ctx, sql, args...)
	return err
}

// SeedPosts creates 500 blog posts.
func (s *Seeder) SeedPosts(ctx context.Context) error {
	titles := []string{
		"Getting Started with %s: A Comprehensive Guide",
		"10 Best Practices for %s Development",
		"Understanding %s: From Basics to Advanced",
		"Why %s is the Future of Software Development",
		"A Deep Dive into %s Architecture",
		"Building Scalable Applications with %s",
		"Common Mistakes in %s and How to Avoid Them",
		"The Complete %s Tutorial for Beginners",
		"Advanced %s Techniques Every Developer Should Know",
		"Comparing %s with Other Solutions",
	}

	topics := []string{"Go", "Python", "JavaScript", "TypeScript", "Rust", "Docker", "Kubernetes",
		"PostgreSQL", "MongoDB", "Redis", "GraphQL", "REST APIs", "Microservices", "Cloud Computing",
		"Machine Learning", "Web Development", "Mobile Development", "DevOps", "Security", "Testing"}

	// First, get all user IDs
	rows, err := s.pool.Query(ctx, `SELECT id FROM public.test_users ORDER BY created_at LIMIT 100`)
	if err != nil {
		return fmt.Errorf("failed to get users for posts: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		return fmt.Errorf("no users found for seeding posts")
	}

	sql := `
	INSERT INTO public.posts (id, author_id, title, slug, content, excerpt, published, view_count, metadata, published_at)
	VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8::jsonb,
		CASE WHEN $6 THEN NOW() - ($9::text || ' days')::interval ELSE NULL END)
	ON CONFLICT (slug) DO NOTHING
	`

	for i := 0; i < 500; i++ {
		title := fmt.Sprintf(titles[i%len(titles)], topics[i%len(topics)])
		slug := fmt.Sprintf("%s-%d-%s", strings.ToLower(strings.ReplaceAll(topics[i%len(topics)], " ", "-")), i, randomHex(4))
		content := generateLoremIpsum(5)
		excerpt := generateLoremIpsum(1)[:200]
		published := i%3 != 0 // ~66% published
		viewCount := (i * 17) % 10000
		metadata := fmt.Sprintf(`{"read_time": %d, "featured": %t}`, (i%10)+1, i%10 == 0)
		daysAgo := fmt.Sprintf("%d", i%365)

		userID := userIDs[i%len(userIDs)]

		_, err = s.pool.Exec(ctx, sql, userID, title, slug, content, excerpt, published, viewCount, metadata, daysAgo)
		if err != nil {
			// Ignore duplicate slug errors
			continue
		}
	}

	return nil
}

// SeedComments creates 2000 comments on posts.
func (s *Seeder) SeedComments(ctx context.Context) error {
	comments := []string{
		"Great article! This really helped me understand the concept better.",
		"Thanks for sharing this. I've been looking for a good tutorial on this topic.",
		"Very well written. I especially liked the section on best practices.",
		"Could you elaborate more on the performance considerations?",
		"I disagree with some points here, but overall a good read.",
		"This saved me hours of debugging. Thank you!",
		"Nice explanation! The code examples were particularly helpful.",
		"I tried this approach and it worked perfectly for my use case.",
		"Have you considered using a different approach for this problem?",
		"Excellent content as always. Looking forward to more posts!",
	}

	sql := `
	INSERT INTO public.comments (id, post_id, author_id, content, approved, metadata)
	SELECT
		gen_random_uuid(),
		p.id,
		u.id,
		$1,
		$2,
		$3::jsonb
	FROM public.posts p, public.test_users u
	WHERE p.id = $4 AND u.id = $5
	`

	// Get all post IDs
	rows, err := s.pool.Query(ctx, `SELECT id FROM public.posts LIMIT 500`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var postIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		postIDs = append(postIDs, id)
	}

	// Get all user IDs
	rows, err = s.pool.Query(ctx, `SELECT id FROM public.test_users LIMIT 100`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		userIDs = append(userIDs, id)
	}

	if len(postIDs) == 0 || len(userIDs) == 0 {
		return nil
	}

	for i := 0; i < 2000; i++ {
		content := comments[i%len(comments)]
		approved := i%10 != 0 // 90% approved
		metadata := fmt.Sprintf(`{"upvotes": %d, "reported": %t}`, i%50, i%100 == 0)

		postID := postIDs[i%len(postIDs)]
		userID := userIDs[i%len(userIDs)]

		_, _ = s.pool.Exec(ctx, sql, content, approved, metadata, postID, userID)
	}

	return nil
}

// SeedPostTags creates random tag assignments for posts.
func (s *Seeder) SeedPostTags(ctx context.Context) error {
	sql := `
	INSERT INTO public.post_tags (post_id, tag_id)
	SELECT p.id, t.id
	FROM (SELECT id FROM public.posts ORDER BY random() LIMIT 400) p
	CROSS JOIN (SELECT id FROM public.tags ORDER BY random() LIMIT 3) t
	ON CONFLICT DO NOTHING
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// SeedTodos creates 1000 todos across users.
func (s *Seeder) SeedTodos(ctx context.Context) error {
	todoTemplates := []string{
		"Review pull request #%d",
		"Update documentation for %s feature",
		"Fix bug in %s module",
		"Write unit tests for %s",
		"Refactor %s component",
		"Deploy %s to staging",
		"Meeting: discuss %s architecture",
		"Research %s alternatives",
		"Implement %s endpoint",
		"Code review: %s changes",
	}

	features := []string{"auth", "payment", "notification", "search", "analytics",
		"user-profile", "dashboard", "settings", "export", "import"}

	sql := `
	INSERT INTO public.todos (id, user_id, title, description, completed, priority, due_date, tags, metadata)
	VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7::text[], $8::jsonb)
	`

	// Get all user IDs
	rows, err := s.pool.Query(ctx, `SELECT id FROM public.test_users LIMIT 100`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		return nil
	}

	for i := 0; i < 1000; i++ {
		title := fmt.Sprintf(todoTemplates[i%len(todoTemplates)], features[i%len(features)])
		if i%len(todoTemplates) == 0 {
			title = fmt.Sprintf(todoTemplates[0], i%1000)
		}
		description := "Description for: " + title
		completed := i%4 == 0 // 25% completed
		priority := (i % 5) + 1
		var dueDate *time.Time
		if i%3 != 0 {
			d := time.Now().AddDate(0, 0, (i%30)-15)
			dueDate = &d
		}
		tags := fmt.Sprintf("{%s,%s}",
			[]string{"work", "personal", "urgent"}[i%3],
			[]string{"high-priority", "low-priority", "medium-priority"}[i%3])
		metadata := fmt.Sprintf(`{"estimated_hours": %d, "category": "%s"}`,
			(i%8)+1, []string{"development", "testing", "documentation", "meeting"}[i%4])

		userID := userIDs[i%len(userIDs)]

		_, _ = s.pool.Exec(ctx, sql, userID, title, description, completed, priority, dueDate, tags, metadata)
	}

	return nil
}

// SeedProducts creates 200 products.
func (s *Seeder) SeedProducts(ctx context.Context) error {
	categories := []string{"Electronics", "Clothing", "Books", "Home & Garden", "Sports",
		"Toys", "Beauty", "Automotive", "Food", "Health"}

	adjectives := []string{"Premium", "Basic", "Professional", "Deluxe", "Standard",
		"Ultra", "Mini", "Max", "Pro", "Lite"}

	items := []string{"Widget", "Gadget", "Device", "Tool", "Kit",
		"Set", "Pack", "Bundle", "System", "Solution"}

	var values []string
	var args []interface{}
	argIdx := 1

	for i := 0; i < 200; i++ {
		sku := fmt.Sprintf("SKU-%s-%04d", strings.ToUpper(categories[i%len(categories)][:3]), i)
		name := fmt.Sprintf("%s %s %s", adjectives[i%len(adjectives)], categories[i%len(categories)], items[i%len(items)])
		description := fmt.Sprintf("High-quality %s for everyday use. %s", name, generateLoremIpsum(1)[:100])
		price := float64((i%100)*10+99) / 100 * 10
		var salePrice *float64
		if i%5 == 0 {
			sp := price * 0.8
			salePrice = &sp
		}
		inventory := (i * 7) % 500
		category := categories[i%len(categories)]
		tags := fmt.Sprintf("{%s,%s}",
			[]string{"featured", "sale", "new"}[i%3],
			[]string{"bestseller", "trending", "recommended"}[i%3])
		metadata := fmt.Sprintf(`{"weight": %.2f, "dimensions": {"l": %d, "w": %d, "h": %d}}`,
			float64((i%50)+1)/10, (i%30)+10, (i%20)+5, (i%15)+2)
		active := i%10 != 0 // 90% active

		if salePrice != nil {
			values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d::text[], $%d::jsonb, $%d)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9, argIdx+10))
			args = append(args, uuid.New().String(), sku, name, description, price, *salePrice, inventory, category, tags, metadata, active)
			argIdx += 11
		} else {
			values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, NULL, $%d, $%d, $%d::text[], $%d::jsonb, $%d)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9))
			args = append(args, uuid.New().String(), sku, name, description, price, inventory, category, tags, metadata, active)
			argIdx += 10
		}
	}

	sql := fmt.Sprintf(`INSERT INTO public.products (id, sku, name, description, price, sale_price, inventory, category, tags, metadata, active) VALUES %s ON CONFLICT (sku) DO NOTHING`,
		strings.Join(values, ", "))

	_, err := s.pool.Exec(ctx, sql, args...)
	return err
}

// SeedOrders creates 500 orders.
func (s *Seeder) SeedOrders(ctx context.Context) error {
	statuses := []string{"pending", "processing", "shipped", "delivered", "cancelled"}

	sql := `
	INSERT INTO public.orders (id, customer_id, order_number, status, subtotal, tax, total, shipping_address, billing_address, notes)
	SELECT
		gen_random_uuid(),
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7::jsonb,
		$8::jsonb,
		$9
	`

	// Get all user IDs
	rows, err := s.pool.Query(ctx, `SELECT id FROM public.test_users LIMIT 100`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		return nil
	}

	for i := 0; i < 500; i++ {
		orderNumber := fmt.Sprintf("ORD-%06d", 100000+i)
		status := statuses[i%len(statuses)]
		subtotal := float64((i%500)+50) + 0.99
		tax := subtotal * 0.08
		total := subtotal + tax

		shippingAddress := fmt.Sprintf(`{"street": "%d Main St", "city": "%s", "state": "%s", "zip": "%05d", "country": "US"}`,
			(i%999)+1,
			[]string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix"}[i%5],
			[]string{"NY", "CA", "IL", "TX", "AZ"}[i%5],
			(i%99999)+10000)

		billingAddress := shippingAddress // Same for simplicity

		var notes *string
		if i%10 == 0 {
			n := "Please handle with care"
			notes = &n
		}

		userID := userIDs[i%len(userIDs)]

		_, _ = s.pool.Exec(ctx, sql, userID, orderNumber, status, subtotal, tax, total, shippingAddress, billingAddress, notes)
	}

	return nil
}

// SeedOrderItems creates order line items.
func (s *Seeder) SeedOrderItems(ctx context.Context) error {
	sql := `
	INSERT INTO public.order_items (order_id, product_id, quantity, unit_price)
	SELECT
		o.id,
		p.id,
		(floor(random() * 5) + 1)::int,
		p.price
	FROM
		(SELECT id FROM public.orders ORDER BY random() LIMIT 400) o
	CROSS JOIN
		(SELECT id, price FROM public.products WHERE active = true ORDER BY random() LIMIT 3) p
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// CreateTestFunctions creates stored procedures for RPC testing.
func (s *Seeder) CreateTestFunctions(ctx context.Context) error {
	sql := `
	-- Simple function: add two numbers
	CREATE OR REPLACE FUNCTION public.add_numbers(a INTEGER, b INTEGER)
	RETURNS INTEGER AS $$
	BEGIN
		RETURN a + b;
	END;
	$$ LANGUAGE plpgsql;

	-- Function returning a set of records
	CREATE OR REPLACE FUNCTION public.get_active_users()
	RETURNS SETOF public.test_users AS $$
	BEGIN
		RETURN QUERY SELECT * FROM public.test_users WHERE status = 'active';
	END;
	$$ LANGUAGE plpgsql;

	-- Function with JSON parameter
	CREATE OR REPLACE FUNCTION public.search_users(filters JSONB DEFAULT '{}')
	RETURNS SETOF public.test_users AS $$
	BEGIN
		RETURN QUERY
		SELECT * FROM public.test_users
		WHERE
			(filters->>'status' IS NULL OR status = filters->>'status')
			AND (filters->>'min_age' IS NULL OR age >= (filters->>'min_age')::int)
			AND (filters->>'max_age' IS NULL OR age <= (filters->>'max_age')::int);
	END;
	$$ LANGUAGE plpgsql;

	-- Function returning a scalar
	CREATE OR REPLACE FUNCTION public.count_posts_by_author(author_uuid UUID)
	RETURNS INTEGER AS $$
	DECLARE
		post_count INTEGER;
	BEGIN
		SELECT COUNT(*) INTO post_count FROM public.posts WHERE author_id = author_uuid;
		RETURN post_count;
	END;
	$$ LANGUAGE plpgsql;

	-- Function with side effects (creates an order)
	CREATE OR REPLACE FUNCTION public.create_order_with_items(
		p_customer_id UUID,
		p_items JSONB
	)
	RETURNS public.orders AS $$
	DECLARE
		v_order public.orders;
		v_item JSONB;
		v_subtotal DECIMAL(10,2) := 0;
	BEGIN
		-- Create order
		INSERT INTO public.orders (customer_id, order_number, status)
		VALUES (p_customer_id, 'ORD-' || LPAD(nextval('orders_id_seq')::text, 6, '0'), 'pending')
		RETURNING * INTO v_order;

		-- Add items
		FOR v_item IN SELECT * FROM jsonb_array_elements(p_items)
		LOOP
			INSERT INTO public.order_items (order_id, product_id, quantity, unit_price)
			SELECT
				v_order.id,
				(v_item->>'product_id')::UUID,
				(v_item->>'quantity')::INTEGER,
				price
			FROM public.products
			WHERE id = (v_item->>'product_id')::UUID;
		END LOOP;

		-- Calculate totals
		SELECT COALESCE(SUM(total), 0) INTO v_subtotal
		FROM public.order_items
		WHERE order_id = v_order.id;

		UPDATE public.orders
		SET
			subtotal = v_subtotal,
			tax = v_subtotal * 0.08,
			total = v_subtotal * 1.08
		WHERE id = v_order.id
		RETURNING * INTO v_order;

		RETURN v_order;
	END;
	$$ LANGUAGE plpgsql;

	-- Void function (no return)
	CREATE OR REPLACE FUNCTION public.update_post_view_count(post_uuid UUID)
	RETURNS VOID AS $$
	BEGIN
		UPDATE public.posts SET view_count = view_count + 1 WHERE id = post_uuid;
	END;
	$$ LANGUAGE plpgsql;

	-- Function returning JSON
	CREATE OR REPLACE FUNCTION public.get_user_stats(user_uuid UUID)
	RETURNS JSONB AS $$
	DECLARE
		result JSONB;
	BEGIN
		SELECT jsonb_build_object(
			'post_count', (SELECT COUNT(*) FROM public.posts WHERE author_id = user_uuid),
			'comment_count', (SELECT COUNT(*) FROM public.comments WHERE author_id = user_uuid),
			'todo_count', (SELECT COUNT(*) FROM public.todos WHERE user_id = user_uuid),
			'completed_todos', (SELECT COUNT(*) FROM public.todos WHERE user_id = user_uuid AND completed = true)
		) INTO result;
		RETURN result;
	END;
	$$ LANGUAGE plpgsql;
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// CreateTestViews creates views for testing.
func (s *Seeder) CreateTestViews(ctx context.Context) error {
	sql := `
	-- Simple view: published posts
	CREATE OR REPLACE VIEW public.published_posts AS
	SELECT p.*, u.name as author_name, u.email as author_email
	FROM public.posts p
	JOIN public.test_users u ON p.author_id = u.id
	WHERE p.published = true;

	-- Aggregation view: user stats
	CREATE OR REPLACE VIEW public.user_stats AS
	SELECT
		u.id,
		u.name,
		u.email,
		COUNT(DISTINCT p.id) as post_count,
		COUNT(DISTINCT c.id) as comment_count,
		COUNT(DISTINCT t.id) as todo_count,
		COUNT(DISTINCT t.id) FILTER (WHERE t.completed) as completed_todo_count
	FROM public.test_users u
	LEFT JOIN public.posts p ON u.id = p.author_id
	LEFT JOIN public.comments c ON u.id = c.author_id
	LEFT JOIN public.todos t ON u.id = t.user_id
	GROUP BY u.id, u.name, u.email;

	-- View with foreign key for embedding tests
	CREATE OR REPLACE VIEW public.post_details AS
	SELECT
		p.*,
		jsonb_build_object(
			'id', u.id,
			'name', u.name,
			'email', u.email
		) as author
	FROM public.posts p
	LEFT JOIN public.test_users u ON p.author_id = u.id;
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// SetupRLS configures Row Level Security for testing.
func (s *Seeder) SetupRLS(ctx context.Context) error {
	sql := `
	-- Enable RLS on todos
	ALTER TABLE public.todos ENABLE ROW LEVEL SECURITY;

	-- Drop existing policies if any
	DROP POLICY IF EXISTS "Users view own todos" ON public.todos;
	DROP POLICY IF EXISTS "Users insert own todos" ON public.todos;
	DROP POLICY IF EXISTS "Users update own todos" ON public.todos;
	DROP POLICY IF EXISTS "Users delete own todos" ON public.todos;

	-- Create RLS policies for todos
	CREATE POLICY "Users view own todos" ON public.todos
		FOR SELECT USING (auth.uid() = user_id);

	CREATE POLICY "Users insert own todos" ON public.todos
		FOR INSERT WITH CHECK (auth.uid() = user_id);

	CREATE POLICY "Users update own todos" ON public.todos
		FOR UPDATE USING (auth.uid() = user_id);

	CREATE POLICY "Users delete own todos" ON public.todos
		FOR DELETE USING (auth.uid() = user_id);

	-- Create a service role that bypasses RLS
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'service_role') THEN
			CREATE ROLE service_role NOINHERIT;
		END IF;
	END $$;

	ALTER TABLE public.todos FORCE ROW LEVEL SECURITY;
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// CleanAll removes all test data.
func (s *Seeder) CleanAll(ctx context.Context) error {
	sql := `
	DROP TABLE IF EXISTS public.order_items CASCADE;
	DROP TABLE IF EXISTS public.orders CASCADE;
	DROP TABLE IF EXISTS public.products CASCADE;
	DROP TABLE IF EXISTS public.todos CASCADE;
	DROP TABLE IF EXISTS public.post_tags CASCADE;
	DROP TABLE IF EXISTS public.comments CASCADE;
	DROP TABLE IF EXISTS public.posts CASCADE;
	DROP TABLE IF EXISTS public.tags CASCADE;
	DROP TABLE IF EXISTS public.profiles CASCADE;
	DROP TABLE IF EXISTS public.test_users CASCADE;
	DROP VIEW IF EXISTS public.published_posts CASCADE;
	DROP VIEW IF EXISTS public.user_stats CASCADE;
	DROP VIEW IF EXISTS public.post_details CASCADE;
	DROP FUNCTION IF EXISTS public.add_numbers CASCADE;
	DROP FUNCTION IF EXISTS public.get_active_users CASCADE;
	DROP FUNCTION IF EXISTS public.search_users CASCADE;
	DROP FUNCTION IF EXISTS public.count_posts_by_author CASCADE;
	DROP FUNCTION IF EXISTS public.create_order_with_items CASCADE;
	DROP FUNCTION IF EXISTS public.update_post_view_count CASCADE;
	DROP FUNCTION IF EXISTS public.get_user_stats CASCADE;
	`

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// Helper functions

func generateLoremIpsum(paragraphs int) string {
	lorem := []string{
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
		"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
		"Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.",
		"Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt.",
		"Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit, sed quia non numquam eius modi tempora incidunt ut labore et dolore magnam aliquam quaerat voluptatem.",
	}

	var result []string
	for i := 0; i < paragraphs; i++ {
		result = append(result, lorem[i%len(lorem)])
	}
	return strings.Join(result, "\n\n")
}

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "0000"
	}
	return hex.EncodeToString(bytes)
}

// randomInt returns a random integer up to max (unused but kept for future use)
var _ = func(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}
