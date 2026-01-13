package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedSSL(ctx context.Context) error {
	slog.Info("seeding SSL certificates and settings")

	certCount := 0
	settingsCount := 0

	// Certificates for each zone
	certs := []struct {
		zoneName string
		certs    []*store.Certificate
	}{
		{
			zoneName: "example.com",
			certs: []*store.Certificate{
				{
					ID:          generateID(),
					Type:        "edge",
					Hosts:       []string{"*.example.com", "example.com"},
					Issuer:      "Localflare CA",
					SerialNum:   "LF-001-2024",
					Signature:   "SHA256withRSA",
					Status:      "active",
					ExpiresAt:   s.timeFuture(90 * 24 * time.Hour),
					Certificate: "-----BEGIN CERTIFICATE-----\nMIIB...(example.com edge cert)...\n-----END CERTIFICATE-----",
					CreatedAt:   s.timeAgo(30 * 24 * time.Hour),
				},
			},
		},
		{
			zoneName: "api.myapp.io",
			certs: []*store.Certificate{
				{
					ID:          generateID(),
					Type:        "edge",
					Hosts:       []string{"*.api.myapp.io", "api.myapp.io"},
					Issuer:      "Localflare CA",
					SerialNum:   "LF-002-2024",
					Signature:   "SHA256withRSA",
					Status:      "active",
					ExpiresAt:   s.timeFuture(180 * 24 * time.Hour),
					Certificate: "-----BEGIN CERTIFICATE-----\nMIIB...(api.myapp.io edge cert)...\n-----END CERTIFICATE-----",
					CreatedAt:   s.timeAgo(60 * 24 * time.Hour),
				},
			},
		},
		{
			zoneName: "store.acme.co",
			certs: []*store.Certificate{
				{
					ID:          generateID(),
					Type:        "edge",
					Hosts:       []string{"*.store.acme.co", "store.acme.co"},
					Issuer:      "Localflare CA",
					SerialNum:   "LF-003-2024",
					Signature:   "SHA256withRSA",
					Status:      "active",
					ExpiresAt:   s.timeFuture(365 * 24 * time.Hour),
					Certificate: "-----BEGIN CERTIFICATE-----\nMIIB...(store.acme.co edge cert)...\n-----END CERTIFICATE-----",
					CreatedAt:   s.timeAgo(90 * 24 * time.Hour),
				},
				{
					ID:          generateID(),
					Type:        "origin",
					Hosts:       []string{"checkout.store.acme.co"},
					Issuer:      "Localflare Origin CA",
					SerialNum:   "LF-ORIGIN-001",
					Signature:   "SHA256withECDSA",
					Status:      "active",
					ExpiresAt:   s.timeFuture(30 * 24 * time.Hour),
					Certificate: "-----BEGIN CERTIFICATE-----\nMIIB...(checkout origin cert)...\n-----END CERTIFICATE-----",
					CreatedAt:   s.timeAgo(7 * 24 * time.Hour),
				},
			},
		},
		{
			zoneName: "internal.corp",
			certs: []*store.Certificate{
				{
					ID:          generateID(),
					Type:        "origin",
					Hosts:       []string{"*.internal.corp"},
					Issuer:      "Internal Corp CA",
					SerialNum:   "CORP-001-2024",
					Signature:   "SHA256withRSA",
					Status:      "active",
					ExpiresAt:   s.timeFuture(60 * 24 * time.Hour),
					Certificate: "-----BEGIN CERTIFICATE-----\nMIIB...(internal.corp origin cert)...\n-----END CERTIFICATE-----",
					CreatedAt:   s.timeAgo(30 * 24 * time.Hour),
				},
			},
		},
	}

	for _, zc := range certs {
		zoneID, ok := s.ids.Zones[zc.zoneName]
		if !ok {
			continue
		}
		for _, cert := range zc.certs {
			cert.ZoneID = zoneID
			if err := s.store.SSL().CreateCertificate(ctx, cert); err == nil {
				certCount++
			}
		}
	}

	// SSL Settings for each zone
	settings := []struct {
		zoneName string
		settings *store.SSLSettings
	}{
		{
			zoneName: "example.com",
			settings: &store.SSLSettings{
				Mode:                   "full",
				AlwaysHTTPS:            true,
				MinTLSVersion:          "1.2",
				TLS13:                  true,
				AutomaticHTTPSRewrites: true,
			},
		},
		{
			zoneName: "api.myapp.io",
			settings: &store.SSLSettings{
				Mode:                   "strict",
				AlwaysHTTPS:            true,
				MinTLSVersion:          "1.2",
				TLS13:                  true,
				AutomaticHTTPSRewrites: true,
			},
		},
		{
			zoneName: "store.acme.co",
			settings: &store.SSLSettings{
				Mode:                   "strict",
				AlwaysHTTPS:            true,
				MinTLSVersion:          "1.3",
				TLS13:                  true,
				AutomaticHTTPSRewrites: true,
			},
		},
		{
			zoneName: "internal.corp",
			settings: &store.SSLSettings{
				Mode:                   "flexible",
				AlwaysHTTPS:            false,
				MinTLSVersion:          "1.2",
				TLS13:                  false,
				AutomaticHTTPSRewrites: false,
			},
		},
	}

	for _, zs := range settings {
		zoneID, ok := s.ids.Zones[zs.zoneName]
		if !ok {
			continue
		}
		zs.settings.ZoneID = zoneID
		if err := s.store.SSL().UpdateSettings(ctx, zs.settings); err == nil {
			settingsCount++
		}
	}

	slog.Info("SSL seeded", "certificates", certCount, "settings", settingsCount)
	return nil
}

