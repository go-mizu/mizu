package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedWorkers(ctx context.Context) error {
	slog.Info("seeding workers")

	workerCount := 0
	routeCount := 0

	// Workers with various scripts
	workers := []*store.Worker{
		{
			ID:   generateID(),
			Name: "hello-world",
			Script: `export default {
  async fetch(request, env, ctx) {
    return new Response('Hello from Localflare!', {
      headers: { 'content-type': 'text/plain' },
    });
  },
};`,
			Routes:    []string{"example.com/*"},
			Bindings:  map[string]string{},
			Enabled:   true,
			CreatedAt: s.timeAgo(30 * 24 * time.Hour),
			UpdatedAt: s.now,
		},
		{
			ID:   generateID(),
			Name: "api-router",
			Script: `export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    // Load config from KV
    const config = await env.CONFIG.get('app:settings', 'json') || {};

    // API versioning
    if (url.pathname.startsWith('/v1/users')) {
      const users = await env.USERS.prepare('SELECT id, email, name FROM users LIMIT 10').all();
      return Response.json({
        version: 'v1',
        data: users.results,
        config: { maintenance: config.maintenance }
      });
    }

    if (url.pathname.startsWith('/v2/users')) {
      const users = await env.USERS.prepare('SELECT * FROM users LIMIT 10').all();
      return Response.json({
        version: 'v2',
        data: users.results,
        meta: { api_version: config.api_version }
      });
    }

    if (url.pathname === '/health') {
      return Response.json({ status: 'healthy', timestamp: Date.now() });
    }

    return new Response('API Router - Not Found', { status: 404 });
  },
};`,
			Routes: []string{"api.myapp.io/v1/*", "api.myapp.io/v2/*", "api.myapp.io/health"},
			Bindings: map[string]string{
				"CONFIG": s.ids.KVNamespaces["CONFIG"],
				"USERS":  s.ids.D1Databases["main"],
			},
			Enabled:   true,
			CreatedAt: s.timeAgo(60 * 24 * time.Hour),
			UpdatedAt: s.now,
		},
		{
			ID:   generateID(),
			Name: "image-optimizer",
			Script: `export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const key = url.pathname.replace('/images/', '');

    // Try to get from R2
    const object = await env.ASSETS.get(key);
    if (!object) {
      return new Response('Image not found', { status: 404 });
    }

    // Determine content type
    const contentType = object.httpMetadata?.contentType || 'application/octet-stream';

    // Add optimization headers
    const headers = new Headers();
    headers.set('content-type', contentType);
    headers.set('cache-control', 'public, max-age=31536000, immutable');
    headers.set('x-optimized', 'true');

    // Return optimized response
    return new Response(object.body, { headers });
  },
};`,
			Routes: []string{"store.acme.co/images/*"},
			Bindings: map[string]string{
				"ASSETS": s.ids.R2Buckets["assets"],
			},
			Enabled:   true,
			CreatedAt: s.timeAgo(45 * 24 * time.Hour),
			UpdatedAt: s.now,
		},
		{
			ID:   generateID(),
			Name: "auth-middleware",
			Script: `export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    // Get session token from cookie or header
    const cookies = request.headers.get('cookie') || '';
    const sessionMatch = cookies.match(/session=([^;]+)/);
    const token = sessionMatch ? sessionMatch[1] : request.headers.get('x-session-token');

    if (!token) {
      return new Response('Unauthorized', {
        status: 401,
        headers: { 'www-authenticate': 'Bearer' }
      });
    }

    // Validate session from KV
    const session = await env.SESSIONS.get('session:' + token, 'json');
    if (!session) {
      return new Response('Session expired', { status: 401 });
    }

    // Check expiration
    if (session.expires < Date.now() / 1000) {
      await env.SESSIONS.delete('session:' + token);
      return new Response('Session expired', { status: 401 });
    }

    // Add user info to request and forward
    const newRequest = new Request(request);
    // In production, you'd forward to origin here

    return new Response(JSON.stringify({
      message: 'Authenticated',
      user: { id: session.user_id, email: session.email, role: session.role }
    }), {
      headers: { 'content-type': 'application/json' }
    });
  },
};`,
			Routes: []string{"internal.corp/*"},
			Bindings: map[string]string{
				"SESSIONS": s.ids.KVNamespaces["SESSIONS"],
			},
			Enabled:   true,
			CreatedAt: s.timeAgo(90 * 24 * time.Hour),
			UpdatedAt: s.now,
		},
		{
			ID:   generateID(),
			Name: "analytics-tracker",
			Script: `export default {
  async fetch(request, env, ctx) {
    // Track the event
    const event = {
      type: 'page_view',
      url: request.url,
      method: request.method,
      timestamp: Date.now(),
      cf: request.cf ? {
        country: request.cf.country,
        city: request.cf.city,
        asn: request.cf.asn,
      } : null,
      headers: {
        userAgent: request.headers.get('user-agent'),
        referer: request.headers.get('referer'),
      },
    };

    // Send to queue asynchronously
    ctx.waitUntil(env.EVENTS.send(event));

    // Return transparent 1x1 pixel or 204
    const accept = request.headers.get('accept') || '';
    if (accept.includes('image')) {
      const pixel = new Uint8Array([
        0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
        0x01, 0x00, 0x80, 0x00, 0x00, 0xFF, 0xFF, 0xFF,
        0x00, 0x00, 0x00, 0x21, 0xF9, 0x04, 0x01, 0x00,
        0x00, 0x00, 0x00, 0x2C, 0x00, 0x00, 0x00, 0x00,
        0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
        0x01, 0x00, 0x3B
      ]);
      return new Response(pixel, {
        headers: { 'content-type': 'image/gif' }
      });
    }

    return new Response(null, { status: 204 });
  },
};`,
			Routes:    []string{"store.acme.co/_track"},
			Bindings:  map[string]string{},
			Enabled:   true,
			CreatedAt: s.timeAgo(20 * 24 * time.Hour),
			UpdatedAt: s.now,
		},
	}

	for _, worker := range workers {
		if err := s.store.Workers().Create(ctx, worker); err == nil {
			s.ids.Workers[worker.Name] = worker.ID
			workerCount++
		}
	}

	// Worker Routes
	routes := []struct {
		zoneName string
		routes   []*store.WorkerRoute
	}{
		{
			zoneName: "example.com",
			routes: []*store.WorkerRoute{
				{ID: generateID(), Pattern: "example.com/*", WorkerID: s.ids.Workers["hello-world"], Enabled: true},
				{ID: generateID(), Pattern: "example.com/blog/*", WorkerID: s.ids.Workers["hello-world"], Enabled: true},
			},
		},
		{
			zoneName: "api.myapp.io",
			routes: []*store.WorkerRoute{
				{ID: generateID(), Pattern: "api.myapp.io/v1/*", WorkerID: s.ids.Workers["api-router"], Enabled: true},
				{ID: generateID(), Pattern: "api.myapp.io/v2/*", WorkerID: s.ids.Workers["api-router"], Enabled: true},
				{ID: generateID(), Pattern: "api.myapp.io/health", WorkerID: s.ids.Workers["api-router"], Enabled: true},
			},
		},
		{
			zoneName: "store.acme.co",
			routes: []*store.WorkerRoute{
				{ID: generateID(), Pattern: "store.acme.co/images/*", WorkerID: s.ids.Workers["image-optimizer"], Enabled: true},
				{ID: generateID(), Pattern: "store.acme.co/_track", WorkerID: s.ids.Workers["analytics-tracker"], Enabled: true},
			},
		},
		{
			zoneName: "internal.corp",
			routes: []*store.WorkerRoute{
				{ID: generateID(), Pattern: "internal.corp/*", WorkerID: s.ids.Workers["auth-middleware"], Enabled: true},
			},
		},
	}

	for _, zr := range routes {
		zoneID, ok := s.ids.Zones[zr.zoneName]
		if !ok {
			continue
		}
		for _, route := range zr.routes {
			route.ZoneID = zoneID
			if err := s.store.Workers().CreateRoute(ctx, route); err == nil {
				routeCount++
			}
		}
	}

	slog.Info("workers seeded", "workers", workerCount, "routes", routeCount)
	return nil
}
