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
	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/blueprints/localflare/store/sqlite"
)

// Config holds server configuration.
type Config struct {
	Addr      string
	DNSPort   int
	HTTPPort  int
	HTTPSPort int
	DataDir   string
	Dev       bool
}

// Server is the HTTP server.
type Server struct {
	app   *mizu.App
	cfg   Config
	store *sqlite.Store

	// Handlers
	zonesHandler           *api.Zones
	dnsHandler             *api.DNS
	sslHandler             *api.SSL
	firewallHandler        *api.Firewall
	cacheHandler           *api.Cache
	workersHandler         *api.Workers
	kvHandler              *api.KV
	r2Handler              *api.R2
	d1Handler              *api.D1
	loadBalancerHandler    *api.LoadBalancer
	analyticsHandler       *api.Analytics
	rulesHandler           *api.Rules
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

	// Create handlers
	s.zonesHandler = api.NewZones(st.Zones())
	s.dnsHandler = api.NewDNS(st.DNS(), st.Zones())
	s.sslHandler = api.NewSSL(st.SSL(), st.Zones())
	s.firewallHandler = api.NewFirewall(st.Firewall())
	s.cacheHandler = api.NewCache(st.Cache())
	s.workersHandler = api.NewWorkers(st.Workers())
	s.kvHandler = api.NewKV(st.KV())
	s.r2Handler = api.NewR2(st.R2())
	s.d1Handler = api.NewD1(st.D1())
	s.loadBalancerHandler = api.NewLoadBalancer(st.LoadBalancer())
	s.analyticsHandler = api.NewAnalytics(st.Analytics())
	s.rulesHandler = api.NewRules(st.Rules())
	s.authHandler = api.NewAuth(st.Users())
	s.durableObjectsHandler = api.NewDurableObjects(st.DurableObjects())
	s.queuesHandler = api.NewQueues(st.Queues())
	s.vectorizeHandler = api.NewVectorize(st.Vectorize())
	s.analyticsEngineHandler = api.NewAnalyticsEngine(st.AnalyticsEngine())
	s.aiHandler = api.NewAI(st.AI())
	s.aiGatewayHandler = api.NewAIGateway(st.AIGateway())
	s.hyperdriveHandler = api.NewHyperdrive(st.Hyperdrive())
	s.cronHandler = api.NewCron(st.Cron())

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
	s.app.Get("/zones", s.serveUI)
	s.app.Get("/zones/{id}", s.serveUI)
	s.app.Get("/zones/{id}/dns", s.serveUI)
	s.app.Get("/zones/{id}/ssl", s.serveUI)
	s.app.Get("/zones/{id}/firewall", s.serveUI)
	s.app.Get("/zones/{id}/speed", s.serveUI)
	s.app.Get("/zones/{id}/caching", s.serveUI)
	s.app.Get("/zones/{id}/rules", s.serveUI)
	s.app.Get("/zones/{id}/analytics", s.serveUI)
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
	s.app.Group("/api", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/login", s.authHandler.Login)
		api.Post("/auth/register", s.authHandler.Register)
		api.Post("/auth/logout", s.authHandler.Logout)
		api.Get("/auth/me", s.authHandler.Me)

		// Zones
		api.Get("/zones", s.zonesHandler.List)
		api.Post("/zones", s.zonesHandler.Create)
		api.Get("/zones/{id}", s.zonesHandler.Get)
		api.Put("/zones/{id}", s.zonesHandler.Update)
		api.Delete("/zones/{id}", s.zonesHandler.Delete)

		// DNS Records
		api.Get("/zones/{zoneID}/dns/records", s.dnsHandler.List)
		api.Post("/zones/{zoneID}/dns/records", s.dnsHandler.Create)
		api.Get("/zones/{zoneID}/dns/records/{id}", s.dnsHandler.Get)
		api.Put("/zones/{zoneID}/dns/records/{id}", s.dnsHandler.Update)
		api.Delete("/zones/{zoneID}/dns/records/{id}", s.dnsHandler.Delete)
		api.Post("/zones/{zoneID}/dns/import", s.dnsHandler.Import)
		api.Get("/zones/{zoneID}/dns/export", s.dnsHandler.Export)

		// SSL/TLS
		api.Get("/zones/{zoneID}/ssl/settings", s.sslHandler.GetSettings)
		api.Put("/zones/{zoneID}/ssl/settings", s.sslHandler.UpdateSettings)
		api.Get("/zones/{zoneID}/ssl/certificates", s.sslHandler.ListCertificates)
		api.Post("/zones/{zoneID}/ssl/certificates", s.sslHandler.CreateCertificate)
		api.Delete("/zones/{zoneID}/ssl/certificates/{id}", s.sslHandler.DeleteCertificate)
		api.Post("/zones/{zoneID}/ssl/origin-ca", s.sslHandler.CreateOriginCA)

		// Firewall
		api.Get("/zones/{zoneID}/firewall/rules", s.firewallHandler.ListRules)
		api.Post("/zones/{zoneID}/firewall/rules", s.firewallHandler.CreateRule)
		api.Put("/zones/{zoneID}/firewall/rules/{id}", s.firewallHandler.UpdateRule)
		api.Delete("/zones/{zoneID}/firewall/rules/{id}", s.firewallHandler.DeleteRule)
		api.Get("/zones/{zoneID}/firewall/ip-access", s.firewallHandler.ListIPAccessRules)
		api.Post("/zones/{zoneID}/firewall/ip-access", s.firewallHandler.CreateIPAccessRule)
		api.Delete("/zones/{zoneID}/firewall/ip-access/{id}", s.firewallHandler.DeleteIPAccessRule)
		api.Get("/zones/{zoneID}/firewall/rate-limits", s.firewallHandler.ListRateLimits)
		api.Post("/zones/{zoneID}/firewall/rate-limits", s.firewallHandler.CreateRateLimit)

		// Cache
		api.Get("/zones/{zoneID}/cache/settings", s.cacheHandler.GetSettings)
		api.Put("/zones/{zoneID}/cache/settings", s.cacheHandler.UpdateSettings)
		api.Get("/zones/{zoneID}/cache/rules", s.cacheHandler.ListRules)
		api.Post("/zones/{zoneID}/cache/rules", s.cacheHandler.CreateRule)
		api.Delete("/zones/{zoneID}/cache/rules/{id}", s.cacheHandler.DeleteRule)
		api.Post("/zones/{zoneID}/cache/purge", s.cacheHandler.Purge)

		// Workers
		api.Get("/workers", s.workersHandler.List)
		api.Post("/workers", s.workersHandler.Create)
		api.Get("/workers/{id}", s.workersHandler.Get)
		api.Put("/workers/{id}", s.workersHandler.Update)
		api.Delete("/workers/{id}", s.workersHandler.Delete)
		api.Get("/workers/{id}/logs", s.workersHandler.Logs)
		api.Post("/workers/{id}/deploy", s.workersHandler.Deploy)
		api.Get("/zones/{zoneID}/workers/routes", s.workersHandler.ListRoutes)
		api.Post("/zones/{zoneID}/workers/routes", s.workersHandler.CreateRoute)
		api.Delete("/zones/{zoneID}/workers/routes/{id}", s.workersHandler.DeleteRoute)

		// KV
		api.Get("/kv/namespaces", s.kvHandler.ListNamespaces)
		api.Post("/kv/namespaces", s.kvHandler.CreateNamespace)
		api.Delete("/kv/namespaces/{id}", s.kvHandler.DeleteNamespace)
		api.Get("/kv/namespaces/{id}/keys", s.kvHandler.ListKeys)
		api.Get("/kv/namespaces/{id}/values/{key}", s.kvHandler.GetValue)
		api.Put("/kv/namespaces/{id}/values/{key}", s.kvHandler.PutValue)
		api.Delete("/kv/namespaces/{id}/values/{key}", s.kvHandler.DeleteValue)

		// R2
		api.Get("/r2/buckets", s.r2Handler.ListBuckets)
		api.Post("/r2/buckets", s.r2Handler.CreateBucket)
		api.Get("/r2/buckets/{id}", s.r2Handler.GetBucket)
		api.Delete("/r2/buckets/{id}", s.r2Handler.DeleteBucket)
		api.Get("/r2/buckets/{id}/objects", s.r2Handler.ListObjects)
		api.Put("/r2/buckets/{id}/objects/{key...}", s.r2Handler.PutObject)
		api.Get("/r2/buckets/{id}/objects/{key...}", s.r2Handler.GetObject)
		api.Delete("/r2/buckets/{id}/objects/{key...}", s.r2Handler.DeleteObject)

		// D1
		api.Get("/d1/databases", s.d1Handler.ListDatabases)
		api.Post("/d1/databases", s.d1Handler.CreateDatabase)
		api.Get("/d1/databases/{id}", s.d1Handler.GetDatabase)
		api.Delete("/d1/databases/{id}", s.d1Handler.DeleteDatabase)
		api.Post("/d1/databases/{id}/query", s.d1Handler.Query)

		// Load Balancer
		api.Get("/zones/{zoneID}/loadbalancers", s.loadBalancerHandler.List)
		api.Post("/zones/{zoneID}/loadbalancers", s.loadBalancerHandler.Create)
		api.Get("/zones/{zoneID}/loadbalancers/{id}", s.loadBalancerHandler.Get)
		api.Put("/zones/{zoneID}/loadbalancers/{id}", s.loadBalancerHandler.Update)
		api.Delete("/zones/{zoneID}/loadbalancers/{id}", s.loadBalancerHandler.Delete)
		api.Get("/origin-pools", s.loadBalancerHandler.ListPools)
		api.Post("/origin-pools", s.loadBalancerHandler.CreatePool)
		api.Get("/origin-pools/{id}", s.loadBalancerHandler.GetPool)
		api.Put("/origin-pools/{id}", s.loadBalancerHandler.UpdatePool)
		api.Delete("/origin-pools/{id}", s.loadBalancerHandler.DeletePool)
		api.Get("/health-checks", s.loadBalancerHandler.ListHealthChecks)
		api.Post("/health-checks", s.loadBalancerHandler.CreateHealthCheck)

		// Analytics
		api.Get("/zones/{zoneID}/analytics/traffic", s.analyticsHandler.Traffic)
		api.Get("/zones/{zoneID}/analytics/security", s.analyticsHandler.Security)
		api.Get("/zones/{zoneID}/analytics/cache", s.analyticsHandler.Cache)

		// Rules
		api.Get("/zones/{zoneID}/rules/page", s.rulesHandler.ListPageRules)
		api.Post("/zones/{zoneID}/rules/page", s.rulesHandler.CreatePageRule)
		api.Delete("/zones/{zoneID}/rules/page/{id}", s.rulesHandler.DeletePageRule)
		api.Get("/zones/{zoneID}/rules/transform", s.rulesHandler.ListTransformRules)
		api.Post("/zones/{zoneID}/rules/transform", s.rulesHandler.CreateTransformRule)
		api.Delete("/zones/{zoneID}/rules/transform/{id}", s.rulesHandler.DeleteTransformRule)

		// Durable Objects
		api.Get("/durable-objects/namespaces", s.durableObjectsHandler.ListNamespaces)
		api.Post("/durable-objects/namespaces", s.durableObjectsHandler.CreateNamespace)
		api.Get("/durable-objects/namespaces/{id}", s.durableObjectsHandler.GetNamespace)
		api.Delete("/durable-objects/namespaces/{id}", s.durableObjectsHandler.DeleteNamespace)
		api.Get("/durable-objects/namespaces/{id}/objects", s.durableObjectsHandler.ListObjects)

		// Queues
		api.Get("/queues", s.queuesHandler.List)
		api.Post("/queues", s.queuesHandler.Create)
		api.Get("/queues/{id}", s.queuesHandler.Get)
		api.Delete("/queues/{id}", s.queuesHandler.Delete)
		api.Post("/queues/{id}/messages", s.queuesHandler.SendMessage)
		api.Post("/queues/{id}/messages/pull", s.queuesHandler.PullMessages)
		api.Post("/queues/{id}/messages/ack", s.queuesHandler.AckMessages)
		api.Get("/queues/{id}/stats", s.queuesHandler.GetStats)

		// Vectorize
		api.Get("/vectorize/indexes", s.vectorizeHandler.ListIndexes)
		api.Post("/vectorize/indexes", s.vectorizeHandler.CreateIndex)
		api.Get("/vectorize/indexes/{name}", s.vectorizeHandler.GetIndex)
		api.Delete("/vectorize/indexes/{name}", s.vectorizeHandler.DeleteIndex)
		api.Post("/vectorize/indexes/{name}/insert", s.vectorizeHandler.InsertVectors)
		api.Post("/vectorize/indexes/{name}/upsert", s.vectorizeHandler.UpsertVectors)
		api.Post("/vectorize/indexes/{name}/query", s.vectorizeHandler.Query)
		api.Post("/vectorize/indexes/{name}/delete-by-ids", s.vectorizeHandler.DeleteVectors)
		api.Post("/vectorize/indexes/{name}/get-by-ids", s.vectorizeHandler.GetByIDs)

		// Analytics Engine
		api.Get("/analytics-engine/datasets", s.analyticsEngineHandler.ListDatasets)
		api.Post("/analytics-engine/datasets", s.analyticsEngineHandler.CreateDataset)
		api.Get("/analytics-engine/datasets/{name}", s.analyticsEngineHandler.GetDataset)
		api.Delete("/analytics-engine/datasets/{name}", s.analyticsEngineHandler.DeleteDataset)
		api.Post("/analytics-engine/datasets/{name}/write", s.analyticsEngineHandler.WriteDataPoints)
		api.Post("/analytics-engine/sql", s.analyticsEngineHandler.Query)

		// Workers AI
		api.Get("/ai/models", s.aiHandler.ListModels)
		api.Get("/ai/models/{model}", s.aiHandler.GetModel)
		api.Post("/ai/run/{model}", s.aiHandler.RunModel)

		// AI Gateway
		api.Get("/ai-gateway", s.aiGatewayHandler.ListGateways)
		api.Post("/ai-gateway", s.aiGatewayHandler.CreateGateway)
		api.Get("/ai-gateway/{id}", s.aiGatewayHandler.GetGateway)
		api.Put("/ai-gateway/{id}", s.aiGatewayHandler.UpdateGateway)
		api.Delete("/ai-gateway/{id}", s.aiGatewayHandler.DeleteGateway)
		api.Get("/ai-gateway/{id}/logs", s.aiGatewayHandler.GetLogs)

		// Hyperdrive
		api.Get("/hyperdrive/configs", s.hyperdriveHandler.ListConfigs)
		api.Post("/hyperdrive/configs", s.hyperdriveHandler.CreateConfig)
		api.Get("/hyperdrive/configs/{id}", s.hyperdriveHandler.GetConfig)
		api.Put("/hyperdrive/configs/{id}", s.hyperdriveHandler.UpdateConfig)
		api.Delete("/hyperdrive/configs/{id}", s.hyperdriveHandler.DeleteConfig)
		api.Get("/hyperdrive/configs/{id}/stats", s.hyperdriveHandler.GetStats)

		// Cron Triggers
		api.Get("/cron/triggers", s.cronHandler.ListTriggers)
		api.Post("/cron/triggers", s.cronHandler.CreateTrigger)
		api.Get("/cron/triggers/{id}", s.cronHandler.GetTrigger)
		api.Put("/cron/triggers/{id}", s.cronHandler.UpdateTrigger)
		api.Delete("/cron/triggers/{id}", s.cronHandler.DeleteTrigger)
		api.Get("/cron/triggers/{id}/executions", s.cronHandler.GetExecutions)
		api.Get("/cron/scripts/{script}/triggers", s.cronHandler.ListTriggersByScript)
	})
}