func (s *Seeder) seedFirewall(ctx context.Context) error {
	slog.Info("seeding firewall rules")

	ruleCount := 0
	ipRuleCount := 0
	rateLimitCount := 0

	// Firewall rules per zone
	firewallRules := []struct {
		zoneName string
		rules    []*store.FirewallRule
	}{
		{
			zoneName: "example.com",
			rules: []*store.FirewallRule{
				{
					ID:          generateID(),
					Description: "Block known bad IPs",
					Expression:  `ip.src in {192.0.2.0/24}`,
					Action:      "block",
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
					UpdatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Description: "Challenge admin access",
					Expression:  `http.request.uri.path contains "/admin"`,
					Action:      "challenge",
					Priority:    2,
					Enabled:     true,
					CreatedAt:   s.now,
					UpdatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Description: "Allow Googlebot",
					Expression:  `http.user_agent contains "Googlebot"`,
					Action:      "allow",
					Priority:    3,
					Enabled:     true,
					CreatedAt:   s.now,
					UpdatedAt:   s.now,
				},
			},
		},
		{
			zoneName: "api.myapp.io",
			rules: []*store.FirewallRule{
				{
					ID:          generateID(),
					Description: "Require API key header",
					Expression:  `http.request.uri.path matches "^/v[12]/.*" and not http.request.headers["x-api-key"]`,
					Action:      "block",
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
					UpdatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Description: "Block missing User-Agent",
					Expression:  `not http.user_agent`,
					Action:      "block",
					Priority:    2,
					Enabled:     true,
					CreatedAt:   s.now,
					UpdatedAt:   s.now,
				},
			},
		},
		{
			zoneName: "store.acme.co",
			rules: []*store.FirewallRule{
				{
					ID:          generateID(),
					Description: "Challenge high-risk payment actions",
					Expression:  `http.request.uri.path contains "/checkout/payment"`,
					Action:      "challenge",
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
					UpdatedAt:   s.now,
				},
				{
					ID:          generateID(),
					Description: "Allow verified partners",
					Expression:  `http.request.headers["x-partner-key"] eq "verified"`,
					Action:      "allow",
					Priority:    2,
					Enabled:     true,
					CreatedAt:   s.now,
					UpdatedAt:   s.now,
				},
			},
		},
		{
			zoneName: "internal.corp",
			rules: []*store.FirewallRule{
				{
					ID:          generateID(),
					Description: "Allow only internal IPs",
					Expression:  `not ip.src in {192.168.0.0/16 10.0.0.0/8}`,
					Action:      "block",
					Priority:    1,
					Enabled:     true,
					CreatedAt:   s.now,
					UpdatedAt:   s.now,
				},
			},
		},
	}

	for _, zr := range firewallRules {
		zoneID, ok := s.ids.Zones[zr.zoneName]
		if !ok {
			continue
		}
		for _, rule := range zr.rules {
			rule.ZoneID = zoneID
			if err := s.store.Firewall().CreateRule(ctx, rule); err == nil {
				ruleCount++
			}
		}
	}

	// IP Access Rules
	ipRules := []struct {
		zoneName string
		rules    []*store.IPAccessRule
	}{
		{
			zoneName: "example.com",
			rules: []*store.IPAccessRule{
				{ID: generateID(), Mode: "block", Target: "ip_range", Value: "192.0.2.0/24", Notes: "Known attackers", CreatedAt: s.now},
				{ID: generateID(), Mode: "whitelist", Target: "ip", Value: "203.0.113.50", Notes: "Office IP", CreatedAt: s.now},
			},
		},
		{
			zoneName: "api.myapp.io",
			rules: []*store.IPAccessRule{
				{ID: generateID(), Mode: "whitelist", Target: "asn", Value: "AS13335", Notes: "Cloudflare", CreatedAt: s.now},
			},
		},
		{
			zoneName: "store.acme.co",
			rules: []*store.IPAccessRule{
				{ID: generateID(), Mode: "block", Target: "country", Value: "XX", Notes: "Test country", CreatedAt: s.now},
			},
		},
		{
			zoneName: "internal.corp",
			rules: []*store.IPAccessRule{
				{ID: generateID(), Mode: "whitelist", Target: "ip_range", Value: "192.168.0.0/16", Notes: "Internal network only", CreatedAt: s.now},
				{ID: generateID(), Mode: "whitelist", Target: "ip_range", Value: "10.0.0.0/8", Notes: "VPN users", CreatedAt: s.now},
			},
		},
	}

	for _, zi := range ipRules {
		zoneID, ok := s.ids.Zones[zi.zoneName]
		if !ok {
			continue
		}
		for _, rule := range zi.rules {
			rule.ZoneID = zoneID
			if err := s.store.Firewall().CreateIPAccessRule(ctx, rule); err == nil {
				ipRuleCount++
			}
		}
	}

	// Rate Limit Rules
	rateLimits := []struct {
		zoneName string
		rules    []*store.RateLimitRule
	}{
		{
			zoneName: "api.myapp.io",
			rules: []*store.RateLimitRule{
				{ID: generateID(), Description: "Rate limit auth endpoints", Expression: `http.request.uri.path matches "^/auth/.*"`, Threshold: 10, Period: 60, Action: "block", ActionTimeout: 300, Enabled: true, CreatedAt: s.now},
				{ID: generateID(), Description: "Rate limit API endpoints", Expression: `http.request.uri.path matches "^/api/.*"`, Threshold: 100, Period: 60, Action: "challenge", ActionTimeout: 60, Enabled: true, CreatedAt: s.now},
			},
		},
		{
			zoneName: "store.acme.co",
			rules: []*store.RateLimitRule{
				{ID: generateID(), Description: "Rate limit checkout", Expression: `http.request.uri.path contains "/checkout"`, Threshold: 5, Period: 60, Action: "challenge", ActionTimeout: 120, Enabled: true, CreatedAt: s.now},
				{ID: generateID(), Description: "Rate limit search", Expression: `http.request.uri.path contains "/search"`, Threshold: 30, Period: 60, Action: "log", ActionTimeout: 0, Enabled: true, CreatedAt: s.now},
			},
		},
	}

	for _, zr := range rateLimits {
		zoneID, ok := s.ids.Zones[zr.zoneName]
		if !ok {
			continue
		}
		for _, rule := range zr.rules {
			rule.ZoneID = zoneID
			if err := s.store.Firewall().CreateRateLimitRule(ctx, rule); err == nil {
				rateLimitCount++
			}
		}
	}

	slog.Info("firewall seeded", "rules", ruleCount, "ip_rules", ipRuleCount, "rate_limits", rateLimitCount)
	return nil
}
