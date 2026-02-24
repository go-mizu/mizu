package qlocal

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type MCPServer struct {
	App       *App
	started   time.Time
	sessionMu sync.Mutex
	sessions  map[string]struct{}
}

type MCPDaemonStatus struct {
	Running bool   `json:"running"`
	PID     int    `json:"pid,omitempty"`
	PIDPath string `json:"pidPath"`
}

func NewMCPServer(app *App) *MCPServer {
	return &MCPServer{App: app, started: time.Now(), sessions: map[string]struct{}{}}
}

func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/health":
		uptime := time.Since(s.started).Seconds()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":     true,
			"status": "ok",
			"uptime": uptime,
		})
		return
	case (r.Method == http.MethodGet || r.Method == http.MethodHead) && r.URL.Path == "/mcp":
		s.maybeEchoSessionHeader(w, r)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc":   "2.0",
			"transport": "streamable-http-subset",
			"server":    "qlocal",
		})
		return
	case r.Method == http.MethodPost && r.URL.Path == "/mcp":
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeRPCResponse(w, rpcResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &rpcError{Code: -32700, Message: "parse error: " + err.Error()},
			})
			return
		}
		s.ensureHTTPMCPResponseSession(w, r, req)
		resp := s.handleRPC(r.Context(), req)
		writeRPCResponse(w, resp)
		return
	case r.Method == http.MethodPost && r.URL.Path == "/query":
		s.serveLegacyQueryEndpoint(w, r)
		return
	default:
		http.NotFound(w, r)
	}
}

