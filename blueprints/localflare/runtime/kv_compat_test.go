package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/blueprints/localflare/store/sqlite"
)

// Test helpers

// setupKVTest creates a store, namespace, and runtime for KV testing.
func setupKVTest(t *testing.T) (store.Store, string, func()) {
	t.Helper()

	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "kv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create store
	s, err := sqlite.New(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	// Ensure schema
	if err := s.Ensure(context.Background()); err != nil {
		s.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Create KV namespace
	nsID := fmt.Sprintf("ns-%d", time.Now().UnixNano())
	ns := &store.KVNamespace{
		ID:        nsID,
		Title:     "test-namespace",
		CreatedAt: time.Now(),
	}
	if err := s.KV().CreateNamespace(context.Background(), ns); err != nil {
		s.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create namespace: %v", err)
	}

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}

	return s, nsID, cleanup
}

// ===========================================================================
// 1. KV.get() - Read Operations
// ===========================================================================

// TestWranglerCompat_KVGET001_BasicGet tests basic get() returns string value.
func TestWranglerCompat_KVGET001_BasicGet(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	// Pre-populate with test data
	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "test-key", Value: []byte("hello world")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("test-key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GET-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "hello world" {
		t.Errorf("KV-GET-001: Expected 'hello world', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVGET002_GetNonExistent tests get() returns null for non-existent key.
func TestWranglerCompat_KVGET002_GetNonExistent(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("nonexistent");
			event.respondWith(new Response(value === null ? "null" : "not-null"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-GET-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "null" {
		t.Errorf("KV-GET-002: Expected 'null', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVGET003_GetTypeText tests get() with type "text".
func TestWranglerCompat_KVGET003_GetTypeText(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "text-key", Value: []byte("text content")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("text-key", "text");
			event.respondWith(new Response(typeof value + ":" + value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GET-003: Execute failed: %v", err)
	}

	if !strings.HasPrefix(string(resp.Body), "string:text content") {
		t.Errorf("KV-GET-003: Expected 'string:text content', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVGET004_GetTypeJSON tests get() with type "json".
func TestWranglerCompat_KVGET004_GetTypeJSON(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "json-key", Value: []byte(`{"name":"test","count":42}`)})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("json-key", "json");
			const result = {
				isObject: typeof value === 'object',
				name: value.name,
				count: value.count
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GET-004: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-GET-004: Failed to parse response: %v", err)
	}

	if result["isObject"] != true {
		t.Errorf("KV-GET-004: Expected isObject to be true")
	}
	if result["name"] != "test" {
		t.Errorf("KV-GET-004: Expected name 'test', got '%v'", result["name"])
	}
	if result["count"] != float64(42) {
		t.Errorf("KV-GET-004: Expected count 42, got '%v'", result["count"])
	}
}

// TestWranglerCompat_KVGET005_GetTypeArrayBuffer tests get() with type "arrayBuffer".
func TestWranglerCompat_KVGET005_GetTypeArrayBuffer(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "binary-key", Value: []byte{0x01, 0x02, 0x03, 0x04}})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("binary-key", "arrayBuffer");
			const bytes = new Uint8Array(value);
			event.respondWith(new Response(bytes.length + ":" + Array.from(bytes).join(",")));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GET-005: Execute failed: %v", err)
	}

	if string(resp.Body) != "4:1,2,3,4" {
		t.Errorf("KV-GET-005: Expected '4:1,2,3,4', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVGET007_GetWithOptionsObject tests get() with options object.
func TestWranglerCompat_KVGET007_GetWithOptionsObject(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "options-key", Value: []byte(`{"data":"value"}`)})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("options-key", { type: "json" });
			event.respondWith(new Response(value.data));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GET-007: Execute failed: %v", err)
	}

	if string(resp.Body) != "value" {
		t.Errorf("KV-GET-007: Expected 'value', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVGET008_GetEmptyString tests get() handles empty string value.
func TestWranglerCompat_KVGET008_GetEmptyString(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "empty-key", Value: []byte("")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("empty-key");
			const isNull = value === null;
			const isEmpty = value === "";
			event.respondWith(new Response("isNull:" + isNull + ",isEmpty:" + isEmpty));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GET-008: Execute failed: %v", err)
	}

	if string(resp.Body) != "isNull:false,isEmpty:true" {
		t.Errorf("KV-GET-008: Expected 'isNull:false,isEmpty:true', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVGET009_SpecialCharKeys tests keys with special characters.
func TestWranglerCompat_KVGET009_SpecialCharKeys(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	specialKey := "user:123:profile:日本語"
	s.KV().Put(ctx, nsID, &store.KVPair{Key: specialKey, Value: []byte("special value")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("user:123:profile:日本語");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GET-009: Execute failed: %v", err)
	}

	if string(resp.Body) != "special value" {
		t.Errorf("KV-GET-009: Expected 'special value', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 2. KV.getWithMetadata() - Read with Metadata
// ===========================================================================

// TestWranglerCompat_KVGWM001_GetWithMetadataNoMeta tests getWithMetadata without metadata.
func TestWranglerCompat_KVGWM001_GetWithMetadataNoMeta(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "meta-key", Value: []byte("data")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.getWithMetadata("meta-key");
			const response = {
				value: result.value,
				metadataIsNull: result.metadata === null
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GWM-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-GWM-001: Failed to parse response: %v", err)
	}

	if result["value"] != "data" {
		t.Errorf("KV-GWM-001: Expected value 'data', got '%v'", result["value"])
	}
	if result["metadataIsNull"] != true {
		t.Errorf("KV-GWM-001: Expected metadataIsNull to be true")
	}
}

// TestWranglerCompat_KVGWM002_GetWithMetadataWithMeta tests getWithMetadata with metadata.
func TestWranglerCompat_KVGWM002_GetWithMetadataWithMeta(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{
		Key:      "meta-key",
		Value:    []byte("data"),
		Metadata: map[string]string{"version": "1", "author": "test"},
	})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.getWithMetadata("meta-key");
			const response = {
				value: result.value,
				hasMetadata: result.metadata !== null,
				version: result.metadata ? result.metadata.version : null,
				author: result.metadata ? result.metadata.author : null
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-GWM-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-GWM-002: Failed to parse response: %v", err)
	}

	if result["value"] != "data" {
		t.Errorf("KV-GWM-002: Expected value 'data', got '%v'", result["value"])
	}
	if result["hasMetadata"] != true {
		t.Errorf("KV-GWM-002: Expected hasMetadata to be true")
	}
	if result["version"] != "1" {
		t.Errorf("KV-GWM-002: Expected version '1', got '%v'", result["version"])
	}
	if result["author"] != "test" {
		t.Errorf("KV-GWM-002: Expected author 'test', got '%v'", result["author"])
	}
}

// TestWranglerCompat_KVGWM003_GetWithMetadataNonExistent tests getWithMetadata for missing key.
func TestWranglerCompat_KVGWM003_GetWithMetadataNonExistent(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.getWithMetadata("nonexistent");
			const response = {
				valueIsNull: result.value === null,
				metadataIsNull: result.metadata === null
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-GWM-003: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-GWM-003: Failed to parse response: %v", err)
	}

	if result["valueIsNull"] != true {
		t.Errorf("KV-GWM-003: Expected valueIsNull to be true")
	}
	if result["metadataIsNull"] != true {
		t.Errorf("KV-GWM-003: Expected metadataIsNull to be true")
	}
}

// ===========================================================================
// 3. KV.put() - Write Operations
// ===========================================================================

// TestWranglerCompat_KVPUT001_BasicPut tests basic put() stores string value.
func TestWranglerCompat_KVPUT001_BasicPut(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			await KV.put("new-key", "new-value");
			const value = await KV.get("new-key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-PUT-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "new-value" {
		t.Errorf("KV-PUT-001: Expected 'new-value', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVPUT002_PutReturnsUndefined tests put() returns undefined.
func TestWranglerCompat_KVPUT002_PutReturnsUndefined(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.put("key", "value");
			event.respondWith(new Response(result === undefined ? "undefined" : "not-undefined"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-PUT-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "undefined" {
		t.Errorf("KV-PUT-002: Expected 'undefined', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVPUT003_PutWithExpiration tests put() with expiration option.
func TestWranglerCompat_KVPUT003_PutWithExpiration(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const expiration = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
			await KV.put("expiring-key", "value", { expiration: expiration });
			const value = await KV.get("expiring-key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-PUT-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "value" {
		t.Errorf("KV-PUT-003: Expected 'value', got '%s'", string(resp.Body))
	}

	// Verify expiration was set in store
	pair, err := s.KV().Get(context.Background(), nsID, "expiring-key")
	if err != nil {
		t.Fatalf("KV-PUT-003: Failed to get pair: %v", err)
	}
	if pair.Expiration == nil {
		t.Errorf("KV-PUT-003: Expected expiration to be set")
	}
}

// TestWranglerCompat_KVPUT004_PutWithExpirationTtl tests put() with expirationTtl option.
func TestWranglerCompat_KVPUT004_PutWithExpirationTtl(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			await KV.put("ttl-key", "value", { expirationTtl: 3600 }); // 1 hour
			const value = await KV.get("ttl-key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-PUT-004: Execute failed: %v", err)
	}

	if string(resp.Body) != "value" {
		t.Errorf("KV-PUT-004: Expected 'value', got '%s'", string(resp.Body))
	}

	// Verify expiration was set in store
	pair, err := s.KV().Get(context.Background(), nsID, "ttl-key")
	if err != nil {
		t.Fatalf("KV-PUT-004: Failed to get pair: %v", err)
	}
	if pair.Expiration == nil {
		t.Errorf("KV-PUT-004: Expected expiration to be set")
	}
	// Should expire roughly 1 hour from now
	expectedExpiry := time.Now().Add(1 * time.Hour)
	if pair.Expiration.Before(expectedExpiry.Add(-1*time.Minute)) || pair.Expiration.After(expectedExpiry.Add(1*time.Minute)) {
		t.Errorf("KV-PUT-004: Expiration should be ~1 hour from now, got %v", pair.Expiration)
	}
}

// TestWranglerCompat_KVPUT006_PutWithMetadata tests put() with metadata option.
func TestWranglerCompat_KVPUT006_PutWithMetadata(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			await KV.put("meta-key", "value", {
				metadata: { version: "1", source: "api" }
			});
			const result = await KV.getWithMetadata("meta-key");
			const response = {
				value: result.value,
				version: result.metadata ? result.metadata.version : null,
				source: result.metadata ? result.metadata.source : null
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-PUT-006: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-PUT-006: Failed to parse response: %v", err)
	}

	if result["value"] != "value" {
		t.Errorf("KV-PUT-006: Expected value 'value', got '%v'", result["value"])
	}
	if result["version"] != "1" {
		t.Errorf("KV-PUT-006: Expected version '1', got '%v'", result["version"])
	}
	if result["source"] != "api" {
		t.Errorf("KV-PUT-006: Expected source 'api', got '%v'", result["source"])
	}
}

// TestWranglerCompat_KVPUT008_PutOverwrites tests put() overwrites existing value.
func TestWranglerCompat_KVPUT008_PutOverwrites(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "overwrite-key", Value: []byte("v1")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			await KV.put("overwrite-key", "v2");
			const value = await KV.get("overwrite-key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-PUT-008: Execute failed: %v", err)
	}

	if string(resp.Body) != "v2" {
		t.Errorf("KV-PUT-008: Expected 'v2', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVPUT010_PutObjectValue tests put() with object value.
func TestWranglerCompat_KVPUT010_PutObjectValue(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			await KV.put("object-key", { data: "test", count: 42 });
			const value = await KV.get("object-key", "json");
			event.respondWith(Response.json(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-PUT-010: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-PUT-010: Failed to parse response: %v", err)
	}

	if result["data"] != "test" {
		t.Errorf("KV-PUT-010: Expected data 'test', got '%v'", result["data"])
	}
	if result["count"] != float64(42) {
		t.Errorf("KV-PUT-010: Expected count 42, got '%v'", result["count"])
	}
}

// ===========================================================================
// 4. KV.delete() - Delete Operations
// ===========================================================================

// TestWranglerCompat_KVDEL001_BasicDelete tests delete() removes key.
func TestWranglerCompat_KVDEL001_BasicDelete(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "delete-key", Value: []byte("to-delete")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			await KV.delete("delete-key");
			const value = await KV.get("delete-key");
			event.respondWith(new Response(value === null ? "deleted" : "still-exists"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-DEL-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "deleted" {
		t.Errorf("KV-DEL-001: Expected 'deleted', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVDEL002_DeleteReturnsUndefined tests delete() returns undefined.
func TestWranglerCompat_KVDEL002_DeleteReturnsUndefined(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "key", Value: []byte("value")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.delete("key");
			event.respondWith(new Response(result === undefined ? "undefined" : "not-undefined"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-DEL-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "undefined" {
		t.Errorf("KV-DEL-002: Expected 'undefined', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVDEL003_DeleteNonExistent tests delete() on non-existent key.
func TestWranglerCompat_KVDEL003_DeleteNonExistent(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			try {
				await KV.delete("nonexistent");
				event.respondWith(new Response("success"));
			} catch (e) {
				event.respondWith(new Response("error: " + e.message));
			}
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-DEL-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "success" {
		t.Errorf("KV-DEL-003: Expected 'success', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVDEL004_DeleteRemovesMetadata tests delete() removes metadata.
func TestWranglerCompat_KVDEL004_DeleteRemovesMetadata(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{
		Key:      "meta-delete-key",
		Value:    []byte("value"),
		Metadata: map[string]string{"key": "value"},
	})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			await KV.delete("meta-delete-key");
			const result = await KV.getWithMetadata("meta-delete-key");
			const response = {
				valueIsNull: result.value === null,
				metadataIsNull: result.metadata === null
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-DEL-004: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-DEL-004: Failed to parse response: %v", err)
	}

	if result["valueIsNull"] != true {
		t.Errorf("KV-DEL-004: Expected valueIsNull to be true")
	}
	if result["metadataIsNull"] != true {
		t.Errorf("KV-DEL-004: Expected metadataIsNull to be true")
	}
}

// ===========================================================================
// 5. KV.list() - List Operations
// ===========================================================================

// TestWranglerCompat_KVLIST001_BasicList tests list() returns keys array.
func TestWranglerCompat_KVLIST001_BasicList(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "key1", Value: []byte("v1")})
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "key2", Value: []byte("v2")})
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "key3", Value: []byte("v3")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.list();
			const response = {
				keysCount: result.keys.length,
				hasListComplete: 'list_complete' in result,
				listComplete: result.list_complete,
				keyNames: result.keys.map(k => k.name)
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-LIST-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-LIST-001: Failed to parse response: %v", err)
	}

	if result["keysCount"] != float64(3) {
		t.Errorf("KV-LIST-001: Expected 3 keys, got '%v'", result["keysCount"])
	}
	if result["hasListComplete"] != true {
		t.Errorf("KV-LIST-001: Expected hasListComplete to be true")
	}
	if result["listComplete"] != true {
		t.Errorf("KV-LIST-001: Expected listComplete to be true")
	}
}

// TestWranglerCompat_KVLIST002_ListWithPrefix tests list() with prefix filter.
func TestWranglerCompat_KVLIST002_ListWithPrefix(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "user:1", Value: []byte("u1")})
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "user:2", Value: []byte("u2")})
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "config:1", Value: []byte("c1")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.list({ prefix: "user:" });
			const response = {
				keysCount: result.keys.length,
				keyNames: result.keys.map(k => k.name)
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-LIST-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-LIST-002: Failed to parse response: %v", err)
	}

	if result["keysCount"] != float64(2) {
		t.Errorf("KV-LIST-002: Expected 2 keys, got '%v'", result["keysCount"])
	}
	keyNames := result["keyNames"].([]interface{})
	for _, name := range keyNames {
		if !strings.HasPrefix(name.(string), "user:") {
			t.Errorf("KV-LIST-002: Expected keys to start with 'user:', got '%v'", name)
		}
	}
}

// TestWranglerCompat_KVLIST003_ListLexicographicOrder tests list() lexicographic ordering.
func TestWranglerCompat_KVLIST003_ListLexicographicOrder(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	// Insert in random order
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "c", Value: []byte("c")})
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "a", Value: []byte("a")})
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "b", Value: []byte("b")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.list();
			const keyNames = result.keys.map(k => k.name);
			event.respondWith(new Response(keyNames.join(",")));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-LIST-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "a,b,c" {
		t.Errorf("KV-LIST-003: Expected 'a,b,c', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVLIST004_ListWithLimit tests list() with limit option.
func TestWranglerCompat_KVLIST004_ListWithLimit(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	for i := 1; i <= 10; i++ {
		s.KV().Put(ctx, nsID, &store.KVPair{Key: fmt.Sprintf("key%02d", i), Value: []byte("v")})
	}

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.list({ limit: 5 });
			const response = {
				keysCount: result.keys.length,
				listComplete: result.list_complete,
				hasCursor: result.cursor !== undefined && result.cursor !== ""
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-LIST-004: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-LIST-004: Failed to parse response: %v", err)
	}

	if result["keysCount"] != float64(5) {
		t.Errorf("KV-LIST-004: Expected 5 keys, got '%v'", result["keysCount"])
	}
	if result["listComplete"] != false {
		t.Errorf("KV-LIST-004: Expected listComplete to be false")
	}
}

// TestWranglerCompat_KVLIST007_ListWithExpiration tests list() returns expiration.
func TestWranglerCompat_KVLIST007_ListWithExpiration(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	exp := time.Now().Add(1 * time.Hour)
	s.KV().Put(ctx, nsID, &store.KVPair{Key: "exp-key", Value: []byte("v"), Expiration: &exp})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.list();
			const key = result.keys[0];
			const response = {
				name: key.name,
				hasExpiration: 'expiration' in key,
				expiration: key.expiration
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-LIST-007: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-LIST-007: Failed to parse response: %v", err)
	}

	if result["name"] != "exp-key" {
		t.Errorf("KV-LIST-007: Expected name 'exp-key', got '%v'", result["name"])
	}
	if result["hasExpiration"] != true {
		t.Errorf("KV-LIST-007: Expected hasExpiration to be true")
	}
	if result["expiration"] == nil {
		t.Errorf("KV-LIST-007: Expected expiration to be set")
	}
}

// TestWranglerCompat_KVLIST008_ListWithMetadata tests list() returns metadata.
func TestWranglerCompat_KVLIST008_ListWithMetadata(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	ctx := context.Background()
	s.KV().Put(ctx, nsID, &store.KVPair{
		Key:      "meta-key",
		Value:    []byte("v"),
		Metadata: map[string]string{"type": "test"},
	})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.list();
			const key = result.keys[0];
			const response = {
				name: key.name,
				hasMetadata: 'metadata' in key,
				metadataType: key.metadata ? key.metadata.type : null
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-LIST-008: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-LIST-008: Failed to parse response: %v", err)
	}

	if result["name"] != "meta-key" {
		t.Errorf("KV-LIST-008: Expected name 'meta-key', got '%v'", result["name"])
	}
	if result["hasMetadata"] != true {
		t.Errorf("KV-LIST-008: Expected hasMetadata to be true")
	}
	if result["metadataType"] != "test" {
		t.Errorf("KV-LIST-008: Expected metadataType 'test', got '%v'", result["metadataType"])
	}
}

// TestWranglerCompat_KVLIST009_ListEmptyNamespace tests list() on empty namespace.
func TestWranglerCompat_KVLIST009_ListEmptyNamespace(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const result = await KV.list();
			const response = {
				keysCount: result.keys.length,
				listComplete: result.list_complete
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-LIST-009: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-LIST-009: Failed to parse response: %v", err)
	}

	if result["keysCount"] != float64(0) {
		t.Errorf("KV-LIST-009: Expected 0 keys, got '%v'", result["keysCount"])
	}
	if result["listComplete"] != true {
		t.Errorf("KV-LIST-009: Expected listComplete to be true")
	}
}

// ===========================================================================
// 6. Promise Behavior
// ===========================================================================

// TestWranglerCompat_KVPROM001_AllMethodsReturnPromises tests all methods return promises.
func TestWranglerCompat_KVPROM001_AllMethodsReturnPromises(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const getPromise = KV.get("key");
			const putPromise = KV.put("key", "value");
			const deletePromise = KV.delete("nonexistent");
			const listPromise = KV.list();
			const getWithMetaPromise = KV.getWithMetadata("key");

			const results = {
				get: getPromise instanceof Promise,
				put: putPromise instanceof Promise,
				delete: deletePromise instanceof Promise,
				list: listPromise instanceof Promise,
				getWithMetadata: getWithMetaPromise instanceof Promise
			};

			// Wait for all promises to settle
			await Promise.all([getPromise, putPromise, deletePromise, listPromise, getWithMetaPromise]);

			event.respondWith(Response.json(results));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-PROM-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-PROM-001: Failed to parse response: %v", err)
	}

	methods := []string{"get", "put", "delete", "list", "getWithMetadata"}
	for _, method := range methods {
		if result[method] != true {
			t.Errorf("KV-PROM-001: Expected %s to return Promise", method)
		}
	}
}

// ===========================================================================
// 7. Edge Cases
// ===========================================================================

// TestWranglerCompat_KVEDGE001_UnicodeKeysValues tests Unicode support.
func TestWranglerCompat_KVEDGE001_UnicodeKeysValues(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const unicodeKey = "テスト-🎉-مرحبا";
			const unicodeValue = "Hello 世界 🌍 مرحبا";

			await KV.put(unicodeKey, unicodeValue);
			const retrieved = await KV.get(unicodeKey);

			event.respondWith(new Response(retrieved === unicodeValue ? "match" : "mismatch"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-EDGE-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "match" {
		t.Errorf("KV-EDGE-001: Unicode handling failed, got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_KVEDGE003_LargeJSONObjects tests large JSON handling.
func TestWranglerCompat_KVEDGE003_LargeJSONObjects(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			// Create a reasonably large object
			const largeObj = {
				users: [],
				metadata: { created: new Date().toISOString() }
			};
			for (let i = 0; i < 100; i++) {
				largeObj.users.push({
					id: i,
					name: "User " + i,
					email: "user" + i + "@example.com",
					data: "x".repeat(100)
				});
			}

			await KV.put("large-key", largeObj);
			const retrieved = await KV.get("large-key", "json");

			const response = {
				usersCount: retrieved.users.length,
				firstUserName: retrieved.users[0].name,
				lastUserName: retrieved.users[99].name
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-EDGE-003: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-EDGE-003: Failed to parse response: %v", err)
	}

	if result["usersCount"] != float64(100) {
		t.Errorf("KV-EDGE-003: Expected 100 users, got '%v'", result["usersCount"])
	}
	if result["firstUserName"] != "User 0" {
		t.Errorf("KV-EDGE-003: Expected first user 'User 0', got '%v'", result["firstUserName"])
	}
	if result["lastUserName"] != "User 99" {
		t.Errorf("KV-EDGE-003: Expected last user 'User 99', got '%v'", result["lastUserName"])
	}
}

// ===========================================================================
// 8. Namespace Isolation
// ===========================================================================

// TestWranglerCompat_KVNS001_NamespaceIsolation tests namespace isolation.
func TestWranglerCompat_KVNS001_NamespaceIsolation(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "kv-ns-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store
	s, err := sqlite.New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	if err := s.Ensure(context.Background()); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	ctx := context.Background()

	// Create two namespaces
	ns1ID := "ns1"
	ns2ID := "ns2"
	s.KV().CreateNamespace(ctx, &store.KVNamespace{ID: ns1ID, Title: "ns1", CreatedAt: time.Now()})
	s.KV().CreateNamespace(ctx, &store.KVNamespace{ID: ns2ID, Title: "ns2", CreatedAt: time.Now()})

	// Put same key in both namespaces with different values
	s.KV().Put(ctx, ns1ID, &store.KVPair{Key: "shared-key", Value: []byte("value-from-ns1")})
	s.KV().Put(ctx, ns2ID, &store.KVPair{Key: "shared-key", Value: []byte("value-from-ns2")})

	// Test NS1
	rt1 := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + ns1ID},
	})
	defer rt1.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("shared-key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp1, err := rt1.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-NS-001: Execute for NS1 failed: %v", err)
	}

	if string(resp1.Body) != "value-from-ns1" {
		t.Errorf("KV-NS-001: NS1 expected 'value-from-ns1', got '%s'", string(resp1.Body))
	}

	// Test NS2
	rt2 := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + ns2ID},
	})
	defer rt2.Close()

	resp2, err := rt2.Execute(ctx, script, req)
	if err != nil {
		t.Fatalf("KV-NS-001: Execute for NS2 failed: %v", err)
	}

	if string(resp2.Body) != "value-from-ns2" {
		t.Errorf("KV-NS-001: NS2 expected 'value-from-ns2', got '%s'", string(resp2.Body))
	}
}

// ===========================================================================
// 9. Complete Workflow Tests
// ===========================================================================

// TestWranglerCompat_KVWF001_FullCRUDWorkflow tests complete CRUD workflow.
func TestWranglerCompat_KVWF001_FullCRUDWorkflow(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const results = [];

			// 1. Create
			await KV.put("workflow-key", "initial-value", { metadata: { step: "created" }});
			let value = await KV.get("workflow-key");
			results.push("create:" + value);

			// 2. Read with metadata
			let meta = await KV.getWithMetadata("workflow-key");
			results.push("metadata:" + meta.metadata.step);

			// 3. Update
			await KV.put("workflow-key", "updated-value", { metadata: { step: "updated" }});
			value = await KV.get("workflow-key");
			results.push("update:" + value);

			// 4. List
			let list = await KV.list();
			results.push("list:" + list.keys.length);

			// 5. Delete
			await KV.delete("workflow-key");
			value = await KV.get("workflow-key");
			results.push("delete:" + (value === null ? "null" : value));

			event.respondWith(new Response(results.join(";")));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-WF-001: Execute failed: %v", err)
	}

	expected := "create:initial-value;metadata:created;update:updated-value;list:1;delete:null"
	if string(resp.Body) != expected {
		t.Errorf("KV-WF-001: Expected '%s', got '%s'", expected, string(resp.Body))
	}
}

// TestWranglerCompat_KVWF002_SessionManagement tests session-like usage pattern.
func TestWranglerCompat_KVWF002_SessionManagement(t *testing.T) {
	s, nsID, cleanup := setupKVTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"SESSIONS": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const sessionId = "session:abc123";

			// Create session
			const sessionData = {
				userId: "user-1",
				createdAt: Date.now(),
				permissions: ["read", "write"]
			};
			await SESSIONS.put(sessionId, sessionData, { expirationTtl: 3600 });

			// Read session
			const session = await SESSIONS.get(sessionId, "json");

			// Verify session
			const isValid = session && session.userId === "user-1";

			// Delete session (logout)
			await SESSIONS.delete(sessionId);

			// Verify deleted
			const afterDelete = await SESSIONS.get(sessionId);

			const response = {
				sessionCreated: true,
				sessionValid: isValid,
				sessionDeleted: afterDelete === null,
				permissions: session.permissions
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("KV-WF-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("KV-WF-002: Failed to parse response: %v", err)
	}

	if result["sessionCreated"] != true {
		t.Errorf("KV-WF-002: Expected sessionCreated to be true")
	}
	if result["sessionValid"] != true {
		t.Errorf("KV-WF-002: Expected sessionValid to be true")
	}
	if result["sessionDeleted"] != true {
		t.Errorf("KV-WF-002: Expected sessionDeleted to be true")
	}
}

// BenchmarkKVGet benchmarks the get operation.
func BenchmarkKVGet(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "kv-bench-*")
	defer os.RemoveAll(tmpDir)

	s, _ := sqlite.New(tmpDir)
	defer s.Close()
	s.Ensure(context.Background())

	nsID := "bench-ns"
	s.KV().CreateNamespace(context.Background(), &store.KVNamespace{ID: nsID, Title: "bench", CreatedAt: time.Now()})
	s.KV().Put(context.Background(), nsID, &store.KVPair{Key: "bench-key", Value: []byte("bench-value")})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const value = await KV.get("bench-key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.Execute(context.Background(), script, req)
	}
}

// BenchmarkKVPut benchmarks the put operation.
func BenchmarkKVPut(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "kv-bench-*")
	defer os.RemoveAll(tmpDir)

	s, _ := sqlite.New(tmpDir)
	defer s.Close()
	s.Ensure(context.Background())

	nsID := "bench-ns"
	s.KV().CreateNamespace(context.Background(), &store.KVNamespace{ID: nsID, Title: "bench", CreatedAt: time.Now()})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"KV": "kv:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			await KV.put("bench-key-" + Math.random(), "value");
			event.respondWith(new Response("ok"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.Execute(context.Background(), script, req)
	}
}
