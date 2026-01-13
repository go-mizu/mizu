package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedHyperdrive(ctx context.Context) error {
	slog.Info("seeding hyperdrive configs")

	configCount := 0

	// Hyperdrive Configurations
	configs := []*store.HyperdriveConfig{
		{
			ID:   generateID(),
			Name: "postgres-prod",
			Origin: store.HyperdriveOrigin{
				Database: "app_production",
				Host:     "prod-db.example.com",
				Port:     5432,
				Scheme:   "postgres",
				User:     "app_user",
				Password: "********",
			},
			Caching: store.HyperdriveCaching{
				Disabled:             false,
				MaxAge:               60, // 60 seconds
				StaleWhileRevalidate: 30,
			},
			CreatedAt: s.timeAgo(60 * 24 * time.Hour),
		},
		{
			ID:   generateID(),
			Name: "postgres-analytics",
			Origin: store.HyperdriveOrigin{
				Database: "analytics",
				Host:     "analytics-db.example.com",
				Port:     5432,
				Scheme:   "postgres",
				User:     "analytics_reader",
				Password: "********",
			},
			Caching: store.HyperdriveCaching{
				Disabled:             true,
				MaxAge:               0,
				StaleWhileRevalidate: 0,
			},
			CreatedAt: s.timeAgo(45 * 24 * time.Hour),
		},
		{
			ID:   generateID(),
			Name: "mysql-legacy",
			Origin: store.HyperdriveOrigin{
				Database: "old_app",
				Host:     "legacy-db.example.com",
				Port:     3306,
				Scheme:   "mysql",
				User:     "legacy_user",
				Password: "********",
			},
			Caching: store.HyperdriveCaching{
				Disabled:             false,
				MaxAge:               300, // 5 minutes
				StaleWhileRevalidate: 60,
			},
			CreatedAt: s.timeAgo(90 * 24 * time.Hour),
		},
		{
			ID:   generateID(),
			Name: "postgres-staging",
			Origin: store.HyperdriveOrigin{
				Database: "app_staging",
				Host:     "staging-db.example.com",
				Port:     5432,
				Scheme:   "postgres",
				User:     "staging_user",
				Password: "********",
			},
			Caching: store.HyperdriveCaching{
				Disabled:             false,
				MaxAge:               30,
				StaleWhileRevalidate: 15,
			},
			CreatedAt: s.timeAgo(30 * 24 * time.Hour),
		},
		{
			ID:   generateID(),
			Name: "postgres-reporting",
			Origin: store.HyperdriveOrigin{
				Database: "reporting",
				Host:     "reporting-db.example.com",
				Port:     5432,
				Scheme:   "postgres",
				User:     "report_reader",
				Password: "********",
			},
			Caching: store.HyperdriveCaching{
				Disabled:             false,
				MaxAge:               600, // 10 minutes
				StaleWhileRevalidate: 120,
			},
			CreatedAt: s.timeAgo(15 * 24 * time.Hour),
		},
	}

	for _, cfg := range configs {
		if err := s.store.Hyperdrive().CreateConfig(ctx, cfg); err == nil {
			s.ids.Hyperdrive[cfg.Name] = cfg.ID
			configCount++
		}
	}

	slog.Info("hyperdrive seeded", "configs", configCount)
	return nil
}
