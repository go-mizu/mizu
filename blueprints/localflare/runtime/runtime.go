// Package runtime provides a high-performance JavaScript runtime for Workers execution.
// It uses goja as the JavaScript engine and provides Cloudflare Workers API compatibility.
package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Runtime represents a JavaScript Worker runtime.
type Runtime struct {
	loop     *eventloop.EventLoop
	vm       *goja.Runtime
	registry *require.Registry
	store    store.Store
	bindings map[string]interface{}
	mu       sync.Mutex
}

// Config holds runtime configuration.
type Config struct {
	Store       store.Store
	Bindings    map[string]string // name -> type mapping (e.g., "KV" -> "kv_namespace_id")
	Environment map[string]string
	MaxMemory   int64 // bytes, 0 for unlimited
	MaxCPUTime  time.Duration
}

// New creates a new JavaScript runtime.
func New(cfg Config) *Runtime {
	registry := require.NewRegistry()

	r := &Runtime{
		loop:     eventloop.NewEventLoop(),
		registry: registry,
		store:    cfg.Store,
		bindings: make(map[string]interface{}),
	}

	r.loop.Run(func(vm *goja.Runtime) {
		r.vm = vm

		// Setup require
		registry.Enable(vm)

		// Setup console
		console.Enable(vm)

		// Setup Web APIs
		r.setupWebAPIs()

		// Setup environment variables
		r.setupEnvironment(cfg.Environment)

		// Setup bindings
		r.setupBindings(cfg.Bindings)
	})

	return r
}

// Execute runs JavaScript code and handles a request.
func (r *Runtime) Execute(ctx context.Context, script string, req *http.Request) (*WorkerResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	resultCh := make(chan *WorkerResponse, 1)
	errCh := make(chan error, 1)

	r.loop.RunOnLoop(func(vm *goja.Runtime) {
		// Compile and run the script
		_, err := vm.RunString(script)
		if err != nil {
			errCh <- fmt.Errorf("script error: %w", err)
			return
		}

		// Create the request object
		reqObj, err := r.createRequest(req)
		if err != nil {
			errCh <- fmt.Errorf("create request: %w", err)
			return
		}

		// Create the event
		event := vm.NewObject()
		event.Set("request", reqObj)
		event.Set("respondWith", func(call goja.FunctionCall) goja.Value {
			// Handle both Promise and direct Response
			arg := call.Argument(0)

			// Check if it's a Promise
			if promiseObj, ok := arg.Export().(map[string]interface{}); ok {
				if then, exists := promiseObj["then"]; exists {
					if thenFunc, ok := then.(func(goja.FunctionCall) goja.Value); ok {
						// It's a promise, handle async
						thenFunc(goja.FunctionCall{
							Arguments: []goja.Value{vm.ToValue(func(resp goja.Value) {
								r.handleResponse(resp, resultCh, errCh)
							})},
						})
						return goja.Undefined()
					}
				}
			}

			// Direct response
			r.handleResponse(arg, resultCh, errCh)
			return goja.Undefined()
		})

		// Call the fetch event handler
		handlers := vm.Get("__fetchHandlers")
		if handlers == nil || goja.IsUndefined(handlers) {
			errCh <- fmt.Errorf("no fetch handler registered")
			return
		}

		handlersArr, ok := handlers.Export().([]interface{})
		if !ok || len(handlersArr) == 0 {
			errCh <- fmt.Errorf("no fetch handler registered")
			return
		}

		for _, h := range handlersArr {
			if handler, ok := h.(func(goja.FunctionCall) goja.Value); ok {
				handler(goja.FunctionCall{
					Arguments: []goja.Value{event},
				})
			}
		}
	})

	// Wait for result with timeout
	select {
	case resp := <-resultCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("worker execution timeout")
	}
}

