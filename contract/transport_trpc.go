package contract

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// MountTRPC mounts a tRPC-like HTTP transport over a contract.Service.
//
// Deprecated: Use trpc.Mount from github.com/go-mizu/mizu/contract/transport/trpc instead.
// The new package provides additional options like custom resolver and invoker.
//
// Endpoint layout (v1):
//   - POST <base>/<proc>       call a procedure
//   - GET  <base>.meta         introspection (methods + schemas)
//
// Procedure names accepted:
//   - "<Method>"                 (e.g. "Create")
//   - "<service>.<Method>"       (e.g. "todo.Create")
//
// Call semantics:
//   - Request body: JSON object (named params) for methods that take input
//   - Request body: empty or "null" for methods that take no input
//
// Response envelope (tRPC-inspired, minimal):
//   - Success: {"result":{"data":<output>}}
//
//     If method has no output, data is null.
//   - Error:   {"error":{"message":"...", "code":"INTERNAL_ERROR"}}
//
// HTTP status:
//   - Always 200 for application-level errors (tRPC-style).
//   - 400 only for transport-level issues (bad JSON, unknown procedure, etc.).
func MountTRPC(mux *http.ServeMux, base string, svc *Service) {
	if mux == nil || svc == nil {
		return
	}
	if base == "" {
		base = "/trpc"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/")

	// Introspection: methods + schemas (useful for CLI and client generators).
	mux.HandleFunc(base+".meta", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service": svc.Name,
			"methods": trpcMethods(svc),
			"schemas": svc.Types.Schemas(),
		})
	})

	// Calls: POST /trpc/<proc>
	mux.HandleFunc(base+"/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		proc := strings.TrimPrefix(r.URL.Path, base+"/")
		proc = strings.TrimSpace(proc)
		if proc == "" {
			writeTRPCTransportError(w, http.StatusBadRequest, "missing procedure")
			return
		}

		m := resolveTRPCMethod(svc, proc)
		if m == nil {
			writeTRPCTransportError(w, http.StatusBadRequest, "unknown procedure")
			return
		}

		ctx := r.Context()
		out, terr := trpcCall(ctx, m, r)
		if terr != nil {
			writeTRPCTransportError(w, http.StatusBadRequest, terr.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}

func trpcMethods(svc *Service) []map[string]any {
	out := make([]map[string]any, 0, len(svc.Methods))
	for _, m := range svc.Methods {
		item := map[string]any{
			"name":     m.Name,
			"fullName": m.FullName,
			"proc":     svc.Name + "." + m.Name,
		}
		if m.Input != nil {
			item["input"] = m.Input
		}
		if m.Output != nil {
			item["output"] = m.Output
		}
		out = append(out, item)
	}
	return out
}

func resolveTRPCMethod(svc *Service, name string) *Method {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	// Accept "service.Method"
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			return nil
		}
		if parts[0] != svc.Name {
			return nil
		}
		return svc.Method(parts[1])
	}

	// Accept "Method"
	return svc.Method(name)
}

func trpcCall(ctx context.Context, m *Method, r *http.Request) (map[string]any, error) {
	var in any

	// Read body into RawMessage (allows empty body).
	var raw json.RawMessage
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&raw); err != nil {
		// If the body is empty, Decode returns EOF. Treat it as empty input.
		if !errors.Is(err, context.Canceled) && strings.Contains(err.Error(), "EOF") {
			raw = nil
		} else if err != nil {
			return nil, err
		}
	}

	raw = TrimJSONSpace(raw)

	if m.Input != nil {
		in = m.NewInput()
		if in == nil {
			return nil, errors.New("failed to allocate input")
		}

		// Accept:
		//   - empty body
		//   - null
		//   - object
		if len(raw) == 0 || IsJSONNull(raw) {
			// Keep zero value input.
		} else if raw[0] == '{' {
			if err := json.Unmarshal(raw, in); err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("input must be a JSON object")
		}
	} else {
		// No input expected: accept empty or null; reject anything else.
		if len(raw) != 0 && !IsJSONNull(raw) {
			return nil, errors.New("procedure takes no input")
		}
	}

	out, err := m.Invoker.Call(ctx, in)
	if err != nil {
		// Application-level error: return tRPC-like error envelope (HTTP 200).
		return map[string]any{
			"error": map[string]any{
				"message": err.Error(),
				"code":    "INTERNAL_ERROR",
			},
		}, nil
	}

	// Success envelope (tRPC-like).
	return map[string]any{
		"result": map[string]any{
			"data": out,
		},
	}, nil
}

func writeTRPCTransportError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": msg,
			"code":    "BAD_REQUEST",
		},
	})
}
