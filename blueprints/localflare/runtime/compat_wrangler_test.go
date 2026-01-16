package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// Cloudflare Workers Wrangler Compatibility Tests
// These tests verify 100% compatibility with Cloudflare Workers (wrangler offline)
// Test IDs correspond to spec/0376_worker_compatible.md

// ===========================================================================
// 1. Request Object Tests
// ===========================================================================

// TestWranglerCompat_REQ001_RequestURL tests Request.url property
func TestWranglerCompat_REQ001_RequestURL(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response(event.request.url));
		});
	`

	req := httptest.NewRequest("GET", "https://example.com/api/users?page=1", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-001: Execute failed: %v", err)
	}

	if !strings.Contains(string(resp.Body), "/api/users") {
		t.Errorf("REQ-001: Expected URL to contain '/api/users', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_REQ002_RequestMethod tests all HTTP methods
func TestWranglerCompat_REQ002_RequestMethod(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response(event.request.method));
		});
	`

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/", nil)
		resp, err := rt.Execute(context.Background(), script, req)
		if err != nil {
			t.Fatalf("REQ-002: Execute failed for %s: %v", method, err)
		}

		if string(resp.Body) != method {
			t.Errorf("REQ-002: Expected method '%s', got '%s'", method, string(resp.Body))
		}
	}
}

// TestWranglerCompat_REQ003_RequestHeaders tests request headers access
func TestWranglerCompat_REQ003_RequestHeaders(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var req = event.request;
			var result = {
				hasHeaders: req.headers instanceof Headers,
				auth: req.headers.get('authorization'),
				contentType: req.headers.get('content-type'),
				custom: req.headers.get('x-custom')
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom", "custom-value")

	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-003: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("REQ-003: Failed to parse response: %v", err)
	}

	if result["hasHeaders"] != true {
		t.Errorf("REQ-003: Expected hasHeaders to be true")
	}
	if result["auth"] != "Bearer token123" {
		t.Errorf("REQ-003: Expected auth 'Bearer token123', got '%v'", result["auth"])
	}
}

// TestWranglerCompat_REQCF001_RequestCfColo tests request.cf.colo
func TestWranglerCompat_REQCF001_RequestCfColo(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var colo = event.request.cf ? event.request.cf.colo : 'undefined';
			event.respondWith(new Response(colo || 'undefined'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-CF-001: Execute failed: %v", err)
	}

	body := string(resp.Body)
	if body == "undefined" || body == "" {
		t.Errorf("REQ-CF-001: Expected cf.colo to be set, got '%s'", body)
	}
	// Colo should be 3-letter IATA code
	if len(body) != 3 {
		t.Errorf("REQ-CF-001: Expected 3-letter colo code, got '%s'", body)
	}
}

// TestWranglerCompat_REQCF002_RequestCfCountry tests request.cf.country
func TestWranglerCompat_REQCF002_RequestCfCountry(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var country = event.request.cf ? event.request.cf.country : 'undefined';
			event.respondWith(new Response(country || 'undefined'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-CF-002: Execute failed: %v", err)
	}

	body := string(resp.Body)
	if body == "undefined" || body == "" {
		t.Errorf("REQ-CF-002: Expected cf.country to be set, got '%s'", body)
	}
	// Country should be 2-letter ISO code
	if len(body) != 2 {
		t.Errorf("REQ-CF-002: Expected 2-letter country code, got '%s'", body)
	}
}

// TestWranglerCompat_REQCF_AllProperties tests all request.cf properties
func TestWranglerCompat_REQCF_AllProperties(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var cf = event.request.cf || {};
			var result = {
				colo: cf.colo,
				country: cf.country,
				asn: cf.asn,
				asOrganization: cf.asOrganization,
				timezone: cf.timezone,
				city: cf.city,
				continent: cf.continent,
				latitude: cf.latitude,
				longitude: cf.longitude,
				region: cf.region,
				regionCode: cf.regionCode,
				tlsVersion: cf.tlsVersion,
				httpProtocol: cf.httpProtocol
			};
			event.respondWith(Response.json(result));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-CF-*: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("REQ-CF-*: Failed to parse response: %v", err)
	}

	// Check required properties exist
	if result["colo"] == nil {
		t.Errorf("REQ-CF-001: cf.colo should be set")
	}
	if result["country"] == nil {
		t.Errorf("REQ-CF-002: cf.country should be set")
	}
}

// TestWranglerCompat_REQBODY001_RequestText tests request.text()
func TestWranglerCompat_REQBODY001_RequestText(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.request.text().then(function(text) {
				event.respondWith(new Response(text));
			});
		});
	`

	req := httptest.NewRequest("POST", "/", strings.NewReader("Hello, World!"))
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-BODY-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "Hello, World!" {
		t.Errorf("REQ-BODY-001: Expected 'Hello, World!', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_REQBODY002_RequestJSON tests request.json()
func TestWranglerCompat_REQBODY002_RequestJSON(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.request.json().then(function(data) {
				event.respondWith(Response.json({ received: data.message, count: data.count }));
			});
		});
	`

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":"test","count":42}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-BODY-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("REQ-BODY-002: Failed to parse response: %v", err)
	}

	if result["received"] != "test" {
		t.Errorf("REQ-BODY-002: Expected received 'test', got '%v'", result["received"])
	}
	if result["count"] != float64(42) {
		t.Errorf("REQ-BODY-002: Expected count 42, got '%v'", result["count"])
	}
}