func (r *Runtime) handleResponse(val goja.Value, resultCh chan *WorkerResponse, errCh chan error) {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		errCh <- fmt.Errorf("invalid response")
		return
	}

	obj := val.ToObject(r.vm)
	resp := &WorkerResponse{
		Status:  200,
		Headers: make(http.Header),
	}

	// Get status
	if status := obj.Get("status"); status != nil && !goja.IsUndefined(status) {
		resp.Status = int(status.ToInteger())
	}

	// Get headers
	if headers := obj.Get("headers"); headers != nil && !goja.IsUndefined(headers) {
		headersObj := headers.ToObject(r.vm)
		if entries := headersObj.Get("entries"); entries != nil {
			if entriesFunc, ok := entries.Export().(func(goja.FunctionCall) goja.Value); ok {
				entriesResult := entriesFunc(goja.FunctionCall{})
				if arr, ok := entriesResult.Export().([]interface{}); ok {
					for _, entry := range arr {
						if pair, ok := entry.([]interface{}); ok && len(pair) == 2 {
							key := fmt.Sprintf("%v", pair[0])
							value := fmt.Sprintf("%v", pair[1])
							resp.Headers.Set(key, value)
						}
					}
				}
			}
		} else {
			// Try as plain object
			for _, key := range headersObj.Keys() {
				val := headersObj.Get(key)
				if val != nil && !goja.IsUndefined(val) {
					resp.Headers.Set(key, val.String())
				}
			}
		}
	}

	// Get body
	if body := obj.Get("body"); body != nil && !goja.IsUndefined(body) {
		resp.Body = []byte(body.String())
	} else if text := obj.Get("_body"); text != nil && !goja.IsUndefined(text) {
		resp.Body = []byte(text.String())
	}

	resultCh <- resp
}

// WorkerResponse represents a Worker's HTTP response.
type WorkerResponse struct {
	Status  int
	Headers http.Header
	Body    []byte
}

func (r *Runtime) createRequest(req *http.Request) (goja.Value, error) {
	obj := r.vm.NewObject()

	// URL
	url := req.URL.String()
	if req.URL.Host == "" {
		url = "http://localhost" + req.URL.String()
	}
	obj.Set("url", url)

	// Method
	obj.Set("method", req.Method)

	// Headers
	headersObj := r.vm.NewObject()
	headersMap := make(map[string]string)
	for key, values := range req.Header {
		if len(values) > 0 {
			headersMap[strings.ToLower(key)] = values[0]
		}
	}
	headersObj.Set("get", func(call goja.FunctionCall) goja.Value {
		key := strings.ToLower(call.Argument(0).String())
		if val, ok := headersMap[key]; ok {
			return r.vm.ToValue(val)
		}
		return goja.Null()
	})
	headersObj.Set("has", func(call goja.FunctionCall) goja.Value {
		key := strings.ToLower(call.Argument(0).String())
		_, ok := headersMap[key]
		return r.vm.ToValue(ok)
	})
	headersObj.Set("entries", func(call goja.FunctionCall) goja.Value {
		var entries [][]string
		for k, v := range headersMap {
			entries = append(entries, []string{k, v})
		}
		return r.vm.ToValue(entries)
	})
	obj.Set("headers", headersObj)

	// Body
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		obj.Set("_bodyBytes", bodyBytes)
		obj.Set("text", func(call goja.FunctionCall) goja.Value {
			return r.vm.ToValue(string(bodyBytes))
		})
		obj.Set("json", func(call goja.FunctionCall) goja.Value {
			var data interface{}
			json.Unmarshal(bodyBytes, &data)
			return r.vm.ToValue(data)
		})
		obj.Set("arrayBuffer", func(call goja.FunctionCall) goja.Value {
			return r.vm.ToValue(bodyBytes)
		})
	}

	// CF object (Cloudflare-specific request properties)
	cf := r.vm.NewObject()
	cf.Set("colo", "LOCAL")
	cf.Set("country", "XX")
	cf.Set("city", "Local")
	cf.Set("continent", "XX")
	cf.Set("latitude", "0")
	cf.Set("longitude", "0")
	cf.Set("postalCode", "00000")
	cf.Set("region", "Local")
	cf.Set("timezone", "UTC")
	cf.Set("asn", 0)
	cf.Set("asOrganization", "Localflare")
	obj.Set("cf", cf)

	return obj, nil
}

