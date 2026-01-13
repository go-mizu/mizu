package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/app/web/handler/api"
	"github.com/go-mizu/blueprints/localflare/assets"
	"github.com/go-mizu/blueprints/localflare/feature/ai"
	"github.com/go-mizu/blueprints/localflare/feature/ai_gateway"
	"github.com/go-mizu/blueprints/localflare/feature/analytics_engine"
	"github.com/go-mizu/blueprints/localflare/feature/auth"
	"github.com/go-mizu/blueprints/localflare/feature/cron"
	"github.com/go-mizu/blueprints/localflare/feature/d1"
	do "github.com/go-mizu/blueprints/localflare/feature/durable_objects"
	"github.com/go-mizu/blueprints/localflare/feature/hyperdrive"
	"github.com/go-mizu/blueprints/localflare/feature/kv"
	"github.com/go-mizu/blueprints/localflare/feature/queues"
	"github.com/go-mizu/blueprints/localflare/feature/r2"
	"github.com/go-mizu/blueprints/localflare/feature/vectorize"
	"github.com/go-mizu/blueprints/localflare/feature/workers"
	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/blueprints/localflare/store/sqlite"
)

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	Dev     bool
}

// Server is the HTTP server.
type Server struct {
	app   *mizu.App
	cfg   Config
	store *sqlite.Store

	// Handlers
	workersHandler         *api.Workers
	kvHandler              *api.KV
	r2Handler              *api.R2
	d1Handler              *api.D1
	authHandler            *api.Auth
	durableObjectsHandler  *api.DurableObjects
	queuesHandler          *api.Queues
	vectorizeHandler       *api.Vectorize
	analyticsEngineHandler *api.AnalyticsEngine
	aiHandler              *api.AI
	aiGatewayHandler       *api.AIGateway
	hyperdriveHandler      *api.Hyperdrive
	cronHandler            *api.Cron
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Create store
	st, err := sqlite.New(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	// Initialize schema
	if err := st.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	s := &Server{
		app:   mizu.New(),
		cfg:   cfg,
		store: st,
	}

	// Create services with store adapters
	workersSvc := workers.NewService(&WorkersStoreAdapter{st: st.Workers()})
	kvSvc := kv.NewService(&KVStoreAdapter{st: st.KV()})
	r2Svc := r2.NewService(&R2StoreAdapter{st: st.R2()})
	d1Svc := d1.NewService(&D1StoreAdapter{st: st.D1()})
	doSvc := do.NewService(&DurableObjectsStoreAdapter{st: st.DurableObjects()})
	queuesSvc := queues.NewService(&QueuesStoreAdapter{st: st.Queues()})
	vectorizeSvc := vectorize.NewService(&VectorizeStoreAdapter{st: st.Vectorize()})
	aeSvc := analytics_engine.NewService(&AnalyticsEngineStoreAdapter{st: st.AnalyticsEngine()})
	aiSvc := ai.NewService(&AIStoreAdapter{st: st.AI()})
	aiGwSvc := ai_gateway.NewService(&AIGatewayStoreAdapter{st: st.AIGateway()})
	hdSvc := hyperdrive.NewService(&HyperdriveStoreAdapter{st: st.Hyperdrive()})
	cronSvc := cron.NewService(&CronStoreAdapter{st: st.Cron()})
	authSvc := auth.NewService(&AuthStoreAdapter{st: st.Users()})

	// Create handlers with services
	s.workersHandler = api.NewWorkers(workersSvc)
	s.kvHandler = api.NewKV(kvSvc)
	s.r2Handler = api.NewR2(r2Svc)
	s.d1Handler = api.NewD1(d1Svc)
	s.durableObjectsHandler = api.NewDurableObjects(doSvc)
	s.queuesHandler = api.NewQueues(queuesSvc)
	s.vectorizeHandler = api.NewVectorize(vectorizeSvc)
	s.analyticsEngineHandler = api.NewAnalyticsEngine(aeSvc)
	s.aiHandler = api.NewAI(aiSvc)
	s.aiGatewayHandler = api.NewAIGateway(aiGwSvc)
	s.hyperdriveHandler = api.NewHyperdrive(hdSvc)
	s.cronHandler = api.NewCron(cronSvc)
	s.authHandler = api.NewAuth(authSvc)

	s.setupRoutes()

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	slog.Info("Starting Localflare server", "addr", s.cfg.Addr)
	return s.app.Listen(s.cfg.Addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler { return s.app }

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Serve static files (frontend)
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(assets.Static())))
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
		c.Writer().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		staticHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	// Frontend routes - serve index.html for SPA
	s.app.Get("/", s.serveUI)
	s.app.Get("/workers", s.serveUI)
	s.app.Get("/workers/{id}", s.serveUI)
	s.app.Get("/r2", s.serveUI)
	s.app.Get("/r2/{id}", s.serveUI)
	s.app.Get("/kv", s.serveUI)
	s.app.Get("/kv/{id}", s.serveUI)
	s.app.Get("/d1", s.serveUI)
	s.app.Get("/d1/{id}", s.serveUI)
	s.app.Get("/durable-objects", s.serveUI)
	s.app.Get("/durable-objects/{id}", s.serveUI)
	s.app.Get("/queues", s.serveUI)
	s.app.Get("/queues/{id}", s.serveUI)
	s.app.Get("/vectorize", s.serveUI)
	s.app.Get("/vectorize/{name}", s.serveUI)
	s.app.Get("/analytics-engine", s.serveUI)
	s.app.Get("/analytics-engine/{name}", s.serveUI)
	s.app.Get("/ai", s.serveUI)
	s.app.Get("/ai-gateway", s.serveUI)
	s.app.Get("/ai-gateway/{id}", s.serveUI)
	s.app.Get("/hyperdrive", s.serveUI)
	s.app.Get("/hyperdrive/{id}", s.serveUI)
	s.app.Get("/cron", s.serveUI)
	s.app.Get("/analytics", s.serveUI)
	s.app.Get("/settings", s.serveUI)
	s.app.Get("/login", s.serveUI)

	// API routes
	s.app.Group("/api", func(apiGroup *mizu.Router) {
		// Auth
		apiGroup.Post("/auth/login", s.authHandler.Login)
		apiGroup.Post("/auth/register", s.authHandler.Register)
		apiGroup.Post("/auth/logout", s.authHandler.Logout)
		apiGroup.Get("/auth/me", s.authHandler.Me)

		// Workers
		apiGroup.Get("/workers", s.workersHandler.List)
		apiGroup.Post("/workers", s.workersHandler.Create)
		apiGroup.Get("/workers/{id}", s.workersHandler.Get)
		apiGroup.Put("/workers/{id}", s.workersHandler.Update)
		apiGroup.Delete("/workers/{id}", s.workersHandler.Delete)
		apiGroup.Get("/workers/{id}/logs", s.workersHandler.Logs)
		apiGroup.Post("/workers/{id}/deploy", s.workersHandler.Deploy)
		apiGroup.Get("/workers/routes", s.workersHandler.ListRoutes)
		apiGroup.Post("/workers/routes", s.workersHandler.CreateRoute)
		apiGroup.Delete("/workers/routes/{id}", s.workersHandler.DeleteRoute)

		// KV
		apiGroup.Get("/kv/namespaces", s.kvHandler.ListNamespaces)
		apiGroup.Post("/kv/namespaces", s.kvHandler.CreateNamespace)
		apiGroup.Delete("/kv/namespaces/{id}", s.kvHandler.DeleteNamespace)
		apiGroup.Get("/kv/namespaces/{id}/keys", s.kvHandler.ListKeys)
		apiGroup.Get("/kv/namespaces/{id}/values/{key}", s.kvHandler.GetValue)
		apiGroup.Put("/kv/namespaces/{id}/values/{key}", s.kvHandler.PutValue)
		apiGroup.Delete("/kv/namespaces/{id}/values/{key}", s.kvHandler.DeleteValue)

		// R2
		apiGroup.Get("/r2/buckets", s.r2Handler.ListBuckets)
		apiGroup.Post("/r2/buckets", s.r2Handler.CreateBucket)
		apiGroup.Get("/r2/buckets/{id}", s.r2Handler.GetBucket)
		apiGroup.Delete("/r2/buckets/{id}", s.r2Handler.DeleteBucket)
		apiGroup.Get("/r2/buckets/{id}/objects", s.r2Handler.ListObjects)
		apiGroup.Put("/r2/buckets/{id}/objects/{key...}", s.r2Handler.PutObject)
		apiGroup.Get("/r2/buckets/{id}/objects/{key...}", s.r2Handler.GetObject)
		apiGroup.Delete("/r2/buckets/{id}/objects/{key...}", s.r2Handler.DeleteObject)

		// D1
		apiGroup.Get("/d1/databases", s.d1Handler.ListDatabases)
		apiGroup.Post("/d1/databases", s.d1Handler.CreateDatabase)
		apiGroup.Get("/d1/databases/{id}", s.d1Handler.GetDatabase)
		apiGroup.Delete("/d1/databases/{id}", s.d1Handler.DeleteDatabase)
		apiGroup.Post("/d1/databases/{id}/query", s.d1Handler.Query)

		// Durable Objects
		apiGroup.Get("/durable-objects/namespaces", s.durableObjectsHandler.ListNamespaces)
		apiGroup.Post("/durable-objects/namespaces", s.durableObjectsHandler.CreateNamespace)
		apiGroup.Get("/durable-objects/namespaces/{id}", s.durableObjectsHandler.GetNamespace)
		apiGroup.Delete("/durable-objects/namespaces/{id}", s.durableObjectsHandler.DeleteNamespace)
		apiGroup.Get("/durable-objects/namespaces/{id}/objects", s.durableObjectsHandler.ListObjects)

		// Queues
		apiGroup.Get("/queues", s.queuesHandler.List)
		apiGroup.Post("/queues", s.queuesHandler.Create)
		apiGroup.Get("/queues/{id}", s.queuesHandler.Get)
		apiGroup.Delete("/queues/{id}", s.queuesHandler.Delete)
		apiGroup.Post("/queues/{id}/messages", s.queuesHandler.SendMessage)
		apiGroup.Post("/queues/{id}/messages/pull", s.queuesHandler.PullMessages)
		apiGroup.Post("/queues/{id}/messages/ack", s.queuesHandler.AckMessages)
		apiGroup.Get("/queues/{id}/stats", s.queuesHandler.GetStats)

		// Vectorize
		apiGroup.Get("/vectorize/indexes", s.vectorizeHandler.ListIndexes)
		apiGroup.Post("/vectorize/indexes", s.vectorizeHandler.CreateIndex)
		apiGroup.Get("/vectorize/indexes/{name}", s.vectorizeHandler.GetIndex)
		apiGroup.Delete("/vectorize/indexes/{name}", s.vectorizeHandler.DeleteIndex)
		apiGroup.Post("/vectorize/indexes/{name}/insert", s.vectorizeHandler.InsertVectors)
		apiGroup.Post("/vectorize/indexes/{name}/upsert", s.vectorizeHandler.UpsertVectors)
		apiGroup.Post("/vectorize/indexes/{name}/query", s.vectorizeHandler.Query)
		apiGroup.Post("/vectorize/indexes/{name}/delete-by-ids", s.vectorizeHandler.DeleteVectors)
		apiGroup.Post("/vectorize/indexes/{name}/get-by-ids", s.vectorizeHandler.GetByIDs)

		// Analytics Engine
		apiGroup.Get("/analytics-engine/datasets", s.analyticsEngineHandler.ListDatasets)
		apiGroup.Post("/analytics-engine/datasets", s.analyticsEngineHandler.CreateDataset)
		apiGroup.Get("/analytics-engine/datasets/{name}", s.analyticsEngineHandler.GetDataset)
		apiGroup.Delete("/analytics-engine/datasets/{name}", s.analyticsEngineHandler.DeleteDataset)
		apiGroup.Post("/analytics-engine/datasets/{name}/write", s.analyticsEngineHandler.WriteDataPoints)
		apiGroup.Post("/analytics-engine/sql", s.analyticsEngineHandler.Query)

		// Workers AI
		apiGroup.Get("/ai/models", s.aiHandler.ListModels)
		apiGroup.Get("/ai/models/{model}", s.aiHandler.GetModel)
		apiGroup.Post("/ai/run/{model}", s.aiHandler.RunModel)

		// AI Gateway
		apiGroup.Get("/ai-gateway", s.aiGatewayHandler.ListGateways)
		apiGroup.Post("/ai-gateway", s.aiGatewayHandler.CreateGateway)
		apiGroup.Get("/ai-gateway/{id}", s.aiGatewayHandler.GetGateway)
		apiGroup.Put("/ai-gateway/{id}", s.aiGatewayHandler.UpdateGateway)
		apiGroup.Delete("/ai-gateway/{id}", s.aiGatewayHandler.DeleteGateway)
		apiGroup.Get("/ai-gateway/{id}/logs", s.aiGatewayHandler.GetLogs)

		// Hyperdrive
		apiGroup.Get("/hyperdrive/configs", s.hyperdriveHandler.ListConfigs)
		apiGroup.Post("/hyperdrive/configs", s.hyperdriveHandler.CreateConfig)
		apiGroup.Get("/hyperdrive/configs/{id}", s.hyperdriveHandler.GetConfig)
		apiGroup.Put("/hyperdrive/configs/{id}", s.hyperdriveHandler.UpdateConfig)
		apiGroup.Delete("/hyperdrive/configs/{id}", s.hyperdriveHandler.DeleteConfig)
		apiGroup.Get("/hyperdrive/configs/{id}/stats", s.hyperdriveHandler.GetStats)

		// Cron Triggers
		apiGroup.Get("/cron/triggers", s.cronHandler.ListTriggers)
		apiGroup.Post("/cron/triggers", s.cronHandler.CreateTrigger)
		apiGroup.Get("/cron/triggers/{id}", s.cronHandler.GetTrigger)
		apiGroup.Put("/cron/triggers/{id}", s.cronHandler.UpdateTrigger)
		apiGroup.Delete("/cron/triggers/{id}", s.cronHandler.DeleteTrigger)
		apiGroup.Get("/cron/triggers/{id}/executions", s.cronHandler.GetExecutions)
		apiGroup.Get("/cron/scripts/{script}/triggers", s.cronHandler.ListTriggersByScript)
	})
}

