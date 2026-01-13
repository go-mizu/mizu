package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// ExecutionContext represents the context of a Worker execution.
type ExecutionContext struct {
	mu                     sync.Mutex
	waitUntilPromises      []goja.Value
	passThroughOnException bool
	cancelled              bool
	deadline               time.Time
}

// NewExecutionContext creates a new execution context.
func NewExecutionContext(timeout time.Duration) *ExecutionContext {
	return &ExecutionContext{
		waitUntilPromises:      make([]goja.Value, 0),
		passThroughOnException: false,
		deadline:               time.Now().Add(timeout),
	}
}

// AddWaitUntil adds a promise to be awaited.
func (ctx *ExecutionContext) AddWaitUntil(promise goja.Value) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.waitUntilPromises = append(ctx.waitUntilPromises, promise)
}

// SetPassThroughOnException enables pass-through on exception.
func (ctx *ExecutionContext) SetPassThroughOnException() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.passThroughOnException = true
}

// ShouldPassThrough returns whether to pass through to origin on exception.
func (ctx *ExecutionContext) ShouldPassThrough() bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.passThroughOnException
}

// Cancel cancels the execution context.
func (ctx *ExecutionContext) Cancel() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.cancelled = true
}

// IsCancelled returns whether the context is cancelled.
func (ctx *ExecutionContext) IsCancelled() bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.cancelled
}

// WaitUntilPromises returns the waitUntil promises.
func (ctx *ExecutionContext) WaitUntilPromises() []goja.Value {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.waitUntilPromises
}

// setupFetchEvent sets up the FetchEvent handler with full Cloudflare Workers compatibility.
func (r *Runtime) setupFetchEvent(execCtx *ExecutionContext) {
	vm := r.vm

	// FetchEvent constructor (internal use)
	vm.Set("__createFetchEvent", func(call goja.FunctionCall) goja.Value {
		reqObj := call.Argument(0)

		event := vm.NewObject()
		event.Set("type", "fetch")
		event.Set("request", reqObj)

		responded := false
		var responseValue goja.Value

		// respondWith(response)
		event.Set("respondWith", func(c goja.FunctionCall) goja.Value {
			if responded {
				panic(vm.NewGoError(fmt.Errorf("respondWith() has already been called")))
			}
			responded = true
			responseValue = c.Argument(0)
			event.Set("__response", responseValue)
			return goja.Undefined()
		})

		// waitUntil(promise)
		event.Set("waitUntil", func(c goja.FunctionCall) goja.Value {
			promise := c.Argument(0)
			execCtx.AddWaitUntil(promise)
			return goja.Undefined()
		})

		// passThroughOnException()
		event.Set("passThroughOnException", func(c goja.FunctionCall) goja.Value {
			if responded {
				// Cannot set passThroughOnException after respondWith
				return goja.Undefined()
			}
			execCtx.SetPassThroughOnException()
			return goja.Undefined()
		})

		return event
	})
}

// ScheduledEvent represents a scheduled event.
type ScheduledEvent struct {
	ScheduledTime int64  // Unix timestamp in milliseconds
	Cron          string // Cron pattern
}

// ExecuteScheduled runs a scheduled event handler.
func (r *Runtime) ExecuteScheduled(ctx context.Context, script string, event ScheduledEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	execCtx := NewExecutionContext(30 * time.Second)
	errCh := make(chan error, 1)
	doneCh := make(chan struct{}, 1)

	r.loop.RunOnLoop(func(vm *goja.Runtime) {
		// Compile and run the script
		_, err := vm.RunString(script)
		if err != nil {
			errCh <- fmt.Errorf("script error: %w", err)
			return
		}

		// Create ScheduledEvent
		scheduledEvent := vm.NewObject()
		scheduledEvent.Set("type", "scheduled")
		scheduledEvent.Set("scheduledTime", event.ScheduledTime)
		scheduledEvent.Set("cron", event.Cron)

		// waitUntil(promise)
		scheduledEvent.Set("waitUntil", func(c goja.FunctionCall) goja.Value {
			promise := c.Argument(0)
			execCtx.AddWaitUntil(promise)
			return goja.Undefined()
		})

		// noRetry() - don't retry this execution
		noRetry := false
		scheduledEvent.Set("noRetry", func(c goja.FunctionCall) goja.Value {
			noRetry = true
			return goja.Undefined()
		})

		// Call the scheduled event handlers
		handlers := vm.Get("__scheduledHandlers")
		if handlers == nil || goja.IsUndefined(handlers) {
			doneCh <- struct{}{}
			return
		}

		handlersArr, ok := handlers.Export().([]interface{})
		if !ok || len(handlersArr) == 0 {
			doneCh <- struct{}{}
			return
		}

		for _, h := range handlersArr {
			if handler, ok := h.(func(goja.FunctionCall) goja.Value); ok {
				handler(goja.FunctionCall{
					Arguments: []goja.Value{scheduledEvent},
				})
			}
		}

		// Process waitUntil promises
		for _, promise := range execCtx.WaitUntilPromises() {
			if promise != nil && !goja.IsUndefined(promise) {
				// Try to await the promise
				if promiseObj, ok := promise.Export().(map[string]interface{}); ok {
					if then, exists := promiseObj["then"]; exists {
						if thenFunc, ok := then.(func(goja.FunctionCall) goja.Value); ok {
							thenFunc(goja.FunctionCall{
								Arguments: []goja.Value{vm.ToValue(func(result goja.Value) {
									// Promise resolved
								})},
							})
						}
					}
				}
			}
		}

		_ = noRetry
		doneCh <- struct{}{}
	})

	// Wait for completion
	select {
	case err := <-errCh:
		return err
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("scheduled execution timeout")
	}
}