func (r *Runtime) setupWebAPIs() {
	vm := r.vm

	// Global fetch handlers storage
	vm.Set("__fetchHandlers", []interface{}{})

	// addEventListener
	vm.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		handler := call.Argument(1)

		if eventType == "fetch" {
			handlers := vm.Get("__fetchHandlers").Export().([]interface{})
			handlers = append(handlers, handler.Export())
			vm.Set("__fetchHandlers", handlers)
		}
		return goja.Undefined()
	})

	// Response constructor
	vm.Set("Response", func(call goja.ConstructorCall) *goja.Object {
		body := ""
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			body = call.Arguments[0].String()
		}

		obj := call.This
		obj.Set("_body", body)
		obj.Set("status", 200)
		obj.Set("statusText", "OK")
		obj.Set("ok", true)

		// Headers
		headers := vm.NewObject()
		headerData := make(map[string]string)

		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			init := call.Arguments[1].ToObject(vm)

			// Status
			if status := init.Get("status"); status != nil && !goja.IsUndefined(status) {
				statusCode := int(status.ToInteger())
				obj.Set("status", statusCode)
				obj.Set("ok", statusCode >= 200 && statusCode < 300)
			}

			// StatusText
			if statusText := init.Get("statusText"); statusText != nil && !goja.IsUndefined(statusText) {
				obj.Set("statusText", statusText.String())
			}

			// Headers
			if h := init.Get("headers"); h != nil && !goja.IsUndefined(h) {
				hObj := h.ToObject(vm)
				for _, key := range hObj.Keys() {
					val := hObj.Get(key)
					if val != nil && !goja.IsUndefined(val) {
						headerData[strings.ToLower(key)] = val.String()
					}
				}
			}
		}

		headers.Set("get", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			if val, ok := headerData[key]; ok {
				return vm.ToValue(val)
			}
			return goja.Null()
		})
		headers.Set("set", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			val := c.Argument(1).String()
			headerData[key] = val
			return goja.Undefined()
		})
		headers.Set("has", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			_, ok := headerData[key]
			return vm.ToValue(ok)
		})
		headers.Set("entries", func(c goja.FunctionCall) goja.Value {
			var entries [][]string
			for k, v := range headerData {
				entries = append(entries, []string{k, v})
			}
			return vm.ToValue(entries)
		})
		obj.Set("headers", headers)

		// Body methods
		obj.Set("text", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue(body))
		})
		obj.Set("json", func(c goja.FunctionCall) goja.Value {
			var data interface{}
			json.Unmarshal([]byte(body), &data)
			return r.createPromise(vm.ToValue(data))
		})
		obj.Set("arrayBuffer", func(c goja.FunctionCall) goja.Value {
			return r.createPromise(vm.ToValue([]byte(body)))
		})
		obj.Set("clone", func(c goja.FunctionCall) goja.Value {
			clone := vm.NewObject()
			clone.Set("_body", body)
			clone.Set("status", obj.Get("status"))
			clone.Set("statusText", obj.Get("statusText"))
			clone.Set("ok", obj.Get("ok"))
			clone.Set("headers", headers)
			return clone
		})

		return obj
	})

	// Response.json static method
	respClass := vm.Get("Response").ToObject(vm)
	respClass.Set("json", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).Export()
		jsonBytes, _ := json.Marshal(data)

		initArg := vm.NewObject()
		headers := vm.NewObject()
		headers.Set("content-type", "application/json")
		initArg.Set("headers", headers)

		if len(call.Arguments) > 1 {
			// Merge with provided init
			init := call.Arguments[1].ToObject(vm)
			if status := init.Get("status"); status != nil && !goja.IsUndefined(status) {
				initArg.Set("status", status)
			}
		}

		resp, _ := vm.New(vm.Get("Response"), vm.ToValue(string(jsonBytes)), initArg)
		return resp
	})

	// Response.redirect static method
	respClass.Set("redirect", func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		status := 302
		if len(call.Arguments) > 1 {
			status = int(call.Argument(1).ToInteger())
		}

		initArg := vm.NewObject()
		initArg.Set("status", status)
		headers := vm.NewObject()
		headers.Set("location", url)
		initArg.Set("headers", headers)

		resp, _ := vm.New(vm.Get("Response"), vm.ToValue(""), initArg)
		return resp
	})

	// Request constructor
	vm.Set("Request", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This

		url := ""
		method := "GET"

		if len(call.Arguments) > 0 {
			url = call.Arguments[0].String()
		}

		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			init := call.Arguments[1].ToObject(vm)
			if m := init.Get("method"); m != nil && !goja.IsUndefined(m) {
				method = m.String()
			}
		}

		obj.Set("url", url)
		obj.Set("method", method)

		return obj
	})

	// Headers constructor
	vm.Set("Headers", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		data := make(map[string]string)

		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			init := call.Arguments[0].ToObject(vm)
			for _, key := range init.Keys() {
				val := init.Get(key)
				if val != nil && !goja.IsUndefined(val) {
					data[strings.ToLower(key)] = val.String()
				}
			}
		}

		obj.Set("get", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			if val, ok := data[key]; ok {
				return vm.ToValue(val)
			}
			return goja.Null()
		})
		obj.Set("set", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			val := c.Argument(1).String()
			data[key] = val
			return goja.Undefined()
		})
		obj.Set("append", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			val := c.Argument(1).String()
			if existing, ok := data[key]; ok {
				data[key] = existing + ", " + val
			} else {
				data[key] = val
			}
			return goja.Undefined()
		})
		obj.Set("delete", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			delete(data, key)
			return goja.Undefined()
		})
		obj.Set("has", func(c goja.FunctionCall) goja.Value {
			key := strings.ToLower(c.Argument(0).String())
			_, ok := data[key]
			return vm.ToValue(ok)
		})
		obj.Set("entries", func(c goja.FunctionCall) goja.Value {
			var entries [][]string
			for k, v := range data {
				entries = append(entries, []string{k, v})
			}
			return vm.ToValue(entries)
		})
		obj.Set("keys", func(c goja.FunctionCall) goja.Value {
			var keys []string
			for k := range data {
				keys = append(keys, k)
			}
			return vm.ToValue(keys)
		})
		obj.Set("values", func(c goja.FunctionCall) goja.Value {
			var values []string
			for _, v := range data {
				values = append(values, v)
			}
			return vm.ToValue(values)
		})
		obj.Set("forEach", func(c goja.FunctionCall) goja.Value {
			callback, ok := goja.AssertFunction(c.Argument(0))
			if ok {
				for k, v := range data {
					callback(nil, vm.ToValue(v), vm.ToValue(k), obj)
				}
			}
			return goja.Undefined()
		})

		return obj
	})

	// fetch function
	vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		return r.fetch(call)
	})

	// URL constructor
	vm.Set("URL", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		urlStr := call.Argument(0).String()

		// Parse URL
		obj.Set("href", urlStr)
		obj.Set("toString", func(c goja.FunctionCall) goja.Value {
			return vm.ToValue(urlStr)
		})

		// Basic URL parsing
		if strings.Contains(urlStr, "://") {
			parts := strings.SplitN(urlStr, "://", 2)
			obj.Set("protocol", parts[0]+":")
			if len(parts) > 1 {
				rest := parts[1]
				pathStart := strings.Index(rest, "/")
				if pathStart > 0 {
					obj.Set("host", rest[:pathStart])
					obj.Set("hostname", strings.Split(rest[:pathStart], ":")[0])
					obj.Set("pathname", rest[pathStart:])
				} else {
					obj.Set("host", rest)
					obj.Set("hostname", strings.Split(rest, ":")[0])
					obj.Set("pathname", "/")
				}
			}
		}

		// SearchParams
		searchParams := vm.NewObject()
		params := make(map[string]string)
		if qIdx := strings.Index(urlStr, "?"); qIdx >= 0 {
			queryStr := urlStr[qIdx+1:]
			for _, pair := range strings.Split(queryStr, "&") {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					params[kv[0]] = kv[1]
				} else if len(kv) == 1 {
					params[kv[0]] = ""
				}
			}
		}
		searchParams.Set("get", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			if val, ok := params[key]; ok {
				return vm.ToValue(val)
			}
			return goja.Null()
		})
		searchParams.Set("has", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			_, ok := params[key]
			return vm.ToValue(ok)
		})
		searchParams.Set("set", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			val := c.Argument(1).String()
			params[key] = val
			return goja.Undefined()
		})
		obj.Set("searchParams", searchParams)

		return obj
	})

	// URLSearchParams constructor
	vm.Set("URLSearchParams", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		params := make(map[string][]string)

		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			arg := call.Arguments[0]
			if s := arg.String(); strings.Contains(s, "=") {
				// Parse query string
				for _, pair := range strings.Split(strings.TrimPrefix(s, "?"), "&") {
					kv := strings.SplitN(pair, "=", 2)
					if len(kv) == 2 {
						params[kv[0]] = append(params[kv[0]], kv[1])
					}
				}
			}
		}

		obj.Set("get", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			if vals, ok := params[key]; ok && len(vals) > 0 {
				return vm.ToValue(vals[0])
			}
			return goja.Null()
		})
		obj.Set("getAll", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			return vm.ToValue(params[key])
		})
		obj.Set("has", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			_, ok := params[key]
			return vm.ToValue(ok)
		})
		obj.Set("set", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			val := c.Argument(1).String()
			params[key] = []string{val}
			return goja.Undefined()
		})
		obj.Set("append", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			val := c.Argument(1).String()
			params[key] = append(params[key], val)
			return goja.Undefined()
		})
		obj.Set("delete", func(c goja.FunctionCall) goja.Value {
			key := c.Argument(0).String()
			delete(params, key)
			return goja.Undefined()
		})
		obj.Set("toString", func(c goja.FunctionCall) goja.Value {
			var pairs []string
			for k, vals := range params {
				for _, v := range vals {
					pairs = append(pairs, k+"="+v)
				}
			}
			return vm.ToValue(strings.Join(pairs, "&"))
		})

		return obj
	})

	// TextEncoder
	vm.Set("TextEncoder", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		obj.Set("encoding", "utf-8")
		obj.Set("encode", func(c goja.FunctionCall) goja.Value {
			str := c.Argument(0).String()
			return vm.ToValue([]byte(str))
		})
		return obj
	})

	// TextDecoder
	vm.Set("TextDecoder", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		encoding := "utf-8"
		if len(call.Arguments) > 0 {
			encoding = call.Arguments[0].String()
		}
		obj.Set("encoding", encoding)
		obj.Set("decode", func(c goja.FunctionCall) goja.Value {
			data := c.Argument(0).Export()
			switch v := data.(type) {
			case []byte:
				return vm.ToValue(string(v))
			case string:
				return vm.ToValue(v)
			default:
				return vm.ToValue("")
			}
		})
		return obj
	})

	// crypto
	crypto := vm.NewObject()
	crypto.Set("randomUUID", func(c goja.FunctionCall) goja.Value {
		return vm.ToValue(generateUUID())
	})
	subtle := vm.NewObject()
	crypto.Set("subtle", subtle)
	vm.Set("crypto", crypto)

	// atob/btoa
	vm.Set("atob", func(call goja.FunctionCall) goja.Value {
		encoded := call.Argument(0).String()
		decoded, _ := base64Decode(encoded)
		return vm.ToValue(string(decoded))
	})
	vm.Set("btoa", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).String()
		return vm.ToValue(base64Encode([]byte(data)))
	})

	// setTimeout/setInterval (simplified - no actual async)
	vm.Set("setTimeout", func(call goja.FunctionCall) goja.Value {
		// In a real implementation, this would use the event loop
		// For now, we execute immediately for synchronous behavior
		if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
			fn(nil)
		}
		return vm.ToValue(1)
	})
	vm.Set("clearTimeout", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})
	vm.Set("setInterval", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(1)
	})
	vm.Set("clearInterval", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})

	// queueMicrotask
	vm.Set("queueMicrotask", func(call goja.FunctionCall) goja.Value {
		if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
			fn(nil)
		}
		return goja.Undefined()
	})

	// structuredClone
	vm.Set("structuredClone", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).Export()
		jsonBytes, _ := json.Marshal(data)
		var cloned interface{}
		json.Unmarshal(jsonBytes, &cloned)
		return vm.ToValue(cloned)
	})

	// Performance
	performance := vm.NewObject()
	startTime := time.Now()
	performance.Set("now", func(c goja.FunctionCall) goja.Value {
		return vm.ToValue(float64(time.Since(startTime).Microseconds()) / 1000.0)
	})
	vm.Set("performance", performance)
}

