package runtime

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockStore implements store.Store for testing
type mockStore struct{}

func (m *mockStore) KV() interface{}             { return nil }
func (m *mockStore) R2() interface{}             { return nil }
func (m *mockStore) D1() interface{}             { return nil }
func (m *mockStore) DurableObjects() interface{} { return nil }
func (m *mockStore) Queues() interface{}         { return nil }
func (m *mockStore) AI() interface{}             { return nil }
func (m *mockStore) Workers() interface{}        { return nil }
func (m *mockStore) Cache() interface{}          { return nil }
func (m *mockStore) Vectorize() interface{}      { return nil }
func (m *mockStore) Hyperdrive() interface{}     { return nil }
func (m *mockStore) AnalyticsEngine() interface{} { return nil }

// TestRuntime_HelloWorld tests basic worker execution
func TestRuntime_HelloWorld(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response('Hello World'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}

	if string(resp.Body) != "Hello World" {
		t.Errorf("Expected body 'Hello World', got '%s'", string(resp.Body))
	}
}

// TestRuntime_ResponseJSON tests Response.json() static method
func TestRuntime_ResponseJSON(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(Response.json({ message: 'Hello', count: 42 }));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}

	contentType := resp.Headers.Get("content-type")
	if contentType != "application/json" {
		t.Errorf("Expected content-type 'application/json', got '%s'", contentType)
	}

	expectedBody := `{"count":42,"message":"Hello"}`
	body := strings.TrimSpace(string(resp.Body))
	if body != expectedBody && body != `{"message":"Hello","count":42}` {
		t.Errorf("Expected JSON body, got '%s'", body)
	}
}

// TestRuntime_ResponseRedirect tests Response.redirect() static method
func TestRuntime_ResponseRedirect(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(Response.redirect('https://example.com', 302));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Status != 302 {
		t.Errorf("Expected status 302, got %d", resp.Status)
	}

	location := resp.Headers.Get("location")
	if location != "https://example.com" {
		t.Errorf("Expected location 'https://example.com', got '%s'", location)
	}
}

// TestRuntime_RequestHeaders tests accessing request headers
func TestRuntime_RequestHeaders(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const auth = event.request.headers.get('authorization');
			event.respondWith(new Response(auth || 'no auth'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token123")
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "Bearer token123" {
		t.Errorf("Expected 'Bearer token123', got '%s'", string(resp.Body))
	}
}

// TestRuntime_RequestMethod tests accessing request method
func TestRuntime_RequestMethod(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response(event.request.method));
		});
	`

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/", nil)
		resp, err := rt.Execute(context.Background(), script, req)

		if err != nil {
			t.Fatalf("Execute failed for %s: %v", method, err)
		}

		if string(resp.Body) != method {
			t.Errorf("Expected '%s', got '%s'", method, string(resp.Body))
		}
	}
}

// TestRuntime_RequestURL tests accessing request URL
func TestRuntime_RequestURL(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const url = new URL(event.request.url);
			event.respondWith(new Response(url.pathname));
		});
	`

	req := httptest.NewRequest("GET", "/api/users?page=1", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "/api/users" {
		t.Errorf("Expected '/api/users', got '%s'", string(resp.Body))
	}
}

