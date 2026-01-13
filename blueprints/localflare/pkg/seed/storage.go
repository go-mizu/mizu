package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedKV(ctx context.Context) error {
	slog.Info("seeding KV namespaces")

	nsCount := 0
	pairCount := 0

	// KV Namespaces
	namespaces := []*store.KVNamespace{
		{ID: generateID(), Title: "CONFIG", CreatedAt: s.timeAgo(60 * 24 * time.Hour)},
		{ID: generateID(), Title: "SESSIONS", CreatedAt: s.timeAgo(45 * 24 * time.Hour)},
		{ID: generateID(), Title: "CACHE", CreatedAt: s.timeAgo(30 * 24 * time.Hour)},
		{ID: generateID(), Title: "RATE_LIMITS", CreatedAt: s.timeAgo(15 * 24 * time.Hour)},
	}

	for _, ns := range namespaces {
		if err := s.store.KV().CreateNamespace(ctx, ns); err == nil {
			s.ids.KVNamespaces[ns.Title] = ns.ID
			nsCount++
		}
	}

	// Sample KV pairs for each namespace
	// CONFIG namespace
	if nsID, ok := s.ids.KVNamespaces["CONFIG"]; ok {
		configPairs := []*store.KVPair{
			{Key: "app:settings", Value: mustJSON(map[string]interface{}{
				"theme":       "dark",
				"maintenance": false,
				"version":     "2.1.0",
				"api_version": "v2",
			})},
			{Key: "feature:flags", Value: mustJSON(map[string]interface{}{
				"new_checkout":    true,
				"dark_mode":       true,
				"beta_api":        false,
				"analytics_v2":    true,
				"social_login":    true,
				"two_factor_auth": false,
			})},
			{Key: "limits:api", Value: mustJSON(map[string]interface{}{
				"requests_per_minute": 100,
				"requests_per_day":    10000,
				"max_payload_size":    1048576,
			})},
			{Key: "providers:email", Value: mustJSON(map[string]interface{}{
				"driver":    "smtp",
				"host":      "smtp.example.com",
				"port":      587,
				"from_name": "Localflare",
			})},
			{Key: "cache:ttl", Value: mustJSON(map[string]interface{}{
				"default":  3600,
				"pages":    7200,
				"api":      300,
				"sessions": 86400,
			})},
		}
		for _, pair := range configPairs {
			if err := s.store.KV().Put(ctx, nsID, pair); err == nil {
				pairCount++
			}
		}
	}

	// SESSIONS namespace
	if nsID, ok := s.ids.KVNamespaces["SESSIONS"]; ok {
		sessionPairs := []*store.KVPair{
			{Key: "session:abc123", Value: mustJSON(map[string]interface{}{
				"user_id": "user_001",
				"email":   "john@example.com",
				"role":    "admin",
				"expires": s.timeFuture(24 * time.Hour).Unix(),
				"ip":      "192.168.1.100",
			}), Metadata: map[string]string{"user": "john"}},
			{Key: "session:def456", Value: mustJSON(map[string]interface{}{
				"user_id": "user_002",
				"email":   "jane@example.com",
				"role":    "user",
				"expires": s.timeFuture(12 * time.Hour).Unix(),
				"ip":      "192.168.1.101",
			}), Metadata: map[string]string{"user": "jane"}},
			{Key: "session:ghi789", Value: mustJSON(map[string]interface{}{
				"user_id": "user_003",
				"email":   "bob@example.com",
				"role":    "user",
				"expires": s.timeFuture(6 * time.Hour).Unix(),
				"ip":      "192.168.1.102",
			}), Metadata: map[string]string{"user": "bob"}},
		}
		for _, pair := range sessionPairs {
			if err := s.store.KV().Put(ctx, nsID, pair); err == nil {
				pairCount++
			}
		}
	}

	// CACHE namespace
	if nsID, ok := s.ids.KVNamespaces["CACHE"]; ok {
		cachePairs := []*store.KVPair{
			{Key: "page:/home", Value: mustJSON(map[string]interface{}{
				"html":      "<html><body>Homepage content</body></html>",
				"cached_at": s.timeAgo(1 * time.Hour).Unix(),
				"ttl":       7200,
			})},
			{Key: "page:/about", Value: mustJSON(map[string]interface{}{
				"html":      "<html><body>About us content</body></html>",
				"cached_at": s.timeAgo(2 * time.Hour).Unix(),
				"ttl":       86400,
			})},
			{Key: "api:products:list", Value: mustJSON(map[string]interface{}{
				"data":      []string{"prod_1", "prod_2", "prod_3"},
				"cached_at": s.timeAgo(30 * time.Minute).Unix(),
				"ttl":       300,
			})},
			{Key: "api:categories", Value: mustJSON(map[string]interface{}{
				"data":      []string{"Electronics", "Clothing", "Books", "Home", "Sports"},
				"cached_at": s.timeAgo(1 * time.Hour).Unix(),
				"ttl":       3600,
			})},
		}
		for _, pair := range cachePairs {
			if err := s.store.KV().Put(ctx, nsID, pair); err == nil {
				pairCount++
			}
		}
	}

	// RATE_LIMITS namespace
	if nsID, ok := s.ids.KVNamespaces["RATE_LIMITS"]; ok {
		ratePairs := []*store.KVPair{
			{Key: "ip:192.168.1.100", Value: mustJSON(map[string]interface{}{
				"count":        42,
				"window_start": s.timeAgo(30 * time.Minute).Unix(),
				"limit":        100,
			})},
			{Key: "ip:192.168.1.101", Value: mustJSON(map[string]interface{}{
				"count":        15,
				"window_start": s.timeAgo(15 * time.Minute).Unix(),
				"limit":        100,
			})},
			{Key: "user:user_001", Value: mustJSON(map[string]interface{}{
				"count":        8,
				"window_start": s.timeAgo(10 * time.Minute).Unix(),
				"limit":        1000,
			})},
			{Key: "api:v1:global", Value: mustJSON(map[string]interface{}{
				"count":        1523,
				"window_start": s.timeAgo(1 * time.Hour).Unix(),
				"limit":        10000,
			})},
		}
		for _, pair := range ratePairs {
			if err := s.store.KV().Put(ctx, nsID, pair); err == nil {
				pairCount++
			}
		}
	}

	slog.Info("KV seeded", "namespaces", nsCount, "pairs", pairCount)
	return nil
}