// QueueEvent represents a queue consumer event.
type QueueEvent struct {
	QueueName string
	Messages  []QueueMessage
}

// QueueMessage represents a queue message.
type QueueMessage struct {
	ID        string
	Timestamp time.Time
	Body      interface{}
	Attempts  int
}

// QueueMessageResult represents the result of processing a message.
type QueueMessageResult struct {
	ID     string
	Acked  bool
	Retry  bool
	Delete bool
}

// ExecuteQueue runs a queue consumer handler.
func (r *Runtime) ExecuteQueue(ctx context.Context, script string, event QueueEvent) ([]QueueMessageResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	execCtx := NewExecutionContext(30 * time.Second)
	errCh := make(chan error, 1)
	resultCh := make(chan []QueueMessageResult, 1)

	r.loop.RunOnLoop(func(vm *goja.Runtime) {
		// Compile and run the script
		_, err := vm.RunString(script)
		if err != nil {
			errCh <- fmt.Errorf("script error: %w", err)
			return
		}

		// Track message results
		results := make(map[string]*QueueMessageResult)
		for _, msg := range event.Messages {
			results[msg.ID] = &QueueMessageResult{ID: msg.ID}
		}

		// Create messages array
		messages := make([]interface{}, len(event.Messages))
		for i, msg := range event.Messages {
			msgObj := vm.NewObject()
			msgObj.Set("id", msg.ID)
			msgObj.Set("timestamp", msg.Timestamp.UnixMilli())
			msgObj.Set("body", msg.Body)
			msgObj.Set("attempts", msg.Attempts)

			// ack() - acknowledge successful processing
			msgID := msg.ID
			msgObj.Set("ack", func(c goja.FunctionCall) goja.Value {
				if r, ok := results[msgID]; ok {
					r.Acked = true
				}
				return goja.Undefined()
			})

			// retry() - retry processing
			msgObj.Set("retry", func(c goja.FunctionCall) goja.Value {
				if r, ok := results[msgID]; ok {
					r.Retry = true
				}
				return goja.Undefined()
			})

			messages[i] = msgObj
		}

		// Create batch object
		batch := vm.NewObject()
		batch.Set("queue", event.QueueName)
		batch.Set("messages", messages)

		// ackAll() - acknowledge all messages
		batch.Set("ackAll", func(c goja.FunctionCall) goja.Value {
			for _, r := range results {
				r.Acked = true
			}
			return goja.Undefined()
		})

		// retryAll() - retry all messages
		batch.Set("retryAll", func(c goja.FunctionCall) goja.Value {
			for _, r := range results {
				r.Retry = true
			}
			return goja.Undefined()
		})

		// Create queue event
		queueEvent := vm.NewObject()
		queueEvent.Set("type", "queue")
		queueEvent.Set("batch", batch)

		// waitUntil(promise)
		queueEvent.Set("waitUntil", func(c goja.FunctionCall) goja.Value {
			promise := c.Argument(0)
			execCtx.AddWaitUntil(promise)
			return goja.Undefined()
		})

		// Call the queue event handlers
		handlers := vm.Get("__queueHandlers")
		if handlers == nil || goja.IsUndefined(handlers) {
			// No handlers, ack all by default
			for _, r := range results {
				r.Acked = true
			}
			resultArr := make([]QueueMessageResult, 0, len(results))
			for _, r := range results {
				resultArr = append(resultArr, *r)
			}
			resultCh <- resultArr
			return
		}

		handlersArr, ok := handlers.Export().([]interface{})
		if !ok || len(handlersArr) == 0 {
			for _, r := range results {
				r.Acked = true
			}
			resultArr := make([]QueueMessageResult, 0, len(results))
			for _, r := range results {
				resultArr = append(resultArr, *r)
			}
			resultCh <- resultArr
			return
		}

		for _, h := range handlersArr {
			if handler, ok := h.(func(goja.FunctionCall) goja.Value); ok {
				handler(goja.FunctionCall{
					Arguments: []goja.Value{queueEvent},
				})
			}
		}

		// Process waitUntil promises
		for _, promise := range execCtx.WaitUntilPromises() {
			if promise != nil && !goja.IsUndefined(promise) {
				if promiseObj, ok := promise.Export().(map[string]interface{}); ok {
					if then, exists := promiseObj["then"]; exists {
						if thenFunc, ok := then.(func(goja.FunctionCall) goja.Value); ok {
							thenFunc(goja.FunctionCall{
								Arguments: []goja.Value{vm.ToValue(func(result goja.Value) {
									// Promise resolved
								})},
							})
						}
					}
				}
			}
		}

		// Collect results
		resultArr := make([]QueueMessageResult, 0, len(results))
		for _, r := range results {
			resultArr = append(resultArr, *r)
		}
		resultCh <- resultArr
	})

	// Wait for completion
	select {
	case err := <-errCh:
		return nil, err
	case result := <-resultCh:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("queue execution timeout")
	}
}

