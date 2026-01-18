package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/oklog/ulid/v2"
)

// PlaygroundHandler handles API playground endpoints.
type PlaygroundHandler struct {
	store   *postgres.Store
	history *RequestHistory
}

// NewPlaygroundHandler creates a new playground handler.
func NewPlaygroundHandler(store *postgres.Store) *PlaygroundHandler {
	return &PlaygroundHandler{
		store:   store,
		history: NewRequestHistory(100),
	}
}

// EndpointCategory represents a category of API endpoints.
type EndpointCategory struct {
	Name        string     `json:"name"`
	Icon        string     `json:"icon"`
	Description string     `json:"description"`
	Endpoints   []Endpoint `json:"endpoints"`
}

// Endpoint represents an API endpoint.
type Endpoint struct {
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Parameters  []Parameter    `json:"parameters,omitempty"`
	RequestBody map[string]any `json:"requestBody,omitempty"`
	Example     string         `json:"example,omitempty"`
}

// Parameter represents a query parameter.
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
	Example     string `json:"example,omitempty"`
}

// ExecuteRequest represents a request to execute.
type ExecuteRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Query   map[string]string `json:"query"`
	Body    json.RawMessage   `json:"body"`
}

// ExecuteResponse represents the response from an executed request.
type ExecuteResponse struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
	DurationMs int64             `json:"duration_ms"`
}

// RequestHistoryEntry represents a saved request in history.
type RequestHistoryEntry struct {
	ID         string          `json:"id"`
	Method     string          `json:"method"`
	Path       string          `json:"path"`
	Status     int             `json:"status"`
	DurationMs int64           `json:"duration_ms"`
	Timestamp  string          `json:"timestamp"`
	Request    ExecuteRequest  `json:"request"`
	Response   ExecuteResponse `json:"response"`
}

// RequestHistory manages request history with a fixed size.
type RequestHistory struct {
	mu      sync.RWMutex
	entries []RequestHistoryEntry
	maxSize int
}

