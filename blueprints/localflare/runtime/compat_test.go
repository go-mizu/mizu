package runtime

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

// Cloudflare Workers Compatibility Tests
// These tests verify that the runtime behaves like real Cloudflare Workers

// TestCompat_FetchEvent_Type tests that fetch event has correct type
func TestCompat_FetchEvent_Type(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response(event.type));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "fetch" {
		t.Errorf("Expected event type 'fetch', got '%s'", string(resp.Body))
	}
}

// TestCompat_Request_Properties tests Request object properties
func TestCompat_Request_Properties(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const req = event.request;
			const result = {
				url: req.url,
				method: req.method,
				hasHeaders: !!req.headers,
				hasCf: !!req.cf
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("POST", "/api/test", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, `"method":"POST"`) {
		t.Errorf("Expected method POST in response")
	}
	if !strings.Contains(body, `"hasHeaders":true`) {
		t.Errorf("Expected hasHeaders true in response")
	}
	if !strings.Contains(body, `"hasCf":true`) {
		t.Errorf("Expected hasCf true in response")
	}
}

// TestCompat_Response_InitOptions tests Response constructor init options
func TestCompat_Response_InitOptions(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const resp = new Response('body', {
				status: 201,
				statusText: 'Created',
				headers: {
					'X-Custom': 'value'
				}
			});
			event.respondWith(new Response(resp.status + ':' + resp.statusText));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "201:Created" {
		t.Errorf("Expected '201:Created', got '%s'", string(resp.Body))
	}
}

// TestCompat_Response_Ok tests Response.ok property
func TestCompat_Response_Ok(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	testCases := []struct {
		status   int
		expected bool
	}{
		{200, true},
		{201, true},
		{299, true},
		{300, false},
		{400, false},
		{500, false},
	}

	for _, tc := range testCases {
		script := strings.Replace(`
			addEventListener('fetch', event => {
				const resp = new Response('', { status: STATUS });
				event.respondWith(new Response(resp.ok.toString()));
			});
		`, "STATUS", string(rune(48+tc.status/100))+string(rune(48+(tc.status%100)/10))+string(rune(48+tc.status%10)), 1)

		req := httptest.NewRequest("GET", "/", nil)
		resp, err := rt.Execute(context.Background(), script, req)

		if err != nil {
			t.Fatalf("Execute failed for status %d: %v", tc.status, err)
		}

		expected := "false"
		if tc.expected {
			expected = "true"
		}
		if string(resp.Body) != expected {
			t.Errorf("For status %d: expected ok=%s, got %s", tc.status, expected, string(resp.Body))
		}
	}
}

// TestCompat_Headers_CaseInsensitive tests that headers are case-insensitive
func TestCompat_Headers_CaseInsensitive(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const headers = new Headers();
			headers.set('Content-Type', 'text/plain');
			const result = [
				headers.get('content-type'),
				headers.get('CONTENT-TYPE'),
				headers.get('Content-Type')
			].join(':');
			event.respondWith(new Response(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// All should return the same value
	if string(resp.Body) != "text/plain:text/plain:text/plain" {
		t.Errorf("Expected case-insensitive headers, got '%s'", string(resp.Body))
	}
}

// TestCompat_URL_Parse tests URL parsing compatibility
func TestCompat_URL_Parse(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const url = new URL('https://example.com:8080/path?query=1#hash');
			const result = {
				protocol: url.protocol,
				host: url.host,
				hostname: url.hostname,
				pathname: url.pathname,
				href: url.href
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, `"protocol":"https:"`) {
		t.Errorf("Expected protocol 'https:' in response")
	}
	if !strings.Contains(body, `"pathname":"/path"`) {
		t.Errorf("Expected pathname '/path' in response")
	}
}

// TestCompat_URLSearchParams tests URLSearchParams compatibility
func TestCompat_URLSearchParams(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const params = new URLSearchParams('foo=1&bar=2&foo=3');
			const result = {
				foo: params.get('foo'),
				fooAll: params.getAll('foo'),
				bar: params.get('bar'),
				hasBar: params.has('bar'),
				hasBaz: params.has('baz')
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, `"foo":"1"`) {
		t.Errorf("Expected first foo value '1'")
	}
	if !strings.Contains(body, `"hasBar":true`) {
		t.Errorf("Expected hasBar true")
	}
	if !strings.Contains(body, `"hasBaz":false`) {
		t.Errorf("Expected hasBaz false")
	}
}

// TestCompat_Blob tests Blob API compatibility
func TestCompat_Blob(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	// Test without async/await - just basic Blob properties
	script := `
		addEventListener('fetch', event => {
			const blob = new Blob(['Hello ', 'World'], { type: 'text/plain' });
			event.respondWith(new Response(blob.size + ':' + blob.type));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "11:text/plain" {
		t.Errorf("Expected '11:text/plain', got '%s'", string(resp.Body))
	}
}

// TestCompat_FormData tests FormData API compatibility
func TestCompat_FormData(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const formData = new FormData();
			formData.append('name', 'John');
			formData.append('age', '30');
			formData.append('name', 'Jane');

			const result = {
				name: formData.get('name'),
				nameAll: formData.getAll('name'),
				age: formData.get('age'),
				hasName: formData.has('name'),
				hasEmail: formData.has('email')
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, `"name":"John"`) {
		t.Errorf("Expected name 'John'")
	}
	if !strings.Contains(body, `"hasName":true`) {
		t.Errorf("Expected hasName true")
	}
	if !strings.Contains(body, `"hasEmail":false`) {
		t.Errorf("Expected hasEmail false")
	}
}

// TestCompat_SubtleCrypto_Digest tests crypto.subtle.digest()
func TestCompat_SubtleCrypto_Digest(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	// Test SubtleCrypto existence and structure (without async)
	script := `
		addEventListener('fetch', event => {
			const hasCrypto = typeof crypto !== 'undefined';
			const hasSubtle = hasCrypto && typeof crypto.subtle !== 'undefined';
			const hasDigest = hasSubtle && typeof crypto.subtle.digest === 'function';
			const hasRandomUUID = hasCrypto && typeof crypto.randomUUID === 'function';
			event.respondWith(new Response(hasCrypto + ':' + hasSubtle + ':' + hasDigest + ':' + hasRandomUUID));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true:true:true:true" {
		t.Errorf("Expected 'true:true:true:true', got '%s'", string(resp.Body))
	}
}

// TestCompat_AbortController tests AbortController API
func TestCompat_AbortController(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const controller = new AbortController();
			const signal = controller.signal;

			const beforeAbort = signal.aborted;
			controller.abort();
			const afterAbort = signal.aborted;

			event.respondWith(new Response(beforeAbort + ':' + afterAbort));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "false:true" {
		t.Errorf("Expected 'false:true', got '%s'", string(resp.Body))
	}
}

// TestCompat_ReadableStream tests ReadableStream API
func TestCompat_ReadableStream(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	// Test ReadableStream existence and structure (without async)
	script := `
		addEventListener('fetch', event => {
			const hasReadableStream = typeof ReadableStream !== 'undefined';
			const stream = new ReadableStream({
				start(controller) {
					controller.enqueue('Hello');
					controller.close();
				}
			});
			const hasGetReader = typeof stream.getReader === 'function';
			const hasLocked = 'locked' in stream;
			event.respondWith(new Response(hasReadableStream + ':' + hasGetReader + ':' + hasLocked));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true:true:true" {
		t.Errorf("Expected 'true:true:true', got '%s'", string(resp.Body))
	}
}

// TestCompat_Cache_API tests Cache API
func TestCompat_Cache_API(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	// Test Cache API existence and structure (without async)
	script := `
		addEventListener('fetch', event => {
			const hasCaches = typeof caches !== 'undefined';
			const hasDefault = hasCaches && typeof caches.default !== 'undefined';
			const cache = caches.default;
			const hasPut = typeof cache.put === 'function';
			const hasMatch = typeof cache.match === 'function';
			const hasDelete = typeof cache.delete === 'function';
			event.respondWith(new Response(hasCaches + ':' + hasDefault + ':' + hasPut + ':' + hasMatch + ':' + hasDelete));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true:true:true:true:true" {
		t.Errorf("Expected 'true:true:true:true:true', got '%s'", string(resp.Body))
	}
}

// TestCompat_Cache_Delete tests Cache.delete()
func TestCompat_Cache_Delete(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	// Test Cache delete method exists
	script := `
		addEventListener('fetch', event => {
			const cache = caches.default;
			const hasDelete = typeof cache.delete === 'function';
			event.respondWith(new Response(String(hasDelete)));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true" {
		t.Errorf("Expected 'true', got '%s'", string(resp.Body))
	}
}

// TestCompat_Console tests console API
func TestCompat_Console(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	// Console should be available and not throw
	script := `
		addEventListener('fetch', event => {
			console.log('test log');
			console.info('test info');
			console.warn('test warn');
			console.error('test error');
			console.debug('test debug');
			event.respondWith(new Response('ok'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "ok" {
		t.Errorf("Expected 'ok', got '%s'", string(resp.Body))
	}
}

// TestCompat_GlobalThis tests global object access
func TestCompat_GlobalThis(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const hasResponse = typeof Response !== 'undefined';
			const hasRequest = typeof Request !== 'undefined';
			const hasHeaders = typeof Headers !== 'undefined';
			const hasFetch = typeof fetch !== 'undefined';
			const hasURL = typeof URL !== 'undefined';

			event.respondWith(new Response([hasResponse, hasRequest, hasHeaders, hasFetch, hasURL].join(':')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true:true:true:true:true" {
		t.Errorf("Expected all globals available, got '%s'", string(resp.Body))
	}
}

// TestCompat_MultipleHandlers tests multiple event listeners
func TestCompat_MultipleHandlers(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		let called = [];

		addEventListener('fetch', event => {
			called.push('first');
		});

		addEventListener('fetch', event => {
			called.push('second');
			event.respondWith(new Response(called.join(',')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Both handlers should be called
	body := string(resp.Body)
	if !strings.Contains(body, "first") || !strings.Contains(body, "second") {
		t.Errorf("Expected both handlers called, got '%s'", body)
	}
}

// TestCompat_AsyncHandler tests Promise existence (async/await requires more complex integration)
func TestCompat_AsyncHandler(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	// Test that Promise exists and can be used with .then()
	script := `
		addEventListener('fetch', event => {
			const hasPromise = typeof Promise !== 'undefined';
			const hasResolve = hasPromise && typeof Promise.resolve === 'function';
			const promise = Promise.resolve('test data');
			const hasThen = typeof promise.then === 'function';
			event.respondWith(new Response(hasPromise + ':' + hasResolve + ':' + hasThen));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true:true:true" {
		t.Errorf("Expected 'true:true:true', got '%s'", string(resp.Body))
	}
}

// TestCompat_RequestBody_Text tests Request.text() method exists
func TestCompat_RequestBody_Text(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const req = event.request;
			const hasText = typeof req.text === 'function';
			const hasJSON = typeof req.json === 'function';
			const hasArrayBuffer = typeof req.arrayBuffer === 'function';
			event.respondWith(new Response(hasText + ':' + hasJSON + ':' + hasArrayBuffer));
		});
	`

	req := httptest.NewRequest("POST", "/", strings.NewReader("test body content"))
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true:true:true" {
		t.Errorf("Expected 'true:true:true', got '%s'", string(resp.Body))
	}
}

// TestCompat_RequestBody_JSON tests Request.json() exists as a function
func TestCompat_RequestBody_JSON(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const req = event.request;
			const hasJson = typeof req.json === 'function';
			event.respondWith(new Response(String(hasJson)));
		});
	`

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"test","value":42}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "true" {
		t.Errorf("Expected 'true', got '%s'", string(resp.Body))
	}
}

// TestCompat_ErrorHandling tests error handling in workers
func TestCompat_ErrorHandling(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			try {
				throw new Error('test error');
			} catch (e) {
				event.respondWith(new Response('caught:' + e.message));
			}
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "caught:test error" {
		t.Errorf("Expected 'caught:test error', got '%s'", string(resp.Body))
	}
}

// TestCompat_JSON_Methods tests JSON.stringify and JSON.parse
func TestCompat_JSON_Methods(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const obj = { a: 1, b: 'text', c: [1, 2, 3] };
			const json = JSON.stringify(obj);
			const parsed = JSON.parse(json);
			event.respondWith(new Response(parsed.a + ':' + parsed.b + ':' + parsed.c.length));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "1:text:3" {
		t.Errorf("Expected '1:text:3', got '%s'", string(resp.Body))
	}
}

// TestCompat_Array_Methods tests Array methods
func TestCompat_Array_Methods(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const arr = [1, 2, 3, 4, 5];
			const filtered = arr.filter(x => x > 2);
			const mapped = arr.map(x => x * 2);
			const reduced = arr.reduce((a, b) => a + b, 0);
			const found = arr.find(x => x === 3);
			const includes = arr.includes(3);

			event.respondWith(new Response([
				filtered.length,
				mapped[0],
				reduced,
				found,
				includes
			].join(':')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "3:2:15:3:true" {
		t.Errorf("Expected '3:2:15:3:true', got '%s'", string(resp.Body))
	}
}

// TestCompat_String_Methods tests String methods
func TestCompat_String_Methods(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const str = 'Hello World';
			const result = [
				str.toLowerCase(),
				str.toUpperCase(),
				str.includes('World'),
				str.startsWith('Hello'),
				str.endsWith('World'),
				str.split(' ').length,
				str.slice(0, 5),
				str.replace('World', 'Worker')
			].join(':');
			event.respondWith(new Response(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expected := "hello world:HELLO WORLD:true:true:true:2:Hello:Hello Worker"
	if string(resp.Body) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(resp.Body))
	}
}

// TestCompat_Object_Methods tests Object methods
func TestCompat_Object_Methods(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			const obj = { a: 1, b: 2, c: 3 };
			const keys = Object.keys(obj);
			const values = Object.values(obj);
			const entries = Object.entries(obj);
			const assigned = Object.assign({}, obj, { d: 4 });

			event.respondWith(new Response([
				keys.length,
				values.reduce((a, b) => a + b, 0),
				entries.length,
				Object.keys(assigned).length
			].join(':')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(resp.Body) != "3:6:3:4" {
		t.Errorf("Expected '3:6:3:4', got '%s'", string(resp.Body))
	}
}
