package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/blueprints/localflare/app/web"
	"github.com/go-mizu/blueprints/localflare/pkg/seed"
)

// testServer creates a test server with seeded data.
func testServer(t *testing.T) *web.Server {
	t.Helper()

	tmpDir := t.TempDir()
	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Seed test data
	seeder := seed.New(srv.Store())
	if err := seeder.Run(context.Background()); err != nil {
		t.Fatalf("Failed to seed data: %v", err)
	}

	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

// apiResponse represents a standard API response.
type apiResponse struct {
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// makeRequest makes an HTTP request to the test server.
func makeRequest(t *testing.T, srv *web.Server, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	return w
}

// parseResponse parses the response body into apiResponse.
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) *apiResponse {
	t.Helper()

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	return &resp
}

// ========== Dashboard Tests ==========

func TestDashboard_GetStats(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/dashboard/stats", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatalf("Expected success=true, got false")
	}

	// Verify stats structure
	if resp.Result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	// Check that all expected service stats are present
	requiredServices := []string{"durable_objects", "queues", "vectorize", "analytics", "ai_gateway", "hyperdrive", "cron"}
	for _, svc := range requiredServices {
		if _, ok := resp.Result[svc]; !ok {
			t.Errorf("Expected stats for service %q", svc)
		}
	}
}

func TestDashboard_GetTimeSeries(t *testing.T) {
	srv := testServer(t)

	// Test default parameters
	w := makeRequest(t, srv, "GET", "/api/dashboard/timeseries", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	data, ok := resp.Result["data"].([]interface{})
	if !ok {
		t.Fatal("Expected result.data to be an array")
	}

	// Default is 24h which should have 24 data points
	if len(data) != 24 {
		t.Errorf("Expected 24 data points, got %d", len(data))
	}
}

func TestDashboard_GetTimeSeries_AllRanges(t *testing.T) {
	srv := testServer(t)

	ranges := []struct {
		param    string
		expected int
	}{
		{"1h", 60},
		{"24h", 24},
		{"7d", 7 * 24},
		{"30d", 30},
	}

	for _, tc := range ranges {
		t.Run(tc.param, func(t *testing.T) {
			w := makeRequest(t, srv, "GET", "/api/dashboard/timeseries?range="+tc.param, nil)
			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", w.Code)
			}

			resp := parseResponse(t, w)
			data, ok := resp.Result["data"].([]interface{})
			if !ok {
				t.Fatal("Expected result.data to be an array")
			}

			if len(data) != tc.expected {
				t.Errorf("Expected %d data points for range %s, got %d", tc.expected, tc.param, len(data))
			}
		})
	}
}

func TestDashboard_GetActivity(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/dashboard/activity", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	_, ok := resp.Result["events"].([]interface{})
	if !ok {
		t.Fatal("Expected result.events to be an array")
	}
}

func TestDashboard_GetStatus(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/dashboard/status", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	services, ok := resp.Result["services"].([]interface{})
	if !ok {
		t.Fatal("Expected result.services to be an array")
	}

	// Should have at least 8 services
	if len(services) < 8 {
		t.Errorf("Expected at least 8 services, got %d", len(services))
	}

	// Verify each service has required fields
	for _, svc := range services {
		s := svc.(map[string]interface{})
		if _, ok := s["service"]; !ok {
			t.Error("Service missing 'service' field")
		}
		if _, ok := s["status"]; !ok {
			t.Error("Service missing 'status' field")
		}
	}
}

// ========== Durable Objects Tests ==========

func TestDurableObjects_ListNamespaces(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/durable-objects/namespaces", nil)

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
		t.Fatal("Expected at least one namespace from seeded data")
	}

	// Verify namespace structure
	ns := namespaces[0].(map[string]interface{})
	requiredFields := []string{"id", "name", "script_name", "class_name", "created_at"}
	for _, field := range requiredFields {
		if _, ok := ns[field]; !ok {
			t.Errorf("Namespace missing required field %q", field)
		}
	}
}