func writeRPCResponse(w http.ResponseWriter, resp rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func StartMCPHTTPServer(ctx context.Context, app *App, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: NewMCPServer(app),
	}
	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	select {
	case <-ctx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func ServeMCPStdio(ctx context.Context, app *App, r io.Reader, w io.Writer) error {
	server := NewMCPServer(app)
	br := bufio.NewReader(r)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, err := readStdioRPCMessage(br)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req rpcRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			resp := rpcResponse{JSONRPC: "2.0", ID: nil, Error: &rpcError{Code: -32700, Message: "parse error"}}
			if err := writeStdioRPCMessage(w, resp); err != nil {
				return err
			}
			continue
		}
		resp := server.handleRPC(ctx, req)
		// Notifications do not require response.
		if req.ID == nil {
			continue
		}
		if err := writeStdioRPCMessage(w, resp); err != nil {
			return err
		}
	}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *MCPServer) handleRPC(ctx context.Context, req rpcRequest) rpcResponse {
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2025-06-18",
			"serverInfo": map[string]any{
				"name":    "qlocal",
				"version": "0.1.0",
			},
			"capabilities": map[string]any{
				"tools":     map[string]any{"listChanged": false},
				"resources": map[string]any{"listChanged": false},
			},
			"instructions": s.buildInstructions(ctx),
		}
	case "notifications/initialized":
		// no response required for notification
		if req.ID == nil {
			return rpcResponse{}
		}
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": s.toolsList()}
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
			ArgsAlias map[string]any `json:"args"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: "invalid params"}
			break
		}
		if len(p.Arguments) == 0 && len(p.ArgsAlias) > 0 {
			p.Arguments = p.ArgsAlias
		}
		result, err := s.callTool(ctx, p.Name, p.Arguments)
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			break
		}
		resp.Result = result
	case "resources/list":
		resp.Result = map[string]any{
			"resources": []map[string]any{
				{
					"uriTemplate": "qmd://{+path}",
					"name":        "qmd documents",
					"description": "Indexed markdown documents addressable by qmd://collection/path",
					"mimeType":    "text/markdown",
				},
			},
		}
	case "resources/read":
		var p struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil || p.URI == "" {
			resp.Error = &rpcError{Code: -32602, Message: "invalid params"}
			break
		}
		uri := normalizeMCPResourceURI(p.URI)
		doc, err := s.App.Get(uri, GetOptions{Full: true})
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			break
		}
		text := addLineNumbers(doc.Body, 1)
		if doc.Context != "" {
			text = "<!-- Context: " + doc.Context + " -->\n\n" + text
		}
		resp.Result = map[string]any{
			"contents": []map[string]any{
				{
					"uri":      p.URI,
					"name":     doc.DisplayPath,
					"title":    doc.Title,
					"mimeType": "text/markdown",
					"text":     text,
				},
			},
		}
	case "shutdown":
		resp.Result = map[string]any{}
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

func (s *MCPServer) buildInstructions(ctx context.Context) string {
	st, err := s.App.Status()
	if err != nil {
		return "qlocal local markdown search MCP server"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Qlocal indexes %d markdown documents across %d collections.\n", st.TotalDocuments, len(st.Collections))
	if st.HasVectorIndex {
		b.WriteString("Vector search is available.\n")
	} else {
		b.WriteString("Vector search needs embeddings. Run `search local embed`.\n")
	}
	names, _ := s.App.DefaultCollectionNames()
	if len(names) > 0 {
		b.WriteString("Default collections: " + strings.Join(names, ", ") + "\n")
	}
	b.WriteString("Use qmd_search / qmd_vector_search / qmd_deep_search and qmd_get / qmd_multi_get.\n")
	_ = ctx
	return strings.TrimSpace(b.String())
}

func (s *MCPServer) toolsList() []map[string]any {
	tool := func(name, title, desc string, schema map[string]any) map[string]any {
		return map[string]any{
			"name":        name,
			"title":       title,
			"description": desc,
			"inputSchema": schema,
		}
	}
	querySchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":      map[string]any{"type": "string"},
			"limit":      map[string]any{"type": "number"},
			"minScore":   map[string]any{"type": "number"},
			"collection": map[string]any{"type": "string"},
		},
		"required": []string{"query"},
	}
	deepSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
			"searches": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"type":  map[string]any{"type": "string"},
						"query": map[string]any{"type": "string"},
					},
				},
			},
			"limit":      map[string]any{"type": "number"},
			"minScore":   map[string]any{"type": "number"},
			"collection": map[string]any{"type": "string"},
		},
	}
	return []map[string]any{
		tool("qmd_search", "QMD Search", "Fast BM25 keyword search", querySchema),
		tool("qmd_vector_search", "QMD Vector Search", "Semantic vector search", querySchema),
		tool("qmd_deep_search", "QMD Deep Search", "Hybrid search with typed subqueries", deepSchema),
		tool("qmd_get", "QMD Get", "Retrieve a document by path or docid", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ref":         map[string]any{"type": "string"},
				"file":        map[string]any{"type": "string"},
				"path":        map[string]any{"type": "string"},
				"lineNumbers": map[string]any{"type": "boolean"},
				"from":        map[string]any{"type": "number"},
				"lines":       map[string]any{"type": "number"},
			},
			"anyOf": []map[string]any{
				{"required": []string{"ref"}},
				{"required": []string{"file"}},
				{"required": []string{"path"}},
			},
		}),
		tool("qmd_multi_get", "QMD Multi Get", "Retrieve multiple docs by glob/list", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern":  map[string]any{"type": "string"},
				"maxBytes": map[string]any{"type": "number"},
				"lines":    map[string]any{"type": "number"},
				"format":   map[string]any{"type": "string"},
			},
			"required": []string{"pattern"},
		}),
		tool("qmd_status", "QMD Status", "Index health and collection info", map[string]any{"type": "object"}),
		// Aliases closer to qmd.ts MCP server implementation
		tool("query", "Query", "Alias of qmd_deep_search", deepSchema),
		tool("get", "Get", "Alias of qmd_get", map[string]any{"type": "object", "properties": map[string]any{
			"ref":  map[string]any{"type": "string"},
			"file": map[string]any{"type": "string"},
			"path": map[string]any{"type": "string"},
		}}),
		tool("multi_get", "Multi Get", "Alias of qmd_multi_get", map[string]any{"type": "object", "properties": map[string]any{"pattern": map[string]any{"type": "string"}}}),
		tool("status", "Status", "Alias of qmd_status", map[string]any{"type": "object"}),
	}
}

func (s *MCPServer) callTool(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "qmd_search":
		query := asString(args["query"])
		results, err := s.App.SearchFTS(query, SearchOptions{
			Limit:       asInt(args["limit"], 10),
			MinScore:    asFloat(args["minScore"], 0),
			Collections: singleCollectionArg(args),
			IncludeBody: true,
		})
		if err != nil {
			return nil, err
		}
		return s.formatToolSearchResult(results, query)
	case "qmd_vector_search":
		query := asString(args["query"])
		results, err := s.App.VectorSearch(query, SearchOptions{
			Limit:       asInt(args["limit"], 10),
			MinScore:    asFloat(args["minScore"], 0.3),
			Collections: singleCollectionArg(args),
			IncludeBody: true,
		})
		if err != nil {
			return nil, err
		}
		return s.formatToolSearchResult(results, query)
	case "qmd_deep_search", "query":
		if searchesArg, ok := args["searches"]; ok {
			if arr, err := coerceStructuredSearches(searchesArg); err == nil && len(arr) > 0 {
				queryText := ""
				if q, ok2 := args["query"]; ok2 {
					queryText = asString(q)
				}
				if queryText == "" && len(arr) > 0 {
					queryText = arr[0].Query
				}
				// Execute typed search document via existing parser path.
				var lines []string
				for _, ssub := range arr {
					lines = append(lines, ssub.Type+": "+ssub.Query)
				}
				results, err := s.App.QueryContext(ctx, strings.Join(lines, "\n"), HybridOptions{
					Limit:       asInt(args["limit"], 10),
					MinScore:    asFloat(args["minScore"], 0),
					Collections: singleCollectionArg(args),
				})
				if err != nil {
					return nil, err
				}
				return s.formatToolSearchResult(results, queryText)
			}
		}
		query := asString(args["query"])
		results, err := s.App.QueryContext(ctx, query, HybridOptions{
			Limit:       asInt(args["limit"], 10),
			MinScore:    asFloat(args["minScore"], 0),
			Collections: singleCollectionArg(args),
		})
		if err != nil {
			return nil, err
		}
		return s.formatToolSearchResult(results, query)
	case "qmd_get", "get":
		ref := asString(args["ref"])
		if ref == "" {
			ref = asString(args["file"])
		}
		if ref == "" {
			ref = asString(args["path"])
		}
		doc, err := s.App.Get(ref, GetOptions{
			Full:        true,
			FromLine:    asInt(args["from"], 0),
			MaxLines:    asInt(args["lines"], 0),
			LineNumbers: asBool(args["lineNumbers"]),
		})
		if err != nil {
			return nil, err
		}
		payload, _ := json.Marshal(doc)
		return map[string]any{
			"content":           []map[string]any{{"type": "text", "text": doc.DisplayPath}},
			"structuredContent": map[string]any{"document": json.RawMessage(payload)},
		}, nil
	case "qmd_multi_get", "multi_get":
		pattern := asString(args["pattern"])
		results, errs, err := s.App.MultiGet(pattern, asInt(args["lines"], 0), asInt(args["maxBytes"], DefaultMultiGetMaxBytes), true)
		if err != nil {
			return nil, err
		}
		format := OutputJSON
		if f := strings.TrimSpace(asString(args["format"])); f != "" {
			format = OutputFormat(f)
		}
		text, _ := FormatMultiGet(results, format, false)
		var structured any = map[string]any{"results": results, "errors": errs}
		if b, err := json.Marshal(structured); err == nil {
			structured = json.RawMessage(b)
		}
		return map[string]any{
			"content":           []map[string]any{{"type": "text", "text": text}},
			"structuredContent": structured,
		}, nil
	case "qmd_status", "status":
		st, err := s.App.Status()
		if err != nil {
			return nil, err
		}
		b, _ := json.Marshal(st)
		return map[string]any{
			"content":           []map[string]any{{"type": "text", "text": "qlocal status"}},
			"structuredContent": json.RawMessage(b),
		}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *MCPServer) formatToolSearchResult(results []SearchResult, query string) (map[string]any, error) {
	out := s.buildSearchItems(results, query)
	text := fmt.Sprintf("Found %d result(s) for %q", len(out), query)
	if b, err := json.Marshal(map[string]any{"results": out}); err == nil {
		return map[string]any{
			"content":           []map[string]any{{"type": "text", "text": text}},
			"structuredContent": json.RawMessage(b),
		}, nil
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}, nil
}

func (s *MCPServer) buildSearchItems(results []SearchResult, query string) []map[string]any {
	type item struct {
		DocID   string  `json:"docid"`
		File    string  `json:"file"`
		Title   string  `json:"title"`
		Score   float64 `json:"score"`
		Context string  `json:"context,omitempty"`
		Snippet string  `json:"snippet"`
	}
	out := make([]item, 0, len(results))
	for _, r := range results {
		body := r.Body
		if body == "" {
			body = r.ChunkText
		}
		sn := extractSnippet(body, query, 300, r.ChunkPos)
		out = append(out, item{
			DocID:   "#" + r.DocID,
			File:    r.DisplayPath,
			Title:   r.Title,
			Score:   round2(r.Score),
			Context: r.Context,
			Snippet: addLineNumbers(sn.Snippet, sn.Line),
		})
	}
	items := make([]map[string]any, 0, len(out))
	for _, it := range out {
		items = append(items, map[string]any{
			"docid":   it.DocID,
			"file":    it.File,
			"title":   it.Title,
			"score":   it.Score,
			"context": it.Context,
			"snippet": it.Snippet,
		})
	}
	_ = query
	return items
}

func singleCollectionArg(args map[string]any) []string {
	if v := asString(args["collection"]); v != "" {
		return []string{v}
	}
	if raw, ok := args["collections"]; ok {
		switch t := raw.(type) {
		case []any:
			var out []string
			for _, v := range t {
				if s := asString(v); s != "" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}

func coerceStructuredSearches(v any) ([]StructuredSubSearch, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var arr []StructuredSubSearch
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, err
	}
	for i := range arr {
		arr[i].Type = strings.TrimSpace(arr[i].Type)
		arr[i].Query = strings.TrimSpace(arr[i].Query)
	}
	return arr, nil
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
}

func normalizeMCPResourceURI(in string) string {
	in = strings.TrimSpace(in)
	if !strings.HasPrefix(in, "qmd://") {
		return in
	}
	raw := strings.TrimPrefix(in, "qmd://")
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return in
	}
	return "qmd://" + decoded
}

func (s *MCPServer) maybeEchoSessionHeader(w http.ResponseWriter, r *http.Request) {
	if sid := strings.TrimSpace(r.Header.Get("mcp-session-id")); sid != "" {
		w.Header().Set("mcp-session-id", sid)
	}
}

func (s *MCPServer) ensureHTTPMCPResponseSession(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	// Echo an existing session id if present, otherwise generate one on initialize.
	if sid := strings.TrimSpace(r.Header.Get("mcp-session-id")); sid != "" {
		s.trackSession(sid)
		w.Header().Set("mcp-session-id", sid)
		return
	}
	if req.Method != "initialize" {
		return
	}
	sid := newMCPHTTPSessionID()
	s.trackSession(sid)
	w.Header().Set("mcp-session-id", sid)
}

func (s *MCPServer) trackSession(id string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.sessions[id] = struct{}{}
}

func newMCPHTTPSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("qlocal-%d", time.Now().UnixNano())
	}
	return "qlocal-" + hex.EncodeToString(b[:])
}

func (s *MCPServer) serveLegacyQueryEndpoint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Searches    any      `json:"searches"`
		Collections []string `json:"collections"`
		Limit       int      `json:"limit"`
		MinScore    float64  `json:"minScore"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Invalid JSON"})
		return
	}
	if body.Searches == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Missing required field: searches (array)"})
		return
	}
	searches, err := coerceStructuredSearches(body.Searches)
	if err != nil || len(searches) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Invalid field: searches"})
		return
	}
	var lines []string
	for _, ss := range searches {
		lines = append(lines, strings.ToLower(strings.TrimSpace(ss.Type))+": "+strings.TrimSpace(ss.Query))
	}
	primary := ""
	for _, ss := range searches {
		if ss.Type == "lex" || ss.Type == "vec" {
			primary = ss.Query
			break
		}
	}
	if primary == "" {
		primary = searches[0].Query
	}
	limit := body.Limit
	if limit <= 0 {
		limit = 10
	}
	results, err := s.App.QueryContext(r.Context(), strings.Join(lines, "\n"), HybridOptions{
		Limit:       limit,
		MinScore:    body.MinScore,
		Collections: body.Collections,
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": s.buildSearchItems(results, primary)})
}

func asInt(v any, fallback int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n
		}
	}
	return fallback
}

