package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// Store implements store.Store using SQLite.
type Store struct {
	db *sql.DB

	search     *SearchStore
	index      *IndexStore
	suggest    *SuggestStore
	knowledge  *KnowledgeStore
	history    *HistoryStore
	preference *PreferenceStore
	cache      *CacheStore

	// AI stores
	session *SessionStore
	canvas  *CanvasStore
	chunker *ChunkerStore

	// Kagi stores
	bang     *BangStore
	summary  *SummaryStore
	widget   *WidgetStore
	smallWeb *SmallWebStore
}

// New creates a new SQLite store.
func New(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite only supports one writer at a time
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Store{db: db}
	s.search = &SearchStore{db: db}
	s.index = &IndexStore{db: db}
	s.suggest = &SuggestStore{db: db}
	s.knowledge = &KnowledgeStore{db: db}
	s.history = &HistoryStore{db: db}
	s.preference = &PreferenceStore{db: db}
	s.cache = NewCacheStore(db)

	// AI stores
	s.session = NewSessionStore(db)
	s.canvas = NewCanvasStore(db)
	s.chunker = NewChunkerStore(db)

	// Kagi stores
	s.bang = &BangStore{db: db}
	s.summary = &SummaryStore{db: db}
	s.widget = &WidgetStore{db: db}
	s.smallWeb = &SmallWebStore{db: db}

	return s, nil
}