func TestDurableObjects_CreateNamespace(t *testing.T) {
	srv := testServer(t)

	input := map[string]string{
		"name":       "test-namespace",
		"script_name": "test-worker",
		"class_name":  "TestClass",
	}

	w := makeRequest(t, srv, "POST", "/api/durable-objects/namespaces", input)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestDurableObjects_GetNamespace(t *testing.T) {
	srv := testServer(t)

	// First get list to find an ID
	w := makeRequest(t, srv, "GET", "/api/durable-objects/namespaces", nil)
	resp := parseResponse(t, w)
	namespaces := resp.Result["namespaces"].([]interface{})
	if len(namespaces) == 0 {
		t.Skip("No namespaces to test")
	}

	ns := namespaces[0].(map[string]interface{})
	id := ns["id"].(string)

	// Get by ID
	w = makeRequest(t, srv, "GET", "/api/durable-objects/namespaces/"+id, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp = parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestDurableObjects_GetNamespace_NotFound(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/durable-objects/namespaces/nonexistent-id", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}
}

func TestDurableObjects_DeleteNamespace(t *testing.T) {
	srv := testServer(t)

	// Create a namespace first
	input := map[string]string{
		"name":       "delete-test",
		"script_name": "test-worker",
		"class_name":  "DeleteTest",
	}
	w := makeRequest(t, srv, "POST", "/api/durable-objects/namespaces", input)
	resp := parseResponse(t, w)

	ns := resp.Result
	id := ns["id"].(string)

	// Delete it
	w = makeRequest(t, srv, "DELETE", "/api/durable-objects/namespaces/"+id, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// ========== Queues Tests ==========

func TestQueues_List(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/queues", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	queues, ok := resp.Result["queues"].([]interface{})
	if !ok {
		t.Fatal("Expected result.queues to be an array")
	}

	if len(queues) == 0 {
		t.Fatal("Expected at least one queue from seeded data")
	}

	// Verify queue structure
	q := queues[0].(map[string]interface{})
	requiredFields := []string{"id", "name", "created_at"}
	for _, field := range requiredFields {
		if _, ok := q[field]; !ok {
			t.Errorf("Queue missing required field %q", field)
		}
	}
}

func TestQueues_Create(t *testing.T) {
	srv := testServer(t)

	input := map[string]interface{}{
		"queue_name": "test-queue",
		"settings": map[string]int{
			"max_retries":     5,
			"max_batch_size":  20,
			"message_ttl":     3600,
		},
	}

	w := makeRequest(t, srv, "POST", "/api/queues", input)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestQueues_Get(t *testing.T) {
	srv := testServer(t)

	// First get list to find an ID
	w := makeRequest(t, srv, "GET", "/api/queues", nil)
	resp := parseResponse(t, w)
	queues := resp.Result["queues"].([]interface{})
	if len(queues) == 0 {
		t.Skip("No queues to test")
	}

	q := queues[0].(map[string]interface{})
	id := q["id"].(string)

	// Get by ID
	w = makeRequest(t, srv, "GET", "/api/queues/"+id, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp = parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestQueues_GetStats(t *testing.T) {
	srv := testServer(t)

	// First get list to find an ID
	w := makeRequest(t, srv, "GET", "/api/queues", nil)
	resp := parseResponse(t, w)
	queues := resp.Result["queues"].([]interface{})
	if len(queues) == 0 {
		t.Skip("No queues to test")
	}

	q := queues[0].(map[string]interface{})
	id := q["id"].(string)

	// Get stats
	w = makeRequest(t, srv, "GET", "/api/queues/"+id+"/stats", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

func TestQueues_SendMessage(t *testing.T) {
	srv := testServer(t)

	// First get a queue
	w := makeRequest(t, srv, "GET", "/api/queues", nil)
	resp := parseResponse(t, w)
	queues := resp.Result["queues"].([]interface{})
	if len(queues) == 0 {
		t.Skip("No queues to test")
	}

	q := queues[0].(map[string]interface{})
	id := q["id"].(string)

	// Send message
	input := map[string]interface{}{
		"body":         "test message",
		"content_type": "text/plain",
	}

	w = makeRequest(t, srv, "POST", "/api/queues/"+id+"/messages", input)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ========== Vectorize Tests ==========

func TestVectorize_ListIndexes(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/vectorize/indexes", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	indexes, ok := resp.Result["indexes"].([]interface{})
	if !ok {
		t.Fatal("Expected result.indexes to be an array")
	}

	if len(indexes) == 0 {
		t.Fatal("Expected at least one index from seeded data")
	}

	// Verify index structure
	idx := indexes[0].(map[string]interface{})
	requiredFields := []string{"id", "name", "dimensions", "metric", "created_at"}
	for _, field := range requiredFields {
		if _, ok := idx[field]; !ok {
			t.Errorf("Index missing required field %q", field)
		}
	}
}

func TestVectorize_CreateIndex(t *testing.T) {
	srv := testServer(t)

	input := map[string]interface{}{
		"name":       "test-index",
		"dimensions": 768,
		"metric":     "cosine",
	}

	w := makeRequest(t, srv, "POST", "/api/vectorize/indexes", input)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestVectorize_GetIndex(t *testing.T) {
	srv := testServer(t)

	// First get list to find an index
	w := makeRequest(t, srv, "GET", "/api/vectorize/indexes", nil)
	resp := parseResponse(t, w)
	indexes := resp.Result["indexes"].([]interface{})
	if len(indexes) == 0 {
		t.Skip("No indexes to test")
	}

	idx := indexes[0].(map[string]interface{})
	name := idx["name"].(string)

	// Get by name
	w = makeRequest(t, srv, "GET", "/api/vectorize/indexes/"+name, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

func TestVectorize_InsertAndQuery(t *testing.T) {
	srv := testServer(t)

	// Create a test index
	createInput := map[string]interface{}{
		"name":       "test-query-index",
		"dimensions": 3,
		"metric":     "cosine",
	}
	w := makeRequest(t, srv, "POST", "/api/vectorize/indexes", createInput)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create index: %s", w.Body.String())
	}

	// Insert vectors
	insertInput := map[string]interface{}{
		"vectors": []map[string]interface{}{
			{"id": "vec1", "values": []float32{0.1, 0.2, 0.3}},
			{"id": "vec2", "values": []float32{0.4, 0.5, 0.6}},
		},
	}
	w = makeRequest(t, srv, "POST", "/api/vectorize/indexes/test-query-index/insert", insertInput)
	if w.Code != http.StatusOK {
		t.Fatalf("Failed to insert vectors: %s", w.Body.String())
	}

	// Query vectors
	queryInput := map[string]interface{}{
		"vector": []float32{0.1, 0.2, 0.3},
		"topK":   2,
	}
	w = makeRequest(t, srv, "POST", "/api/vectorize/indexes/test-query-index/query", queryInput)
	if w.Code != http.StatusOK {
		t.Fatalf("Failed to query: %s", w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

// ========== Analytics Engine Tests ==========

func TestAnalyticsEngine_ListDatasets(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/analytics-engine/datasets", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	datasets, ok := resp.Result["datasets"].([]interface{})
	if !ok {
		t.Fatal("Expected result.datasets to be an array")
	}

	if len(datasets) == 0 {
		t.Fatal("Expected at least one dataset from seeded data")
	}
}

func TestAnalyticsEngine_CreateDataset(t *testing.T) {
	srv := testServer(t)

	input := map[string]interface{}{
		"name": "test-dataset",
	}

	w := makeRequest(t, srv, "POST", "/api/analytics-engine/datasets", input)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestAnalyticsEngine_WriteAndQuery(t *testing.T) {
	srv := testServer(t)

	// Create a dataset
	createInput := map[string]interface{}{
		"name": "test-write-dataset",
	}
	w := makeRequest(t, srv, "POST", "/api/analytics-engine/datasets", createInput)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create dataset: %s", w.Body.String())
	}

	// Write data points
	writeInput := map[string]interface{}{
		"data_points": []map[string]interface{}{
			{
				"indexes": []string{"page_view", "home"},
				"doubles": []float64{1.0, 100.0},
			},
		},
	}
	w = makeRequest(t, srv, "POST", "/api/analytics-engine/datasets/test-write-dataset/write", writeInput)
	if w.Code != http.StatusOK {
		t.Fatalf("Failed to write data points: %s", w.Body.String())
	}
}

// ========== AI Gateway Tests ==========

func TestAIGateway_List(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/ai-gateway", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	gateways, ok := resp.Result["gateways"].([]interface{})
	if !ok {
		t.Fatal("Expected result.gateways to be an array")
	}

	if len(gateways) == 0 {
		t.Fatal("Expected at least one gateway from seeded data")
	}
}

func TestAIGateway_Create(t *testing.T) {
	srv := testServer(t)

	input := map[string]interface{}{
		"name":          "test-gateway",
		"collect_logs":  true,
		"cache_enabled": true,
		"cache_ttl":     3600,
	}

	w := makeRequest(t, srv, "POST", "/api/ai-gateway", input)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestAIGateway_GetLogs(t *testing.T) {
	srv := testServer(t)

	// First get list to find a gateway
	w := makeRequest(t, srv, "GET", "/api/ai-gateway", nil)
	resp := parseResponse(t, w)
	gateways := resp.Result["gateways"].([]interface{})
	if len(gateways) == 0 {
		t.Skip("No gateways to test")
	}

	gw := gateways[0].(map[string]interface{})
	id := gw["id"].(string)

	// Get logs
	w = makeRequest(t, srv, "GET", "/api/ai-gateway/"+id+"/logs", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp = parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	// Verify logs structure
	_, ok := resp.Result["logs"].([]interface{})
	if !ok {
		t.Fatal("Expected result.logs to be an array")
	}
}

// ========== Hyperdrive Tests ==========

func TestHyperdrive_List(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/hyperdrive/configs", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	configs, ok := resp.Result["configs"].([]interface{})
	if !ok {
		t.Fatal("Expected result.configs to be an array")
	}

	if len(configs) == 0 {
		t.Fatal("Expected at least one config from seeded data")
	}
}

func TestHyperdrive_Create(t *testing.T) {
	srv := testServer(t)

	input := map[string]interface{}{
		"name": "test-hyperdrive",
		"origin": map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"database": "testdb",
			"user":     "testuser",
			"scheme":   "postgres",
		},
	}

	w := makeRequest(t, srv, "POST", "/api/hyperdrive/configs", input)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestHyperdrive_GetStats(t *testing.T) {
	srv := testServer(t)

	// First get list to find a config
	w := makeRequest(t, srv, "GET", "/api/hyperdrive/configs", nil)
	resp := parseResponse(t, w)
	configs := resp.Result["configs"].([]interface{})
	if len(configs) == 0 {
		t.Skip("No configs to test")
	}

	cfg := configs[0].(map[string]interface{})
	id := cfg["id"].(string)

	// Get stats
	w = makeRequest(t, srv, "GET", "/api/hyperdrive/configs/"+id+"/stats", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// ========== Cron Tests ==========

func TestCron_List(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/cron/triggers", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	triggers, ok := resp.Result["triggers"].([]interface{})
	if !ok {
		t.Fatal("Expected result.triggers to be an array")
	}

	if len(triggers) == 0 {
		t.Fatal("Expected at least one trigger from seeded data")
	}
}

func TestCron_Create(t *testing.T) {
	srv := testServer(t)

	input := map[string]interface{}{
		"cron":        "*/5 * * * *",
		"script_name": "test-worker",
		"enabled":     true,
	}

	w := makeRequest(t, srv, "POST", "/api/cron/triggers", input)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}
}

func TestCron_GetExecutions(t *testing.T) {
	srv := testServer(t)

	// First get list to find a trigger
	w := makeRequest(t, srv, "GET", "/api/cron/triggers", nil)
	resp := parseResponse(t, w)
	triggers := resp.Result["triggers"].([]interface{})
	if len(triggers) == 0 {
		t.Skip("No triggers to test")
	}

	trigger := triggers[0].(map[string]interface{})
	id := trigger["id"].(string)

	// Get executions
	w = makeRequest(t, srv, "GET", "/api/cron/triggers/"+id+"/executions", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp = parseResponse(t, w)
	if !resp.Success {
		t.Fatal("Expected success=true")
	}

	_, ok := resp.Result["executions"].([]interface{})
	if !ok {
		t.Fatal("Expected result.executions to be an array")
	}
}

// ========== Response Structure Tests ==========

func TestResponseStructure_DurableObjects(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/durable-objects/namespaces", nil)

	var raw map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &raw)

	if _, ok := raw["success"]; !ok {
		t.Error("Response missing 'success' field")
	}
	if _, ok := raw["result"]; !ok {
		t.Error("Response missing 'result' field")
	}

	result := raw["result"].(map[string]interface{})
	if _, ok := result["namespaces"]; !ok {
		t.Error("Result missing 'namespaces' field")
	}
}

func TestResponseStructure_Queues(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/queues", nil)

	var raw map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &raw)

	result := raw["result"].(map[string]interface{})
	if _, ok := result["queues"]; !ok {
		t.Error("Result missing 'queues' field")
	}
}

func TestResponseStructure_Vectorize(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/vectorize/indexes", nil)

	var raw map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &raw)

	result := raw["result"].(map[string]interface{})
	if _, ok := result["indexes"]; !ok {
		t.Error("Result missing 'indexes' field")
	}
}

func TestResponseStructure_AnalyticsEngine(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/analytics-engine/datasets", nil)

	var raw map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &raw)

	result := raw["result"].(map[string]interface{})
	if _, ok := result["datasets"]; !ok {
		t.Error("Result missing 'datasets' field")
	}
}

func TestResponseStructure_AIGateway(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/ai-gateway", nil)

	var raw map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &raw)

	result := raw["result"].(map[string]interface{})
	if _, ok := result["gateways"]; !ok {
		t.Error("Result missing 'gateways' field")
	}
}

func TestResponseStructure_Hyperdrive(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/hyperdrive/configs", nil)

	var raw map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &raw)

	result := raw["result"].(map[string]interface{})
	if _, ok := result["configs"]; !ok {
		t.Error("Result missing 'configs' field")
	}
}

func TestResponseStructure_Cron(t *testing.T) {
	srv := testServer(t)
	w := makeRequest(t, srv, "GET", "/api/cron/triggers", nil)

	var raw map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &raw)

	result := raw["result"].(map[string]interface{})
	if _, ok := result["triggers"]; !ok {
		t.Error("Result missing 'triggers' field (got 'schedules' which is incorrect)")
	}
}

// ========== Integration Tests ==========

func TestDashboard_WithSeededData(t *testing.T) {
	srv := testServer(t)

	// Verify dashboard stats include seeded data
	w := makeRequest(t, srv, "GET", "/api/dashboard/stats", nil)
	resp := parseResponse(t, w)

	// Check that DO stats are populated
	doStats := resp.Result["durable_objects"].(map[string]interface{})
	namespaces := doStats["namespaces"].(float64)
	if namespaces == 0 {
		t.Error("Expected seeded DO namespaces in dashboard stats")
	}

	// Check that queues stats are populated
	queueStats := resp.Result["queues"].(map[string]interface{})
	queueCount := queueStats["count"].(float64)
	if queueCount == 0 {
		t.Error("Expected seeded queues in dashboard stats")
	}

	// Check that vectorize stats are populated
	vectorStats := resp.Result["vectorize"].(map[string]interface{})
	indexCount := vectorStats["indexes"].(float64)
	if indexCount == 0 {
		t.Error("Expected seeded vector indexes in dashboard stats")
	}
}

// TestMain ensures test database cleanup
func TestMain(m *testing.M) {
	// Create temp directory for all tests
	tmpDir, err := os.MkdirTemp("", "localflare-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set working directory to project root for asset loading
	// Note: Tests may need to be run from project root
	os.Exit(m.Run())
}

// Ensure temp dirs are cleaned up
func init() {
	// Clean up any leftover test directories
	pattern := filepath.Join(os.TempDir(), "localflare-test-*")
	matches, _ := filepath.Glob(pattern)
	for _, m := range matches {
		os.RemoveAll(m)
	}
}