// setupScheduledHandlers sets up the scheduled event handler registration.
func (r *Runtime) setupScheduledHandlers() {
	vm := r.vm

	// Initialize scheduled handlers storage
	vm.Set("__scheduledHandlers", []interface{}{})

	// Override addEventListener to handle scheduled events
	originalAddEventListener := vm.Get("addEventListener")
	vm.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		handler := call.Argument(1)

		switch eventType {
		case "scheduled":
			handlers := vm.Get("__scheduledHandlers").Export().([]interface{})
			handlers = append(handlers, handler.Export())
			vm.Set("__scheduledHandlers", handlers)
		case "queue":
			handlers := vm.Get("__queueHandlers").Export().([]interface{})
			handlers = append(handlers, handler.Export())
			vm.Set("__queueHandlers", handlers)
		default:
			// Call original addEventListener for fetch events
			if fn, ok := goja.AssertFunction(originalAddEventListener); ok {
				fn(nil, call.Arguments...)
			}
		}

		return goja.Undefined()
	})

	// Initialize queue handlers storage
	vm.Set("__queueHandlers", []interface{}{})
}

// EmailEvent represents an incoming email event.
type EmailEvent struct {
	From    string
	To      string
	Headers map[string]string
	Raw     []byte
}

// EmailMessage represents an email message that can be forwarded or rejected.
type EmailMessage struct {
	vm       *goja.Runtime
	From     string
	To       string
	Headers  map[string]string
	Raw      []byte
	RawSize  int
	rejected bool
	forward  string
}

// ExecuteEmail runs an email event handler.
func (r *Runtime) ExecuteEmail(ctx context.Context, script string, event EmailEvent) (*EmailMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	execCtx := NewExecutionContext(30 * time.Second)
	errCh := make(chan error, 1)
	resultCh := make(chan *EmailMessage, 1)

	r.loop.RunOnLoop(func(vm *goja.Runtime) {
		// Compile and run the script
		_, err := vm.RunString(script)
		if err != nil {
			errCh <- fmt.Errorf("script error: %w", err)
			return
		}

		// Create email message object
		msg := &EmailMessage{
			vm:      vm,
			From:    event.From,
			To:      event.To,
			Headers: event.Headers,
			Raw:     event.Raw,
			RawSize: len(event.Raw),
		}

		msgObj := vm.NewObject()
		msgObj.Set("from", event.From)
		msgObj.Set("to", event.To)

		// Headers
		headers := vm.NewObject()
		for k, v := range event.Headers {
			headers.Set(k, v)
		}
		msgObj.Set("headers", headers)

		msgObj.Set("raw", event.Raw)
		msgObj.Set("rawSize", len(event.Raw))

		// forward(address, headers?)
		msgObj.Set("forward", func(c goja.FunctionCall) goja.Value {
			address := c.Argument(0).String()
			msg.forward = address
			return r.createPromise(goja.Undefined())
		})

		// reply(message)
		msgObj.Set("reply", func(c goja.FunctionCall) goja.Value {
			// Not implemented - would send a reply
			return r.createPromise(goja.Undefined())
		})

		// setReject(reason)
		msgObj.Set("setReject", func(c goja.FunctionCall) goja.Value {
			msg.rejected = true
			return goja.Undefined()
		})

		// Create email event
		emailEvent := vm.NewObject()
		emailEvent.Set("type", "email")
		emailEvent.Set("message", msgObj)

		// waitUntil(promise)
		emailEvent.Set("waitUntil", func(c goja.FunctionCall) goja.Value {
			promise := c.Argument(0)
			execCtx.AddWaitUntil(promise)
			return goja.Undefined()
		})

		// Call the email event handlers
		handlers := vm.Get("__emailHandlers")
		if handlers == nil || goja.IsUndefined(handlers) {
			resultCh <- msg
			return
		}

		handlersArr, ok := handlers.Export().([]interface{})
		if !ok || len(handlersArr) == 0 {
			resultCh <- msg
			return
		}

		for _, h := range handlersArr {
			if handler, ok := h.(func(goja.FunctionCall) goja.Value); ok {
				handler(goja.FunctionCall{
					Arguments: []goja.Value{emailEvent},
				})
			}
		}

		resultCh <- msg
	})

	// Wait for completion
	select {
	case err := <-errCh:
		return nil, err
	case result := <-resultCh:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("email execution timeout")
	}
}