func (s *Server) serveUI(c *mizu.Ctx) error {
	html := assets.IndexHTML()
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer().Write(html)
	return nil
}

// Store returns the underlying store for seeding purposes.
func (s *Server) Store() store.Store {
	return s.store
}

// SeedData seeds sample data for development.
func (s *Server) SeedData(ctx context.Context) error {
	now := time.Now()

	// Create sample worker
	worker := &store.Worker{
		ID:   generateID(),
		Name: "hello-world",
		Script: `addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request))
})

async function handleRequest(request) {
  return new Response('Hello from Localflare Worker!', {
    headers: { 'content-type': 'text/plain' },
  })
}`,
		Routes:    []string{"example.com/*"},
		Bindings:  map[string]string{},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.store.Workers().Create(ctx, worker)

	// Create KV namespace
	kvNS := &store.KVNamespace{
		ID:        generateID(),
		Title:     "MY_KV_NAMESPACE",
		CreatedAt: now,
	}
	s.store.KV().CreateNamespace(ctx, kvNS)

	// Add sample KV pairs
	kvPairs := []*store.KVPair{
		{Key: "greeting", Value: []byte("Hello, World!")},
		{Key: "config:theme", Value: []byte("dark")},
		{Key: "user:1", Value: []byte(`{"name":"John","email":"john@example.com"}`)},
	}
	for _, pair := range kvPairs {
		s.store.KV().Put(ctx, kvNS.ID, pair)
	}

	// Create R2 bucket
	r2Bucket := &store.R2Bucket{
		ID:        generateID(),
		Name:      "my-bucket",
		Location:  "auto",
		CreatedAt: now,
	}
	s.store.R2().CreateBucket(ctx, r2Bucket)

	// Add sample object
	s.store.R2().PutObject(ctx, r2Bucket.ID, "welcome.txt", []byte("Welcome to Localflare R2 Storage!"), &store.R2PutOptions{
		HTTPMetadata: &store.R2HTTPMetadata{ContentType: "text/plain"},
	})

	// Create D1 database
	d1DB := &store.D1Database{
		ID:        generateID(),
		Name:      "my-database",
		Version:   "1",
		NumTables: 0,
		FileSize:  0,
		CreatedAt: now,
	}
	s.store.D1().CreateDatabase(ctx, d1DB)

	// Create sample table in D1
	s.store.D1().Exec(ctx, d1DB.ID, `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`, nil)
	s.store.D1().Exec(ctx, d1DB.ID, `INSERT INTO users (name, email) VALUES (?, ?)`, []interface{}{"John Doe", "john@example.com"})

	// Create Durable Objects namespace
	doNS := &store.DurableObjectNamespace{
		ID:        generateID(),
		Name:      "COUNTER",
		ClassName: "Counter",
		Script:    "counter-worker",
		CreatedAt: now,
	}
	s.store.DurableObjects().CreateNamespace(ctx, doNS)

	// Create Queue
	queue := &store.Queue{
		ID:   generateID(),
		Name: "my-queue",
		Settings: store.QueueSettings{
			DeliveryDelay: 0,
			MessageTTL:    3600,
			MaxRetries:    3,
		},
		CreatedAt: now,
	}
	s.store.Queues().CreateQueue(ctx, queue)

	slog.Info("Sample data seeded successfully")
	return nil
}

func generateID() string {
	return ulid.Make().String()
}