// TestRuntime_URLSearchParams tests URL search params
func TestRuntime_URLSearchParams(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const url = new URL(event.request.url);
			const page = url.searchParams.get('page');
			const limit = url.searchParams.get('limit');
			event.respondWith(new Response(page + ':' + limit));
		});
	`

	req := httptest.NewRequest("GET", "/api/users?page=2&limit=10", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "2:10" {
		t.Errorf("Expected '2:10', got '%s'", string(resp.Body))
	}
}

// TestRuntime_ResponseStatus tests setting response status
func TestRuntime_ResponseStatus(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	testCases := []struct {
		status int
		ok     bool
	}{
		{200, true},
		{201, true},
		{204, true},
		{301, false},
		{400, false},
		{404, false},
		{500, false},
	}

	for _, tc := range testCases {
		script := `
			addEventListener('fetch', event => {
				event.respondWith(new Response('', { status: ` + string(rune(tc.status+'0')) + ` }));
			});
		`
		// Use format instead
		script = strings.Replace(`
			addEventListener('fetch', event => {
				event.respondWith(new Response('', { status: STATUS }));
			});
		`, "STATUS", string(rune(48+tc.status/100))+string(rune(48+(tc.status%100)/10))+string(rune(48+tc.status%10)), 1)

		req := httptest.NewRequest("GET", "/", nil)
		resp, err := rt.Execute(context.Background(), script, req)

		if err != nil {
			t.Fatalf("Execute failed for status %d: %v", tc.status, err)
		}

		if resp.Status != tc.status {
			t.Errorf("Expected status %d, got %d", tc.status, resp.Status)
		}
	}
}

// TestRuntime_ResponseHeaders tests setting response headers
func TestRuntime_ResponseHeaders(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response('body', {
				headers: {
					'Content-Type': 'text/plain',
					'X-Custom-Header': 'custom-value'
				}
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	ct := resp.Headers.Get("content-type")
	if ct != "text/plain" {
		t.Errorf("Expected content-type 'text/plain', got '%s'", ct)
	}

	custom := resp.Headers.Get("x-custom-header")
	if custom != "custom-value" {
		t.Errorf("Expected x-custom-header 'custom-value', got '%s'", custom)
	}
}

// TestRuntime_Headers tests Headers API
func TestRuntime_Headers(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const headers = new Headers();
			headers.set('Content-Type', 'application/json');
			headers.append('Accept', 'text/html');
			headers.append('Accept', 'application/json');

			const hasContentType = headers.has('content-type');
			const accept = headers.get('accept');

			headers.delete('accept');
			const hasAccept = headers.has('accept');

			event.respondWith(new Response(hasContentType + ':' + accept + ':' + hasAccept));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Headers append joins with ", "
	expected := "true:text/html, application/json:false"
	if string(resp.Body) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(resp.Body))
	}
}

// TestRuntime_TextEncoder tests TextEncoder API
func TestRuntime_TextEncoder(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const encoder = new TextEncoder();
			const bytes = encoder.encode('hello');
			event.respondWith(new Response(bytes.length.toString()));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "5" {
		t.Errorf("Expected '5', got '%s'", string(resp.Body))
	}
}

// TestRuntime_TextDecoder tests TextDecoder API
func TestRuntime_TextDecoder(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const encoder = new TextEncoder();
			const decoder = new TextDecoder();
			const bytes = encoder.encode('hello');
			const text = decoder.decode(bytes);
			event.respondWith(new Response(text));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", string(resp.Body))
	}
}

// TestRuntime_Base64 tests atob/btoa
func TestRuntime_Base64(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const encoded = btoa('hello world');
			const decoded = atob(encoded);
			event.respondWith(new Response(encoded + ':' + decoded));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expected := "aGVsbG8gd29ybGQ=:hello world"
	if string(resp.Body) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(resp.Body))
	}
}

// TestRuntime_CryptoRandomUUID tests crypto.randomUUID()
func TestRuntime_CryptoRandomUUID(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const uuid = crypto.randomUUID();
			// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
			const valid = uuid.length === 36 && uuid.split('-').length === 5;
			event.respondWith(new Response(valid.toString()));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true" {
		t.Errorf("Expected valid UUID, got invalid format")
	}
}

// TestRuntime_StructuredClone tests structuredClone()
func TestRuntime_StructuredClone(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const obj = { a: 1, b: { c: 2 } };
			const clone = structuredClone(obj);
			clone.b.c = 3;
			// Original should not be affected
			event.respondWith(new Response(obj.b.c.toString()));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "2" {
		t.Errorf("Expected '2', got '%s'", string(resp.Body))
	}
}

// TestRuntime_PerformanceNow tests performance.now()
func TestRuntime_PerformanceNow(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const start = performance.now();
			// Some work
			for (let i = 0; i < 1000; i++) {}
			const end = performance.now();
			const elapsed = end - start;
			event.respondWith(new Response((elapsed >= 0).toString()));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true" {
		t.Errorf("Expected positive elapsed time")
	}
}

// TestRuntime_WaitUntil tests event.waitUntil()
func TestRuntime_WaitUntil(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			let completed = false;
			event.waitUntil(new Promise((resolve) => {
				completed = true;
				resolve();
			}));
			event.respondWith(new Response('done'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "done" {
		t.Errorf("Expected 'done', got '%s'", string(resp.Body))
	}
}

// TestRuntime_PassThroughOnException tests event.passThroughOnException()
func TestRuntime_PassThroughOnException(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.passThroughOnException();
			event.respondWith(new Response('success'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "success" {
		t.Errorf("Expected 'success', got '%s'", string(resp.Body))
	}
}

// TestRuntime_CFObject tests request.cf object
func TestRuntime_CFObject(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const cf = event.request.cf;
			const colo = cf.colo;
			const country = cf.country;
			event.respondWith(new Response(colo + ':' + country));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "LOCAL:XX" {
		t.Errorf("Expected 'LOCAL:XX', got '%s'", string(resp.Body))
	}
}

// TestRuntime_Timeout tests execution timeout
func TestRuntime_Timeout(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			// This should complete quickly
			event.respondWith(new Response('done'));
		});
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(ctx, script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "done" {
		t.Errorf("Expected 'done', got '%s'", string(resp.Body))
	}
}

// TestRuntime_NoHandler tests error when no fetch handler is registered
func TestRuntime_NoHandler(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		// No fetch handler registered
		console.log('No handler');
	`

	req := httptest.NewRequest("GET", "/", nil)
	_, err := rt.Execute(context.Background(), script, req)

	if err == nil {
		t.Error("Expected error for missing fetch handler")
	}

	if !strings.Contains(err.Error(), "no fetch handler") {
		t.Errorf("Expected 'no fetch handler' error, got: %v", err)
	}
}

