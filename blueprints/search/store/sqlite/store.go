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

	return s, nil
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
	return createSchema(ctx, s.db)
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
	}

	for _, q := range suggestions {
		s.suggest.RecordQuery(ctx, q)
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
	}

	for _, lens := range lenses {
		s.preference.CreateLens(ctx, &lens)
	}

	return nil
}