func (s *Server) serveUI(c *mizu.Ctx) error {
	html := assets.IndexHTML()
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer().Write(html)
	return nil
}

// SeedData seeds sample data for development.
func (s *Server) SeedData(ctx context.Context) error {
	now := time.Now()

	// Create sample zones
	zones := []*store.Zone{
		{
			ID:          generateID(),
			Name:        "example.com",
			Status:      "active",
			Plan:        "free",
			NameServers: []string{"ns1.localflare.local", "ns2.localflare.local"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          generateID(),
			Name:        "test.local",
			Status:      "active",
			Plan:        "pro",
			NameServers: []string{"ns1.localflare.local", "ns2.localflare.local"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, zone := range zones {
		if err := s.store.Zones().Create(ctx, zone); err != nil {
			slog.Warn("failed to create zone", "name", zone.Name, "error", err)
			continue
		}

		// Create DNS records for each zone
		dnsRecords := []*store.DNSRecord{
			{ID: generateID(), ZoneID: zone.ID, Type: "A", Name: "@", Content: "192.168.1.1", TTL: 300, Proxied: true, CreatedAt: now, UpdatedAt: now},
			{ID: generateID(), ZoneID: zone.ID, Type: "A", Name: "www", Content: "192.168.1.1", TTL: 300, Proxied: true, CreatedAt: now, UpdatedAt: now},
			{ID: generateID(), ZoneID: zone.ID, Type: "AAAA", Name: "@", Content: "2001:db8::1", TTL: 300, Proxied: true, CreatedAt: now, UpdatedAt: now},
			{ID: generateID(), ZoneID: zone.ID, Type: "CNAME", Name: "mail", Content: "mail.example.com", TTL: 300, Proxied: false, CreatedAt: now, UpdatedAt: now},
			{ID: generateID(), ZoneID: zone.ID, Type: "MX", Name: "@", Content: "mail.example.com", TTL: 300, Priority: 10, Proxied: false, CreatedAt: now, UpdatedAt: now},
			{ID: generateID(), ZoneID: zone.ID, Type: "TXT", Name: "@", Content: "v=spf1 include:_spf.example.com ~all", TTL: 300, Proxied: false, CreatedAt: now, UpdatedAt: now},
		}
		for _, record := range dnsRecords {
			s.store.DNS().Create(ctx, record)
		}

		// Create SSL settings
		s.store.SSL().UpdateSettings(ctx, &store.SSLSettings{
			ZoneID:                 zone.ID,
			Mode:                   "full",
			AlwaysHTTPS:            true,
			MinTLSVersion:          "1.2",
			TLS13:                  true,
			AutomaticHTTPSRewrites: true,
		})

		// Create firewall rules
		firewallRules := []*store.FirewallRule{
			{
				ID:          generateID(),
				ZoneID:      zone.ID,
				Description: "Block known bad IPs",
				Expression:  `ip.src in {192.0.2.0/24}`,
				Action:      "block",
				Priority:    1,
				Enabled:     true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          generateID(),
				ZoneID:      zone.ID,
				Description: "Challenge suspicious requests",
				Expression:  `http.request.uri.path contains "/admin"`,
				Action:      "challenge",
				Priority:    2,
				Enabled:     true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}
		for _, rule := range firewallRules {
			s.store.Firewall().CreateRule(ctx, rule)
		}

		// Create cache settings
		s.store.Cache().UpdateSettings(ctx, &store.CacheSettings{
			ZoneID:          zone.ID,
			CacheLevel:      "standard",
			BrowserTTL:      14400,
			EdgeTTL:         7200,
			DevelopmentMode: false,
			AlwaysOnline:    true,
		})
	}

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
	s.store.R2().PutObject(ctx, r2Bucket.ID, "welcome.txt", []byte("Welcome to Localflare R2 Storage!"), map[string]string{"content-type": "text/plain"})

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

	// Create sample analytics data
	for i := 0; i < 24; i++ {
		ts := now.Add(-time.Duration(i) * time.Hour)
		data := &store.AnalyticsData{
			Timestamp:    ts,
			Requests:     int64(1000 + i*100),
			Bandwidth:    int64(1024*1024*(10+i)),
			Threats:      int64(i % 5),
			PageViews:    int64(500 + i*50),
			UniqueVisits: int64(200 + i*20),
			CacheHits:    int64(800 + i*80),
			CacheMisses:  int64(200 + i*20),
			StatusCodes:  map[int]int64{200: int64(900 + i*90), 404: int64(50 + i), 500: int64(10)},
		}
		s.store.Analytics().Record(ctx, zones[0].ID, data)
	}

	slog.Info("Sample data seeded successfully")
	return nil
}

func generateID() string {
	return ulid.Make().String()
}

func isDevMode() bool {
	return os.Getenv("DEV") == "1" || os.Getenv("DEV") == "true"
}
