package qlocal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("health status=%d", resp.StatusCode)
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