// Cache returns the cache store.
func (s *Store) Cache() *CacheStore {
	return s.cache
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateExtensions is a no-op for SQLite (FTS5 is built-in).
func (s *Store) CreateExtensions(ctx context.Context) error {
	return nil
}

// Ensure creates all required tables and FTS indexes.
func (s *Store) Ensure(ctx context.Context) error {
	if err := createSchema(ctx, s.db); err != nil {
		return err
	}
	return createAISchema(ctx, s.db)
}

// Feature store accessors

func (s *Store) Search() store.SearchStore {
	return s.search
}

func (s *Store) Index() store.IndexStore {
	return s.index
}

func (s *Store) Suggest() store.SuggestStore {
	return s.suggest
}

func (s *Store) Knowledge() store.KnowledgeStore {
	return s.knowledge
}

func (s *Store) History() store.HistoryStore {
	return s.history
}

func (s *Store) Preference() store.PreferenceStore {
	return s.preference
}

func (s *Store) Bang() store.BangStore {
	return s.bang
}

func (s *Store) Summary() store.SummaryStore {
	return s.summary
}

func (s *Store) Widget() store.WidgetStore {
	return s.widget
}

func (s *Store) SmallWeb() store.SmallWebStore {
	return s.smallWeb
}

// AI store accessors

func (s *Store) Session() *SessionStore {
	return s.session
}

func (s *Store) Canvas() *CanvasStore {
	return s.canvas
}

func (s *Store) Chunker() *ChunkerStore {
	return s.chunker
}

// SeedDocuments seeds sample documents.
func (s *Store) SeedDocuments(ctx context.Context) error {
	docs := []store.Document{
		{
			URL:         "https://golang.org/",
			Title:       "The Go Programming Language",
			Description: "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
			Content:     "Go is expressive, concise, clean, and efficient. Its concurrency mechanisms make it easy to write programs that get the most out of multicore and networked machines. Go compiles quickly to machine code yet has the convenience of garbage collection and the power of run-time reflection.",
			Domain:      "golang.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://golang.org/favicon.ico",
		},
		{
			URL:         "https://rust-lang.org/",
			Title:       "Rust Programming Language",
			Description: "A language empowering everyone to build reliable and efficient software.",
			Content:     "Rust is blazingly fast and memory-efficient: with no runtime or garbage collector, it can power performance-critical services, run on embedded devices, and easily integrate with other languages.",
			Domain:      "rust-lang.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://rust-lang.org/favicon.ico",
		},
		{
			URL:         "https://python.org/",
			Title:       "Welcome to Python.org",
			Description: "The official home of the Python Programming Language",
			Content:     "Python is a programming language that lets you work quickly and integrate systems more effectively. Python is powerful and fast, plays well with others, runs everywhere, is friendly and easy to learn, and is open source.",
			Domain:      "python.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://python.org/favicon.ico",
		},
		{
			URL:         "https://developer.mozilla.org/en-US/docs/Web/JavaScript",
			Title:       "JavaScript | MDN",
			Description: "JavaScript (JS) is a lightweight interpreted programming language with first-class functions.",
			Content:     "JavaScript is a prototype-based, multi-paradigm, single-threaded, dynamic language, supporting object-oriented, imperative, and declarative styles. The standards for JavaScript are the ECMAScript Language Specification.",
			Domain:      "developer.mozilla.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://developer.mozilla.org/favicon.ico",
		},
		{
			URL:         "https://www.typescriptlang.org/",
			Title:       "TypeScript: JavaScript With Syntax For Types",
			Description: "TypeScript extends JavaScript by adding types to the language.",
			Content:     "TypeScript is a strongly typed programming language that builds on JavaScript, giving you better tooling at any scale. TypeScript code converts to JavaScript, which runs anywhere JavaScript runs.",
			Domain:      "typescriptlang.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://typescriptlang.org/favicon.ico",
		},
		{
			URL:         "https://reactjs.org/",
			Title:       "React - A JavaScript library for building user interfaces",
			Description: "React is a JavaScript library for building user interfaces, created by Facebook.",
			Content:     "React makes it painless to create interactive UIs. Design simple views for each state in your application, and React will efficiently update and render just the right components when your data changes. Build encapsulated components that manage their own state, then compose them to make complex UIs.",
			Domain:      "reactjs.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://reactjs.org/favicon.ico",
		},
		{
			URL:         "https://vuejs.org/",
			Title:       "Vue.js - The Progressive JavaScript Framework",
			Description: "Vue.js is a progressive, incrementally-adoptable JavaScript framework for building UI on the web.",
			Content:     "Vue is a progressive framework for building user interfaces. Unlike other monolithic frameworks, Vue is designed from the ground up to be incrementally adoptable. The core library is focused on the view layer only.",
			Domain:      "vuejs.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://vuejs.org/favicon.ico",
		},
		{
			URL:         "https://angular.io/",
			Title:       "Angular",
			Description: "Angular is a platform for building mobile and desktop web applications.",
			Content:     "Angular is a platform and framework for building single-page client applications using HTML and TypeScript. Angular is written in TypeScript. It implements core and optional functionality as a set of TypeScript libraries that you import into your applications.",
			Domain:      "angular.io",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://angular.io/favicon.ico",
		},
		{
			URL:         "https://nodejs.org/",
			Title:       "Node.js",
			Description: "Node.js is a JavaScript runtime built on Chrome's V8 JavaScript engine.",
			Content:     "As an asynchronous event-driven JavaScript runtime, Node.js is designed to build scalable network applications. Node.js is similar in design to, and influenced by, systems like Ruby's Event Machine and Python's Twisted.",
			Domain:      "nodejs.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://nodejs.org/favicon.ico",
		},
		{
			URL:         "https://www.postgresql.org/",
			Title:       "PostgreSQL: The World's Most Advanced Open Source Database",
			Description: "PostgreSQL is a powerful, open source object-relational database system with a strong reputation for reliability and features.",
			Content:     "PostgreSQL is a powerful, open source object-relational database system with over 35 years of active development. PostgreSQL has earned a strong reputation for its proven architecture, reliability, data integrity, robust feature set, extensibility, and the dedication of the open source community.",
			Domain:      "postgresql.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://postgresql.org/favicon.ico",
		},
		{
			URL:         "https://www.mongodb.com/",
			Title:       "MongoDB: The Developer Data Platform",
			Description: "MongoDB is a document database with the scalability and flexibility that you want.",
			Content:     "MongoDB is a general purpose, document-based, distributed database built for modern application developers and for the cloud era. MongoDB stores data in flexible, JSON-like documents, meaning fields can vary from document to document and data structure can be changed over time.",
			Domain:      "mongodb.com",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://mongodb.com/favicon.ico",
		},
		{
			URL:         "https://redis.io/",
			Title:       "Redis - The Real-time Data Platform",
			Description: "Redis is an open source, in-memory data structure store used as a database, cache, message broker, and streaming engine.",
			Content:     "Redis is an in-memory data structure store, used as a distributed, in-memory key-value database, cache and message broker, with optional durability. Redis supports different kinds of abstract data structures, such as strings, lists, maps, sets, sorted sets, HyperLogLogs, bitmaps, streams, and spatial indexes.",
			Domain:      "redis.io",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://redis.io/favicon.ico",
		},
		{
			URL:         "https://kubernetes.io/",
			Title:       "Kubernetes - Production-Grade Container Orchestration",
			Description: "Kubernetes is an open-source system for automating deployment, scaling, and management of containerized applications.",
			Content:     "Kubernetes, also known as K8s, is an open-source system for automating deployment, scaling, and management of containerized applications. It groups containers that make up an application into logical units for easy management and discovery.",
			Domain:      "kubernetes.io",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://kubernetes.io/favicon.ico",
		},
		{
			URL:         "https://www.docker.com/",
			Title:       "Docker: Accelerated Container Application Development",
			Description: "Docker is a platform designed to help developers build, share, and run container applications.",
			Content:     "Docker is a set of platform as a service products that use OS-level virtualization to deliver software in packages called containers. The service has both free and premium tiers. The software that hosts the containers is called Docker Engine.",
			Domain:      "docker.com",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://docker.com/favicon.ico",
		},
		{
			URL:         "https://github.com/",
			Title:       "GitHub: Let's build from here",
			Description: "GitHub is where over 100 million developers shape the future of software, together.",
			Content:     "GitHub is a developer platform that allows developers to create, store, manage and share their code. It uses Git software, providing the distributed version control of Git plus access control, bug tracking, software feature requests, task management, continuous integration.",
			Domain:      "github.com",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://github.com/favicon.ico",
		},
		{
			URL:         "https://svelte.dev/",
			Title:       "Svelte - Cybernetically enhanced web apps",
			Description: "Svelte is a radical new approach to building user interfaces.",
			Content:     "Svelte is a radical new approach to building user interfaces. Whereas traditional frameworks like React and Vue do the bulk of their work in the browser, Svelte shifts that work into a compile step that happens when you build your app.",
			Domain:      "svelte.dev",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://svelte.dev/favicon.ico",
		},
		{
			URL:         "https://nextjs.org/",
			Title:       "Next.js by Vercel - The React Framework",
			Description: "Next.js gives you the best developer experience with all the features you need for production.",
			Content:     "Next.js enables you to create full-stack Web applications by extending the latest React features, and integrating powerful Rust-based JavaScript tooling for the fastest builds.",
			Domain:      "nextjs.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://nextjs.org/favicon.ico",
		},
		{
			URL:         "https://tailwindcss.com/",
			Title:       "Tailwind CSS - Rapidly build modern websites without ever leaving your HTML",
			Description: "A utility-first CSS framework packed with classes that can be composed to build any design, directly in your markup.",
			Content:     "Tailwind CSS is a utility-first CSS framework for rapidly building custom user interfaces. It provides low-level utility classes that let you build completely custom designs without ever leaving your HTML.",
			Domain:      "tailwindcss.com",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://tailwindcss.com/favicon.ico",
		},
		{
			URL:         "https://graphql.org/",
			Title:       "GraphQL | A query language for your API",
			Description: "GraphQL is a query language for APIs and a runtime for fulfilling those queries with your existing data.",
			Content:     "GraphQL provides a complete and understandable description of the data in your API, gives clients the power to ask for exactly what they need and nothing more, makes it easier to evolve APIs over time, and enables powerful developer tools.",
			Domain:      "graphql.org",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://graphql.org/favicon.ico",
		},
		{
			URL:         "https://www.elastic.co/elasticsearch/",
			Title:       "Elasticsearch: The Official Distributed Search & Analytics Engine",
			Description: "Elasticsearch is a distributed, RESTful search and analytics engine capable of solving a growing number of use cases.",
			Content:     "Elasticsearch is the distributed search and analytics engine at the heart of the Elastic Stack. It provides near real-time search and analytics for all types of data.",
			Domain:      "elastic.co",
			Language:    "en",
			ContentType: "text/html",
			Favicon:     "https://elastic.co/favicon.ico",
		},
	}

	for _, doc := range docs {
		if err := s.index.IndexDocument(ctx, &doc); err != nil {
			// Ignore duplicate errors
			continue
		}
	}

	// Seed some suggestions
	suggestions := []string{
		"golang",
		"go programming",
		"go tutorial",
		"python",
		"python tutorial",
		"javascript",
		"react",
		"react hooks",
		"vue.js",
		"typescript",
		"node.js",
		"docker",
		"kubernetes",
		"postgresql",
		"mongodb",
		"redis",
		"github",
		"programming languages",
		"web development",
		"database",
		"machine learning",
		"artificial intelligence",
		"data science",
		"cloud computing",
		"devops",
		"api design",
		"microservices",
		"graphql",
		"rest api",
		"authentication",
	}

	for _, q := range suggestions {
		s.suggest.RecordQuery(ctx, q)
	}

	// Seed images
	if err := s.seedImages(ctx); err != nil {
		return err
	}

	// Seed videos
	if err := s.seedVideos(ctx); err != nil {
		return err
	}

	// Seed news
	if err := s.seedNews(ctx); err != nil {
		return err
	}

	return nil
}

// seedImages seeds sample images.
func (s *Store) seedImages(ctx context.Context) error {
	images := []struct {
		url, thumb, title, source, domain, format string
		width, height                             int
		size                                      int64
	}{
		{"https://go.dev/blog/go-brand/Go-Logo/PNG/Go-Logo_Blue.png", "https://go.dev/blog/go-brand/Go-Logo/PNG/Go-Logo_Blue.png", "Go Programming Language Logo", "https://golang.org", "golang.org", "png", 800, 300, 45000},
		{"https://www.python.org/static/community_logos/python-logo-master-v3-TM.png", "https://www.python.org/static/community_logos/python-logo-master-v3-TM.png", "Python Logo", "https://python.org", "python.org", "png", 601, 203, 32000},
		{"https://upload.wikimedia.org/wikipedia/commons/6/6a/JavaScript-logo.png", "https://upload.wikimedia.org/wikipedia/commons/6/6a/JavaScript-logo.png", "JavaScript Logo", "https://developer.mozilla.org", "wikipedia.org", "png", 512, 512, 25000},
		{"https://www.docker.com/wp-content/uploads/2022/03/Moby-logo.png", "https://www.docker.com/wp-content/uploads/2022/03/Moby-logo.png", "Docker Moby Logo", "https://docker.com", "docker.com", "png", 800, 600, 50000},
		{"https://www.postgresql.org/media/img/about/press/elephant.png", "https://www.postgresql.org/media/img/about/press/elephant.png", "PostgreSQL Elephant Logo", "https://postgresql.org", "postgresql.org", "png", 450, 450, 35000},
		{"https://raw.githubusercontent.com/kubernetes/kubernetes/master/logo/logo.png", "https://raw.githubusercontent.com/kubernetes/kubernetes/master/logo/logo.png", "Kubernetes Logo", "https://kubernetes.io", "github.com", "png", 722, 702, 48000},
		{"https://reactjs.org/logo-og.png", "https://reactjs.org/logo-og.png", "React Logo", "https://reactjs.org", "reactjs.org", "png", 1200, 630, 55000},
		{"https://vuejs.org/images/logo.png", "https://vuejs.org/images/logo.png", "Vue.js Logo", "https://vuejs.org", "vuejs.org", "png", 400, 400, 28000},
		{"https://angular.io/assets/images/logos/angular/angular.png", "https://angular.io/assets/images/logos/angular/angular.png", "Angular Logo", "https://angular.io", "angular.io", "png", 500, 500, 42000},
		{"https://nodejs.org/static/images/logo.svg", "https://nodejs.org/static/images/logo.svg", "Node.js Logo", "https://nodejs.org", "nodejs.org", "svg", 442, 270, 12000},
		{"https://www.rust-lang.org/static/images/rust-logo-blk.svg", "https://www.rust-lang.org/static/images/rust-logo-blk.svg", "Rust Programming Language Logo", "https://rust-lang.org", "rust-lang.org", "svg", 512, 512, 15000},
		{"https://www.typescriptlang.org/assets/images/icons/apple-touch-icon-180x180.png", "https://www.typescriptlang.org/assets/images/icons/apple-touch-icon-180x180.png", "TypeScript Logo", "https://typescriptlang.org", "typescriptlang.org", "png", 180, 180, 8000},
		{"https://upload.wikimedia.org/wikipedia/commons/9/91/Octicons-mark-github.svg", "https://upload.wikimedia.org/wikipedia/commons/9/91/Octicons-mark-github.svg", "GitHub Logo", "https://github.com", "wikipedia.org", "svg", 1024, 1024, 5000},
		{"https://redis.io/images/redis-logo.svg", "https://redis.io/images/redis-logo.svg", "Redis Logo", "https://redis.io", "redis.io", "svg", 200, 67, 3000},
		{"https://webassets.mongodb.com/_com_assets/cms/MongoDB_Logo_FullColorBlack_RGB-4td3yuxzjs.png", "https://webassets.mongodb.com/_com_assets/cms/MongoDB_Logo_FullColorBlack_RGB-4td3yuxzjs.png", "MongoDB Logo", "https://mongodb.com", "mongodb.com", "png", 1400, 400, 35000},
		{"https://www.elastic.co/static-res/images/elastic-logo-200.png", "https://www.elastic.co/static-res/images/elastic-logo-200.png", "Elasticsearch Logo", "https://elastic.co", "elastic.co", "png", 200, 200, 18000},
		{"https://graphql.org/img/logo.svg", "https://graphql.org/img/logo.svg", "GraphQL Logo", "https://graphql.org", "graphql.org", "svg", 400, 400, 6000},
		{"https://tailwindcss.com/_next/static/media/tailwindcss-mark.3c5441fc7a190e4a.svg", "https://tailwindcss.com/_next/static/media/tailwindcss-mark.3c5441fc7a190e4a.svg", "Tailwind CSS Logo", "https://tailwindcss.com", "tailwindcss.com", "svg", 262, 262, 4500},
		{"https://nextjs.org/static/favicon/favicon-32x32.png", "https://nextjs.org/static/favicon/favicon-32x32.png", "Next.js Logo", "https://nextjs.org", "nextjs.org", "png", 32, 32, 1200},
		{"https://svelte.dev/svelte-logo-horizontal.svg", "https://svelte.dev/svelte-logo-horizontal.svg", "Svelte Logo", "https://svelte.dev", "svelte.dev", "svg", 400, 100, 5000},
	}

	for i, img := range images {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO images (id, url, thumbnail_url, title, source_url, source_domain, width, height, file_size, format)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(url) DO NOTHING
		`, fmt.Sprintf("img_%d", i+1), img.url, img.thumb, img.title, img.source, img.domain, img.width, img.height, img.size, img.format)
		if err != nil {
			continue
		}
	}

	return nil
}

// seedVideos seeds sample videos.
func (s *Store) seedVideos(ctx context.Context) error {
	videos := []struct {
		url, thumb, title, desc, channel string
		duration                         int
		views                            int64
		published                        string
	}{
		{"https://www.youtube.com/watch?v=YS4e4q9oBaU", "https://i.ytimg.com/vi/YS4e4q9oBaU/maxresdefault.jpg", "Learn Go Programming - Golang Tutorial for Beginners", "Complete Go programming tutorial for beginners. Learn all the fundamentals of Go (Golang).", "freeCodeCamp.org", 27000, 3500000, "2024-01-15"},
		{"https://www.youtube.com/watch?v=rfscVS0vtbw", "https://i.ytimg.com/vi/rfscVS0vtbw/maxresdefault.jpg", "Learn Python - Full Course for Beginners", "This course will give you a full introduction into all of the core concepts in python.", "freeCodeCamp.org", 16200, 45000000, "2023-06-20"},
		{"https://www.youtube.com/watch?v=PkZNo7MFNFg", "https://i.ytimg.com/vi/PkZNo7MFNFg/maxresdefault.jpg", "Learn JavaScript - Full Course for Beginners", "This complete 134-part JavaScript tutorial for beginners will teach you everything you need to know.", "freeCodeCamp.org", 12600, 18000000, "2023-04-10"},
		{"https://www.youtube.com/watch?v=Tn6-PIqc4UM", "https://i.ytimg.com/vi/Tn6-PIqc4UM/maxresdefault.jpg", "React Course - Beginner's Tutorial for React JavaScript Library", "Learn React JS in this full course for beginners. React is a popular JavaScript library.", "freeCodeCamp.org", 43200, 8500000, "2024-02-28"},
		{"https://www.youtube.com/watch?v=qz0aGYrrlhU", "https://i.ytimg.com/vi/qz0aGYrrlhU/maxresdefault.jpg", "Docker Tutorial for Beginners - A Full DevOps Course", "This full Docker course for beginners covers everything you need to know to get started with Docker.", "TechWorld with Nana", 10800, 5200000, "2023-11-05"},
		{"https://www.youtube.com/watch?v=X48VuDVv0do", "https://i.ytimg.com/vi/X48VuDVv0do/maxresdefault.jpg", "Kubernetes Tutorial for Beginners", "Full Kubernetes Course covering all the main K8s components and concepts.", "TechWorld with Nana", 14400, 4800000, "2023-09-12"},
		{"https://www.youtube.com/watch?v=zsjvFFKOm3c", "https://i.ytimg.com/vi/zsjvFFKOm3c/maxresdefault.jpg", "PostgreSQL Tutorial Full Course 2024", "Learn PostgreSQL in this full database tutorial. PostgreSQL is one of the most popular databases.", "Programming with Mosh", 10200, 2100000, "2024-01-08"},
		{"https://www.youtube.com/watch?v=c2M-rlkkT5o", "https://i.ytimg.com/vi/c2M-rlkkT5o/maxresdefault.jpg", "MongoDB Tutorial for Beginners", "Complete MongoDB tutorial covering CRUD operations, aggregation, indexing, and more.", "Traversy Media", 7200, 1800000, "2023-08-22"},
		{"https://www.youtube.com/watch?v=OXGznpKZ_sA", "https://i.ytimg.com/vi/OXGznpKZ_sA/maxresdefault.jpg", "Vue.js Course for Beginners 2024", "Learn Vue.js in this full course. We'll cover all the fundamentals of Vue 3.", "freeCodeCamp.org", 18000, 1500000, "2024-03-01"},
		{"https://www.youtube.com/watch?v=dGcsHMXbSOA", "https://i.ytimg.com/vi/dGcsHMXbSOA/maxresdefault.jpg", "TypeScript Full Course for Beginners", "Learn TypeScript in this comprehensive tutorial for beginners.", "Dave Gray", 28800, 2300000, "2023-10-18"},
		{"https://www.youtube.com/watch?v=Ke90Tje7VS0", "https://i.ytimg.com/vi/Ke90Tje7VS0/maxresdefault.jpg", "React Redux Tutorial", "Complete Redux Toolkit tutorial with React for state management.", "Programming with Mosh", 3600, 980000, "2024-02-14"},
		{"https://www.youtube.com/watch?v=wm5gMKuwSYk", "https://i.ytimg.com/vi/wm5gMKuwSYk/maxresdefault.jpg", "GraphQL Full Course - Novice to Expert", "Learn GraphQL from scratch in this comprehensive course.", "freeCodeCamp.org", 14400, 1200000, "2023-07-30"},
		{"https://www.youtube.com/watch?v=KJgsSFOSQv0", "https://i.ytimg.com/vi/KJgsSFOSQv0/maxresdefault.jpg", "CSS Full Course - Includes Flexbox and Grid", "Complete CSS tutorial including Flexbox and CSS Grid.", "freeCodeCamp.org", 39600, 6500000, "2023-05-15"},
		{"https://www.youtube.com/watch?v=RGOj5yH7evk", "https://i.ytimg.com/vi/RGOj5yH7evk/maxresdefault.jpg", "Git and GitHub for Beginners - Crash Course", "Learn Git and GitHub in this crash course tutorial.", "freeCodeCamp.org", 3900, 4200000, "2023-03-25"},
		{"https://www.youtube.com/watch?v=BZP1rYjoBgI", "https://i.ytimg.com/vi/BZP1rYjoBgI/maxresdefault.jpg", "AWS Certified Cloud Practitioner Training", "Full AWS cloud practitioner certification course.", "freeCodeCamp.org", 50400, 3100000, "2024-01-22"},
	}

	for i, vid := range videos {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO videos (id, url, thumbnail_url, title, description, duration_seconds, channel, views, published_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(url) DO NOTHING
		`, fmt.Sprintf("vid_%d", i+1), vid.url, vid.thumb, vid.title, vid.desc, vid.duration, vid.channel, vid.views, vid.published)
		if err != nil {
			continue
		}
	}

	return nil
}