func asFloat(v any, fallback float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		if n, err := t.Float64(); err == nil {
			return n
		}
	case string:
		if n, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return n
		}
	}
	return fallback
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, _ := strconv.ParseBool(strings.TrimSpace(t))
		return b
	default:
		return false
	}
}

func readStdioRPCMessage(br *bufio.Reader) ([]byte, error) {
	var contentLen int
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			if n, err := strconv.Atoi(v); err == nil {
				contentLen = n
			}
		}
	}
	if contentLen <= 0 {
		return nil, fmt.Errorf("invalid Content-Length")
	}
	buf := make([]byte, contentLen)
	if _, err := io.ReadFull(br, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeStdioRPCMessage(w io.Writer, resp rpcResponse) error {
	if resp.JSONRPC == "" && resp.ID == nil && resp.Result == nil && resp.Error == nil {
		return nil
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(b))
	if _, err := io.Copy(w, bytes.NewBufferString(header)); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (a *App) MCPStatus() string {
	return "qlocal mcp: available (stdio + HTTP /mcp + /health)"
}

func MCPPIDPathForIndex(indexName string) string {
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		home, _ := os.UserHomeDir()
		cache = filepath.Join(home, ".cache")
	}
	safe := sanitizeIndexName(indexName)
	if safe == "" {
		safe = "index"
	}
	return filepath.Join(cache, "mizu", "qlocal", safe+".mcp.pid")
}

func mcpPIDPath(indexName string) string { return MCPPIDPathForIndex(indexName) }

func GetMCPDaemonStatus(indexName string) MCPDaemonStatus {
	pidPath := MCPPIDPathForIndex(indexName)
	out := MCPDaemonStatus{PIDPath: pidPath}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return out
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	if pid <= 0 {
		return out
	}
	out.PID = pid
	proc, err := os.FindProcess(pid)
	if err != nil {
		return out
	}
	if err := proc.Signal(syscall.Signal(0)); err == nil {
		out.Running = true
	}
	return out
}
