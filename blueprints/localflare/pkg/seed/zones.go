package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedZones(ctx context.Context) error {
	slog.Info("seeding zones")

	zones := []*store.Zone{
		{
			ID:          generateID(),
			Name:        "example.com",
			Status:      "active",
			Plan:        "free",
			NameServers: []string{"ns1.localflare.local", "ns2.localflare.local"},
			CreatedAt:   s.timeAgo(30 * 24 * time.Hour),
			UpdatedAt:   s.now,
		},
		{
			ID:          generateID(),
			Name:        "api.myapp.io",
			Status:      "active",
			Plan:        "pro",
			NameServers: []string{"ns1.localflare.local", "ns2.localflare.local"},
			CreatedAt:   s.timeAgo(60 * 24 * time.Hour),
			UpdatedAt:   s.now,
		},
		{
			ID:          generateID(),
			Name:        "store.acme.co",
			Status:      "active",
			Plan:        "business",
			NameServers: []string{"ns1.localflare.local", "ns2.localflare.local"},
			CreatedAt:   s.timeAgo(90 * 24 * time.Hour),
			UpdatedAt:   s.now,
		},
		{
			ID:          generateID(),
			Name:        "internal.corp",
			Status:      "active",
			Plan:        "enterprise",
			NameServers: []string{"ns1.localflare.local", "ns2.localflare.local"},
			CreatedAt:   s.timeAgo(180 * 24 * time.Hour),
			UpdatedAt:   s.now,
		},
	}

	for _, zone := range zones {
		if err := s.store.Zones().Create(ctx, zone); err != nil {
			slog.Warn("failed to create zone", "name", zone.Name, "error", err)
			continue
		}
		s.ids.Zones[zone.Name] = zone.ID
		slog.Debug("created zone", "name", zone.Name, "id", zone.ID)
	}

	slog.Info("zones seeded", "count", len(s.ids.Zones))
	return nil
}

func (s *Seeder) seedDNS(ctx context.Context) error {
	slog.Info("seeding DNS records")

	count := 0

	// example.com DNS records
	if zoneID, ok := s.ids.Zones["example.com"]; ok {
		records := []*store.DNSRecord{
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "@", Content: "192.168.1.10", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "CNAME", Name: "www", Content: "example.com", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "blog", Content: "192.168.1.11", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "MX", Name: "@", Content: "mail.example.com", TTL: 300, Priority: 10, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "TXT", Name: "@", Content: "v=spf1 include:_spf.example.com ~all", TTL: 300, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "TXT", Name: "_dmarc", Content: "v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com", TTL: 300, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "CNAME", Name: "cdn", Content: "cdn.localflare.local", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "AAAA", Name: "@", Content: "2001:db8::1", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
		}
		for _, r := range records {
			if err := s.store.DNS().Create(ctx, r); err == nil {
				count++
			}
		}
	}

	// api.myapp.io DNS records
	if zoneID, ok := s.ids.Zones["api.myapp.io"]; ok {
		records := []*store.DNSRecord{
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "@", Content: "10.0.0.100", TTL: 60, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "CNAME", Name: "v1", Content: "api.myapp.io", TTL: 60, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "CNAME", Name: "v2", Content: "api.myapp.io", TTL: 60, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "ws", Content: "10.0.0.101", TTL: 60, Proxied: true, Comment: "WebSocket server", CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "health", Content: "10.0.0.102", TTL: 60, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "TXT", Name: "@", Content: "api-version=2.0", TTL: 300, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
		}
		for _, r := range records {
			if err := s.store.DNS().Create(ctx, r); err == nil {
				count++
			}
		}
	}

	// store.acme.co DNS records
	if zoneID, ok := s.ids.Zones["store.acme.co"]; ok {
		records := []*store.DNSRecord{
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "@", Content: "172.16.0.50", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "CNAME", Name: "www", Content: "store.acme.co", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "checkout", Content: "172.16.0.51", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "CNAME", Name: "images", Content: "r2.localflare.local", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "CNAME", Name: "api", Content: "api.store.acme.co", TTL: 300, Proxied: true, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "MX", Name: "@", Content: "mx1.acme.co", TTL: 300, Priority: 10, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "MX", Name: "@", Content: "mx2.acme.co", TTL: 300, Priority: 20, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "TXT", Name: "@", Content: "v=spf1 include:_spf.acme.co ~all", TTL: 300, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
		}
		for _, r := range records {
			if err := s.store.DNS().Create(ctx, r); err == nil {
				count++
			}
		}
	}

	// internal.corp DNS records
	if zoneID, ok := s.ids.Zones["internal.corp"]; ok {
		records := []*store.DNSRecord{
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "@", Content: "192.168.100.1", TTL: 300, Proxied: false, CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "gitlab", Content: "192.168.100.10", TTL: 300, Proxied: false, Comment: "GitLab server", CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "jenkins", Content: "192.168.100.11", TTL: 300, Proxied: false, Comment: "Jenkins CI", CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "grafana", Content: "192.168.100.12", TTL: 300, Proxied: false, Comment: "Grafana monitoring", CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "vault", Content: "192.168.100.13", TTL: 300, Proxied: false, Comment: "HashiCorp Vault", CreatedAt: s.now, UpdatedAt: s.now},
			{ID: generateID(), ZoneID: zoneID, Type: "A", Name: "prometheus", Content: "192.168.100.14", TTL: 300, Proxied: false, Comment: "Prometheus metrics", CreatedAt: s.now, UpdatedAt: s.now},
		}
		for _, r := range records {
			if err := s.store.DNS().Create(ctx, r); err == nil {
				count++
			}
		}
	}

	slog.Info("DNS records seeded", "count", count)
	return nil
}