// TestWranglerCompat_REQBODY003_RequestArrayBuffer tests request.arrayBuffer()
func TestWranglerCompat_REQBODY003_RequestArrayBuffer(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.request.arrayBuffer().then(function(buffer) {
				var bytes = new Uint8Array(buffer);
				event.respondWith(new Response(bytes.length.toString()));
			});
		});
	`

	req := httptest.NewRequest("POST", "/", strings.NewReader("12345"))
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-BODY-003: Execute failed: %v", err)
	}

	if string(resp.Body) != "5" {
		t.Errorf("REQ-BODY-003: Expected '5', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_REQNEW001_RequestConstructor tests new Request(url)
func TestWranglerCompat_REQNEW001_RequestConstructor(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var newReq = new Request('https://example.com/api');
			event.respondWith(Response.json({
				url: newReq.url,
				method: newReq.method
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-NEW-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("REQ-NEW-001: Failed to parse response: %v", err)
	}

	if !strings.Contains(result["url"].(string), "example.com/api") {
		t.Errorf("REQ-NEW-001: Expected URL to contain 'example.com/api'")
	}
	if result["method"] != "GET" {
		t.Errorf("REQ-NEW-001: Expected default method GET")
	}
}

// TestWranglerCompat_REQNEW002_RequestWithInit tests new Request(url, init)
func TestWranglerCompat_REQNEW002_RequestWithInit(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var newReq = new Request('https://example.com/api', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ test: true })
			});
			event.respondWith(Response.json({
				method: newReq.method,
				hasBody: newReq.body !== null
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("REQ-NEW-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("REQ-NEW-002: Failed to parse response: %v", err)
	}

	if result["method"] != "POST" {
		t.Errorf("REQ-NEW-002: Expected method POST, got '%v'", result["method"])
	}
}

// ===========================================================================
// 2. Response Object Tests
// ===========================================================================

// TestWranglerCompat_RES001_ResponseStatus tests Response.status
func TestWranglerCompat_RES001_ResponseStatus(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	testCases := []int{200, 201, 204, 301, 400, 404, 500}

	for _, status := range testCases {
		script := fmt.Sprintf(`
			addEventListener('fetch', function(event) {
				var resp = new Response('body', { status: %d });
				event.respondWith(new Response(resp.status.toString()));
			});
		`, status)

		req := httptest.NewRequest("GET", "/", nil)
		resp, err := rt.Execute(context.Background(), script, req)
		if err != nil {
			t.Fatalf("RES-001: Execute failed for status %d: %v", status, err)
		}

		if string(resp.Body) != fmt.Sprintf("%d", status) {
			t.Errorf("RES-001: Expected status '%d', got '%s'", status, string(resp.Body))
		}
	}
}

// TestWranglerCompat_RES002_ResponseStatusText tests Response.statusText
func TestWranglerCompat_RES002_ResponseStatusText(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var resp = new Response('body', {
				status: 201,
				statusText: 'Created'
			});
			event.respondWith(new Response(resp.statusText));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("RES-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "Created" {
		t.Errorf("RES-002: Expected 'Created', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_RES003_ResponseOk tests Response.ok property
func TestWranglerCompat_RES003_ResponseOk(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	testCases := []struct {
		status int
		ok     bool
	}{
		{200, true},
		{201, true},
		{299, true},
		{300, false},
		{400, false},
		{500, false},
	}

	for _, tc := range testCases {
		script := fmt.Sprintf(`
			addEventListener('fetch', function(event) {
				var resp = new Response('', { status: %d });
				event.respondWith(new Response(resp.ok.toString()));
			});
		`, tc.status)

		req := httptest.NewRequest("GET", "/", nil)
		resp, err := rt.Execute(context.Background(), script, req)
		if err != nil {
			t.Fatalf("RES-003: Execute failed for status %d: %v", tc.status, err)
		}

		expected := "false"
		if tc.ok {
			expected = "true"
		}
		if string(resp.Body) != expected {
			t.Errorf("RES-003: For status %d, expected ok=%s, got '%s'", tc.status, expected, string(resp.Body))
		}
	}
}

// TestWranglerCompat_RES004_ResponseHeaders tests Response.headers
func TestWranglerCompat_RES004_ResponseHeaders(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var resp = new Response('body', {
				headers: {
					'Content-Type': 'text/plain',
					'X-Custom': 'value'
				}
			});
			event.respondWith(Response.json({
				hasHeaders: resp.headers instanceof Headers,
				contentType: resp.headers.get('content-type'),
				custom: resp.headers.get('x-custom')
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("RES-004: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("RES-004: Failed to parse response: %v", err)
	}

	if result["hasHeaders"] != true {
		t.Errorf("RES-004: Expected hasHeaders to be true")
	}
	if result["contentType"] != "text/plain" {
		t.Errorf("RES-004: Expected content-type 'text/plain', got '%v'", result["contentType"])
	}
}

// TestWranglerCompat_RESBODY001_ResponseText tests response.text()
func TestWranglerCompat_RESBODY001_ResponseText(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var resp = new Response('Hello, World!');
			resp.text().then(function(text) {
				event.respondWith(new Response(text));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("RES-BODY-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "Hello, World!" {
		t.Errorf("RES-BODY-001: Expected 'Hello, World!', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_RESBODY002_ResponseJSON tests response.json()
func TestWranglerCompat_RESBODY002_ResponseJSON(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var resp = new Response('{"message":"test","count":42}');
			resp.json().then(function(data) {
				event.respondWith(Response.json({ received: data.message, count: data.count }));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("RES-BODY-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("RES-BODY-002: Failed to parse response: %v", err)
	}

	if result["received"] != "test" {
		t.Errorf("RES-BODY-002: Expected received 'test', got '%v'", result["received"])
	}
}

// TestWranglerCompat_RESBODY006_ResponseClone tests response.clone()
func TestWranglerCompat_RESBODY006_ResponseClone(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var resp = new Response('original', {
				status: 201,
				headers: { 'X-Custom': 'value' }
			});
			var clone = resp.clone();
			Promise.all([resp.text(), clone.text()]).then(function(texts) {
				event.respondWith(Response.json({
					text1: texts[0],
					text2: texts[1],
					status: clone.status
				}));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("RES-BODY-006: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("RES-BODY-006: Failed to parse response: %v", err)
	}

	if result["text1"] != "original" || result["text2"] != "original" {
		t.Errorf("RES-BODY-006: Expected both texts to be 'original'")
	}
	if result["status"] != float64(201) {
		t.Errorf("RES-BODY-006: Expected status 201, got '%v'", result["status"])
	}
}

// TestWranglerCompat_RESSTATIC001_ResponseJSON tests Response.json()
func TestWranglerCompat_RESSTATIC001_ResponseJSON(t *testing.T) {
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
		t.Fatalf("RES-STATIC-001: Execute failed: %v", err)
	}

	contentType := resp.Headers.Get("content-type")
	if contentType != "application/json" {
		t.Errorf("RES-STATIC-001: Expected content-type 'application/json', got '%s'", contentType)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("RES-STATIC-001: Failed to parse response: %v", err)
	}

	if result["message"] != "Hello" || result["count"] != float64(42) {
		t.Errorf("RES-STATIC-001: Unexpected JSON content")
	}
}

// TestWranglerCompat_RESSTATIC003_ResponseRedirect tests Response.redirect()
func TestWranglerCompat_RESSTATIC003_ResponseRedirect(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(Response.redirect('https://example.com/new-path', 302));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("RES-STATIC-003: Execute failed: %v", err)
	}

	if resp.Status != 302 {
		t.Errorf("RES-STATIC-003: Expected status 302, got %d", resp.Status)
	}

	location := resp.Headers.Get("location")
	if location != "https://example.com/new-path" {
		t.Errorf("RES-STATIC-003: Expected location 'https://example.com/new-path', got '%s'", location)
	}
}

// ===========================================================================
// 3. Headers Object Tests
// ===========================================================================

// TestWranglerCompat_HDR001_HeadersGet tests headers.get()
func TestWranglerCompat_HDR001_HeadersGet(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var headers = new Headers();
			headers.set('Content-Type', 'application/json');
			event.respondWith(new Response(headers.get('content-type')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("HDR-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "application/json" {
		t.Errorf("HDR-001: Expected 'application/json', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_HDR003_HeadersAppend tests headers.append()
func TestWranglerCompat_HDR003_HeadersAppend(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var headers = new Headers();
			headers.append('Accept', 'text/html');
			headers.append('Accept', 'application/json');
			event.respondWith(new Response(headers.get('accept')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("HDR-003: Execute failed: %v", err)
	}

	// Headers.append should join with ", "
	if string(resp.Body) != "text/html, application/json" {
		t.Errorf("HDR-003: Expected 'text/html, application/json', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_HDR005_HeadersHas tests headers.has()
func TestWranglerCompat_HDR005_HeadersHas(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var headers = new Headers();
			headers.set('Content-Type', 'text/plain');
			event.respondWith(Response.json({
				has: headers.has('content-type'),
				hasNot: headers.has('accept')
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("HDR-005: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("HDR-005: Failed to parse response: %v", err)
	}

	if result["has"] != true || result["hasNot"] != false {
		t.Errorf("HDR-005: Unexpected has() results")
	}
}

// TestWranglerCompat_HDR010_HeadersCaseInsensitive tests case-insensitive header access
func TestWranglerCompat_HDR010_HeadersCaseInsensitive(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var headers = new Headers();
			headers.set('Content-Type', 'text/plain');
			event.respondWith(Response.json({
				lower: headers.get('content-type'),
				upper: headers.get('CONTENT-TYPE'),
				mixed: headers.get('Content-Type')
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("HDR-010: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("HDR-010: Failed to parse response: %v", err)
	}

	if result["lower"] != "text/plain" || result["upper"] != "text/plain" || result["mixed"] != "text/plain" {
		t.Errorf("HDR-010: Headers should be case-insensitive")
	}
}

// TestWranglerCompat_HDR006_HeadersEntries tests headers.entries()
func TestWranglerCompat_HDR006_HeadersEntries(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var headers = new Headers();
			headers.set('Content-Type', 'text/plain');
			headers.set('Accept', 'application/json');

			var entries = [];
			var iter = headers.entries();
			var result = iter.next();
			while (!result.done) {
				entries.push(result.value[0] + ':' + result.value[1]);
				result = iter.next();
			}
			event.respondWith(new Response(entries.sort().join('|')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("HDR-006: Execute failed: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "content-type:text/plain") || !strings.Contains(body, "accept:application/json") {
		t.Errorf("HDR-006: Expected both headers in entries, got '%s'", body)
	}
}

// ===========================================================================
// 4. URL & URLSearchParams Tests
// ===========================================================================

// TestWranglerCompat_URL001_URLHref tests url.href
func TestWranglerCompat_URL001_URLHref(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var url = new URL('https://user:pass@example.com:8080/path?query=1#hash');
			event.respondWith(Response.json({
				href: url.href,
				protocol: url.protocol,
				host: url.host,
				hostname: url.hostname,
				port: url.port,
				pathname: url.pathname,
				search: url.search,
				hash: url.hash,
				origin: url.origin
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("URL-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("URL-001: Failed to parse response: %v", err)
	}

	// Verify all URL parts
	if result["protocol"] != "https:" {
		t.Errorf("URL-002: Expected protocol 'https:', got '%v'", result["protocol"])
	}
	if result["host"] != "example.com:8080" {
		t.Errorf("URL-003: Expected host 'example.com:8080', got '%v'", result["host"])
	}
	if result["hostname"] != "example.com" {
		t.Errorf("URL-004: Expected hostname 'example.com', got '%v'", result["hostname"])
	}
	if result["port"] != "8080" {
		t.Errorf("URL-005: Expected port '8080', got '%v'", result["port"])
	}
	if result["pathname"] != "/path" {
		t.Errorf("URL-006: Expected pathname '/path', got '%v'", result["pathname"])
	}
	if result["search"] != "?query=1" {
		t.Errorf("URL-007: Expected search '?query=1', got '%v'", result["search"])
	}
	if result["hash"] != "#hash" {
		t.Errorf("URL-008: Expected hash '#hash', got '%v'", result["hash"])
	}
	if result["origin"] != "https://example.com:8080" {
		t.Errorf("URL-009: Expected origin 'https://example.com:8080', got '%v'", result["origin"])
	}
}

// TestWranglerCompat_USP001_URLSearchParamsGet tests params.get()
func TestWranglerCompat_USP001_URLSearchParamsGet(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var url = new URL('https://example.com?name=test&count=42');
			event.respondWith(Response.json({
				name: url.searchParams.get('name'),
				count: url.searchParams.get('count'),
				missing: url.searchParams.get('missing')
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("USP-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("USP-001: Failed to parse response: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("USP-001: Expected name 'test', got '%v'", result["name"])
	}
	if result["count"] != "42" {
		t.Errorf("USP-001: Expected count '42', got '%v'", result["count"])
	}
	if result["missing"] != nil {
		t.Errorf("USP-001: Expected missing to be null, got '%v'", result["missing"])
	}
}

// TestWranglerCompat_USP002_URLSearchParamsGetAll tests params.getAll()
func TestWranglerCompat_USP002_URLSearchParamsGetAll(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var url = new URL('https://example.com?tag=a&tag=b&tag=c');
			var tags = url.searchParams.getAll('tag');
			event.respondWith(Response.json({ tags: tags }));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("USP-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("USP-002: Failed to parse response: %v", err)
	}

	tags := result["tags"].([]interface{})
	if len(tags) != 3 || tags[0] != "a" || tags[1] != "b" || tags[2] != "c" {
		t.Errorf("USP-002: Expected ['a','b','c'], got '%v'", tags)
	}
}

// TestWranglerCompat_USP010_URLSearchParamsToString tests params.toString()
func TestWranglerCompat_USP010_URLSearchParamsToString(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var params = new URLSearchParams();
			params.set('name', 'test');
			params.set('count', '42');
			event.respondWith(new Response(params.toString()));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("USP-010: Execute failed: %v", err)
	}

	body := string(resp.Body)
	if !strings.Contains(body, "name=test") || !strings.Contains(body, "count=42") {
		t.Errorf("USP-010: Expected query string with name and count, got '%s'", body)
	}
}

// ===========================================================================
// 6. Event Handler Tests
// ===========================================================================

// TestWranglerCompat_EVT001_AddEventListener tests addEventListener('fetch')
func TestWranglerCompat_EVT001_AddEventListener(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response('addEventListener works'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("EVT-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "addEventListener works" {
		t.Errorf("EVT-001: Expected 'addEventListener works', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_EVT004_WaitUntil tests event.waitUntil()
func TestWranglerCompat_EVT004_WaitUntil(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var completed = false;
			event.waitUntil(new Promise(function(resolve) {
				completed = true;
				resolve();
			}));
			event.respondWith(new Response(completed.toString()));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("EVT-004: Execute failed: %v", err)
	}

	if string(resp.Body) != "true" {
		t.Errorf("EVT-004: Expected waitUntil to work")
	}
}

// TestWranglerCompat_EVT006_EventType tests event.type
func TestWranglerCompat_EVT006_EventType(t *testing.T) {
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
		t.Fatalf("EVT-006: Execute failed: %v", err)
	}

	if string(resp.Body) != "fetch" {
		t.Errorf("EVT-006: Expected event type 'fetch', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 7. Web Crypto API Tests
// ===========================================================================

// TestWranglerCompat_CRYPTO001_GetRandomValues tests crypto.getRandomValues()
func TestWranglerCompat_CRYPTO001_GetRandomValues(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var buffer = new Uint8Array(16);
			crypto.getRandomValues(buffer);
			// Check that at least some values are non-zero
			var hasNonZero = false;
			for (var i = 0; i < buffer.length; i++) {
				if (buffer[i] !== 0) hasNonZero = true;
			}
			event.respondWith(new Response(hasNonZero.toString()));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("CRYPTO-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "true" {
		t.Errorf("CRYPTO-001: Expected random values to contain non-zero")
	}
}

// TestWranglerCompat_CRYPTO002_RandomUUID tests crypto.randomUUID()
func TestWranglerCompat_CRYPTO002_RandomUUID(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var uuid = crypto.randomUUID();
			// UUID format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
			var valid = uuid.length === 36 &&
				uuid.split('-').length === 5 &&
				uuid[14] === '4'; // Version 4
			event.respondWith(Response.json({ uuid: uuid, valid: valid }));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("CRYPTO-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("CRYPTO-002: Failed to parse response: %v", err)
	}

	if result["valid"] != true {
		t.Errorf("CRYPTO-002: Invalid UUID format: %v", result["uuid"])
	}
}

// TestWranglerCompat_DIGEST001_SHA1 tests crypto.subtle.digest('SHA-1')
func TestWranglerCompat_DIGEST001_SHA1(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var encoder = new TextEncoder();
			var data = encoder.encode('hello');
			crypto.subtle.digest('SHA-1', data).then(function(hash) {
				var hashArray = new Uint8Array(hash);
				var hashHex = '';
				for (var i = 0; i < hashArray.length; i++) {
					hashHex += hashArray[i].toString(16).padStart(2, '0');
				}
				event.respondWith(new Response(hashHex));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DIGEST-001: Execute failed: %v", err)
	}

	// SHA-1 of "hello" = aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d
	expected := "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"
	if string(resp.Body) != expected {
		t.Errorf("DIGEST-001: Expected SHA-1 '%s', got '%s'", expected, string(resp.Body))
	}
}

// TestWranglerCompat_DIGEST002_SHA256 tests crypto.subtle.digest('SHA-256')
func TestWranglerCompat_DIGEST002_SHA256(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var encoder = new TextEncoder();
			var data = encoder.encode('hello');
			crypto.subtle.digest('SHA-256', data).then(function(hash) {
				var hashArray = new Uint8Array(hash);
				var hashHex = '';
				for (var i = 0; i < hashArray.length; i++) {
					hashHex += hashArray[i].toString(16).padStart(2, '0');
				}
				event.respondWith(new Response(hashHex));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DIGEST-002: Execute failed: %v", err)
	}

	// SHA-256 of "hello" = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if string(resp.Body) != expected {
		t.Errorf("DIGEST-002: Expected SHA-256 '%s', got '%s'", expected, string(resp.Body))
	}
}

// TestWranglerCompat_DIGEST005_MD5 tests crypto.subtle.digest('MD5')
func TestWranglerCompat_DIGEST005_MD5(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var encoder = new TextEncoder();
			var data = encoder.encode('hello');
			crypto.subtle.digest('MD5', data).then(function(hash) {
				var hashArray = new Uint8Array(hash);
				var hashHex = '';
				for (var i = 0; i < hashArray.length; i++) {
					hashHex += hashArray[i].toString(16).padStart(2, '0');
				}
				event.respondWith(new Response(hashHex));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("DIGEST-005: Execute failed: %v", err)
	}

	// MD5 of "hello" = 5d41402abc4b2a76b9719d911017c592
	expected := "5d41402abc4b2a76b9719d911017c592"
	if string(resp.Body) != expected {
		t.Errorf("DIGEST-005: Expected MD5 '%s', got '%s'", expected, string(resp.Body))
	}
}

// ===========================================================================
// 8. Encoding API Tests
// ===========================================================================

// TestWranglerCompat_ENCTXT001_TextEncoder tests TextEncoder.encode()
func TestWranglerCompat_ENCTXT001_TextEncoder(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var encoder = new TextEncoder();
			var bytes = encoder.encode('hello');
			event.respondWith(Response.json({
				length: bytes.length,
				first: bytes[0],
				last: bytes[4]
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("ENC-TXT-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("ENC-TXT-001: Failed to parse response: %v", err)
	}

	if result["length"] != float64(5) {
		t.Errorf("ENC-TXT-001: Expected length 5, got '%v'", result["length"])
	}
	if result["first"] != float64(104) { // 'h' = 104
		t.Errorf("ENC-TXT-001: Expected first byte 104, got '%v'", result["first"])
	}
}

// TestWranglerCompat_ENCTXT002_TextDecoder tests TextDecoder.decode()
func TestWranglerCompat_ENCTXT002_TextDecoder(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var encoder = new TextEncoder();
			var decoder = new TextDecoder();
			var bytes = encoder.encode('Hello, World!');
			var text = decoder.decode(bytes);
			event.respondWith(new Response(text));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("ENC-TXT-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "Hello, World!" {
		t.Errorf("ENC-TXT-002: Expected 'Hello, World!', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_B64001_Btoa tests btoa()
func TestWranglerCompat_B64001_Btoa(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response(btoa('hello world')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("B64-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "aGVsbG8gd29ybGQ=" {
		t.Errorf("B64-001: Expected 'aGVsbG8gd29ybGQ=', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_B64002_Atob tests atob()
func TestWranglerCompat_B64002_Atob(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(new Response(atob('aGVsbG8gd29ybGQ=')));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("B64-002: Execute failed: %v", err)
	}

	if string(resp.Body) != "hello world" {
		t.Errorf("B64-002: Expected 'hello world', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 10. FormData API Tests
// ===========================================================================

// TestWranglerCompat_FD001_FormDataNew tests new FormData()
func TestWranglerCompat_FD001_FormDataNew(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var formData = new FormData();
			formData.append('name', 'John');
			formData.append('age', '30');
			event.respondWith(Response.json({
				name: formData.get('name'),
				age: formData.get('age'),
				has: formData.has('name'),
				missing: formData.has('email')
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("FD-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("FD-001: Failed to parse response: %v", err)
	}

	if result["name"] != "John" {
		t.Errorf("FD-001: Expected name 'John', got '%v'", result["name"])
	}
	if result["has"] != true || result["missing"] != false {
		t.Errorf("FD-001: has() returned unexpected results")
	}
}

// TestWranglerCompat_FD005_FormDataGetAll tests formData.getAll()
func TestWranglerCompat_FD005_FormDataGetAll(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var formData = new FormData();
			formData.append('tag', 'a');
			formData.append('tag', 'b');
			formData.append('tag', 'c');
			event.respondWith(Response.json({
				tags: formData.getAll('tag'),
				single: formData.get('tag')
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("FD-005: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("FD-005: Failed to parse response: %v", err)
	}

	tags := result["tags"].([]interface{})
	if len(tags) != 3 {
		t.Errorf("FD-005: Expected 3 tags, got %d", len(tags))
	}
	if result["single"] != "a" {
		t.Errorf("FD-005: get() should return first value, got '%v'", result["single"])
	}
}

// ===========================================================================
// 11. Blob API Tests
// ===========================================================================

// TestWranglerCompat_BLOB001_BlobNew tests new Blob()
func TestWranglerCompat_BLOB001_BlobNew(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var blob = new Blob(['Hello, ', 'World!'], { type: 'text/plain' });
			event.respondWith(Response.json({
				size: blob.size,
				type: blob.type
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("BLOB-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("BLOB-001: Failed to parse response: %v", err)
	}

	if result["size"] != float64(13) { // "Hello, World!" = 13 chars
		t.Errorf("BLOB-001: Expected size 13, got '%v'", result["size"])
	}
	if result["type"] != "text/plain" {
		t.Errorf("BLOB-001: Expected type 'text/plain', got '%v'", result["type"])
	}
}

// TestWranglerCompat_BLOB007_BlobText tests blob.text()
func TestWranglerCompat_BLOB007_BlobText(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var blob = new Blob(['Hello, World!']);
			blob.text().then(function(text) {
				event.respondWith(new Response(text));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("BLOB-007: Execute failed: %v", err)
	}

	if string(resp.Body) != "Hello, World!" {
		t.Errorf("BLOB-007: Expected 'Hello, World!', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_BLOB006_BlobSlice tests blob.slice()
func TestWranglerCompat_BLOB006_BlobSlice(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var blob = new Blob(['Hello, World!']);
			var slice = blob.slice(0, 5);
			slice.text().then(function(text) {
				event.respondWith(new Response(text));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("BLOB-006: Execute failed: %v", err)
	}

	if string(resp.Body) != "Hello" {
		t.Errorf("BLOB-006: Expected 'Hello', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 12. AbortController Tests
// ===========================================================================

// TestWranglerCompat_ABORT001_AbortControllerNew tests new AbortController()
func TestWranglerCompat_ABORT001_AbortControllerNew(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var controller = new AbortController();
			event.respondWith(Response.json({
				hasSignal: !!controller.signal,
				aborted: controller.signal.aborted
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("ABORT-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("ABORT-001: Failed to parse response: %v", err)
	}

	if result["hasSignal"] != true {
		t.Errorf("ABORT-001: Expected signal to exist")
	}
	if result["aborted"] != false {
		t.Errorf("ABORT-001: Expected aborted to be false initially")
	}
}

// TestWranglerCompat_ABORT003_Abort tests controller.abort()
func TestWranglerCompat_ABORT003_Abort(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var controller = new AbortController();
			var beforeAbort = controller.signal.aborted;
			controller.abort();
			var afterAbort = controller.signal.aborted;
			event.respondWith(Response.json({
				before: beforeAbort,
				after: afterAbort
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("ABORT-003: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("ABORT-003: Failed to parse response: %v", err)
	}

	if result["before"] != false || result["after"] != true {
		t.Errorf("ABORT-003: Expected before=false, after=true")
	}
}

// ===========================================================================
// 16. Console API Tests
// ===========================================================================

// TestWranglerCompat_CON001_ConsoleLog tests console.log()
func TestWranglerCompat_CON001_ConsoleLog(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	// Console.log shouldn't throw, it should work silently
	script := `
		addEventListener('fetch', event => {
			console.log('test message');
			console.log('multiple', 'args', 123);
			console.log({ object: true });
			event.respondWith(new Response('OK'));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("CON-001: Execute failed: %v", err)
	}

	if string(resp.Body) != "OK" {
		t.Errorf("CON-001: Expected 'OK', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 17. Performance & Timers Tests
// ===========================================================================

// TestWranglerCompat_PERF001_PerformanceNow tests performance.now()
func TestWranglerCompat_PERF001_PerformanceNow(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var start = performance.now();
			// Small delay
			for (var i = 0; i < 1000; i++) {}
			var end = performance.now();
			event.respondWith(Response.json({
				start: typeof start,
				end: typeof end,
				elapsed: end - start >= 0
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("PERF-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("PERF-001: Failed to parse response: %v", err)
	}

	if result["start"] != "number" || result["end"] != "number" {
		t.Errorf("PERF-001: performance.now() should return numbers")
	}
	if result["elapsed"] != true {
		t.Errorf("PERF-001: Elapsed time should be >= 0")
	}
}

// TestWranglerCompat_PERF002_DateNow tests Date.now()
func TestWranglerCompat_PERF002_DateNow(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var now = Date.now();
			// Should be a reasonable timestamp (after 2020)
			var valid = now > 1577836800000;
			event.respondWith(Response.json({
				timestamp: now,
				valid: valid,
				type: typeof now
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("PERF-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("PERF-002: Failed to parse response: %v", err)
	}

	if result["type"] != "number" || result["valid"] != true {
		t.Errorf("PERF-002: Date.now() should return valid timestamp")
	}
}

// ===========================================================================
// 18. JSON API Tests
// ===========================================================================

// TestWranglerCompat_JSON001_Parse tests JSON.parse()
func TestWranglerCompat_JSON001_Parse(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var obj = JSON.parse('{"name":"test","count":42,"active":true}');
			event.respondWith(Response.json({
				name: obj.name,
				count: obj.count,
				active: obj.active
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("JSON-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("JSON-001: Failed to parse response: %v", err)
	}

	if result["name"] != "test" || result["count"] != float64(42) || result["active"] != true {
		t.Errorf("JSON-001: JSON.parse() returned unexpected values")
	}
}

// TestWranglerCompat_JSON002_Stringify tests JSON.stringify()
func TestWranglerCompat_JSON002_Stringify(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var obj = { name: 'test', count: 42 };
			var json = JSON.stringify(obj);
			event.respondWith(new Response(json));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("JSON-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("JSON-002: Failed to parse stringified JSON: %v", err)
	}

	if result["name"] != "test" || result["count"] != float64(42) {
		t.Errorf("JSON-002: JSON.stringify() produced incorrect output")
	}
}

// TestWranglerCompat_JSON006_ParseError tests JSON.parse error handling
func TestWranglerCompat_JSON006_ParseError(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var error = null;
			try {
				JSON.parse('invalid json');
			} catch (e) {
				error = e.name;
			}
			event.respondWith(new Response(error));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("JSON-006: Execute failed: %v", err)
	}

	if string(resp.Body) != "SyntaxError" {
		t.Errorf("JSON-006: Expected 'SyntaxError', got '%s'", string(resp.Body))
	}
}

// ===========================================================================
// 19. structuredClone Tests
// ===========================================================================

// TestWranglerCompat_CLONE001_Object tests structuredClone(object)
func TestWranglerCompat_CLONE001_Object(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var obj = { a: 1, b: { c: 2 } };
			var clone = structuredClone(obj);
			clone.b.c = 3;
			event.respondWith(Response.json({
				original: obj.b.c,
				cloned: clone.b.c
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("CLONE-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("CLONE-001: Failed to parse response: %v", err)
	}

	if result["original"] != float64(2) || result["cloned"] != float64(3) {
		t.Errorf("CLONE-001: structuredClone should create deep copy")
	}
}

// TestWranglerCompat_CLONE002_Array tests structuredClone(array)
func TestWranglerCompat_CLONE002_Array(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var arr = [1, [2, 3], { nested: true }];
			var clone = structuredClone(arr);
			clone[1][0] = 99;
			event.respondWith(Response.json({
				original: arr[1][0],
				cloned: clone[1][0]
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("CLONE-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("CLONE-002: Failed to parse response: %v", err)
	}

	if result["original"] != float64(2) || result["cloned"] != float64(99) {
		t.Errorf("CLONE-002: structuredClone should create deep copy of arrays")
	}
}

// TestWranglerCompat_CLONE003_Date tests structuredClone(date)
func TestWranglerCompat_CLONE003_Date(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var date = new Date('2024-01-15T12:00:00Z');
			var clone = structuredClone(date);
			event.respondWith(Response.json({
				isDate: clone instanceof Date,
				time: clone.getTime(),
				iso: clone.toISOString()
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("CLONE-003: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("CLONE-003: Failed to parse response: %v", err)
	}

	if result["isDate"] != true {
		t.Errorf("CLONE-003: Clone should be a Date instance")
	}
	if result["iso"] != "2024-01-15T12:00:00.000Z" {
		t.Errorf("CLONE-003: Clone should preserve date value")
	}
}

// ===========================================================================
// 22. Global Objects Tests
// ===========================================================================

// TestWranglerCompat_GLB001_GlobalThis tests globalThis
func TestWranglerCompat_GLB001_GlobalThis(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(Response.json({
				hasGlobalThis: typeof globalThis === 'object',
				hasResponse: typeof globalThis.Response === 'function',
				hasRequest: typeof globalThis.Request === 'function',
				hasFetch: typeof globalThis.fetch === 'function'
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("GLB-001: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("GLB-001: Failed to parse response: %v", err)
	}

	if result["hasGlobalThis"] != true {
		t.Errorf("GLB-001: globalThis should exist")
	}
	if result["hasResponse"] != true || result["hasRequest"] != true {
		t.Errorf("GLB-001: globalThis should have Request/Response")
	}
}

// TestWranglerCompat_GLB002_Self tests self reference
func TestWranglerCompat_GLB002_Self(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			event.respondWith(Response.json({
				hasSelf: typeof self === 'object',
				selfEqualsGlobal: self === globalThis
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("GLB-002: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("GLB-002: Failed to parse response: %v", err)
	}

	if result["hasSelf"] != true {
		t.Errorf("GLB-002: self should exist")
	}
}

// ===========================================================================
// Additional Compatibility Tests
// ===========================================================================

// TestWranglerCompat_PromiseAll tests Promise.all()
func TestWranglerCompat_PromiseAll(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var promises = [
				Promise.resolve(1),
				Promise.resolve(2),
				Promise.resolve(3)
			];
			Promise.all(promises).then(function(results) {
				event.respondWith(new Response(results.join(',')));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("PromiseAll: Execute failed: %v", err)
	}

	if string(resp.Body) != "1,2,3" {
		t.Errorf("PromiseAll: Expected '1,2,3', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_PromiseRace tests Promise.race()
func TestWranglerCompat_PromiseRace(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			Promise.race([
				Promise.resolve('first'),
				new Promise(function(resolve) { setTimeout(function() { resolve('second'); }, 100); })
			]).then(function(first) {
				event.respondWith(new Response(first));
			});
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("PromiseRace: Execute failed: %v", err)
	}

	if string(resp.Body) != "first" {
		t.Errorf("PromiseRace: Expected 'first', got '%s'", string(resp.Body))
	}
}

// TestWranglerCompat_ArrayMethods tests modern array methods
func TestWranglerCompat_ArrayMethods(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var arr = [1, 2, 3, 4, 5];
			event.respondWith(Response.json({
				map: arr.map(function(x) { return x * 2; }),
				filter: arr.filter(function(x) { return x > 2; }),
				reduce: arr.reduce(function(a, b) { return a + b; }, 0),
				find: arr.find(function(x) { return x > 3; }),
				findIndex: arr.findIndex(function(x) { return x > 3; }),
				some: arr.some(function(x) { return x > 3; }),
				every: arr.every(function(x) { return x > 0; }),
				includes: arr.includes(3)
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("ArrayMethods: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("ArrayMethods: Failed to parse response: %v", err)
	}

	if result["reduce"] != float64(15) {
		t.Errorf("ArrayMethods: reduce() returned wrong result")
	}
	if result["find"] != float64(4) {
		t.Errorf("ArrayMethods: find() returned wrong result")
	}
}

// TestWranglerCompat_ObjectMethods tests Object static methods
func TestWranglerCompat_ObjectMethods(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var obj = { a: 1, b: 2, c: 3 };
			event.respondWith(Response.json({
				keys: Object.keys(obj),
				values: Object.values(obj),
				entries: Object.entries(obj)
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("ObjectMethods: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("ObjectMethods: Failed to parse response: %v", err)
	}

	keys := result["keys"].([]interface{})
	if len(keys) != 3 {
		t.Errorf("ObjectMethods: Object.keys() returned wrong count")
	}
}

// TestWranglerCompat_StringMethods tests modern string methods
func TestWranglerCompat_StringMethods(t *testing.T) {
	rt := New(Config{})
	defer rt.Close()

	script := `
		addEventListener('fetch', event => {
			var str = '  Hello, World!  ';
			event.respondWith(Response.json({
				trim: str.trim(),
				startsWith: str.trim().startsWith('Hello'),
				endsWith: str.trim().endsWith('!'),
				includes: str.includes('World'),
				padStart: 'abc'.padStart(6, '0'),
				padEnd: 'abc'.padEnd(6, '0'),
				repeat: 'ab'.repeat(3)
			}));
		});
	`

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := rt.Execute(context.Background(), script, req)
	if err != nil {
		t.Fatalf("StringMethods: Execute failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("StringMethods: Failed to parse response: %v", err)
	}

	if result["trim"] != "Hello, World!" {
		t.Errorf("StringMethods: trim() returned wrong result")
	}
	if result["padStart"] != "000abc" {
		t.Errorf("StringMethods: padStart() returned wrong result")
	}
}
