// Package seed provides comprehensive data seeding for Localflare.
package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Seeder orchestrates data seeding for all Localflare products.
type Seeder struct {
	store store.Store
	now   time.Time
	ids   *IDRegistry
}

// IDRegistry tracks generated IDs for cross-references between entities.
type IDRegistry struct {
	Zones         map[string]string // name -> id
	Workers       map[string]string // name -> id
	KVNamespaces  map[string]string // title -> id
	R2Buckets     map[string]string // name -> id
	D1Databases   map[string]string // name -> id
	OriginPools   map[string]string // name -> id
	HealthChecks  map[string]string // description -> id
	Queues        map[string]string // name -> id
	VectorIndexes map[string]string // name -> id
	AIGateways    map[string]string // name -> id
	DONamespaces  map[string]string // name -> id
	CronTriggers  map[string]string // script:cron -> id
	Datasets      map[string]string // name -> id
	Hyperdrive    map[string]string // name -> id
}

// New creates a new Seeder instance.
func New(s store.Store) *Seeder {
	return &Seeder{
		store: s,
		now:   time.Now(),
		ids: &IDRegistry{
			Zones:         make(map[string]string),
			Workers:       make(map[string]string),
			KVNamespaces:  make(map[string]string),
			R2Buckets:     make(map[string]string),
			D1Databases:   make(map[string]string),
			OriginPools:   make(map[string]string),
			HealthChecks:  make(map[string]string),
			Queues:        make(map[string]string),
			VectorIndexes: make(map[string]string),
			AIGateways:    make(map[string]string),
			DONamespaces:  make(map[string]string),
			CronTriggers:  make(map[string]string),
			Datasets:      make(map[string]string),
			Hyperdrive:    make(map[string]string),
		},
	}
}

// Stats tracks seeding statistics.
type Stats struct {
	Created int
	Skipped int
	Errors  int
}

// Run executes the complete seeding process.
func (s *Seeder) Run(ctx context.Context) error {
	slog.Info("starting seed process")

	// Phase 1: Core infrastructure (zones, DNS)
	if err := s.seedZones(ctx); err != nil {
		return err
	}
	if err := s.seedDNS(ctx); err != nil {
		return err
	}

	// Phase 2: Security (SSL, Firewall, Rate Limits)
	if err := s.seedSSL(ctx); err != nil {
		return err
	}
	if err := s.seedFirewall(ctx); err != nil {
		return err
	}

	// Phase 3: Performance (Cache, Load Balancer)
	if err := s.seedCache(ctx); err != nil {
		return err
	}
	if err := s.seedLoadBalancer(ctx); err != nil {
		return err
	}

	// Phase 4: Storage (KV, R2, D1)
	if err := s.seedKV(ctx); err != nil {
		return err
	}
	if err := s.seedR2(ctx); err != nil {
		return err
	}
	if err := s.seedD1(ctx); err != nil {
		return err
	}

	// Phase 5: Workers (after storage for bindings)
	if err := s.seedWorkers(ctx); err != nil {
		return err
	}

	// Phase 6: Rules (Page Rules, Transform Rules)
	if err := s.seedRules(ctx); err != nil {
		return err
	}

	// Phase 7: Compute (Durable Objects, Queues)
	if err := s.seedDurableObjects(ctx); err != nil {
		return err
	}
	if err := s.seedQueues(ctx); err != nil {
		return err
	}

	// Phase 8: AI & Search (Vectorize, AI Gateway)
	if err := s.seedVectorize(ctx); err != nil {
		return err
	}
	if err := s.seedAIGateway(ctx); err != nil {
		return err
	}

	// Phase 9: Data & Analytics
	if err := s.seedAnalyticsEngine(ctx); err != nil {
		return err
	}
	if err := s.seedAnalytics(ctx); err != nil {
		return err
	}

	// Phase 10: Network (Hyperdrive)
	if err := s.seedHyperdrive(ctx); err != nil {
		return err
	}

	// Phase 11: Scheduling (Cron)
	if err := s.seedCron(ctx); err != nil {
		return err
	}

	slog.Info("seed process complete")
	return nil
}

// generateID creates a new ULID.
func generateID() string {
	return ulid.Make().String()
}

// timeAgo returns a time in the past.
func (s *Seeder) timeAgo(d time.Duration) time.Time {
	return s.now.Add(-d)
}

// timeFuture returns a time in the future.
func (s *Seeder) timeFuture(d time.Duration) time.Time {
	return s.now.Add(d)
}