func (s *Seeder) seedR2(ctx context.Context) error {
	slog.Info("seeding R2 buckets")

	bucketCount := 0
	objectCount := 0

	// R2 Buckets
	buckets := []*store.R2Bucket{
		{ID: generateID(), Name: "assets", Location: "auto", CreatedAt: s.timeAgo(90 * 24 * time.Hour)},
		{ID: generateID(), Name: "uploads", Location: "wnam", CreatedAt: s.timeAgo(60 * 24 * time.Hour)},
		{ID: generateID(), Name: "backups", Location: "eeur", CreatedAt: s.timeAgo(120 * 24 * time.Hour)},
		{ID: generateID(), Name: "logs", Location: "auto", CreatedAt: s.timeAgo(30 * 24 * time.Hour)},
	}

	for _, bucket := range buckets {
		if err := s.store.R2().CreateBucket(ctx, bucket); err == nil {
			s.ids.R2Buckets[bucket.Name] = bucket.ID
			bucketCount++
		}
	}

	// Sample objects for assets bucket
	if bucketID, ok := s.ids.R2Buckets["assets"]; ok {
		objects := []struct {
			key      string
			data     []byte
			metadata map[string]string
		}{
			{"images/logo.png", []byte("PNG...logo image data..."), map[string]string{"content-type": "image/png"}},
			{"images/hero.jpg", make([]byte, 45*1024), map[string]string{"content-type": "image/jpeg"}},
			{"css/main.css", []byte("body { margin: 0; padding: 0; font-family: sans-serif; }"), map[string]string{"content-type": "text/css"}},
			{"js/app.js", []byte("console.log('Localflare App');"), map[string]string{"content-type": "application/javascript"}},
			{"fonts/inter.woff2", make([]byte, 24*1024), map[string]string{"content-type": "font/woff2"}},
			{"favicon.ico", make([]byte, 1*1024), map[string]string{"content-type": "image/x-icon"}},
		}
		for _, obj := range objects {
			if err := s.store.R2().PutObject(ctx, bucketID, obj.key, obj.data, obj.metadata); err == nil {
				objectCount++
			}
		}
	}

	// Sample objects for uploads bucket
	if bucketID, ok := s.ids.R2Buckets["uploads"]; ok {
		objects := []struct {
			key      string
			data     []byte
			metadata map[string]string
		}{
			{"user_001/avatar.jpg", make([]byte, 15*1024), map[string]string{"content-type": "image/jpeg", "user": "user_001"}},
			{"user_001/documents/invoice.pdf", make([]byte, 85*1024), map[string]string{"content-type": "application/pdf", "user": "user_001"}},
			{"user_002/avatar.png", make([]byte, 22*1024), map[string]string{"content-type": "image/png", "user": "user_002"}},
			{"user_003/resume.pdf", make([]byte, 120*1024), map[string]string{"content-type": "application/pdf", "user": "user_003"}},
		}
		for _, obj := range objects {
			if err := s.store.R2().PutObject(ctx, bucketID, obj.key, obj.data, obj.metadata); err == nil {
				objectCount++
			}
		}
	}

	// Sample objects for backups bucket
	if bucketID, ok := s.ids.R2Buckets["backups"]; ok {
		objects := []struct {
			key      string
			data     []byte
			metadata map[string]string
		}{
			{fmt.Sprintf("db/%s.sql.gz", s.timeAgo(24*time.Hour).Format("2006-01-02")), make([]byte, 512*1024), map[string]string{"content-type": "application/gzip"}},
			{fmt.Sprintf("db/%s.sql.gz", s.timeAgo(48*time.Hour).Format("2006-01-02")), make([]byte, 508*1024), map[string]string{"content-type": "application/gzip"}},
			{fmt.Sprintf("db/%s.sql.gz", s.timeAgo(72*time.Hour).Format("2006-01-02")), make([]byte, 495*1024), map[string]string{"content-type": "application/gzip"}},
		}
		for _, obj := range objects {
			if err := s.store.R2().PutObject(ctx, bucketID, obj.key, obj.data, obj.metadata); err == nil {
				objectCount++
			}
		}
	}

	// Sample objects for logs bucket
	if bucketID, ok := s.ids.R2Buckets["logs"]; ok {
		today := s.now.Format("2006/01/02")
		objects := []struct {
			key      string
			data     []byte
			metadata map[string]string
		}{
			{today + "/access.log", []byte("192.168.1.1 - - [15/Jan/2024:10:00:00] \"GET / HTTP/1.1\" 200 1234\n"), map[string]string{"content-type": "text/plain"}},
			{today + "/error.log", []byte("[ERROR] 2024-01-15 10:05:23 Connection timeout\n"), map[string]string{"content-type": "text/plain"}},
			{today + "/app.log", []byte("[INFO] Application started\n[DEBUG] Config loaded\n"), map[string]string{"content-type": "text/plain"}},
		}
		for _, obj := range objects {
			if err := s.store.R2().PutObject(ctx, bucketID, obj.key, obj.data, obj.metadata); err == nil {
				objectCount++
			}
		}
	}

	slog.Info("R2 seeded", "buckets", bucketCount, "objects", objectCount)
	return nil
}