func (r *Runtime) createPromise(value goja.Value) goja.Value {
	promise := r.vm.NewObject()
	promise.Set("then", func(call goja.FunctionCall) goja.Value {
		if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
			result, _ := fn(nil, value)
			return result
		}
		return value
	})
	promise.Set("catch", func(call goja.FunctionCall) goja.Value {
		return promise
	})
	promise.Set("finally", func(call goja.FunctionCall) goja.Value {
		if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
			fn(nil)
		}
		return promise
	})
	return promise
}

func (r *Runtime) fetch(call goja.FunctionCall) goja.Value {
	vm := r.vm
	url := call.Argument(0).String()

	method := "GET"
	var bodyData []byte
	headers := make(map[string]string)

	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
		opts := call.Arguments[1].ToObject(vm)

		if m := opts.Get("method"); m != nil && !goja.IsUndefined(m) {
			method = m.String()
		}

		if h := opts.Get("headers"); h != nil && !goja.IsUndefined(h) {
			hObj := h.ToObject(vm)
			for _, key := range hObj.Keys() {
				val := hObj.Get(key)
				if val != nil && !goja.IsUndefined(val) {
					headers[key] = val.String()
				}
			}
		}

		if b := opts.Get("body"); b != nil && !goja.IsUndefined(b) {
			bodyData = []byte(b.String())
		}
	}

	// Make HTTP request
	var reqBody io.Reader
	if len(bodyData) > 0 {
		reqBody = strings.NewReader(string(bodyData))
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return r.createRejectedPromise(err.Error())
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return r.createRejectedPromise(err.Error())
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Create Response object
	respObj := vm.NewObject()
	respObj.Set("status", resp.StatusCode)
	respObj.Set("statusText", resp.Status)
	respObj.Set("ok", resp.StatusCode >= 200 && resp.StatusCode < 300)
	respObj.Set("_body", string(respBody))

	// Headers
	respHeaders := vm.NewObject()
	headerData := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headerData[strings.ToLower(k)] = v[0]
		}
	}
	respHeaders.Set("get", func(c goja.FunctionCall) goja.Value {
		key := strings.ToLower(c.Argument(0).String())
		if val, ok := headerData[key]; ok {
			return vm.ToValue(val)
		}
		return goja.Null()
	})
	respHeaders.Set("has", func(c goja.FunctionCall) goja.Value {
		key := strings.ToLower(c.Argument(0).String())
		_, ok := headerData[key]
		return vm.ToValue(ok)
	})
	respHeaders.Set("entries", func(c goja.FunctionCall) goja.Value {
		var entries [][]string
		for k, v := range headerData {
			entries = append(entries, []string{k, v})
		}
		return vm.ToValue(entries)
	})
	respObj.Set("headers", respHeaders)

	// Body methods
	respObj.Set("text", func(c goja.FunctionCall) goja.Value {
		return r.createPromise(vm.ToValue(string(respBody)))
	})
	respObj.Set("json", func(c goja.FunctionCall) goja.Value {
		var data interface{}
		json.Unmarshal(respBody, &data)
		return r.createPromise(vm.ToValue(data))
	})
	respObj.Set("arrayBuffer", func(c goja.FunctionCall) goja.Value {
		return r.createPromise(vm.ToValue(respBody))
	})

	return r.createPromise(respObj)
}