// NewRequestHistory creates a new request history manager.
func NewRequestHistory(maxSize int) *RequestHistory {
	return &RequestHistory{
		entries: make([]RequestHistoryEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a new entry to the history.
func (h *RequestHistory) Add(entry RequestHistoryEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to front
	h.entries = append([]RequestHistoryEntry{entry}, h.entries...)

	// Trim if needed
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[:h.maxSize]
	}
}

// List returns all history entries.
func (h *RequestHistory) List(limit, offset int) []RequestHistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if offset >= len(h.entries) {
		return []RequestHistoryEntry{}
	}

	end := offset + limit
	if end > len(h.entries) {
		end = len(h.entries)
	}

	return h.entries[offset:end]
}

// Clear removes all history entries.
func (h *RequestHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = make([]RequestHistoryEntry, 0, h.maxSize)
}

// TableInfo represents table information for dynamic endpoints.
type TableInfo struct {
	Schema     string       `json:"schema"`
	Name       string       `json:"name"`
	Columns    []ColumnInfo `json:"columns"`
	RLSEnabled bool         `json:"rls_enabled"`
}

// ColumnInfo represents column information.
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	IsNullable   bool   `json:"is_nullable"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// FunctionInfo represents database function information.
type FunctionInfo struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	Arguments  string `json:"arguments,omitempty"`
	ReturnType string `json:"return_type"`
}

// GetEndpoints returns all available API endpoints.
func (h *PlaygroundHandler) GetEndpoints(c *mizu.Ctx) error {
	categories := []EndpointCategory{
		{
			Name:        "Authentication",
			Icon:        "shield",
			Description: "User authentication and session management",
			Endpoints: []Endpoint{
				{Method: "POST", Path: "/auth/v1/signup", Description: "Register a new user", Category: "Authentication", RequestBody: map[string]any{"email": "user@example.com", "password": "password123"}},
				{Method: "POST", Path: "/auth/v1/token?grant_type=password", Description: "Sign in with email/password", Category: "Authentication", RequestBody: map[string]any{"email": "user@example.com", "password": "password123"}},
				{Method: "POST", Path: "/auth/v1/logout", Description: "Sign out the current user", Category: "Authentication"},
				{Method: "GET", Path: "/auth/v1/user", Description: "Get current authenticated user", Category: "Authentication"},
				{Method: "PUT", Path: "/auth/v1/user", Description: "Update the current user", Category: "Authentication", RequestBody: map[string]any{"data": map[string]any{"display_name": "John Doe"}}},
				{Method: "POST", Path: "/auth/v1/recover", Description: "Send password recovery email", Category: "Authentication", RequestBody: map[string]any{"email": "user@example.com"}},
				{Method: "POST", Path: "/auth/v1/otp", Description: "Send OTP code", Category: "Authentication", RequestBody: map[string]any{"email": "user@example.com"}},
				{Method: "POST", Path: "/auth/v1/verify", Description: "Verify OTP/magic link", Category: "Authentication", RequestBody: map[string]any{"type": "signup", "token": "..."}},
				{Method: "GET", Path: "/auth/v1/admin/users", Description: "List all users (service_role)", Category: "Authentication"},
				{Method: "POST", Path: "/auth/v1/admin/users", Description: "Create a user (service_role)", Category: "Authentication", RequestBody: map[string]any{"email": "new@example.com", "password": "password123", "email_confirm": true}},
			},
		},
		{
			Name:        "Database",
			Icon:        "database",
			Description: "CRUD operations on database tables",
			Endpoints: []Endpoint{
				{Method: "GET", Path: "/rest/v1/{table}", Description: "Select rows from a table", Category: "Database", Parameters: []Parameter{
					{Name: "select", Type: "string", Description: "Columns to return", Example: "*"},
					{Name: "limit", Type: "integer", Description: "Max rows to return", Example: "10"},
					{Name: "offset", Type: "integer", Description: "Rows to skip", Example: "0"},
					{Name: "order", Type: "string", Description: "Order by column", Example: "created_at.desc"},
				}},
				{Method: "POST", Path: "/rest/v1/{table}", Description: "Insert rows into a table", Category: "Database", RequestBody: map[string]any{"column1": "value1", "column2": "value2"}},
				{Method: "PATCH", Path: "/rest/v1/{table}?id=eq.{id}", Description: "Update rows matching filter", Category: "Database", RequestBody: map[string]any{"column1": "newvalue"}},
				{Method: "DELETE", Path: "/rest/v1/{table}?id=eq.{id}", Description: "Delete rows matching filter", Category: "Database"},
				{Method: "POST", Path: "/rest/v1/rpc/{function}", Description: "Call a PostgreSQL function", Category: "Database", RequestBody: map[string]any{"param1": "value1"}},
			},
		},
		{
			Name:        "Storage",
			Icon:        "folder",
			Description: "File storage operations",
			Endpoints: []Endpoint{
				{Method: "GET", Path: "/storage/v1/bucket", Description: "List all buckets", Category: "Storage"},
				{Method: "POST", Path: "/storage/v1/bucket", Description: "Create a new bucket", Category: "Storage", RequestBody: map[string]any{"name": "my-bucket", "public": false}},
				{Method: "GET", Path: "/storage/v1/bucket/{id}", Description: "Get bucket details", Category: "Storage"},
				{Method: "PUT", Path: "/storage/v1/bucket/{id}", Description: "Update bucket", Category: "Storage", RequestBody: map[string]any{"public": true}},
				{Method: "DELETE", Path: "/storage/v1/bucket/{id}", Description: "Delete a bucket", Category: "Storage"},
				{Method: "POST", Path: "/storage/v1/object/list/{bucket}", Description: "List objects in bucket", Category: "Storage", RequestBody: map[string]any{"prefix": "", "limit": 100}},
				{Method: "GET", Path: "/storage/v1/object/{bucket}/{path}", Description: "Download an object", Category: "Storage"},
				{Method: "DELETE", Path: "/storage/v1/object/{bucket}/{path}", Description: "Delete an object", Category: "Storage"},
				{Method: "POST", Path: "/storage/v1/object/sign/{bucket}/{path}", Description: "Create signed URL", Category: "Storage", RequestBody: map[string]any{"expiresIn": 3600}},
			},
		},
		{
			Name:        "Edge Functions",
			Icon:        "bolt",
			Description: "Serverless function invocation",
			Endpoints: []Endpoint{
				{Method: "POST", Path: "/functions/v1/{function_name}", Description: "Invoke an edge function", Category: "Edge Functions", RequestBody: map[string]any{"key": "value"}},
				{Method: "GET", Path: "/functions/v1/{function_name}", Description: "GET invoke function", Category: "Edge Functions"},
				{Method: "GET", Path: "/api/functions", Description: "List all functions (service_role)", Category: "Edge Functions"},
				{Method: "POST", Path: "/api/functions", Description: "Create a function (service_role)", Category: "Edge Functions", RequestBody: map[string]any{"name": "my-function", "slug": "my-function"}},
			},
		},
		{
			Name:        "Realtime",
			Icon:        "broadcast",
			Description: "Realtime subscriptions and channels",
			Endpoints: []Endpoint{
				{Method: "GET", Path: "/api/realtime/channels", Description: "List active channels", Category: "Realtime"},
				{Method: "GET", Path: "/api/realtime/stats", Description: "Get realtime statistics", Category: "Realtime"},
			},
		},
		{
			Name:        "Dashboard",
			Icon:        "dashboard",
			Description: "Dashboard and monitoring endpoints",
			Endpoints: []Endpoint{
				{Method: "GET", Path: "/api/dashboard/stats", Description: "Get dashboard statistics", Category: "Dashboard"},
				{Method: "GET", Path: "/api/dashboard/health", Description: "Get health status", Category: "Dashboard"},
				{Method: "GET", Path: "/health", Description: "Basic health check", Category: "Dashboard"},
			},
		},
	}

	return c.JSON(200, map[string]any{
		"categories": categories,
	})
}

// GetTables returns tables with their columns for dynamic REST API docs.
func (h *PlaygroundHandler) GetTables(c *mizu.Ctx) error {
	ctx := c.Context()

	// Get tables from the public schema
	tables, err := h.store.Database().ListTables(ctx, "public")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	result := make([]TableInfo, 0, len(tables))
	for _, t := range tables {
		columns := make([]ColumnInfo, 0)
		if t.Columns != nil {
			for _, col := range t.Columns {
				columns = append(columns, ColumnInfo{
					Name:         col.Name,
					Type:         col.Type,
					IsNullable:   col.IsNullable,
					IsPrimaryKey: col.IsPrimaryKey,
				})
			}
		}
		result = append(result, TableInfo{
			Schema:     t.Schema,
			Name:       t.Name,
			Columns:    columns,
			RLSEnabled: t.RLSEnabled,
		})
	}

	return c.JSON(200, map[string]any{
		"tables": result,
	})
}

// GetFunctions returns available PostgreSQL RPC functions.
func (h *PlaygroundHandler) GetFunctions(c *mizu.Ctx) error {
	ctx := c.Context()

	// Get database functions
	functions, err := h.store.PGMeta().ListDatabaseFunctions(ctx, nil)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	result := make([]FunctionInfo, 0, len(functions))
	for _, fn := range functions {
		// Only include functions from public schema that can be called via RPC
		if fn.Schema == "public" {
			result = append(result, FunctionInfo{
				Schema:     fn.Schema,
				Name:       fn.Name,
				Arguments:  fn.Arguments,
				ReturnType: fn.ReturnType,
			})
		}
	}

	return c.JSON(200, map[string]any{
		"functions": result,
	})
}

// Execute proxies an API request and returns the response.
func (h *PlaygroundHandler) Execute(c *mizu.Ctx) error {
	// Parse request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "failed to read request body"})
	}
	defer c.Request().Body.Close()

	var req ExecuteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	// Validate method
	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	if !validMethods[strings.ToUpper(req.Method)] {
		return c.JSON(400, map[string]string{"error": "invalid HTTP method"})
	}

	// Build query string
	queryParts := make([]string, 0)
	for k, v := range req.Query {
		if v != "" {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, v))
		}
	}

	url := req.Path
	if len(queryParts) > 0 {
		if strings.Contains(url, "?") {
			url += "&" + strings.Join(queryParts, "&")
		} else {
			url += "?" + strings.Join(queryParts, "&")
		}
	}

	// Security: Only allow requests to local endpoints
	if strings.Contains(url, "://") && !strings.HasPrefix(url, "/") {
		return c.JSON(400, map[string]string{"error": "only local endpoints are allowed"})
	}

	// Create the internal request
	start := time.Now()

	var bodyReader io.Reader
	if len(req.Body) > 0 && string(req.Body) != "null" {
		bodyReader = bytes.NewReader(req.Body)
	}

	// Build full URL for the request
	host := c.Request().Host
	fullURL := fmt.Sprintf("http://%s%s", host, url)
	if strings.HasPrefix(url, "/") {
		fullURL = fmt.Sprintf("http://%s%s", host, url)
	}

	clientReq, err := http.NewRequestWithContext(c.Context(), req.Method, fullURL, bodyReader)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	for k, v := range req.Headers {
		clientReq.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(clientReq)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()

	duration := time.Since(start).Milliseconds()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Get response headers
	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	// Build response
	execResp := ExecuteResponse{
		Status:     resp.StatusCode,
		StatusText: resp.Status,
		Headers:    respHeaders,
		Body:       respBody,
		DurationMs: duration,
	}

	// Add to history
	entry := RequestHistoryEntry{
		ID:         ulid.Make().String(),
		Method:     req.Method,
		Path:       url,
		Status:     resp.StatusCode,
		DurationMs: duration,
		Timestamp:  time.Now().Format(time.RFC3339),
		Request:    req,
		Response:   execResp,
	}
	h.history.Add(entry)

	return c.JSON(200, execResp)
}

// GetHistory returns the request history.
func (h *PlaygroundHandler) GetHistory(c *mizu.Ctx) error {
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			offset = v
		}
	}

	entries := h.history.List(limit, offset)

	return c.JSON(200, map[string]any{
		"history": entries,
		"total":   len(entries),
	})
}

// ClearHistory clears the request history.
func (h *PlaygroundHandler) ClearHistory(c *mizu.Ctx) error {
	h.history.Clear()
	return c.JSON(200, map[string]string{"status": "ok"})
}

// SaveHistory saves a request to history.
func (h *PlaygroundHandler) SaveHistory(c *mizu.Ctx) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "failed to read request body"})
	}
	defer c.Request().Body.Close()

	var entry RequestHistoryEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	entry.ID = ulid.Make().String()
	entry.Timestamp = time.Now().Format(time.RFC3339)
	h.history.Add(entry)

	return c.JSON(200, entry)
}

// GetTableDocs returns auto-generated documentation for a specific table.
func (h *PlaygroundHandler) GetTableDocs(c *mizu.Ctx) error {
	schema := c.Param("schema")
	if schema == "" {
		schema = "public"
	}
	tableName := c.Param("table")

	ctx := c.Context()

	// Get table info
	table, err := h.store.Database().GetTable(ctx, schema, tableName)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "table not found"})
	}

	// Build column docs
	columns := make([]map[string]any, 0)
	if table.Columns != nil {
		for _, col := range table.Columns {
			columns = append(columns, map[string]any{
				"name":           col.Name,
				"type":           col.Type,
				"is_nullable":    col.IsNullable,
				"is_primary_key": col.IsPrimaryKey,
				"is_unique":      col.IsUnique,
				"default_value":  col.DefaultValue,
				"comment":        col.Comment,
			})
		}
	}

	// Build endpoints
	basePath := fmt.Sprintf("/rest/v1/%s", tableName)
	endpoints := []map[string]any{
		{
			"method":      "GET",
			"path":        basePath,
			"description": fmt.Sprintf("Select rows from %s", tableName),
			"parameters": []map[string]any{
				{"name": "select", "type": "string", "description": "Columns to return"},
				{"name": "limit", "type": "integer", "description": "Max rows to return"},
				{"name": "offset", "type": "integer", "description": "Rows to skip"},
				{"name": "order", "type": "string", "description": "Order by column (e.g., created_at.desc)"},
			},
		},
		{
			"method":      "POST",
			"path":        basePath,
			"description": fmt.Sprintf("Insert rows into %s", tableName),
		},
		{
			"method":      "PATCH",
			"path":        basePath + "?{filters}",
			"description": fmt.Sprintf("Update rows in %s matching filters", tableName),
		},
		{
			"method":      "DELETE",
			"path":        basePath + "?{filters}",
			"description": fmt.Sprintf("Delete rows from %s matching filters", tableName),
		},
	}

	return c.JSON(200, map[string]any{
		"table":       tableName,
		"schema":      schema,
		"columns":     columns,
		"endpoints":   endpoints,
		"rls_enabled": table.RLSEnabled,
	})
}
