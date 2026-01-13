package seed

import (
	"context"
	"log/slog"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedCache(ctx context.Context) error {
	slog.Info("seeding cache settings and rules")

	settingsCount := 0
	ruleCount := 0

	// Cache Settings per zone
	settings := []struct {
		zoneName string
		settings *store.CacheSettings
	}{
		{
			zoneName: "example.com",
			settings: &store.CacheSettings{
				CacheLevel:      "standard",
				BrowserTTL:      14400, // 4 hours
				EdgeTTL:         7200,  // 2 hours
				DevelopmentMode: false,
				AlwaysOnline:    true,
			},
		},
		{
			zoneName: "api.myapp.io",
			settings: &store.CacheSettings{
				CacheLevel:      "no_query_string",
				BrowserTTL:      0,
				EdgeTTL:         3600, // 1 hour
				DevelopmentMode: false,
				AlwaysOnline:    false,
			},
		},
		{
			zoneName: "store.acme.co",
			settings: &store.CacheSettings{
				CacheLevel:      "standard",
				BrowserTTL:      3600,  // 1 hour
				EdgeTTL:         14400, // 4 hours
				DevelopmentMode: false,
				AlwaysOnline:    true,
			},
		},
		{
			zoneName: "internal.corp",
			settings: &store.CacheSettings{
				CacheLevel:      "ignore_query_string",
				BrowserTTL:      0,
				EdgeTTL:         0,
				DevelopmentMode: true,
				AlwaysOnline:    false,
			},
		},
	}

	for _, zs := range settings {
		zoneID, ok := s.ids.Zones[zs.zoneName]
		if !ok {
			continue
		}
		zs.settings.ZoneID = zoneID
		if err := s.store.Cache().UpdateSettings(ctx, zs.settings); err == nil {
			settingsCount++
		}
	}

	// Cache Rules
	cacheRules := []struct {
		zoneName string
		rules    []*store.CacheRule
	}{
		{
			zoneName: "example.com",
			rules: []*store.CacheRule{
				{
					ID:          generateID(),
					Name:        "Cache static assets",
					Expression:  `http.request.uri.path matches "^/static/.*"`,
					CacheLevel:  "cache_everything",
					EdgeTTL:     604800, // 1 week
					BrowserTTL:  86400,  // 1 day
					BypassCache: false,
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Name:        "Bypass API cache",
					Expression:  `http.request.uri.path matches "^/api/.*"`,
					CacheLevel:  "bypass",
					EdgeTTL:     0,
					BrowserTTL:  0,
					BypassCache: true,
					Priority:    2,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Name:        "Cache images long-term",
					Expression:  `http.request.uri.path matches ".*\\.(jpg|jpeg|png|gif|webp)$"`,
					CacheLevel:  "cache_everything",
					EdgeTTL:     2592000, // 30 days
					BrowserTTL:  604800,  // 1 week
					BypassCache: false,
					Priority:    3,
					Enabled:     true,
					CreatedAt:   s.now,
				},
			},
		},
		{
			zoneName: "store.acme.co",
			rules: []*store.CacheRule{
				{
					ID:          generateID(),
					Name:        "Bypass cart cache",
					Expression:  `http.request.uri.path matches "^/cart/.*"`,
					CacheLevel:  "bypass",
					EdgeTTL:     0,
					BrowserTTL:  0,
					BypassCache: true,
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Name:        "Cache product pages",
					Expression:  `http.request.uri.path matches "^/products/.*"`,
					CacheLevel:  "standard",
					EdgeTTL:     3600, // 1 hour
					BrowserTTL:  1800, // 30 min
					BypassCache: false,
					Priority:    2,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Name:        "Cache product images",
					Expression:  `http.request.uri.path matches "^/images/products/.*"`,
					CacheLevel:  "cache_everything",
					EdgeTTL:     604800, // 7 days
					BrowserTTL:  86400,  // 1 day
					BypassCache: false,
					Priority:    3,
					Enabled:     true,
					CreatedAt:   s.now,
				},
			},
		},
	}

	for _, zr := range cacheRules {
		zoneID, ok := s.ids.Zones[zr.zoneName]
		if !ok {
			continue
		}
		for _, rule := range zr.rules {
			rule.ZoneID = zoneID
			if err := s.store.Cache().CreateRule(ctx, rule); err == nil {
				ruleCount++
			}
		}
	}

	slog.Info("cache seeded", "settings", settingsCount, "rules", ruleCount)
	return nil
}

func (s *Seeder) seedLoadBalancer(ctx context.Context) error {
	slog.Info("seeding load balancers")

	hcCount := 0
	poolCount := 0
	lbCount := 0

	// Health Checks
	healthChecks := []*store.HealthCheck{
		{
			ID:              generateID(),
			Description:     "HTTP Health Check",
			Type:            "http",
			Method:          "GET",
			Path:            "/health",
			Port:            80,
			Timeout:         5,
			Retries:         2,
			Interval:        60,
			ExpectedBody:    "",
			ExpectedCodes:   "200",
			FollowRedirects: true,
			AllowInsecure:   false,
		},
		{
			ID:              generateID(),
			Description:     "HTTPS Ping Check",
			Type:            "https",
			Method:          "GET",
			Path:            "/ping",
			Port:            443,
			Timeout:         10,
			Retries:         3,
			Interval:        30,
			ExpectedBody:    "pong",
			ExpectedCodes:   "200",
			FollowRedirects: false,
			AllowInsecure:   false,
		},
		{
			ID:              generateID(),
			Description:     "TCP Port Check",
			Type:            "tcp",
			Method:          "",
			Path:            "",
			Port:            443,
			Timeout:         5,
			Retries:         2,
			Interval:        60,
			ExpectedBody:    "",
			ExpectedCodes:   "",
			FollowRedirects: false,
			AllowInsecure:   false,
		},
	}

	for _, hc := range healthChecks {
		if err := s.store.LoadBalancer().CreateHealthCheck(ctx, hc); err == nil {
			s.ids.HealthChecks[hc.Description] = hc.ID
			hcCount++
		}
	}

	// Origin Pools
	pools := []*store.OriginPool{
		{
			ID:   generateID(),
			Name: "api-primary",
			Origins: []store.Origin{
				{Name: "api-us-west-1", Address: "10.0.1.10", Weight: 1.0, Enabled: true},
				{Name: "api-us-east-1", Address: "10.0.2.10", Weight: 1.0, Enabled: true},
				{Name: "api-us-central-1", Address: "10.0.3.10", Weight: 1.0, Enabled: true},
			},
			CheckRegions: []string{"WNAM", "ENAM"},
			Description:  "Primary API servers in US",
			Enabled:      true,
			MinOrigins:   2,
			Monitor:      s.ids.HealthChecks["HTTP Health Check"],
			NotifyEmail:  "ops@myapp.io",
			CreatedAt:    s.now,
		},
		{
			ID:   generateID(),
			Name: "api-fallback",
			Origins: []store.Origin{
				{Name: "api-eu-west-1", Address: "10.1.1.10", Weight: 0.5, Enabled: true},
				{Name: "api-eu-central-1", Address: "10.1.2.10", Weight: 0.5, Enabled: true},
			},
			CheckRegions: []string{"EEUR"},
			Description:  "Fallback API servers in EU",
			Enabled:      true,
			MinOrigins:   1,
			Monitor:      s.ids.HealthChecks["HTTP Health Check"],
			NotifyEmail:  "ops@myapp.io",
			CreatedAt:    s.now,
		},
		{
			ID:   generateID(),
			Name: "store-us",
			Origins: []store.Origin{
				{Name: "store-us-west", Address: "172.16.1.10", Weight: 1.0, Enabled: true},
				{Name: "store-us-east", Address: "172.16.2.10", Weight: 1.0, Enabled: true},
			},
			CheckRegions: []string{"WNAM"},
			Description:  "US Store servers",
			Enabled:      true,
			MinOrigins:   1,
			Monitor:      s.ids.HealthChecks["HTTPS Ping Check"],
			NotifyEmail:  "ops@acme.co",
			CreatedAt:    s.now,
		},
		{
			ID:   generateID(),
			Name: "store-eu",
			Origins: []store.Origin{
				{Name: "store-eu-west", Address: "172.16.3.10", Weight: 1.0, Enabled: true},
				{Name: "store-eu-central", Address: "172.16.4.10", Weight: 1.0, Enabled: true},
			},
			CheckRegions: []string{"EEUR"},
			Description:  "EU Store servers",
			Enabled:      true,
			MinOrigins:   1,
			Monitor:      s.ids.HealthChecks["HTTPS Ping Check"],
			NotifyEmail:  "ops@acme.co",
			CreatedAt:    s.now,
		},
		{
			ID:   generateID(),
			Name: "internal-pool",
			Origins: []store.Origin{
				{Name: "internal-1", Address: "192.168.100.20", Weight: 1.0, Enabled: true},
				{Name: "internal-2", Address: "192.168.100.21", Weight: 1.0, Enabled: true},
				{Name: "internal-3", Address: "192.168.100.22", Weight: 1.0, Enabled: true},
			},
			CheckRegions: []string{"WNAM"},
			Description:  "Internal services pool",
			Enabled:      true,
			MinOrigins:   2,
			Monitor:      s.ids.HealthChecks["TCP Port Check"],
			NotifyEmail:  "ops@internal.corp",
			CreatedAt:    s.now,
		},
	}

	for _, pool := range pools {
		if err := s.store.LoadBalancer().CreatePool(ctx, pool); err == nil {
			s.ids.OriginPools[pool.Name] = pool.ID
			poolCount++
		}
	}

	// Load Balancers
	loadBalancers := []struct {
		zoneName string
		lb       *store.LoadBalancer
	}{
		{
			zoneName: "api.myapp.io",
			lb: &store.LoadBalancer{
				ID:              generateID(),
				Name:            "api-lb",
				Fallback:        s.ids.OriginPools["api-fallback"],
				DefaultPools:    []string{s.ids.OriginPools["api-primary"]},
				SessionAffinity: "cookie",
				SteeringPolicy:  "dynamic",
				Enabled:         true,
				CreatedAt:       s.now,
			},
		},
		{
			zoneName: "store.acme.co",
			lb: &store.LoadBalancer{
				ID:              generateID(),
				Name:            "store-lb",
				Fallback:        s.ids.OriginPools["store-eu"],
				DefaultPools:    []string{s.ids.OriginPools["store-us"], s.ids.OriginPools["store-eu"]},
				SessionAffinity: "ip_cookie",
				SteeringPolicy:  "geo",
				Enabled:         true,
				CreatedAt:       s.now,
			},
		},
		{
			zoneName: "internal.corp",
			lb: &store.LoadBalancer{
				ID:              generateID(),
				Name:            "internal-lb",
				Fallback:        "",
				DefaultPools:    []string{s.ids.OriginPools["internal-pool"]},
				SessionAffinity: "none",
				SteeringPolicy:  "off",
				Enabled:         true,
				CreatedAt:       s.now,
			},
		},
	}

	for _, zlb := range loadBalancers {
		zoneID, ok := s.ids.Zones[zlb.zoneName]
		if !ok {
			continue
		}
		zlb.lb.ZoneID = zoneID
		if err := s.store.LoadBalancer().Create(ctx, zlb.lb); err == nil {
			lbCount++
		}
	}

	slog.Info("load balancers seeded", "health_checks", hcCount, "pools", poolCount, "load_balancers", lbCount)
	return nil
}