// setupEmailHandlers sets up the email event handler registration.
func (r *Runtime) setupEmailHandlers() {
	vm := r.vm
	vm.Set("__emailHandlers", []interface{}{})
}

// TailEvent represents a tail event for Tail Workers.
type TailEvent struct {
	Events    []TailItem
	Timestamp time.Time
}

// TailItem represents a single tail item.
type TailItem struct {
	Event      string                 // "fetch", "scheduled", "queue", etc.
	EventTime  time.Time              // When the event occurred
	Request    map[string]interface{} // Request details for fetch events
	Response   map[string]interface{} // Response details
	Logs       []TailLog              // Console logs
	Exceptions []TailException        // Exceptions thrown
	Outcome    string                 // "ok", "exception", "exceededCpu", etc.
}

// TailLog represents a console log entry.
type TailLog struct {
	Level     string        // "log", "debug", "info", "warn", "error"
	Message   []interface{} // Log arguments
	Timestamp time.Time
}

// TailException represents an exception.
type TailException struct {
	Name      string
	Message   string
	Timestamp time.Time
}

// ExecuteTail runs a tail event handler.
func (r *Runtime) ExecuteTail(ctx context.Context, script string, event TailEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	errCh := make(chan error, 1)
	doneCh := make(chan struct{}, 1)

	r.loop.RunOnLoop(func(vm *goja.Runtime) {
		// Compile and run the script
		_, err := vm.RunString(script)
		if err != nil {
			errCh <- fmt.Errorf("script error: %w", err)
			return
		}

		// Create events array
		events := make([]interface{}, len(event.Events))
		for i, item := range event.Events {
			eventObj := vm.NewObject()
			eventObj.Set("event", item.Event)
			eventObj.Set("eventTimestamp", item.EventTime.UnixMilli())

			if item.Request != nil {
				eventObj.Set("request", item.Request)
			}
			if item.Response != nil {
				eventObj.Set("response", item.Response)
			}

			// Logs
			logs := make([]interface{}, len(item.Logs))
			for j, log := range item.Logs {
				logObj := vm.NewObject()
				logObj.Set("level", log.Level)
				logObj.Set("message", log.Message)
				logObj.Set("timestamp", log.Timestamp.UnixMilli())
				logs[j] = logObj
			}
			eventObj.Set("logs", logs)

			// Exceptions
			exceptions := make([]interface{}, len(item.Exceptions))
			for j, exc := range item.Exceptions {
				excObj := vm.NewObject()
				excObj.Set("name", exc.Name)
				excObj.Set("message", exc.Message)
				excObj.Set("timestamp", exc.Timestamp.UnixMilli())
				exceptions[j] = excObj
			}
			eventObj.Set("exceptions", exceptions)

			eventObj.Set("outcome", item.Outcome)
			events[i] = eventObj
		}

		// Create tail event
		tailEvent := vm.NewObject()
		tailEvent.Set("type", "tail")
		tailEvent.Set("events", events)
		tailEvent.Set("timestamp", event.Timestamp.UnixMilli())

		// Call the tail event handlers
		handlers := vm.Get("__tailHandlers")
		if handlers == nil || goja.IsUndefined(handlers) {
			doneCh <- struct{}{}
			return
		}

		handlersArr, ok := handlers.Export().([]interface{})
		if !ok || len(handlersArr) == 0 {
			doneCh <- struct{}{}
			return
		}

		for _, h := range handlersArr {
			if handler, ok := h.(func(goja.FunctionCall) goja.Value); ok {
				handler(goja.FunctionCall{
					Arguments: []goja.Value{tailEvent},
				})
			}
		}

		doneCh <- struct{}{}
	})

	// Wait for completion
	select {
	case err := <-errCh:
		return err
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("tail execution timeout")
	}
}

// setupTailHandlers sets up the tail event handler registration.
func (r *Runtime) setupTailHandlers() {
	vm := r.vm
	vm.Set("__tailHandlers", []interface{}{})
}