// seedNews seeds sample news articles.
func (s *Store) seedNews(ctx context.Context) error {
	news := []struct {
		url, title, snippet, source, image, published string
	}{
		{"https://blog.golang.org/go1.22", "Go 1.22 Released with Range Over Functions", "Go 1.22 introduces range over functions, making iterator patterns more natural in Go code.", "Go Blog", "https://go.dev/images/gophers/biplane.svg", "2024-02-06"},
		{"https://www.python.org/downloads/release/python-312", "Python 3.12 Brings Performance Improvements", "Python 3.12 introduces significant performance improvements and new typing features.", "Python.org", "https://www.python.org/static/img/python-logo.png", "2024-01-15"},
		{"https://github.blog/news-insights", "GitHub Copilot Now Available for Free", "GitHub announces free tier for Copilot AI assistant for all developers.", "GitHub Blog", "https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png", "2024-03-01"},
		{"https://reactjs.org/blog/2024", "React 19 Introduces Server Components", "The React team announces React 19 with built-in support for Server Components.", "React Blog", "https://reactjs.org/logo-og.png", "2024-02-20"},
		{"https://kubernetes.io/blog/2024", "Kubernetes 1.30 Released", "Kubernetes 1.30 brings enhanced security features and improved resource management.", "Kubernetes Blog", "https://kubernetes.io/images/kubernetes-horizontal-color.png", "2024-03-15"},
		{"https://www.docker.com/blog/2024", "Docker Desktop 5.0 Announcement", "Docker Desktop 5.0 brings AI-powered development tools and improved WSL integration.", "Docker Blog", "https://www.docker.com/wp-content/uploads/2022/03/Moby-logo.png", "2024-02-28"},
		{"https://techcrunch.com/ai-programming", "AI Transforms Software Development in 2024", "How AI coding assistants are changing the way developers write code.", "TechCrunch", "https://techcrunch.com/wp-content/uploads/2024/01/tc-logo.png", "2024-03-10"},
		{"https://www.infoworld.com/typescript", "TypeScript 5.4 Adds New Type Features", "Microsoft releases TypeScript 5.4 with improved type inference and new utility types.", "InfoWorld", "https://www.infoworld.com/images/logo.png", "2024-03-05"},
		{"https://blog.rust-lang.org/2024", "Rust Foundation Announces New Initiatives", "The Rust Foundation announces new programs to support Rust adoption in enterprises.", "Rust Blog", "https://www.rust-lang.org/static/images/rust-logo-blk.svg", "2024-02-25"},
		{"https://devblogs.microsoft.com/vscode", "VS Code February 2024 Release", "Visual Studio Code February 2024 update brings improved AI features and performance.", "Microsoft DevBlogs", "https://code.visualstudio.com/assets/images/code-stable.png", "2024-02-08"},
		{"https://www.mongodb.com/blog/atlas", "MongoDB Atlas Introduces Vector Search", "MongoDB adds vector search capabilities for AI and machine learning applications.", "MongoDB Blog", "https://webassets.mongodb.com/_com_assets/cms/MongoDB_Logo_FullColorBlack_RGB-4td3yuxzjs.png", "2024-03-12"},
		{"https://redis.io/blog/redis-8", "Redis 8.0 Preview Announced", "Redis announces preview of version 8.0 with new data structures and improved clustering.", "Redis Blog", "https://redis.io/images/redis-logo.svg", "2024-02-18"},
		{"https://www.postgresql.org/about/news", "PostgreSQL 17 Beta Released", "PostgreSQL 17 beta introduces improved JSON support and query performance.", "PostgreSQL News", "https://www.postgresql.org/media/img/about/press/elephant.png", "2024-03-08"},
		{"https://nextjs.org/blog/next-15", "Next.js 15 Introduces Turbopack", "Vercel releases Next.js 15 with Turbopack as the default bundler.", "Next.js Blog", "https://nextjs.org/static/favicon/favicon-32x32.png", "2024-03-18"},
		{"https://aws.amazon.com/blogs/compute", "AWS Lambda Adds Support for Node.js 22", "AWS Lambda now supports Node.js 22 runtime with improved cold start performance.", "AWS Blog", "https://a0.awsstatic.com/libra-css/images/logos/aws_logo_smile.png", "2024-03-20"},
		{"https://cloud.google.com/blog/products/gcp", "Google Cloud Introduces New AI Services", "Google Cloud announces new AI and ML services for enterprise developers.", "Google Cloud Blog", "https://www.gstatic.com/devrel-devsite/prod/v93e1bba94ddeb00b71e9c9e5b86fdd5e27f66cee527816ec7e46f06eaa3ad4fc/cloud/images/cloud-logo.svg", "2024-03-22"},
		{"https://azure.microsoft.com/blog", "Azure Kubernetes Service Updates", "Microsoft announces new features for Azure Kubernetes Service including enhanced monitoring.", "Azure Blog", "https://azure.microsoft.com/svghandler/azure-kubernetes-service/", "2024-03-14"},
		{"https://tailwindcss.com/blog/tailwindcss-v4", "Tailwind CSS v4.0 Alpha Released", "Tailwind CSS team releases alpha version 4.0 with new JIT engine.", "Tailwind Blog", "https://tailwindcss.com/_next/static/media/tailwindcss-mark.3c5441fc7a190e4a.svg", "2024-03-25"},
		{"https://vuejs.org/blog/vue-3-4", "Vue 3.4 Released", "Vue.js 3.4 brings improved TypeScript support and reactivity enhancements.", "Vue.js Blog", "https://vuejs.org/images/logo.png", "2024-02-22"},
		{"https://svelte.dev/blog/svelte-5-preview", "Svelte 5 Preview: Runes", "Svelte 5 preview introduces Runes, a new way to handle reactivity.", "Svelte Blog", "https://svelte.dev/svelte-logo-horizontal.svg", "2024-03-02"},
	}

	for i, n := range news {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO news (id, url, title, snippet, source, image_url, published_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(url) DO NOTHING
		`, fmt.Sprintf("news_%d", i+1), n.url, n.title, n.snippet, n.source, n.image, n.published)
		if err != nil {
			continue
		}
	}

	return nil
}

// SeedKnowledge seeds sample knowledge entities.
func (s *Store) SeedKnowledge(ctx context.Context) error {
	entities := []store.Entity{
		{
			Name:        "Go",
			Type:        "programming_language",
			Description: "Go is a statically typed, compiled high-level programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson.",
			Image:       "https://go.dev/blog/go-brand/Go-Logo/PNG/Go-Logo_Blue.png",
			Facts: map[string]any{
				"Designed by":    "Robert Griesemer, Rob Pike, Ken Thompson",
				"First appeared": "2009",
				"Developer":      "Google",
				"Typing":         "Static, strong, inferred",
				"License":        "BSD-style",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://golang.org"},
				{Title: "Documentation", URL: "https://golang.org/doc"},
				{Title: "GitHub", URL: "https://github.com/golang/go"},
			},
		},
		{
			Name:        "Python",
			Type:        "programming_language",
			Description: "Python is a high-level, general-purpose programming language. Its design philosophy emphasizes code readability with the use of significant indentation.",
			Image:       "https://www.python.org/static/community_logos/python-logo-master-v3-TM.png",
			Facts: map[string]any{
				"Designed by":    "Guido van Rossum",
				"First appeared": "1991",
				"Developer":      "Python Software Foundation",
				"Typing":         "Dynamic, strong",
				"License":        "Python Software Foundation License",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://python.org"},
				{Title: "Documentation", URL: "https://docs.python.org"},
				{Title: "PyPI", URL: "https://pypi.org"},
			},
		},
		{
			Name:        "JavaScript",
			Type:        "programming_language",
			Description: "JavaScript, often abbreviated as JS, is a programming language that is one of the core technologies of the World Wide Web, alongside HTML and CSS.",
			Image:       "https://upload.wikimedia.org/wikipedia/commons/6/6a/JavaScript-logo.png",
			Facts: map[string]any{
				"Designed by":    "Brendan Eich",
				"First appeared": "1995",
				"Developer":      "Netscape, Mozilla Foundation, Ecma International",
				"Typing":         "Dynamic, weak",
				"License":        "ECMAScript specification",
			},
			Links: []store.Link{
				{Title: "MDN Docs", URL: "https://developer.mozilla.org/en-US/docs/Web/JavaScript"},
				{Title: "ECMAScript", URL: "https://www.ecma-international.org/publications-and-standards/standards/ecma-262/"},
			},
		},
		{
			Name:        "PostgreSQL",
			Type:        "software",
			Description: "PostgreSQL is a free and open-source relational database management system emphasizing extensibility and SQL compliance.",
			Image:       "https://www.postgresql.org/media/img/about/press/elephant.png",
			Facts: map[string]any{
				"Developer":       "PostgreSQL Global Development Group",
				"Initial release": "1996",
				"Written in":      "C",
				"License":         "PostgreSQL License",
				"Type":            "ORDBMS",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://postgresql.org"},
				{Title: "Documentation", URL: "https://www.postgresql.org/docs/"},
			},
		},
		{
			Name:        "Docker",
			Type:        "software",
			Description: "Docker is a set of platform as a service products that use OS-level virtualization to deliver software in packages called containers.",
			Image:       "https://www.docker.com/wp-content/uploads/2022/03/Moby-logo.png",
			Facts: map[string]any{
				"Developer":       "Docker, Inc.",
				"Initial release": "2013",
				"Written in":      "Go",
				"License":         "Apache License 2.0",
				"Type":            "Container platform",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://docker.com"},
				{Title: "Docker Hub", URL: "https://hub.docker.com"},
				{Title: "Documentation", URL: "https://docs.docker.com"},
			},
		},
		{
			Name:        "React",
			Type:        "software",
			Description: "React is a free and open-source front-end JavaScript library for building user interfaces based on components. It is maintained by Meta and a community of developers.",
			Image:       "https://reactjs.org/logo-og.png",
			Facts: map[string]any{
				"Developer":       "Meta (Facebook)",
				"Initial release": "2013",
				"Written in":      "JavaScript",
				"License":         "MIT License",
				"Type":            "JavaScript library",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://reactjs.org"},
				{Title: "Documentation", URL: "https://react.dev"},
				{Title: "GitHub", URL: "https://github.com/facebook/react"},
			},
		},
		{
			Name:        "Kubernetes",
			Type:        "software",
			Description: "Kubernetes is an open-source container orchestration system for automating software deployment, scaling, and management.",
			Image:       "https://kubernetes.io/images/kubernetes-horizontal-color.png",
			Facts: map[string]any{
				"Developer":       "Google, Cloud Native Computing Foundation",
				"Initial release": "2014",
				"Written in":      "Go",
				"License":         "Apache License 2.0",
				"Type":            "Container orchestration",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://kubernetes.io"},
				{Title: "Documentation", URL: "https://kubernetes.io/docs"},
				{Title: "GitHub", URL: "https://github.com/kubernetes/kubernetes"},
			},
		},
		{
			Name:        "TypeScript",
			Type:        "programming_language",
			Description: "TypeScript is a strongly typed programming language that builds on JavaScript, giving you better tooling at any scale.",
			Image:       "https://www.typescriptlang.org/assets/images/icons/apple-touch-icon-180x180.png",
			Facts: map[string]any{
				"Designed by":    "Anders Hejlsberg",
				"First appeared": "2012",
				"Developer":      "Microsoft",
				"Typing":         "Static, strong",
				"License":        "Apache License 2.0",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://typescriptlang.org"},
				{Title: "Documentation", URL: "https://www.typescriptlang.org/docs"},
				{Title: "GitHub", URL: "https://github.com/microsoft/TypeScript"},
			},
		},
		{
			Name:        "Rust",
			Type:        "programming_language",
			Description: "Rust is a multi-paradigm, general-purpose programming language that emphasizes performance, type safety, and concurrency.",
			Image:       "https://www.rust-lang.org/static/images/rust-logo-blk.svg",
			Facts: map[string]any{
				"Designed by":    "Graydon Hoare",
				"First appeared": "2010",
				"Developer":      "Rust Foundation",
				"Typing":         "Static, strong, inferred",
				"License":        "MIT or Apache 2.0",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://rust-lang.org"},
				{Title: "Documentation", URL: "https://doc.rust-lang.org"},
				{Title: "GitHub", URL: "https://github.com/rust-lang/rust"},
			},
		},
		{
			Name:        "Vue.js",
			Type:        "software",
			Description: "Vue.js is an open-source model-view-viewmodel front end JavaScript framework for building user interfaces and single-page applications.",
			Image:       "https://vuejs.org/images/logo.png",
			Facts: map[string]any{
				"Developer":       "Evan You",
				"Initial release": "2014",
				"Written in":      "JavaScript, TypeScript",
				"License":         "MIT License",
				"Type":            "JavaScript framework",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://vuejs.org"},
				{Title: "Documentation", URL: "https://vuejs.org/guide"},
				{Title: "GitHub", URL: "https://github.com/vuejs/vue"},
			},
		},
		{
			Name:        "MongoDB",
			Type:        "software",
			Description: "MongoDB is a source-available cross-platform document-oriented database program. Classified as a NoSQL database.",
			Image:       "https://webassets.mongodb.com/_com_assets/cms/MongoDB_Logo_FullColorBlack_RGB-4td3yuxzjs.png",
			Facts: map[string]any{
				"Developer":       "MongoDB Inc.",
				"Initial release": "2009",
				"Written in":      "C++, JavaScript, Python",
				"License":         "Server Side Public License",
				"Type":            "Document database",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://mongodb.com"},
				{Title: "Documentation", URL: "https://docs.mongodb.com"},
				{Title: "MongoDB Atlas", URL: "https://www.mongodb.com/atlas"},
			},
		},
		{
			Name:        "Redis",
			Type:        "software",
			Description: "Redis is an in-memory data structure store, used as a distributed, in-memory key-value database, cache and message broker.",
			Image:       "https://redis.io/images/redis-logo.svg",
			Facts: map[string]any{
				"Developer":       "Redis Ltd.",
				"Initial release": "2009",
				"Written in":      "C",
				"License":         "BSD 3-Clause",
				"Type":            "In-memory database",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://redis.io"},
				{Title: "Documentation", URL: "https://redis.io/docs"},
				{Title: "GitHub", URL: "https://github.com/redis/redis"},
			},
		},
		{
			Name:        "Node.js",
			Type:        "software",
			Description: "Node.js is a cross-platform, open-source JavaScript runtime environment that runs on the V8 JavaScript engine.",
			Image:       "https://nodejs.org/static/images/logo.svg",
			Facts: map[string]any{
				"Developer":       "OpenJS Foundation",
				"Initial release": "2009",
				"Written in":      "C, C++, JavaScript",
				"License":         "MIT License",
				"Type":            "JavaScript runtime",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://nodejs.org"},
				{Title: "Documentation", URL: "https://nodejs.org/docs"},
				{Title: "GitHub", URL: "https://github.com/nodejs/node"},
			},
		},
		{
			Name:        "GraphQL",
			Type:        "technology",
			Description: "GraphQL is a query language for APIs and a runtime for fulfilling those queries with your existing data.",
			Image:       "https://graphql.org/img/logo.svg",
			Facts: map[string]any{
				"Developer":       "Facebook (Meta)",
				"Initial release": "2015",
				"Type":            "Query language",
				"License":         "MIT License",
				"Specification":   "graphql.github.io/graphql-spec",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://graphql.org"},
				{Title: "Documentation", URL: "https://graphql.org/learn"},
				{Title: "GitHub", URL: "https://github.com/graphql"},
			},
		},
		{
			Name:        "GitHub",
			Type:        "service",
			Description: "GitHub is a developer platform that allows developers to create, store, manage and share their code using Git version control.",
			Image:       "https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png",
			Facts: map[string]any{
				"Owner":          "Microsoft",
				"Founded":        "2008",
				"Headquarters":   "San Francisco, California",
				"Users":          "100+ million",
				"Type":           "Code hosting platform",
			},
			Links: []store.Link{
				{Title: "Official Website", URL: "https://github.com"},
				{Title: "GitHub Docs", URL: "https://docs.github.com"},
				{Title: "GitHub Blog", URL: "https://github.blog"},
			},
		},
	}

	for _, entity := range entities {
		if err := s.knowledge.CreateEntity(ctx, &entity); err != nil {
			continue
		}
	}

	// Seed lenses
	lenses := []store.SearchLens{
		{
			Name:        "Forums",
			Description: "Search discussions and forums",
			Domains:     []string{"reddit.com", "stackoverflow.com", "news.ycombinator.com", "lobste.rs"},
			IsPublic:    true,
			IsBuiltIn:   true,
		},
		{
			Name:        "Academic",
			Description: "Search academic and research content",
			Domains:     []string{"arxiv.org", "scholar.google.com", "researchgate.net", "academia.edu"},
			IsPublic:    true,
			IsBuiltIn:   true,
		},
		{
			Name:        "Docs",
			Description: "Search documentation sites",
			Domains:     []string{"docs.github.com", "developer.mozilla.org", "docs.python.org", "pkg.go.dev"},
			IsPublic:    true,
			IsBuiltIn:   true,
		},
		{
			Name:        "Tech News",
			Description: "Search technology news sites",
			Domains:     []string{"techcrunch.com", "theverge.com", "arstechnica.com", "wired.com", "hackernews.com"},
			IsPublic:    true,
			IsBuiltIn:   true,
		},
		{
			Name:        "Videos",
			Description: "Search video platforms",
			Domains:     []string{"youtube.com", "vimeo.com", "twitch.tv"},
			IsPublic:    true,
			IsBuiltIn:   true,
		},
		{
			Name:        "Social",
			Description: "Search social media",
			Domains:     []string{"twitter.com", "linkedin.com", "facebook.com", "mastodon.social"},
			IsPublic:    true,
			IsBuiltIn:   true,
		},
		{
			Name:        "Shopping",
			Description: "Search shopping sites",
			Domains:     []string{"amazon.com", "ebay.com", "walmart.com", "target.com"},
			IsPublic:    true,
			IsBuiltIn:   true,
		},
		{
			Name:        "Recipes",
			Description: "Search recipe and cooking sites",
			Domains:     []string{"allrecipes.com", "foodnetwork.com", "epicurious.com", "bonappetit.com"},
			IsPublic:    true,
			IsBuiltIn:   true,
		},
	}

	for _, lens := range lenses {
		s.preference.CreateLens(ctx, &lens)
	}

	return nil
}