func (r *Runtime) createRejectedPromise(errMsg string) goja.Value {
	vm := r.vm
	promise := vm.NewObject()
	err := vm.NewGoError(errors.New(errMsg))

	promise.Set("then", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 1 {
			if fn, ok := goja.AssertFunction(call.Argument(1)); ok {
				result, _ := fn(nil, err)
				return result
			}
		}
		return promise
	})
	promise.Set("catch", func(call goja.FunctionCall) goja.Value {
		if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
			result, _ := fn(nil, err)
			return result
		}
		return promise
	})
	promise.Set("finally", func(call goja.FunctionCall) goja.Value {
		if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
			fn(nil)
		}
		return promise
	})
	return promise
}

func (r *Runtime) setupEnvironment(env map[string]string) {
	envObj := r.vm.NewObject()
	for k, v := range env {
		envObj.Set(k, v)
	}
	r.vm.Set("env", envObj)
}

func (r *Runtime) setupBindings(bindings map[string]string) {
	// This will be extended to support actual bindings
	// For now, create placeholder objects
	for name, bindingType := range bindings {
		switch {
		case strings.HasPrefix(bindingType, "kv:"):
			r.setupKVBinding(name, strings.TrimPrefix(bindingType, "kv:"))
		case strings.HasPrefix(bindingType, "r2:"):
			r.setupR2Binding(name, strings.TrimPrefix(bindingType, "r2:"))
		case strings.HasPrefix(bindingType, "d1:"):
			r.setupD1Binding(name, strings.TrimPrefix(bindingType, "d1:"))
		case strings.HasPrefix(bindingType, "do:"):
			r.setupDOBinding(name, strings.TrimPrefix(bindingType, "do:"))
		case strings.HasPrefix(bindingType, "queue:"):
			r.setupQueueBinding(name, strings.TrimPrefix(bindingType, "queue:"))
		case strings.HasPrefix(bindingType, "ai:"):
			r.setupAIBinding(name)
		}
	}
}

