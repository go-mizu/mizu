package seed

import (
	"context"
	"log/slog"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedRules(ctx context.Context) error {
	slog.Info("seeding page rules and transform rules")

	pageRuleCount := 0
	transformRuleCount := 0

	// Page Rules
	pageRules := []struct {
		zoneName string
		rules    []*store.PageRule
	}{
		{
			zoneName: "example.com",
			rules: []*store.PageRule{
				{
					ID:       generateID(),
					Targets:  []string{"example.com/wp-admin/*"},
					Actions:  map[string]interface{}{"ssl": "full", "security_level": "high", "cache_level": "bypass"},
					Priority: 1,
					Status:   "active",
					CreatedAt: s.now,
				},
				{
					ID:       generateID(),
					Targets:  []string{"example.com/static/*"},
					Actions:  map[string]interface{}{"cache_level": "cache_everything", "edge_cache_ttl": 2592000, "browser_cache_ttl": 86400},
					Priority: 2,
					Status:   "active",
					CreatedAt: s.now,
				},
				{
					ID:       generateID(),
					Targets:  []string{"example.com/api/*"},
					Actions:  map[string]interface{}{"cache_level": "bypass", "disable_apps": true},
					Priority: 3,
					Status:   "active",
					CreatedAt: s.now,
				},
			},
		},
		{
			zoneName: "store.acme.co",
			rules: []*store.PageRule{
				{
					ID:       generateID(),
					Targets:  []string{"store.acme.co/checkout/*"},
					Actions:  map[string]interface{}{"ssl": "strict", "disable_apps": true, "security_level": "high"},
					Priority: 1,
					Status:   "active",
					CreatedAt: s.now,
				},
				{
					ID:       generateID(),
					Targets:  []string{"store.acme.co/images/*"},
					Actions:  map[string]interface{}{"polish": "lossy", "mirage": true, "cache_level": "cache_everything"},
					Priority: 2,
					Status:   "active",
					CreatedAt: s.now,
				},
				{
					ID:       generateID(),
					Targets:  []string{"store.acme.co/sale/*"},
					Actions:  map[string]interface{}{"forwarding_url": map[string]interface{}{"status_code": 301, "url": "https://store.acme.co/deals/$1"}},
					Priority: 3,
					Status:   "active",
					CreatedAt: s.now,
				},
				{
					ID:       generateID(),
					Targets:  []string{"store.acme.co/old-product/*"},
					Actions:  map[string]interface{}{"forwarding_url": map[string]interface{}{"status_code": 302, "url": "https://store.acme.co/products/$1"}},
					Priority: 4,
					Status:   "disabled",
					CreatedAt: s.now,
				},
			},
		},
		{
			zoneName: "api.myapp.io",
			rules: []*store.PageRule{
				{
					ID:       generateID(),
					Targets:  []string{"api.myapp.io/*"},
					Actions:  map[string]interface{}{"ssl": "strict", "disable_apps": true, "cache_level": "bypass"},
					Priority: 1,
					Status:   "active",
					CreatedAt: s.now,
				},
			},
		},
	}

	for _, zr := range pageRules {
		zoneID, ok := s.ids.Zones[zr.zoneName]
		if !ok {
			continue
		}
		for _, rule := range zr.rules {
			rule.ZoneID = zoneID
			if err := s.store.Rules().CreatePageRule(ctx, rule); err == nil {
				pageRuleCount++
			}
		}
	}

	// Transform Rules
	transformRules := []struct {
		zoneName string
		rules    []*store.TransformRule
	}{
		{
			zoneName: "api.myapp.io",
			rules: []*store.TransformRule{
				{
					ID:          generateID(),
					Type:        "modify_request_header",
					Expression:  "true",
					Action:      "set",
					ActionValue: "X-Request-ID: ${cf.ray}",
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Type:        "modify_response_header",
					Expression:  "true",
					Action:      "set",
					ActionValue: "X-Response-Time: ${cf.edge.server_timing}",
					Priority:    2,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Type:        "modify_response_header",
					Expression:  "true",
					Action:      "set",
					ActionValue: "X-API-Version: v2.0",
					Priority:    3,
					Enabled:     true,
					CreatedAt:   s.now,
				},
			},
		},
		{
			zoneName: "store.acme.co",
			rules: []*store.TransformRule{
				{
					ID:          generateID(),
					Type:        "rewrite_url",
					Expression:  `http.request.uri.path matches "^/old/(.*)"`,
					Action:      "rewrite",
					ActionValue: "/new/${1}",
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Type:        "modify_response_header",
					Expression:  "true",
					Action:      "set",
					ActionValue: "X-Frame-Options: SAMEORIGIN",
					Priority:    2,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Type:        "modify_response_header",
					Expression:  "true",
					Action:      "set",
					ActionValue: "X-Content-Type-Options: nosniff",
					Priority:    3,
					Enabled:     true,
					CreatedAt:   s.now,
				},
			},
		},
		{
			zoneName: "example.com",
			rules: []*store.TransformRule{
				{
					ID:          generateID(),
					Type:        "modify_response_header",
					Expression:  "true",
					Action:      "set",
					ActionValue: "X-Frame-Options: DENY",
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Type:        "modify_response_header",
					Expression:  "true",
					Action:      "set",
					ActionValue: "Strict-Transport-Security: max-age=31536000; includeSubDomains",
					Priority:    2,
					Enabled:     true,
					CreatedAt:   s.now,
				},
			},
		},
	}

	for _, zr := range transformRules {
		zoneID, ok := s.ids.Zones[zr.zoneName]
		if !ok {
			continue
		}
		for _, rule := range zr.rules {
			rule.ZoneID = zoneID
			if err := s.store.Rules().CreateTransformRule(ctx, rule); err == nil {
				transformRuleCount++
			}
		}
	}

	slog.Info("rules seeded", "page_rules", pageRuleCount, "transform_rules", transformRuleCount)
	return nil
}
