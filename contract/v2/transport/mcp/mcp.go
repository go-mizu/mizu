// Package mcp provides MCP (Model Context Protocol) transport for contract services.
//
// MCP is built on JSON-RPC 2.0 and enables LLM applications to invoke contract
// methods as AI tools. Each contract method becomes an MCP tool with automatic
// JSON Schema generation for inputs.
//
// Tool naming convention:
//   - MCP tool name is "{resource}_{method}"
//     Example: "todos_list", "todos_create", "users_get"
//
// Supported MCP methods:
//   - initialize: Client/server capability negotiation
//   - notifications/initialized: Client confirms initialization
//   - tools/list: List available tools
//   - tools/call: Invoke a tool
package mcp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
)

// protocolVersion is the MCP protocol version we support.
const protocolVersion = "2024-11-05"

// Mount registers an MCP endpoint on a mizu router.
// The endpoint accepts POST requests with MCP JSON-RPC payloads.
func Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error {
	if r == nil {
		return errors.New("mcp: nil router")
	}
	handler, err := Handler(inv, opts...)
	if err != nil {
		return err
	}
	if path == "" {
		path = "/"
	}
	r.Post(path, handler)
	return nil
}

// Handler returns a mizu.Handler for MCP requests.
// This is the primary API when you need direct control.
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error) {
	if inv == nil {
		return nil, errors.New("mcp: nil invoker")
	}
	svc := inv.Descriptor()
	if svc == nil {
		return nil, errors.New("mcp: nil descriptor")
	}

	o := applyOptions(opts)
	tools := buildTools(svc)

	return func(c *mizu.Ctx) error {
		if c.Request().Method != http.MethodPost {
			c.Header().Set("Allow", "POST")
			c.Status(http.StatusMethodNotAllowed)
			_, _ = c.Write([]byte("method not allowed"))
			return nil
		}

		// Read request body with size limit
		body, err := io.ReadAll(io.LimitReader(c.Request().Body, o.maxBodySize+1))
		if err != nil {
			return writeError(c, nil, errParse, "parse error", err.Error())
		}
		if int64(len(body)) > o.maxBodySize {
			return writeError(c, nil, errInvalidRequest, "invalid request", "body too large")
		}

		raw := strings.TrimSpace(string(body))
		if raw == "" {
			return writeError(c, nil, errInvalidRequest, "invalid request", "empty body")
		}

		var req request
		if err := json.Unmarshal(body, &req); err != nil {
			return writeError(c, nil, errParse, "parse error", err.Error())
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			return writeError(c, req.ID, errInvalidRequest, "invalid request", "missing jsonrpc=2.0")
		}

		// Route to appropriate handler
		switch req.Method {
		case "initialize":
			return handleInitialize(c, &req, o)
		case "notifications/initialized":
			return c.NoContent()
		case "tools/list":
			return handleToolsList(c, &req, tools)
		case "tools/call":
			return handleToolsCall(c, &req, inv, svc, o)
		default:
			if !req.hasID() {
				// Notification - no response
				return c.NoContent()
			}
			return writeError(c, req.ID, errMethodNotFound, "method not found", nil)
		}
	}, nil
}

// handleInitialize handles the initialize request.
func handleInitialize(c *mizu.Ctx, req *request, o *options) error {
	result := initializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities: serverCapabilities{
			Tools: &toolsCapability{},
		},
	}
	if o.serverName != "" {
		result.ServerInfo = &serverInfo{
			Name:    o.serverName,
			Version: o.serverVersion,
		}
	}
	return writeResult(c, req.ID, result)
}

// handleToolsList handles the tools/list request.
func handleToolsList(c *mizu.Ctx, req *request, tools []tool) error {
	return writeResult(c, req.ID, toolsListResult{Tools: tools})
}

// handleToolsCall handles the tools/call request.
func handleToolsCall(c *mizu.Ctx, req *request, inv contract.Invoker, svc *contract.Service, o *options) error {
	var params toolsCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return writeError(c, req.ID, errInvalidParams, "invalid params", err.Error())
	}

	// Parse tool name: resource_method
	resource, method, ok := parseTool(params.Name)
	if !ok {
		return writeError(c, req.ID, errInvalidParams, "invalid params", "invalid tool name format")
	}

	// Find method in descriptor
	methodDesc := svc.Method(resource, method)
	if methodDesc == nil {
		return writeError(c, req.ID, errInvalidParams, "invalid params", "unknown tool: "+params.Name)
	}

	// Allocate input
	var in any
	if methodDesc.Input != "" {
		var err error
		in, err = inv.NewInput(resource, method)
		if err != nil || in == nil {
			return writeError(c, req.ID, errInternal, "internal error", "failed to allocate input")
		}
	}

	// Decode arguments
	if in != nil && len(params.Arguments) > 0 && string(params.Arguments) != "null" {
		if err := json.Unmarshal(params.Arguments, in); err != nil {
			return writeError(c, req.ID, errInvalidParams, "invalid params", err.Error())
		}
	}

	// Invoke the contract method
	out, err := inv.Call(c.Context(), resource, method, in)
	if err != nil {
		isErr, msg := o.errorMapper(err)
		return writeResult(c, req.ID, toolsCallResult{
			Content: []content{{Type: "text", Text: msg}},
			IsError: isErr,
		})
	}

	// Marshal output
	var text string
	if out != nil {
		b, err := json.Marshal(out)
		if err != nil {
			return writeError(c, req.ID, errInternal, "internal error", err.Error())
		}
		text = string(b)
	} else {
		text = "null"
	}

	return writeResult(c, req.ID, toolsCallResult{
		Content: []content{{Type: "text", Text: text}},
		IsError: false,
	})
}

// writeResult writes a JSON-RPC success response.
func writeResult(c *mizu.Ctx, id any, result any) error {
	b, err := json.Marshal(result)
	if err != nil {
		return writeError(c, id, errInternal, "internal error", err.Error())
	}
	return c.JSON(http.StatusOK, response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  json.RawMessage(b),
	})
}

// writeError writes a JSON-RPC error response.
func writeError(c *mizu.Ctx, id any, code int, message string, data any) error {
	return c.JSON(http.StatusOK, response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message, Data: data},
	})
}
