package qlocal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestMCPHTTP_InitializeToolsAndSearch(t *testing.T) {
	env := newTestEnv(t)
	env.writeFile(t, "kb/readme.md", "# KB\n\nCompiler parsing notes and tokenizer details.\n")
	if _, err := env.App.CollectionAdd(env.RootDir+"/kb", "kb", "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewMCPServer(env.App))
	defer srv.Close()

	// health
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	var health map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("health status=%d", resp.StatusCode)
	}
	if health["status"] != "ok" {
		t.Fatalf("health status field=%v want ok", health["status"])
	}
	if _, ok := health["uptime"].(float64); !ok {
		t.Fatalf("health uptime type=%T want number", health["uptime"])
	}

	call := func(method string, params any) map[string]any {
		t.Helper()
		body, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  method,
			"params":  params,
		})
		r, err := http.Post(srv.URL+"/mcp", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		var out map[string]any
		if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if out["error"] != nil {
			t.Fatalf("rpc %s error: %#v", method, out["error"])
		}
		return out
	}

	initResp := call("initialize", map[string]any{})
	if initResp["result"] == nil {
		t.Fatal("initialize missing result")
	}

	toolsResp := call("tools/list", map[string]any{})
	resultObj := toolsResp["result"].(map[string]any)
	tools := resultObj["tools"].([]any)
	if len(tools) < 6 {
		t.Fatalf("expected tools, got %d", len(tools))
	}
	resListResp := call("resources/list", map[string]any{})
	if resListResp["result"] == nil {
		t.Fatal("resources/list missing result")
	}

	searchResp := call("tools/call", map[string]any{
		"name": "qmd_search",
		"arguments": map[string]any{
			"query": "compiler",
			"limit": 5,
		},
	})
	if searchResp["result"] == nil {
		t.Fatal("search tool missing result")
	}
	deepResp := call("tools/call", map[string]any{
		"name": "qmd_deep_search",
		"arguments": map[string]any{
			"searches": []map[string]any{
				{"type": "lex", "query": "compiler"},
				{"type": "vec", "query": "parsing tokenizer"},
			},
			"limit": 5,
		},
	})
	if deepResp["result"] == nil {
		t.Fatal("deep search tool missing result")
	}

	readResp := call("resources/read", map[string]any{
		"uri": "qmd://kb/readme.md",
	})
	result := readResp["result"].(map[string]any)
	contents := result["contents"].([]any)
	if len(contents) != 1 {
		t.Fatalf("expected one resource content, got %d", len(contents))
	}
	txt := contents[0].(map[string]any)["text"].(string)
	if !strings.Contains(txt, "Compiler parsing notes") {
		t.Fatalf("unexpected resource text: %q", txt)
	}
}

func TestMCPHTTP_SessionHeader_QueryEndpoint_GetPathAlias_EncodedResource(t *testing.T) {
	env := newTestEnv(t)
	env.writeFile(t, "kb/readme.md", "# KB\n\nCompiler parsing notes and tokenizer details.\n")
	env.writeFile(t, "kb/space file.md", "# Space\n\nEncoded resource path test.\n")
	if _, err := env.App.CollectionAdd(env.RootDir+"/kb", "kb", "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewMCPServer(env.App))
	defer srv.Close()

	postJSON := func(path string, headers map[string]string, payload any) (*http.Response, map[string]any) {
		t.Helper()
		b, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		data, _ := io.ReadAll(res.Body)
		if len(data) > 0 {
			if err := json.Unmarshal(data, &out); err != nil {
				t.Fatalf("decode %s response: %v body=%q", path, err, string(data))
			}
		}
		return res, out
	}

	// GET /mcp should be accepted in streamable-http subset mode.
	getResp, err := http.Get(srv.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("GET /mcp status=%d want 200", getResp.StatusCode)
	}
	getResp.Body.Close()

	initRes, initBody := postJSON("/mcp", map[string]string{
		"Accept": "application/json, text/event-stream",
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
		},
	})
	if initRes.StatusCode != 200 {
		t.Fatalf("initialize status=%d want 200", initRes.StatusCode)
	}
	sid := initRes.Header.Get("mcp-session-id")
	if strings.TrimSpace(sid) == "" {
		t.Fatal("initialize missing mcp-session-id header")
	}
	if initBody["result"] == nil {
		t.Fatalf("initialize missing result: %#v", initBody)
	}

	listRes, listBody := postJSON("/mcp", map[string]string{
		"Accept":         "application/json, text/event-stream",
		"mcp-session-id": sid,
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if got := listRes.Header.Get("mcp-session-id"); got != sid {
		t.Fatalf("tools/list session header=%q want %q", got, sid)
	}
	if listBody["result"] == nil {
		t.Fatalf("tools/list missing result: %#v", listBody)
	}

	// qmd-compatible /query convenience endpoint
	qRes, qBody := postJSON("/query", nil, map[string]any{
		"searches": []map[string]any{
			{"type": "lex", "query": "compiler"},
			{"type": "vec", "query": "tokenizer parsing"},
		},
		"limit": 5,
	})
	if qRes.StatusCode != 200 {
		t.Fatalf("/query status=%d want 200 body=%#v", qRes.StatusCode, qBody)
	}
	if _, ok := qBody["results"].([]any); !ok {
		t.Fatalf("/query results missing: %#v", qBody)
	}

	// qmd_get/get path alias support (`path`, not only `ref`/`file`)
	getToolRes, getToolBody := postJSON("/mcp", map[string]string{
		"mcp-session-id": sid,
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get",
			"arguments": map[string]any{
				"path": "kb/readme.md",
			},
		},
	})
	if getToolRes.StatusCode != 200 {
		t.Fatalf("tools/call get status=%d want 200 body=%#v", getToolRes.StatusCode, getToolBody)
	}
	if getToolBody["error"] != nil {
		t.Fatalf("tools/call get returned rpc error: %#v", getToolBody["error"])
	}

	// URL-encoded qmd:// resource paths (common MCP client behavior)
	encRes, encBody := postJSON("/mcp", map[string]string{
		"mcp-session-id": sid,
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "resources/read",
		"params": map[string]any{
			"uri": "qmd://kb/space%20file.md",
		},
	})
	if encRes.StatusCode != 200 {
		t.Fatalf("resources/read encoded status=%d want 200 body=%#v", encRes.StatusCode, encBody)
	}
	if encBody["error"] != nil {
		t.Fatalf("resources/read encoded returned rpc error: %#v", encBody["error"])
	}
}

func TestMCPStdio_ContentLengthFraming(t *testing.T) {
	env := newTestEnv(t)
	inReq, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})
	var in bytes.Buffer
	in.WriteString("Content-Length: ")
	in.WriteString(strconv.Itoa(len(inReq)))
	in.WriteString("\r\n\r\n")
	in.Write(inReq)

	var out bytes.Buffer
	if err := ServeMCPStdio(context.Background(), env.App, &in, &out); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(&out)
	msg, err := readStdioRPCMessage(br)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != nil || resp["result"] == nil {
		t.Fatalf("unexpected stdio rpc response: %#v", resp)
	}
}