// Close shuts down the runtime.
func (r *Runtime) Close() {
	if r.loop != nil {
		r.loop.Stop()
	}
}

// Helper functions

func generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano()&0xffffffff,
		time.Now().UnixNano()>>32&0xffff,
		0x4000|(time.Now().UnixNano()>>48&0x0fff),
		0x8000|(time.Now().UnixNano()>>60&0x3fff),
		time.Now().UnixNano())
}

func base64Encode(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	for i := 0; i < len(data); i += 3 {
		var n uint32
		remaining := len(data) - i
		if remaining >= 3 {
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			result.WriteByte(alphabet[n>>18&0x3f])
			result.WriteByte(alphabet[n>>12&0x3f])
			result.WriteByte(alphabet[n>>6&0x3f])
			result.WriteByte(alphabet[n&0x3f])
		} else if remaining == 2 {
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8
			result.WriteByte(alphabet[n>>18&0x3f])
			result.WriteByte(alphabet[n>>12&0x3f])
			result.WriteByte(alphabet[n>>6&0x3f])
			result.WriteByte('=')
		} else {
			n = uint32(data[i]) << 16
			result.WriteByte(alphabet[n>>18&0x3f])
			result.WriteByte(alphabet[n>>12&0x3f])
			result.WriteString("==")
		}
	}
	return result.String()
}

func base64Decode(s string) ([]byte, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result []byte
	s = strings.TrimRight(s, "=")
	lookup := make(map[byte]int)
	for i, c := range []byte(alphabet) {
		lookup[c] = i
	}
	for i := 0; i < len(s); i += 4 {
		var n uint32
		end := i + 4
		if end > len(s) {
			end = len(s)
		}
		chunk := s[i:end]
		for j, c := range []byte(chunk) {
			n |= uint32(lookup[c]) << (18 - 6*j)
		}
		switch len(chunk) {
		case 4:
			result = append(result, byte(n>>16), byte(n>>8), byte(n))
		case 3:
			result = append(result, byte(n>>16), byte(n>>8))
		case 2:
			result = append(result, byte(n>>16))
		}
	}
	return result, nil
}
