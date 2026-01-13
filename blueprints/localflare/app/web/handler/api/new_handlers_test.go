package api_test

import (
	"net/http"
	"testing"
)

// ========== KV Tests ==========

func TestKV_ListNamespaces(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/kv/namespaces", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	namespaces, ok := resp.Result["namespaces"].([]interface{})
	if !ok {
		t.Fatal("Expected result.namespaces to be an array")
	}

	if len(namespaces) == 0 {
		t.Fatal("Expected at least one KV namespace from seeded data")
	}
}

func TestKV_ListKeys(t *testing.T) {
	srv := testServer(t)

	// First get a namespace
	w := makeRequest(t, srv, "GET", "/api/kv/namespaces", nil)
	resp := parseResponse(t, w)
	namespaces := resp.Result["namespaces"].([]interface{})
	if len(namespaces) == 0 {
		t.Skip("No KV namespaces to test")
	}

	ns := namespaces[0].(map[string]interface{})
	id := ns["id"].(string)

	// Get keys
	w = makeRequest(t, srv, "GET", "/api/kv/namespaces/"+id+"/keys", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp = parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	_, ok := resp.Result["keys"].([]interface{})
	if !ok {
		t.Fatal("Expected result.keys to be an array")
	}
}

// ========== R2 Tests ==========

func TestR2_ListBuckets(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/r2/buckets", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	buckets, ok := resp.Result["buckets"].([]interface{})
	if !ok {
		t.Fatal("Expected result.buckets to be an array")
	}

	if len(buckets) == 0 {
		t.Fatal("Expected at least one R2 bucket from seeded data")
	}
}

// ========== D1 Tests ==========

func TestD1_ListDatabases(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/d1/databases", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	databases, ok := resp.Result["databases"].([]interface{})
	if !ok {
		t.Fatal("Expected result.databases to be an array")
	}

	if len(databases) == 0 {
		t.Fatal("Expected at least one D1 database from seeded data")
	}
}

// ========== Workers Tests ==========

func TestWorkers_List(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/workers", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	workers, ok := resp.Result["workers"].([]interface{})
	if !ok {
		t.Fatal("Expected result.workers to be an array")
	}

	if len(workers) == 0 {
		t.Fatal("Expected at least one worker from seeded data")
	}
}

// ========== Pages Tests ==========

func TestPages_ListProjects(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/pages/projects", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	projects, ok := resp.Result["projects"].([]interface{})
	if !ok {
		t.Fatal("Expected result.projects to be an array")
	}

	if len(projects) == 0 {
		t.Fatal("Expected at least one Pages project")
	}
}

func TestPages_GetProject(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/pages/projects/my-blog", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestPages_GetDeployments(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/pages/projects/my-blog/deployments", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	deployments, ok := resp.Result["deployments"].([]interface{})
	if !ok {
		t.Fatal("Expected result.deployments to be an array")
	}

	if len(deployments) == 0 {
		t.Fatal("Expected at least one deployment")
	}
}

// ========== Images Tests ==========

func TestImages_List(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/images", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	images, ok := resp.Result["images"].([]interface{})
	if !ok {
		t.Fatal("Expected result.images to be an array")
	}

	if len(images) == 0 {
		t.Fatal("Expected at least one image")
	}
}

func TestImages_ListVariants(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/images/variants", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	variants, ok := resp.Result["variants"].([]interface{})
	if !ok {
		t.Fatal("Expected result.variants to be an array")
	}

	if len(variants) == 0 {
		t.Fatal("Expected at least one variant")
	}
}

// ========== Stream Tests ==========

func TestStream_ListVideos(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/stream/videos", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	videos, ok := resp.Result["videos"].([]interface{})
	if !ok {
		t.Fatal("Expected result.videos to be an array")
	}

	if len(videos) == 0 {
		t.Fatal("Expected at least one video")
	}
}

func TestStream_ListLiveInputs(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/stream/live", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	liveInputs, ok := resp.Result["live_inputs"].([]interface{})
	if !ok {
		t.Fatal("Expected result.live_inputs to be an array")
	}

	if len(liveInputs) == 0 {
		t.Fatal("Expected at least one live input")
	}
}

// ========== Observability Tests ==========

func TestObservability_GetLogs(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/observability/logs", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	logs, ok := resp.Result["logs"].([]interface{})
	if !ok {
		t.Fatal("Expected result.logs to be an array")
	}

	if len(logs) == 0 {
		t.Fatal("Expected at least one log entry")
	}
}

func TestObservability_GetTraces(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/observability/traces", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	traces, ok := resp.Result["traces"].([]interface{})
	if !ok {
		t.Fatal("Expected result.traces to be an array")
	}

	if len(traces) == 0 {
		t.Fatal("Expected at least one trace")
	}
}

func TestObservability_GetMetrics(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/observability/metrics?range=24h", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	data, ok := resp.Result["data"].([]interface{})
	if !ok {
		t.Fatal("Expected result.data to be an array")
	}

	if len(data) == 0 {
		t.Fatal("Expected at least one data point")
	}
}

// ========== Settings Tests ==========

func TestSettings_ListTokens(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/settings/tokens", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	tokens, ok := resp.Result["tokens"].([]interface{})
	if !ok {
		t.Fatal("Expected result.tokens to be an array")
	}

	if len(tokens) == 0 {
		t.Fatal("Expected at least one token")
	}
}

func TestSettings_ListMembers(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/settings/members", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	members, ok := resp.Result["members"].([]interface{})
	if !ok {
		t.Fatal("Expected result.members to be an array")
	}

	if len(members) == 0 {
		t.Fatal("Expected at least one member")
	}
}
