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

// Test helpers for Durable Objects

// setupDOTest creates a store, namespace, and runtime for DO testing.
func setupDOTest(t *testing.T) (store.Store, string, func()) {
	t.Helper()

	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "do-test-*")
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

	// Create DO namespace
	nsID := fmt.Sprintf("ns-%d", time.Now().UnixNano())
	ns := &store.DurableObjectNamespace{
		ID:        nsID,
		Name:      "test-namespace",
		Script:    "test-script",
		ClassName: "TestDO",
		CreatedAt: time.Now(),
	}
	if err := s.DurableObjects().CreateNamespace(context.Background(), ns); err != nil {
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
// 1. DurableObjectNamespace API Tests
// ===========================================================================

// TestWranglerCompat_DONS001_IdFromNameDeterministic tests idFromName creates deterministic IDs.
func TestWranglerCompat_DONS001_IdFromNameDeterministic(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id1 = DO.idFromName("my-object");
			const id2 = DO.idFromName("my-object");
			const same = id1.toString() === id2.toString();
			event.respondWith(new Response(same ? "deterministic" : "not-deterministic"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-NS-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "deterministic" {
		t.Errorf("DO-NS-001: Expected 'deterministic', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DONS002_IdFromNameHasNameProperty tests ID has name property.
func TestWranglerCompat_DONS002_IdFromNameHasNameProperty(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("test-name");
			event.respondWith(new Response(id.name));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-NS-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "test-name" {
		t.Errorf("DO-NS-002: Expected 'test-name', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DONS003_NewUniqueIdCreatesUniqueIds tests newUniqueId creates unique IDs.
func TestWranglerCompat_DONS003_NewUniqueIdCreatesUniqueIds(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id1 = DO.newUniqueId();
			const id2 = DO.newUniqueId();
			const different = id1.toString() !== id2.toString();
			event.respondWith(new Response(different ? "unique" : "not-unique"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-NS-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "unique" {
		t.Errorf("DO-NS-003: Expected 'unique', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DONS004_NewUniqueIdHasNoName tests unique ID has no name.
func TestWranglerCompat_DONS004_NewUniqueIdHasNoName(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.newUniqueId();
			const hasNoName = id.name === undefined;
			event.respondWith(new Response(hasNoName ? "no-name" : "has-name:" + id.name));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-NS-004: Execute failed: %v", err)
	}

	if string(resp.Body) != "no-name" {
		t.Errorf("DO-NS-004: Expected 'no-name', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DONS005_IdFromStringReconstructs tests idFromString reconstruction.
func TestWranglerCompat_DONS005_IdFromStringReconstructs(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const original = DO.newUniqueId();
			const str = original.toString();
			const reconstructed = DO.idFromString(str);
			const same = reconstructed.toString() === str;
			event.respondWith(new Response(same ? "reconstructed" : "mismatch"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-NS-005: Execute failed: %v", err)
	}

	if string(resp.Body) != "reconstructed" {
		t.Errorf("DO-NS-005: Expected 'reconstructed', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DONS006_IdFromStringWithNamedId tests idFromString with named IDs.
func TestWranglerCompat_DONS006_IdFromStringWithNamedId(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const original = DO.idFromName("test-object");
			const str = original.toString();
			const reconstructed = DO.idFromString(str);
			const same = reconstructed.toString() === str;
			event.respondWith(new Response(same ? "reconstructed" : "mismatch"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-NS-006: Execute failed: %v", err)
	}

	if string(resp.Body) != "reconstructed" {
		t.Errorf("DO-NS-006: Expected 'reconstructed', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DONS007_GetReturnsStubImmediately tests get returns stub.
func TestWranglerCompat_DONS007_GetReturnsStubImmediately(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("test");
			const stub = DO.get(id);
			const hasStub = stub !== null && stub !== undefined;
			const hasFetch = typeof stub.fetch === 'function';
			const hasStorage = typeof stub.storage === 'object';
			event.respondWith(new Response(
				hasStub && hasFetch && hasStorage ? "valid-stub" : "invalid-stub"
			));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-NS-007: Execute failed: %v", err)
	}

	if string(resp.Body) != "valid-stub" {
		t.Errorf("DO-NS-007: Expected 'valid-stub', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DONS008_DifferentNamesCreateDifferentIds tests different names.
func TestWranglerCompat_DONS008_DifferentNamesCreateDifferentIds(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id1 = DO.idFromName("object-a");
			const id2 = DO.idFromName("object-b");
			const different = id1.toString() !== id2.toString();
			event.respondWith(new Response(different ? "different" : "same"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-NS-008: Execute failed: %v", err)
	}

	if string(resp.Body) != "different" {
		t.Errorf("DO-NS-008: Expected 'different', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 2. DurableObjectId Tests
// ===========================================================================

// TestWranglerCompat_DOID001_ToStringReturnsString tests toString returns string.
func TestWranglerCompat_DOID001_ToStringReturnsString(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("test");
			const str = id.toString();
			const isString = typeof str === 'string' && str.length > 0;
			event.respondWith(new Response(isString ? "valid-string" : "invalid"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ID-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "valid-string" {
		t.Errorf("DO-ID-001: Expected 'valid-string', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOID002_UniqueIdFormat tests unique ID format.
func TestWranglerCompat_DOID002_UniqueIdFormat(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.newUniqueId();
			const str = id.toString();
			// Should be a valid UUID format or at least 32+ chars
			const validFormat = str.length >= 32;
			event.respondWith(new Response(validFormat ? str : "invalid-format"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ID-002: Execute failed: %v", err)
	}

	if len(resp.Body) < 32 {
		t.Errorf("DO-ID-002: Expected valid ID format, got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOID003_NamedIdContainsName tests named ID contains name.
func TestWranglerCompat_DOID003_NamedIdContainsName(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("my-special-name");
			const str = id.toString();
			const containsName = str.includes("my-special-name");
			event.respondWith(new Response(containsName ? "contains-name" : "no-name:" + str));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ID-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "contains-name" {
		t.Errorf("DO-ID-003: Expected 'contains-name', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 3. DurableObjectStub Tests
// ===========================================================================

// TestWranglerCompat_DOSTUB001_StubHasIdProperty tests stub has id property.
func TestWranglerCompat_DOSTUB001_StubHasIdProperty(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("test");
			const stub = DO.get(id);
			const hasId = stub.id !== undefined;
			const sameId = stub.id.toString() === id.toString();
			event.respondWith(new Response(hasId && sameId ? "valid-id" : "invalid-id"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STUB-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "valid-id" {
		t.Errorf("DO-STUB-001: Expected 'valid-id', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTUB002_StubHasNameProperty tests stub has name property.
func TestWranglerCompat_DOSTUB002_StubHasNameProperty(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("my-name");
			const stub = DO.get(id);
			event.respondWith(new Response(stub.name));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STUB-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "my-name" {
		t.Errorf("DO-STUB-002: Expected 'my-name', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTUB003_StubHasFetchMethod tests stub has fetch method.
func TestWranglerCompat_DOSTUB003_StubHasFetchMethod(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("test");
			const stub = DO.get(id);
			const hasFetch = typeof stub.fetch === 'function';
			event.respondWith(new Response(hasFetch ? "has-fetch" : "no-fetch"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STUB-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "has-fetch" {
		t.Errorf("DO-STUB-003: Expected 'has-fetch', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTUB004_FetchReturnsPromise tests fetch returns Promise.
func TestWranglerCompat_DOSTUB004_FetchReturnsPromise(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("test");
			const stub = DO.get(id);
			const result = stub.fetch(new Request('http://example.com'));
			const isPromise = result instanceof Promise;
			await result; // Settle the promise
			event.respondWith(new Response(isPromise ? "is-promise" : "not-promise"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STUB-004: Execute failed: %v", err)
	}

	if string(resp.Body) != "is-promise" {
		t.Errorf("DO-STUB-004: Expected 'is-promise', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTUB005_StubHasStorageProperty tests stub has storage.
func TestWranglerCompat_DOSTUB005_StubHasStorageProperty(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("test");
			const stub = DO.get(id);
			const hasStorage = stub.storage !== undefined;
			const hasGet = typeof stub.storage.get === 'function';
			const hasPut = typeof stub.storage.put === 'function';
			const hasDelete = typeof stub.storage.delete === 'function';
			const hasDeleteAll = typeof stub.storage.deleteAll === 'function';
			const hasList = typeof stub.storage.list === 'function';
			const all = hasStorage && hasGet && hasPut && hasDelete && hasDeleteAll && hasList;
			event.respondWith(new Response(all ? "all-methods" : "missing-methods"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STUB-005: Execute failed: %v", err)
	}

	if string(resp.Body) != "all-methods" {
		t.Errorf("DO-STUB-005: Expected 'all-methods', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 4. Storage KV API Tests
// ===========================================================================

// TestWranglerCompat_DOSTOR001_PutAndGetString tests put and get string.
func TestWranglerCompat_DOSTOR001_PutAndGetString(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			await stub.storage.put("key", "value");
			const value = await stub.storage.get("key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "value" {
		t.Errorf("DO-STOR-001: Expected 'value', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR002_GetReturnsUndefinedForMissingKey tests missing key.
func TestWranglerCompat_DOSTOR002_GetReturnsUndefinedForMissingKey(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			const value = await stub.storage.get("nonexistent");
			event.respondWith(new Response(value === undefined ? "undefined" : "not-undefined"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "undefined" {
		t.Errorf("DO-STOR-002: Expected 'undefined', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR003_PutAndGetObject tests object storage.
func TestWranglerCompat_DOSTOR003_PutAndGetObject(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			const obj = { name: "test", count: 42, nested: { a: 1 } };
			await stub.storage.put("object-key", obj);
			const value = await stub.storage.get("object-key");
			const result = {
				name: value.name,
				count: value.count,
				nestedA: value.nested.a
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-003: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-STOR-003: Failed to parse response: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("DO-STOR-003: Expected name 'test', got '%v'", result["name"])
	}
	if result["count"] != float64(42) {
		t.Errorf("DO-STOR-003: Expected count 42, got '%v'", result["count"])
	}
	if result["nestedA"] != float64(1) {
		t.Errorf("DO-STOR-003: Expected nestedA 1, got '%v'", result["nestedA"])
	}
}

// TestWranglerCompat_DOSTOR004_PutAndGetArray tests array storage.
func TestWranglerCompat_DOSTOR004_PutAndGetArray(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			const arr = [1, 2, 3, "four", { five: 5 }];
			await stub.storage.put("array-key", arr);
			const value = await stub.storage.get("array-key");
			const result = {
				isArray: Array.isArray(value),
				length: value.length,
				third: value[2],
				fourth: value[3]
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-004: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-STOR-004: Failed to parse response: %v", err)
	}

	if result["isArray"] != true {
		t.Errorf("DO-STOR-004: Expected isArray true")
	}
	if result["length"] != float64(5) {
		t.Errorf("DO-STOR-004: Expected length 5, got '%v'", result["length"])
	}
	if result["third"] != float64(3) {
		t.Errorf("DO-STOR-004: Expected third 3, got '%v'", result["third"])
	}
	if result["fourth"] != "four" {
		t.Errorf("DO-STOR-004: Expected fourth 'four', got '%v'", result["fourth"])
	}
}

// TestWranglerCompat_DOSTOR005_PutAndGetNumber tests number storage.
func TestWranglerCompat_DOSTOR005_PutAndGetNumber(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			await stub.storage.put("int-key", 42);
			await stub.storage.put("float-key", 3.14159);
			const intVal = await stub.storage.get("int-key");
			const floatVal = await stub.storage.get("float-key");
			event.respondWith(new Response(intVal + ":" + floatVal));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-005: Execute failed: %v", err)
	}

	if string(resp.Body) != "42:3.14159" {
		t.Errorf("DO-STOR-005: Expected '42:3.14159', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR006_PutAndGetBoolean tests boolean storage.
func TestWranglerCompat_DOSTOR006_PutAndGetBoolean(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			await stub.storage.put("true-key", true);
			await stub.storage.put("false-key", false);
			const trueVal = await stub.storage.get("true-key");
			const falseVal = await stub.storage.get("false-key");
			event.respondWith(new Response(trueVal + ":" + falseVal));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-006: Execute failed: %v", err)
	}

	if string(resp.Body) != "true:false" {
		t.Errorf("DO-STOR-006: Expected 'true:false', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR007_PutAndGetNull tests null storage.
func TestWranglerCompat_DOSTOR007_PutAndGetNull(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			await stub.storage.put("null-key", null);
			const value = await stub.storage.get("null-key");
			event.respondWith(new Response(value === null ? "null" : "not-null"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-007: Execute failed: %v", err)
	}

	if string(resp.Body) != "null" {
		t.Errorf("DO-STOR-007: Expected 'null', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR008_DeleteRemovesKey tests delete removes key.
func TestWranglerCompat_DOSTOR008_DeleteRemovesKey(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			await stub.storage.put("delete-me", "value");
			await stub.storage.delete("delete-me");
			const value = await stub.storage.get("delete-me");
			event.respondWith(new Response(value === undefined ? "deleted" : "still-exists"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-008: Execute failed: %v", err)
	}

	if string(resp.Body) != "deleted" {
		t.Errorf("DO-STOR-008: Expected 'deleted', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR009_DeleteReturnsTrue tests delete returns true.
func TestWranglerCompat_DOSTOR009_DeleteReturnsTrue(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			await stub.storage.put("key", "value");
			const result = await stub.storage.delete("key");
			event.respondWith(new Response(result === true ? "true" : "not-true:" + result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-009: Execute failed: %v", err)
	}

	if string(resp.Body) != "true" {
		t.Errorf("DO-STOR-009: Expected 'true', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR010_DeleteAllClearsStorage tests deleteAll.
func TestWranglerCompat_DOSTOR010_DeleteAllClearsStorage(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			await stub.storage.put("key1", "value1");
			await stub.storage.put("key2", "value2");
			await stub.storage.put("key3", "value3");
			await stub.storage.deleteAll();
			const v1 = await stub.storage.get("key1");
			const v2 = await stub.storage.get("key2");
			const v3 = await stub.storage.get("key3");
			const allDeleted = v1 === undefined && v2 === undefined && v3 === undefined;
			event.respondWith(new Response(allDeleted ? "all-deleted" : "not-deleted"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-010: Execute failed: %v", err)
	}

	if string(resp.Body) != "all-deleted" {
		t.Errorf("DO-STOR-010: Expected 'all-deleted', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR011_ListReturnsAllKeys tests list returns all.
func TestWranglerCompat_DOSTOR011_ListReturnsAllKeys(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("list-test"));
			await stub.storage.deleteAll();
			await stub.storage.put("a", 1);
			await stub.storage.put("b", 2);
			await stub.storage.put("c", 3);
			const entries = await stub.storage.list();
			const result = {
				a: entries.get("a"),
				b: entries.get("b"),
				c: entries.get("c"),
				size: entries.size
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-011: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-STOR-011: Failed to parse response: %v", err)
	}

	if result["a"] != float64(1) {
		t.Errorf("DO-STOR-011: Expected a=1, got '%v'", result["a"])
	}
	if result["b"] != float64(2) {
		t.Errorf("DO-STOR-011: Expected b=2, got '%v'", result["b"])
	}
	if result["c"] != float64(3) {
		t.Errorf("DO-STOR-011: Expected c=3, got '%v'", result["c"])
	}
	if result["size"] != float64(3) {
		t.Errorf("DO-STOR-011: Expected size=3, got '%v'", result["size"])
	}
}

// TestWranglerCompat_DOSTOR012_ListReturnsEmptyForEmptyStorage tests empty list.
func TestWranglerCompat_DOSTOR012_ListReturnsEmptyForEmptyStorage(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("empty-test"));
			const entries = await stub.storage.list();
			event.respondWith(new Response(entries.size === 0 ? "empty" : "not-empty:" + entries.size));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-012: Execute failed: %v", err)
	}

	if string(resp.Body) != "empty" {
		t.Errorf("DO-STOR-012: Expected 'empty', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR013_PutOverwritesExistingValue tests overwrite.
func TestWranglerCompat_DOSTOR013_PutOverwritesExistingValue(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			await stub.storage.put("key", "original");
			await stub.storage.put("key", "updated");
			const value = await stub.storage.get("key");
			event.respondWith(new Response(value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-013: Execute failed: %v", err)
	}

	if string(resp.Body) != "updated" {
		t.Errorf("DO-STOR-013: Expected 'updated', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR014_UnicodeKeysAndValues tests Unicode support.
func TestWranglerCompat_DOSTOR014_UnicodeKeysAndValues(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			const key = "key-日本語-emoji-🎉";
			const value = "value-中文-العربية";
			await stub.storage.put(key, value);
			const result = await stub.storage.get(key);
			event.respondWith(new Response(result === value ? "match" : "mismatch"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-014: Execute failed: %v", err)
	}

	if string(resp.Body) != "match" {
		t.Errorf("DO-STOR-014: Expected 'match', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOSTOR015_LargeValueStorage tests large value storage.
func TestWranglerCompat_DOSTOR015_LargeValueStorage(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			const largeObj = { data: "x".repeat(10000), items: [] };
			for (let i = 0; i < 100; i++) {
				largeObj.items.push({ id: i, value: "item-" + i });
			}
			await stub.storage.put("large-key", largeObj);
			const result = await stub.storage.get("large-key");
			const response = {
				dataLength: result.data.length,
				itemsLength: result.items.length
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-015: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-STOR-015: Failed to parse response: %v", err)
	}

	if result["dataLength"] != float64(10000) {
		t.Errorf("DO-STOR-015: Expected dataLength 10000, got '%v'", result["dataLength"])
	}
	if result["itemsLength"] != float64(100) {
		t.Errorf("DO-STOR-015: Expected itemsLength 100, got '%v'", result["itemsLength"])
	}
}

// TestWranglerCompat_DOSTOR016_AllMethodsReturnPromises tests promises.
func TestWranglerCompat_DOSTOR016_AllMethodsReturnPromises(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			const getP = stub.storage.get("key");
			const putP = stub.storage.put("key", "value");
			const deleteP = stub.storage.delete("key");
			const deleteAllP = stub.storage.deleteAll();
			const listP = stub.storage.list();

			const results = {
				get: getP instanceof Promise,
				put: putP instanceof Promise,
				delete: deleteP instanceof Promise,
				deleteAll: deleteAllP instanceof Promise,
				list: listP instanceof Promise
			};

			await Promise.all([getP, putP, deleteP, deleteAllP, listP]);
			event.respondWith(Response.json(results));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-STOR-016: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-STOR-016: Failed to parse response: %v", err)
	}

	methods := []string{"get", "put", "delete", "deleteAll", "list"}
	for _, method := range methods {
		if result[method] != true {
			t.Errorf("DO-STOR-016: Expected %s to return Promise", method)
		}
	}
}

// ===========================================================================
// 5. Batch Operations Tests
// ===========================================================================

// TestWranglerCompat_DOBATCH001_GetMultipleKeys tests batch get.
func TestWranglerCompat_DOBATCH001_GetMultipleKeys(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("batch-test"));
			await stub.storage.put("key1", "value1");
			await stub.storage.put("key2", "value2");
			await stub.storage.put("key3", "value3");

			const result = await stub.storage.get(["key1", "key2", "key3"]);
			const response = {
				size: result.size,
				key1: result.get("key1"),
				key2: result.get("key2"),
				key3: result.get("key3"),
				hasKey1: result.has("key1"),
				hasKey4: result.has("key4")
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-BATCH-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-BATCH-001: Failed to parse response: %v", err)
	}

	if result["size"] != float64(3) {
		t.Errorf("DO-BATCH-001: Expected size 3, got '%v'", result["size"])
	}
	if result["key1"] != "value1" {
		t.Errorf("DO-BATCH-001: Expected key1='value1', got '%v'", result["key1"])
	}
	if result["hasKey1"] != true {
		t.Errorf("DO-BATCH-001: Expected hasKey1=true")
	}
	if result["hasKey4"] != false {
		t.Errorf("DO-BATCH-001: Expected hasKey4=false")
	}
}

// TestWranglerCompat_DOBATCH002_PutMultipleEntries tests batch put.
func TestWranglerCompat_DOBATCH002_PutMultipleEntries(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("batch-put-test"));
			await stub.storage.deleteAll();

			await stub.storage.put({
				"batch1": "value1",
				"batch2": "value2",
				"batch3": { nested: true }
			});

			const v1 = await stub.storage.get("batch1");
			const v2 = await stub.storage.get("batch2");
			const v3 = await stub.storage.get("batch3");

			const response = {
				batch1: v1,
				batch2: v2,
				batch3Nested: v3.nested
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-BATCH-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-BATCH-002: Failed to parse response: %v", err)
	}

	if result["batch1"] != "value1" {
		t.Errorf("DO-BATCH-002: Expected batch1='value1', got '%v'", result["batch1"])
	}
	if result["batch2"] != "value2" {
		t.Errorf("DO-BATCH-002: Expected batch2='value2', got '%v'", result["batch2"])
	}
	if result["batch3Nested"] != true {
		t.Errorf("DO-BATCH-002: Expected batch3Nested=true, got '%v'", result["batch3Nested"])
	}
}

// TestWranglerCompat_DOBATCH003_DeleteMultipleKeys tests batch delete.
func TestWranglerCompat_DOBATCH003_DeleteMultipleKeys(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("batch-delete-test"));
			await stub.storage.put("del1", "v1");
			await stub.storage.put("del2", "v2");
			await stub.storage.put("del3", "v3");
			await stub.storage.put("keep", "keep-value");

			const deleted = await stub.storage.delete(["del1", "del2", "del3"]);

			const v1 = await stub.storage.get("del1");
			const v2 = await stub.storage.get("del2");
			const keep = await stub.storage.get("keep");

			const response = {
				deleted: deleted,
				del1Gone: v1 === undefined,
				del2Gone: v2 === undefined,
				keepExists: keep === "keep-value"
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-BATCH-003: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-BATCH-003: Failed to parse response: %v", err)
	}

	if result["deleted"] != float64(3) {
		t.Errorf("DO-BATCH-003: Expected deleted=3, got '%v'", result["deleted"])
	}
	if result["del1Gone"] != true {
		t.Errorf("DO-BATCH-003: Expected del1Gone=true")
	}
	if result["keepExists"] != true {
		t.Errorf("DO-BATCH-003: Expected keepExists=true")
	}
}

// TestWranglerCompat_DOBATCH004_GetMultipleReturnsMapLikeObject tests Map interface.
func TestWranglerCompat_DOBATCH004_GetMultipleReturnsMapLikeObject(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("map-test"));
			await stub.storage.put("a", 1);
			await stub.storage.put("b", 2);

			const result = await stub.storage.get(["a", "b"]);

			const keys = result.keys();
			const values = result.values();
			const entries = result.entries();

			const response = {
				hasKeys: Array.isArray(keys),
				hasValues: Array.isArray(values),
				hasEntries: Array.isArray(entries),
				keysLength: keys.length,
				valuesLength: values.length
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-BATCH-004: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-BATCH-004: Failed to parse response: %v", err)
	}

	if result["hasKeys"] != true {
		t.Errorf("DO-BATCH-004: Expected hasKeys=true")
	}
	if result["hasValues"] != true {
		t.Errorf("DO-BATCH-004: Expected hasValues=true")
	}
	if result["keysLength"] != float64(2) {
		t.Errorf("DO-BATCH-004: Expected keysLength=2, got '%v'", result["keysLength"])
	}
}

// ===========================================================================
// 6. List Options Tests
// ===========================================================================

// TestWranglerCompat_DOLIST001_ListWithPrefix tests list with prefix.
func TestWranglerCompat_DOLIST001_ListWithPrefix(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("list-prefix-test"));
			await stub.storage.deleteAll();
			await stub.storage.put("user:1", "alice");
			await stub.storage.put("user:2", "bob");
			await stub.storage.put("config:theme", "dark");

			const users = await stub.storage.list({ prefix: "user:" });
			const response = {
				size: users.size,
				hasUser1: users.has("user:1"),
				hasConfig: users.has("config:theme")
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-LIST-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-LIST-001: Failed to parse response: %v", err)
	}

	if result["size"] != float64(2) {
		t.Errorf("DO-LIST-001: Expected size=2, got '%v'", result["size"])
	}
	if result["hasUser1"] != true {
		t.Errorf("DO-LIST-001: Expected hasUser1=true")
	}
	if result["hasConfig"] != false {
		t.Errorf("DO-LIST-001: Expected hasConfig=false")
	}
}

// TestWranglerCompat_DOLIST002_ListWithLimit tests list with limit.
func TestWranglerCompat_DOLIST002_ListWithLimit(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("list-limit-test"));
			await stub.storage.deleteAll();
			for (let i = 0; i < 10; i++) {
				await stub.storage.put("key" + i, i);
			}

			const limited = await stub.storage.list({ limit: 3 });
			event.respondWith(new Response(limited.size.toString()));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-LIST-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "3" {
		t.Errorf("DO-LIST-002: Expected '3', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOLIST003_ListWithStartEnd tests list with start/end range.
func TestWranglerCompat_DOLIST003_ListWithStartEnd(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("list-range-test"));
			await stub.storage.deleteAll();
			await stub.storage.put("a", 1);
			await stub.storage.put("b", 2);
			await stub.storage.put("c", 3);
			await stub.storage.put("d", 4);
			await stub.storage.put("e", 5);

			const ranged = await stub.storage.list({ start: "b", end: "d" });
			const response = {
				size: ranged.size,
				hasA: ranged.has("a"),
				hasB: ranged.has("b"),
				hasC: ranged.has("c"),
				hasD: ranged.has("d"),
				hasE: ranged.has("e")
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-LIST-003: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-LIST-003: Failed to parse response: %v", err)
	}

	if result["size"] != float64(2) {
		t.Errorf("DO-LIST-003: Expected size=2, got '%v'", result["size"])
	}
	if result["hasA"] != false {
		t.Errorf("DO-LIST-003: Expected hasA=false (before start)")
	}
	if result["hasB"] != true {
		t.Errorf("DO-LIST-003: Expected hasB=true (at start)")
	}
	if result["hasC"] != true {
		t.Errorf("DO-LIST-003: Expected hasC=true (in range)")
	}
	if result["hasD"] != false {
		t.Errorf("DO-LIST-003: Expected hasD=false (at end, exclusive)")
	}
}

// TestWranglerCompat_DOLIST004_ListReturnsMapLike tests list returns Map-like.
func TestWranglerCompat_DOLIST004_ListReturnsMapLike(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("list-map-test"));
			await stub.storage.deleteAll();
			await stub.storage.put("x", 10);
			await stub.storage.put("y", 20);

			const result = await stub.storage.list();

			const response = {
				hasSize: typeof result.size === 'number',
				hasGet: typeof result.get === 'function',
				hasHas: typeof result.has === 'function',
				hasKeys: typeof result.keys === 'function',
				hasValues: typeof result.values === 'function',
				hasEntries: typeof result.entries === 'function',
				hasForEach: typeof result.forEach === 'function'
			};
			event.respondWith(Response.json(response));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-LIST-004: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-LIST-004: Failed to parse response: %v", err)
	}

	methods := []string{"hasSize", "hasGet", "hasHas", "hasKeys", "hasValues", "hasEntries", "hasForEach"}
	for _, method := range methods {
		if result[method] != true {
			t.Errorf("DO-LIST-004: Expected %s=true", method)
		}
	}
}

// TestWranglerCompat_DOLIST005_SyncMethod tests sync method exists.
func TestWranglerCompat_DOLIST005_SyncMethod(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("sync-test"));
			await stub.storage.put("key", "value");
			const syncResult = await stub.storage.sync();
			const isUndefined = syncResult === undefined;
			event.respondWith(new Response(isUndefined ? "sync-works" : "sync-failed"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-LIST-005: Execute failed: %v", err)
	}

	if string(resp.Body) != "sync-works" {
		t.Errorf("DO-LIST-005: Expected 'sync-works', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 7. Alarm API Tests
// ===========================================================================

// TestWranglerCompat_DOALARM001_SetAlarmSchedulesAlarm tests setAlarm schedules.
func TestWranglerCompat_DOALARM001_SetAlarmSchedulesAlarm(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("alarm-test"));
			const futureTime = Date.now() + 60000; // 1 minute from now
			await stub.storage.setAlarm(futureTime);
			const alarm = await stub.storage.getAlarm();
			const hasAlarm = alarm !== null && typeof alarm === 'number';
			event.respondWith(new Response(hasAlarm ? "alarm-set" : "no-alarm"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ALARM-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "alarm-set" {
		t.Errorf("DO-ALARM-001: Expected 'alarm-set', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOALARM002_GetAlarmReturnsNullWhenUnset tests null when no alarm.
func TestWranglerCompat_DOALARM002_GetAlarmReturnsNullWhenUnset(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("no-alarm-test"));
			const alarm = await stub.storage.getAlarm();
			event.respondWith(new Response(alarm === null ? "null" : "not-null:" + alarm));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ALARM-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "null" {
		t.Errorf("DO-ALARM-002: Expected 'null', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOALARM003_DeleteAlarmRemovesAlarm tests deleteAlarm.
func TestWranglerCompat_DOALARM003_DeleteAlarmRemovesAlarm(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("delete-alarm-test"));
			await stub.storage.setAlarm(Date.now() + 60000);
			await stub.storage.deleteAlarm();
			const alarm = await stub.storage.getAlarm();
			event.respondWith(new Response(alarm === null ? "deleted" : "still-exists"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ALARM-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "deleted" {
		t.Errorf("DO-ALARM-003: Expected 'deleted', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOALARM004_SetAlarmOverwritesPrevious tests overwrite.
func TestWranglerCompat_DOALARM004_SetAlarmOverwritesPrevious(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("overwrite-alarm-test"));
			const time1 = Date.now() + 60000;
			const time2 = Date.now() + 120000;
			await stub.storage.setAlarm(time1);
			await stub.storage.setAlarm(time2);
			const alarm = await stub.storage.getAlarm();
			// Should be closer to time2
			const diff = Math.abs(alarm - time2);
			event.respondWith(new Response(diff < 1000 ? "overwritten" : "not-overwritten:" + diff));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ALARM-004: Execute failed: %v", err)
	}

	if string(resp.Body) != "overwritten" {
		t.Errorf("DO-ALARM-004: Expected 'overwritten', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOALARM005_AlarmMethodsReturnPromises tests promises.
func TestWranglerCompat_DOALARM005_AlarmMethodsReturnPromises(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			const getP = stub.storage.getAlarm();
			const setP = stub.storage.setAlarm(Date.now() + 60000);
			const deleteP = stub.storage.deleteAlarm();

			const results = {
				getAlarm: getP instanceof Promise,
				setAlarm: setP instanceof Promise,
				deleteAlarm: deleteP instanceof Promise
			};

			await Promise.all([getP, setP, deleteP]);
			event.respondWith(Response.json(results));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ALARM-005: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("DO-ALARM-005: Failed to parse response: %v", err)
	}

	methods := []string{"getAlarm", "setAlarm", "deleteAlarm"}
	for _, method := range methods {
		if result[method] != true {
			t.Errorf("DO-ALARM-005: Expected %s to return Promise", method)
		}
	}
}

// ===========================================================================
// 6. Instance Isolation Tests
// ===========================================================================

// TestWranglerCompat_DOISO001_DifferentNamesHaveIsolatedStorage tests isolation.
func TestWranglerCompat_DOISO001_DifferentNamesHaveIsolatedStorage(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub1 = DO.get(DO.idFromName("object-a"));
			const stub2 = DO.get(DO.idFromName("object-b"));

			await stub1.storage.put("key", "value-from-a");
			await stub2.storage.put("key", "value-from-b");

			const v1 = await stub1.storage.get("key");
			const v2 = await stub2.storage.get("key");

			const isolated = v1 === "value-from-a" && v2 === "value-from-b";
			event.respondWith(new Response(isolated ? "isolated" : "not-isolated:" + v1 + ":" + v2));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ISO-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "isolated" {
		t.Errorf("DO-ISO-001: Expected 'isolated', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOISO002_DeleteAllOnlyAffectsOwnStorage tests deleteAll isolation.
func TestWranglerCompat_DOISO002_DeleteAllOnlyAffectsOwnStorage(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub1 = DO.get(DO.idFromName("object-a"));
			const stub2 = DO.get(DO.idFromName("object-b"));

			await stub1.storage.put("key", "value-a");
			await stub2.storage.put("key", "value-b");

			await stub1.storage.deleteAll();

			const v1 = await stub1.storage.get("key");
			const v2 = await stub2.storage.get("key");

			const isolated = v1 === undefined && v2 === "value-b";
			event.respondWith(new Response(isolated ? "isolated" : "not-isolated:" + v1 + ":" + v2));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ISO-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "isolated" {
		t.Errorf("DO-ISO-002: Expected 'isolated', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOISO003_SameNameSameStorage tests same name shares storage.
func TestWranglerCompat_DOISO003_SameNameSameStorage(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub1 = DO.get(DO.idFromName("shared"));
			const stub2 = DO.get(DO.idFromName("shared"));

			await stub1.storage.put("counter", 1);
			const value1 = await stub2.storage.get("counter");

			await stub2.storage.put("counter", 2);
			const value2 = await stub1.storage.get("counter");

			const shared = value1 === 1 && value2 === 2;
			event.respondWith(new Response(shared ? "shared" : "not-shared:" + value1 + ":" + value2));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ISO-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "shared" {
		t.Errorf("DO-ISO-003: Expected 'shared', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOISO004_UniqueIdsAreIsolated tests unique IDs isolation.
func TestWranglerCompat_DOISO004_UniqueIdsAreIsolated(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id1 = DO.newUniqueId();
			const id2 = DO.newUniqueId();
			const stub1 = DO.get(id1);
			const stub2 = DO.get(id2);

			await stub1.storage.put("data", "first");
			await stub2.storage.put("data", "second");

			const v1 = await stub1.storage.get("data");
			const v2 = await stub2.storage.get("data");

			const isolated = v1 === "first" && v2 === "second";
			event.respondWith(new Response(isolated ? "isolated" : "not-isolated:" + v1 + ":" + v2));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ISO-004: Execute failed: %v", err)
	}

	if string(resp.Body) != "isolated" {
		t.Errorf("DO-ISO-004: Expected 'isolated', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 6. Persistence Tests
// ===========================================================================

// TestWranglerCompat_DOPERS001_StoragePersistsAcrossGets tests persistence.
func TestWranglerCompat_DOPERS001_StoragePersistsAcrossGets(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const id = DO.idFromName("persistent");

			// First access
			const stub1 = DO.get(id);
			await stub1.storage.put("persistent-key", "persistent-value");

			// Second access (new stub)
			const stub2 = DO.get(id);
			const value = await stub2.storage.get("persistent-key");

			event.respondWith(new Response(value === "persistent-value" ? "persistent" : "not-persistent:" + value));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-PERS-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "persistent" {
		t.Errorf("DO-PERS-001: Expected 'persistent', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 7. Error Handling Tests
// ===========================================================================

// TestWranglerCompat_DOERR001_GetWithEmptyKey tests empty key.
func TestWranglerCompat_DOERR001_GetWithEmptyKey(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			try {
				await stub.storage.put("", "empty-key");
				const value = await stub.storage.get("");
				event.respondWith(new Response(value === "empty-key" ? "works" : "wrong-value"));
			} catch (e) {
				event.respondWith(new Response("error:" + e.message));
			}
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-ERR-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "works" {
		t.Errorf("DO-ERR-001: Expected 'works', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 8. Complete Workflow Tests
// ===========================================================================

// TestWranglerCompat_DOWF001_CounterWorkflow tests counter pattern.
func TestWranglerCompat_DOWF001_CounterWorkflow(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("counter"));
			await stub.storage.deleteAll();

			// Increment counter
			let count = (await stub.storage.get("count")) || 0;
			count++;
			await stub.storage.put("count", count);
			const count1 = count;

			// Increment again
			count = (await stub.storage.get("count")) || 0;
			count++;
			await stub.storage.put("count", count);
			const count2 = count;

			event.respondWith(new Response(count1 + ":" + count2));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-WF-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "1:2" {
		t.Errorf("DO-WF-001: Expected '1:2', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOWF002_SessionManagementWorkflow tests session pattern.
func TestWranglerCompat_DOWF002_SessionManagementWorkflow(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("session:user-123"));

			// Create session
			const session = {
				userId: "user-123",
				createdAt: Date.now(),
				data: { theme: "dark", language: "en" }
			};
			await stub.storage.put("session", session);

			// Read session
			const retrieved = await stub.storage.get("session");
			const step1 = retrieved.userId === "user-123" && retrieved.data.theme === "dark";

			// Update session
			retrieved.data.theme = "light";
			await stub.storage.put("session", retrieved);

			// Verify update
			const updated = await stub.storage.get("session");
			const step2 = updated.data.theme === "light";

			// Delete session
			await stub.storage.deleteAll();
			const afterDelete = await stub.storage.get("session");
			const step3 = afterDelete === undefined;

			const result = step1 && step2 && step3;
			event.respondWith(new Response(result ? "workflow-complete" : "workflow-failed"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-WF-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "workflow-complete" {
		t.Errorf("DO-WF-002: Expected 'workflow-complete', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_DOWF003_ChatRoomWorkflow tests chat room pattern.
func TestWranglerCompat_DOWF003_ChatRoomWorkflow(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("room:general"));
			await stub.storage.deleteAll();

			// Add messages
			const messages = [
				{ id: 1, user: "alice", text: "Hello!" },
				{ id: 2, user: "bob", text: "Hi Alice!" },
				{ id: 3, user: "alice", text: "How are you?" }
			];

			for (const msg of messages) {
				await stub.storage.put("msg:" + msg.id, msg);
			}

			// List all messages using Map-like API
			const entries = await stub.storage.list({ prefix: "msg:" });
			const msgCount = entries.size;

			event.respondWith(new Response(msgCount === 3 ? "chat-works" : "chat-failed:" + msgCount));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-WF-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "chat-works" {
		t.Errorf("DO-WF-003: Expected 'chat-works', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 9. Namespace Isolation Tests
// ===========================================================================

// TestWranglerCompat_DONSISO001_MultipleNamespacesIsolated tests namespace isolation.
func TestWranglerCompat_DONSISO001_MultipleNamespacesIsolated(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "do-ns-test-*")
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
	s.DurableObjects().CreateNamespace(ctx, &store.DurableObjectNamespace{
		ID:        ns1ID,
		Name:      "ns1",
		Script:    "script",
		ClassName: "TestDO",
		CreatedAt: time.Now(),
	})
	s.DurableObjects().CreateNamespace(ctx, &store.DurableObjectNamespace{
		ID:        ns2ID,
		Name:      "ns2",
		Script:    "script",
		ClassName: "TestDO",
		CreatedAt: time.Now(),
	})

	// Test with NS1
	rt1 := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + ns1ID},
	})
	defer rt1.Close()

	script1 := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("shared-name"));
			await stub.storage.put("key", "value-from-ns1");
			event.respondWith(new Response("ns1-done"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp1, err := rt1.Execute(ctx, script1, req)
	if err != nil {
		t.Fatalf("NS1 execute failed: %v", err)
	}
	if string(resp1.Body) != "ns1-done" {
		t.Errorf("Expected 'ns1-done', got '%s'", string(resp1.Body))
	}

	// Test with NS2 - same object name but different namespace
	rt2 := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + ns2ID},
	})
	defer rt2.Close()

	script2 := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("shared-name"));
			await stub.storage.put("key", "value-from-ns2");
			const value = await stub.storage.get("key");
			event.respondWith(new Response(value));
		});
	`

	resp2, err := rt2.Execute(ctx, script2, req)
	if err != nil {
		t.Fatalf("NS2 execute failed: %v", err)
	}
	if string(resp2.Body) != "value-from-ns2" {
		t.Errorf("Expected 'value-from-ns2', got '%s' (namespaces may not be isolated)", string(resp2.Body))
	}

	// Verify NS1 still has its value
	script1Check := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("shared-name"));
			const value = await stub.storage.get("key");
			event.respondWith(new Response(value));
		});
	`

	resp1Check, err := rt1.Execute(ctx, script1Check, req)
	if err != nil {
		t.Fatalf("NS1 check execute failed: %v", err)
	}
	if string(resp1Check.Body) != "value-from-ns1" {
		t.Errorf("Expected 'value-from-ns1', got '%s' (namespaces not isolated)", string(resp1Check.Body))
	}
}

// ===========================================================================
// 10. Concurrent Access Tests
// ===========================================================================

// TestWranglerCompat_DOCONC001_ConcurrentPuts tests concurrent writes.
func TestWranglerCompat_DOCONC001_ConcurrentPuts(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("concurrent"));
			await stub.storage.deleteAll();

			// Perform concurrent puts
			const promises = [];
			for (let i = 0; i < 10; i++) {
				promises.push(stub.storage.put("key" + i, "value" + i));
			}
			await Promise.all(promises);

			// Verify all values
			let success = true;
			for (let i = 0; i < 10; i++) {
				const value = await stub.storage.get("key" + i);
				if (value !== "value" + i) {
					success = false;
					break;
				}
			}

			event.respondWith(new Response(success ? "concurrent-success" : "concurrent-failed"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DO-CONC-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "concurrent-success" {
		t.Errorf("DO-CONC-001: Expected 'concurrent-success', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// Benchmarks
// ===========================================================================

// BenchmarkDOGet benchmarks the get operation.
func BenchmarkDOGet(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "do-bench-*")
	defer os.RemoveAll(tmpDir)

	s, _ := sqlite.New(tmpDir)
	defer s.Close()
	s.Ensure(context.Background())

	nsID := "bench-ns"
	s.DurableObjects().CreateNamespace(context.Background(), &store.DurableObjectNamespace{
		ID:        nsID,
		Name:      "bench",
		Script:    "script",
		ClassName: "Bench",
		CreatedAt: time.Now(),
	})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	// Setup: put a value
	setupScript := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("bench"));
			await stub.storage.put("bench-key", "bench-value");
			event.respondWith(new Response("ok"));
		});
	`
	req := httptest.NewRequest("GET", "/", nil)
	rt.Execute(context.Background(), setupScript, req)

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("bench"));
			const value = await stub.storage.get("bench-key");
			event.respondWith(new Response(value));
		});
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.Execute(context.Background(), script, req)
	}
}

// BenchmarkDOPut benchmarks the put operation.
func BenchmarkDOPut(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "do-bench-*")
	defer os.RemoveAll(tmpDir)

	s, _ := sqlite.New(tmpDir)
	defer s.Close()
	s.Ensure(context.Background())

	nsID := "bench-ns"
	s.DurableObjects().CreateNamespace(context.Background(), &store.DurableObjectNamespace{
		ID:        nsID,
		Name:      "bench",
		Script:    "script",
		ClassName: "Bench",
		CreatedAt: time.Now(),
	})

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("bench"));
			await stub.storage.put("bench-key-" + Math.random(), "value");
			event.respondWith(new Response("ok"));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.Execute(context.Background(), script, req)
	}
}

// ===========================================================================
// Helper to test specific wrangler behaviors
// ===========================================================================

// TestWranglerCompat_DOSpecialCases tests edge cases and special behaviors.
func TestWranglerCompat_DOSpecialCases(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	tests := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name: "nested objects",
			script: `
				addEventListener('fetch', async event => {
					const stub = DO.get(DO.idFromName("test"));
					const deep = { a: { b: { c: { d: { e: 5 } } } } };
					await stub.storage.put("deep", deep);
					const value = await stub.storage.get("deep");
					event.respondWith(new Response(value.a.b.c.d.e));
				});
			`,
			expected: "5",
		},
		{
			name: "array of objects",
			script: `
				addEventListener('fetch', async event => {
					const stub = DO.get(DO.idFromName("test"));
					const arr = [{ x: 1 }, { x: 2 }, { x: 3 }];
					await stub.storage.put("arr", arr);
					const value = await stub.storage.get("arr");
					event.respondWith(new Response(value.map(o => o.x).join(",")));
				});
			`,
			expected: "1,2,3",
		},
		{
			name: "special characters in key",
			script: `
				addEventListener('fetch', async event => {
					const stub = DO.get(DO.idFromName("test"));
					const key = "key/with/slashes:and:colons?and=queries";
					await stub.storage.put(key, "special");
					const value = await stub.storage.get(key);
					event.respondWith(new Response(value));
				});
			`,
			expected: "special",
		},
		{
			name: "consecutive operations",
			script: `
				addEventListener('fetch', async event => {
					const stub = DO.get(DO.idFromName("test"));
					await stub.storage.put("seq", 1);
					await stub.storage.put("seq", 2);
					await stub.storage.put("seq", 3);
					const value = await stub.storage.get("seq");
					event.respondWith(new Response(value));
				});
			`,
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			resp, err := rt.Execute(context.Background(), tt.script, req)
			if err != nil {
				t.Fatalf("%s: Execute failed: %v", tt.name, err)
			}
			if string(resp.Body) != tt.expected {
				t.Errorf("%s: Expected '%s', got '%s'", tt.name, tt.expected, string(resp.Body))
			}
		})
	}
}

// TestWranglerCompat_DOStubFetchResponse tests fetch response.
func TestWranglerCompat_DOStubFetchResponse(t *testing.T) {
	s, nsID, cleanup := setupDOTest(t)
	defer cleanup()

	rt := New(Config{
		Store:    s,
		Bindings: map[string]string{"DO": "do:" + nsID},
	})
	defer rt.Close()

	script := `
		addEventListener('fetch', async event => {
			const stub = DO.get(DO.idFromName("test"));
			const response = await stub.fetch(new Request('http://example.com'));
			const results = [];
			results.push("status:" + response.status);
			results.push("ok:" + response.ok);
			event.respondWith(new Response(results.join(",")));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "status:200") {
		t.Errorf("Expected status:200 in response, got '%s'", body)
	}
	if !strings.Contains(body, "ok:true") {
		t.Errorf("Expected ok:true in response, got '%s'", body)
	}
}