func (s *Seeder) seedD1(ctx context.Context) error {
	slog.Info("seeding D1 databases")

	dbCount := 0

	// D1 Databases
	databases := []*store.D1Database{
		{ID: generateID(), Name: "main", Version: "1", NumTables: 0, FileSize: 0, CreatedAt: s.timeAgo(90 * 24 * time.Hour)},
		{ID: generateID(), Name: "ecommerce", Version: "1", NumTables: 0, FileSize: 0, CreatedAt: s.timeAgo(60 * 24 * time.Hour)},
		{ID: generateID(), Name: "analytics", Version: "1", NumTables: 0, FileSize: 0, CreatedAt: s.timeAgo(30 * 24 * time.Hour)},
	}

	for _, db := range databases {
		if err := s.store.D1().CreateDatabase(ctx, db); err == nil {
			s.ids.D1Databases[db.Name] = db.ID
			dbCount++
		}
	}

	// Seed main database (blog CMS)
	if dbID, ok := s.ids.D1Databases["main"]; ok {
		// Create tables
		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)

		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			author_id INTEGER REFERENCES users(id),
			title TEXT NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			content TEXT,
			status TEXT DEFAULT 'draft',
			published_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)

		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER REFERENCES posts(id),
			author_name TEXT NOT NULL,
			author_email TEXT,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)

		// Insert users
		users := []struct{ email, name, role string }{
			{"admin@example.com", "Admin User", "admin"},
			{"john@example.com", "John Doe", "author"},
			{"jane@example.com", "Jane Smith", "author"},
			{"bob@example.com", "Bob Wilson", "user"},
			{"alice@example.com", "Alice Brown", "user"},
		}
		for _, u := range users {
			s.store.D1().Exec(ctx, dbID, `INSERT INTO users (email, name, role) VALUES (?, ?, ?)`, []interface{}{u.email, u.name, u.role})
		}

		// Insert posts
		posts := []struct {
			authorID                int
			title, slug, content    string
			status                  string
			published               bool
		}{
			{2, "Getting Started with Localflare", "getting-started", "Learn how to set up Localflare for local development...", "published", true},
			{2, "DNS Management Best Practices", "dns-best-practices", "A comprehensive guide to managing DNS records...", "published", true},
			{3, "Understanding Workers", "understanding-workers", "Deep dive into Cloudflare Workers...", "published", true},
			{3, "R2 Storage Guide", "r2-storage-guide", "Everything you need to know about R2 object storage...", "published", true},
			{2, "D1 Database Tutorial", "d1-database-tutorial", "Build applications with D1 serverless database...", "published", true},
			{3, "Load Balancing Strategies", "load-balancing", "Configure load balancers for high availability...", "published", true},
			{2, "Security with WAF", "security-waf", "Protect your applications with Web Application Firewall...", "published", true},
			{3, "Caching Strategies", "caching-strategies", "Optimize performance with intelligent caching...", "published", true},
			{2, "Upcoming Features", "upcoming-features", "Preview of new features coming soon...", "draft", false},
			{3, "Advanced Analytics", "advanced-analytics", "Draft post about analytics...", "draft", false},
		}
		for i, p := range posts {
			var publishedAt interface{}
			if p.published {
				publishedAt = s.timeAgo(time.Duration(len(posts)-i) * 24 * time.Hour).Format("2006-01-02 15:04:05")
			}
			s.store.D1().Exec(ctx, dbID, `INSERT INTO posts (author_id, title, slug, content, status, published_at) VALUES (?, ?, ?, ?, ?, ?)`,
				[]interface{}{p.authorID, p.title, p.slug, p.content, p.status, publishedAt})
		}

		// Insert comments
		comments := []struct {
			postID      int
			name, email string
			content     string
		}{
			{1, "Mike", "mike@test.com", "Great tutorial! Very helpful."},
			{1, "Sarah", "sarah@test.com", "This saved me hours of work."},
			{1, "Tom", "tom@test.com", "Clear and concise. Thanks!"},
			{2, "Lisa", "lisa@test.com", "DNS can be tricky, this helps a lot."},
			{2, "Chris", "chris@test.com", "Good examples!"},
			{3, "Emma", "emma@test.com", "Workers are amazing!"},
			{3, "David", "david@test.com", "Edge computing FTW."},
			{3, "Amy", "amy@test.com", "Very detailed explanation."},
			{4, "James", "james@test.com", "R2 is a game changer."},
			{4, "Kate", "kate@test.com", "S3-compatible and cheaper!"},
			{5, "Steve", "steve@test.com", "D1 makes serverless DB easy."},
			{5, "Laura", "laura@test.com", "SQLite everywhere!"},
			{6, "Peter", "peter@test.com", "Great load balancing tips."},
			{7, "Nina", "nina@test.com", "Security first, always."},
			{8, "Alex", "alex@test.com", "Cache all the things!"},
		}
		for _, c := range comments {
			s.store.D1().Exec(ctx, dbID, `INSERT INTO comments (post_id, author_name, author_email, content) VALUES (?, ?, ?, ?)`,
				[]interface{}{c.postID, c.name, c.email, c.content})
		}
	}

	// Seed ecommerce database
	if dbID, ok := s.ids.D1Databases["ecommerce"]; ok {
		// Create tables
		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sku TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			price REAL NOT NULL,
			inventory INTEGER DEFAULT 0,
			category TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)

		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS customers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)

		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER REFERENCES customers(id),
			status TEXT DEFAULT 'pending',
			total REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)

		// Insert products
		products := []struct {
			sku, name, desc string
			price           float64
			inventory       int
			category        string
		}{
			{"ELEC-001", "Wireless Bluetooth Headphones", "Premium wireless headphones with ANC", 149.99, 50, "Electronics"},
			{"ELEC-002", "USB-C Charging Cable", "Fast charging cable, 6ft", 19.99, 200, "Electronics"},
			{"ELEC-003", "Portable Power Bank", "10000mAh battery pack", 39.99, 75, "Electronics"},
			{"ELEC-004", "Smart Watch Pro", "Fitness tracker with GPS", 299.99, 30, "Electronics"},
			{"CLOTH-001", "Classic Cotton T-Shirt", "100% cotton, various colors", 24.99, 150, "Clothing"},
			{"CLOTH-002", "Denim Jeans", "Slim fit, dark wash", 59.99, 80, "Clothing"},
			{"CLOTH-003", "Running Shoes", "Lightweight athletic shoes", 89.99, 45, "Clothing"},
			{"CLOTH-004", "Winter Jacket", "Insulated waterproof jacket", 129.99, 25, "Clothing"},
			{"BOOK-001", "Clean Code", "Robert C. Martin", 34.99, 60, "Books"},
			{"BOOK-002", "The Go Programming Language", "Donovan & Kernighan", 44.99, 40, "Books"},
			{"BOOK-003", "Designing Data-Intensive Apps", "Martin Kleppmann", 49.99, 35, "Books"},
			{"BOOK-004", "System Design Interview", "Alex Xu", 39.99, 55, "Books"},
			{"HOME-001", "Coffee Maker", "12-cup programmable", 79.99, 20, "Home"},
			{"HOME-002", "Robot Vacuum", "Smart navigation vacuum", 249.99, 15, "Home"},
			{"HOME-003", "Air Purifier", "HEPA filter, large room", 159.99, 22, "Home"},
			{"HOME-004", "Standing Desk", "Electric adjustable height", 449.99, 10, "Home"},
			{"SPORT-001", "Yoga Mat", "Extra thick, non-slip", 29.99, 100, "Sports"},
			{"SPORT-002", "Resistance Bands Set", "5 levels of resistance", 24.99, 85, "Sports"},
			{"SPORT-003", "Dumbbells Set", "Adjustable 5-25 lbs", 149.99, 30, "Sports"},
			{"SPORT-004", "Cycling Helmet", "Lightweight ventilated", 79.99, 40, "Sports"},
		}
		for _, p := range products {
			s.store.D1().Exec(ctx, dbID, `INSERT INTO products (sku, name, description, price, inventory, category) VALUES (?, ?, ?, ?, ?, ?)`,
				[]interface{}{p.sku, p.name, p.desc, p.price, p.inventory, p.category})
		}

		// Insert customers
		customers := []struct{ email, name, address string }{
			{"customer1@test.com", "Customer One", "123 Main St, City A"},
			{"customer2@test.com", "Customer Two", "456 Oak Ave, City B"},
			{"customer3@test.com", "Customer Three", "789 Pine Rd, City C"},
			{"customer4@test.com", "Customer Four", "321 Elm St, City D"},
			{"customer5@test.com", "Customer Five", "654 Maple Dr, City E"},
			{"customer6@test.com", "Customer Six", "987 Cedar Ln, City F"},
			{"customer7@test.com", "Customer Seven", "147 Birch Blvd, City G"},
			{"customer8@test.com", "Customer Eight", "258 Walnut Way, City H"},
			{"customer9@test.com", "Customer Nine", "369 Cherry Ct, City I"},
			{"customer10@test.com", "Customer Ten", "741 Spruce St, City J"},
		}
		for _, c := range customers {
			s.store.D1().Exec(ctx, dbID, `INSERT INTO customers (email, name, address) VALUES (?, ?, ?)`,
				[]interface{}{c.email, c.name, c.address})
		}

		// Insert orders
		orders := []struct {
			customerID int
			status     string
			total      float64
		}{
			{1, "completed", 169.98},
			{1, "completed", 89.99},
			{2, "completed", 344.97},
			{2, "shipped", 79.99},
			{3, "completed", 59.99},
			{3, "processing", 449.99},
			{4, "completed", 154.98},
			{4, "completed", 39.99},
			{5, "shipped", 299.99},
			{5, "pending", 24.99},
			{6, "completed", 249.99},
			{7, "completed", 129.99},
			{7, "processing", 179.98},
			{8, "completed", 44.99},
			{9, "shipped", 159.99},
			{10, "completed", 84.98},
		}
		for _, o := range orders {
			s.store.D1().Exec(ctx, dbID, `INSERT INTO orders (customer_id, status, total) VALUES (?, ?, ?)`,
				[]interface{}{o.customerID, o.status, o.total})
		}
	}

	// Seed analytics database
	if dbID, ok := s.ids.D1Databases["analytics"]; ok {
		// Create tables
		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			user_id TEXT,
			properties TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)

		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			pages_viewed INTEGER DEFAULT 0,
			device TEXT,
			browser TEXT
		)`, nil)

		s.store.D1().Exec(ctx, dbID, `CREATE TABLE IF NOT EXISTS pageviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT REFERENCES sessions(id),
			path TEXT NOT NULL,
			referrer TEXT,
			duration INTEGER,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, nil)

		// Insert sample events
		eventTypes := []string{"page_view", "click", "form_submit", "purchase", "signup", "login"}
		for i := 0; i < 50; i++ {
			eventType := eventTypes[i%len(eventTypes)]
			userID := fmt.Sprintf("user_%03d", (i%10)+1)
			props := mustJSON(map[string]interface{}{
				"page":   fmt.Sprintf("/page-%d", i%5),
				"source": []string{"google", "direct", "social", "email"}[i%4],
			})
			s.store.D1().Exec(ctx, dbID, `INSERT INTO events (event_type, user_id, properties) VALUES (?, ?, ?)`,
				[]interface{}{eventType, userID, string(props)})
		}
	}

	slog.Info("D1 seeded", "databases", dbCount)
	return nil
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