// TestRuntime_ScriptError tests script compilation error
func TestRuntime_ScriptError(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		this is not valid javascript { } ( ]
	`

	req := httptest.NewRequest("GET", "/", nil)
	_, err := rt.Execute(context.Background(), script, req)

	if err == nil {
		t.Error("Expected error for invalid script")
	}

	if !strings.Contains(err.Error(), "script error") {
		t.Errorf("Expected 'script error', got: %v", err)
	}
}

// TestPool_ConcurrentExecution tests concurrent execution in pool
func TestPool_ConcurrentExecution(t *testing.T) {
	pool := NewPool(PoolConfig{PoolSize: 5})
	defer pool.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response('concurrent'));
		});
	`

	pool.CacheScript("test", script)

	// Execute concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/", nil)
			_, err := pool.ExecuteCached(context.Background(), "test", req, nil)
			done <- (err == nil)
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		if !<-done {
			t.Error("Concurrent execution failed")
		}
	}
}

// TestPool_Stats tests pool statistics
func TestPool_Stats(t *testing.T) {
	pool := NewPool(PoolConfig{PoolSize: 5})
	defer pool.Close()

	stats := pool.Stats()

	if stats.PoolSize != 5 {
		t.Errorf("Expected pool size 5, got %d", stats.PoolSize)
	}

	if stats.Available != 5 {
		t.Errorf("Expected 5 available, got %d", stats.Available)
	}

	pool.CacheScript("test", "")
	stats = pool.Stats()

	if stats.CachedScripts != 1 {
		t.Errorf("Expected 1 cached script, got %d", stats.CachedScripts)
	}
}

// TestPool_InvalidateScript tests script cache invalidation
func TestPool_InvalidateScript(t *testing.T) {
	pool := NewPool(PoolConfig{PoolSize: 1})
	defer pool.Close()

	pool.CacheScript("test", "script")
	stats := pool.Stats()
	if stats.CachedScripts != 1 {
		t.Errorf("Expected 1 cached script, got %d", stats.CachedScripts)
	}

	pool.InvalidateScript("test")
	stats = pool.Stats()
	if stats.CachedScripts != 0 {
		t.Errorf("Expected 0 cached scripts after invalidation, got %d", stats.CachedScripts)
	}
}

// TestExecutionContext_WaitUntil tests execution context
func TestExecutionContext_WaitUntil(t *testing.T) {
	ctx := NewExecutionContext(30 * time.Second)

	// Add promises
	ctx.AddWaitUntil(nil)
	ctx.AddWaitUntil(nil)

	promises := ctx.WaitUntilPromises()
	if len(promises) != 2 {
		t.Errorf("Expected 2 promises, got %d", len(promises))
	}
}

// TestExecutionContext_PassThrough tests pass-through functionality
func TestExecutionContext_PassThrough(t *testing.T) {
	ctx := NewExecutionContext(30 * time.Second)

	if ctx.ShouldPassThrough() {
		t.Error("Expected ShouldPassThrough to be false initially")
	}

	ctx.SetPassThroughOnException()

	if !ctx.ShouldPassThrough() {
		t.Error("Expected ShouldPassThrough to be true after setting")
	}
}

// TestExecutionContext_Cancel tests context cancellation
func TestExecutionContext_Cancel(t *testing.T) {
	ctx := NewExecutionContext(30 * time.Second)

	if ctx.IsCancelled() {
		t.Error("Expected IsCancelled to be false initially")
	}

	ctx.Cancel()

	if !ctx.IsCancelled() {
		t.Error("Expected IsCancelled to be true after cancel")
	}
}

// Benchmark for basic execution
func BenchmarkRuntime_HelloWorld(b *testing.B) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response('Hello World'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.Execute(context.Background(), script, req)
	}
}

// Benchmark for pool execution
func BenchmarkPool_Execution(b *testing.B) {
	pool := NewPool(PoolConfig{PoolSize: 10})
	defer pool.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response('Hello World'));
		});
	`

	pool.CacheScript("test", script)
	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.ExecuteCached(context.Background(), "test", req, nil)
	}
}
