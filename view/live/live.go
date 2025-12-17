package live

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/view"
)

// Default configuration values.
const (
	DefaultSessionTimeout   = 30 * time.Minute
	DefaultHeartbeatInterval = 30 * time.Second
	DefaultMaxMessageSize   = 64 * 1024
)

// Options configures the live engine.
type Options struct {
	// View is the view engine for template rendering.
	View *view.Engine

	// PubSub is the message broker for server-push.
	// Default: NewInmemPubSub()
	PubSub PubSub

	// SessionStore persists sessions.
	// Default: NewMemoryStore()
	SessionStore SessionStore

	// Dev enables development mode features.
	Dev bool

	// SessionTimeout is how long idle sessions are kept.
	// Default: 30 minutes
	SessionTimeout time.Duration

	// HeartbeatInterval is the WebSocket ping interval.
	// Default: 30 seconds
	HeartbeatInterval time.Duration

	// MaxMessageSize is the maximum WebSocket message size.
	// Default: 64KB
	MaxMessageSize int64

	// CSRFTokenFunc extracts the CSRF token from a request.
	// Default: reads from _csrf form field or X-CSRF-Token header
	CSRFTokenFunc func(c *mizu.Ctx) string

	// OnError is called when an error occurs.
	OnError func(ctx *Ctx, err error)

	// OnSessionStart is called when a new session starts.
	OnSessionStart func(ctx *Ctx, sessionID string)

	// OnSessionEnd is called when a session ends.
	OnSessionEnd func(sessionID string)
}

func (o *Options) applyDefaults() {
	if o.SessionTimeout == 0 {
		o.SessionTimeout = DefaultSessionTimeout
	}
	if o.HeartbeatInterval == 0 {
		o.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if o.MaxMessageSize == 0 {
		o.MaxMessageSize = DefaultMaxMessageSize
	}
}

// Live is the live view engine.
type Live struct {
	opts   Options
	view   *view.Engine
	pubsub PubSub
	store  SessionStore

	// pages maps URL path to page handler.
	pages map[string]pageHandler
}

// New creates a new live engine.
func New(opts Options) *Live {
	opts.applyDefaults()

	pubsub := opts.PubSub
	if pubsub == nil {
		pubsub = NewInmemPubSub()
	}

	store := opts.SessionStore
	if store == nil {
		store = NewMemoryStore()
	}

	return &Live{
		opts:   opts,
		view:   opts.View,
		pubsub: pubsub,
		store:  store,
		pages:  make(map[string]pageHandler),
	}
}

// Mount registers live routes on the Mizu app.
// Registers:
//   - GET /_live/runtime.js - Client runtime
//   - WS  /_live/websocket  - WebSocket endpoint
func (l *Live) Mount(app *mizu.App) {
	// Serve client runtime.
	app.Get("/_live/runtime.js", l.runtimeHandler())

	// WebSocket endpoint.
	app.Compat.Handle("/_live/websocket", http.HandlerFunc(l.websocketHTTPHandler))
}

// Handle creates a handler for a live page.
func Handle[T any](l *Live, page Page[T]) mizu.Handler {
	wrapper := &pageWrapper[T]{page: page}

	return func(c *mizu.Ctx) error {
		// Check if this is a WebSocket upgrade request.
		if isWebSocketUpgrade(c.Request()) {
			return nil // Will be handled by websocket handler.
		}

		// Initial HTTP request - render full page.
		return l.renderInitialPage(c, wrapper)
	}
}

// RegisterPage registers a page for a path (used by websocket handler).
func (l *Live) RegisterPage(path string, handler pageHandler) {
	l.pages[path] = handler
}

// PubSub returns the pubsub instance for publishing messages.
func (l *Live) PubSub() PubSub {
	return l.pubsub
}

// Store returns the session store.
func (l *Live) Store() SessionStore {
	return l.store
}

// renderInitialPage renders the full HTML page for the initial request.
func (l *Live) renderInitialPage(c *mizu.Ctx, wrapper pageHandler) error {
	// Create a temporary session for initial render.
	sessionID := generateSessionID()
	session := wrapper.newSession(sessionID)

	ctx := newCtx(c, sessionID, l, nil)

	session.lock()
	defer session.unlock()

	// Call Mount for initial state.
	if err := wrapper.mount(ctx, session); err != nil {
		return fmt.Errorf("mount: %w", err)
	}

	// Get view configuration.
	v, err := wrapper.render(ctx, session)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	// Build template data.
	data := view.Data{
		"State": getSessionState(session.session),
		"Flash": session.getFlash(),
		"Live": map[string]any{
			"SessionID": sessionID,
			"Connected": false,
		},
	}

	// Render the page.
	var buf bytes.Buffer
	if err := l.view.Render(&buf, v.Page, data, view.Layout(v.Layout)); err != nil {
		return err
	}

	// Inject live runtime script.
	html := l.injectRuntime(buf.String(), sessionID, c.Request().URL.Path)

	return c.HTML(http.StatusOK, html)
}

// injectRuntime adds the live runtime script to the HTML.
func (l *Live) injectRuntime(html, sessionID, path string) string {
	script := fmt.Sprintf(`<script src="/_live/runtime.js"></script>
<script>
if (window.MizuLive) {
  MizuLive.connect({
    url: %q,
    sessionId: %q
  });
}
</script>`, path, sessionID)

	// Inject before </body> or at end.
	if idx := strings.LastIndex(html, "</body>"); idx != -1 {
		return html[:idx] + script + html[idx:]
	}
	return html + script
}

// runtimeHandler serves the client JavaScript runtime.
func (l *Live) runtimeHandler() mizu.Handler {
	return func(c *mizu.Ctx) error {
		c.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		c.Header().Set("Cache-Control", "public, max-age=86400")
		return c.Text(http.StatusOK, clientRuntime)
	}
}

// websocketHTTPHandler handles WebSocket upgrade requests.
func (l *Live) websocketHTTPHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgradeHTTP(w, r)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	l.handleWebSocket(conn)
}

// handleWebSocket handles a WebSocket connection.
func (l *Live) handleWebSocket(conn *wsConn) {
	// Read first message (should be JOIN).
	data, err := conn.ReadMessage()
	if err != nil {
		return
	}

	msg, err := decodeMessage([]byte(data))
	if err != nil {
		return
	}

	if msg.Type != MsgTypeJoin {
		sendWsError(conn, "expected_join", "First message must be JOIN")
		return
	}

	var join JoinPayload
	if err := msg.parsePayload(&join); err != nil {
		sendWsError(conn, "invalid_payload", "Invalid JOIN payload")
		return
	}

	// Find the page handler.
	page, ok := l.pages[join.URL]
	if !ok {
		sendWsError(conn, "page_not_found", "Page not found: "+join.URL)
		return
	}

	// Create Mizu context from WebSocket request.
	req := conn.Request()
	mc := mizu.CtxFromRequest(nil, req)

	// Create session handler.
	handler := newSessionHandler(l, page, mc)

	// Run session event loop.
	ctx := req.Context()
	if err := handler.run(ctx, conn, &join); err != nil {
		// Log error if handler set.
		if l.opts.OnError != nil {
			l.opts.OnError(handler.ctx, err)
		}
	}
}

func sendWsError(conn *wsConn, code, message string) {
	data, _ := encodeMessage(MsgTypeError, 0, ErrorPayload{
		Code:    code,
		Message: message,
	})
	conn.WriteMessage(string(data))
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade.
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// TemplateFuncs returns template functions for live pages.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"lvClick":    lvClick,
		"lvSubmit":   lvSubmit,
		"lvChange":   lvChange,
		"lvKeydown":  lvKeydown,
		"lvKeyup":    lvKeyup,
		"lvFocus":    lvFocus,
		"lvBlur":     lvBlur,
		"lvVal":      lvVal,
		"lvDebounce": lvDebounce,
		"lvThrottle": lvThrottle,
		"lvLoading":  lvLoading,
		"lvTarget":   lvTarget,
	}
}

// Template helper functions.

func lvClick(event string, values ...map[string]any) template.HTMLAttr {
	return buildEventAttr("click", event, values...)
}

func lvSubmit(event string, values ...map[string]any) template.HTMLAttr {
	return buildEventAttr("submit", event, values...)
}

func lvChange(event string, values ...map[string]any) template.HTMLAttr {
	return buildEventAttr("change", event, values...)
}

func lvKeydown(event string, key string) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`data-lv-keydown="%s" data-lv-key="%s"`, event, key))
}

func lvKeyup(event string, key string) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`data-lv-keyup="%s" data-lv-key="%s"`, event, key))
}

func lvFocus(event string) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`data-lv-focus="%s"`, event))
}

func lvBlur(event string) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`data-lv-blur="%s"`, event))
}

func lvVal(key string, value any) map[string]any {
	return map[string]any{key: value}
}

func lvDebounce(ms int) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`data-lv-debounce="%d"`, ms))
}

func lvThrottle(ms int) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`data-lv-throttle="%d"`, ms))
}

func lvLoading(class string) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`data-lv-loading-class="%s"`, class))
}

func lvTarget(target string) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`data-lv-target="%s"`, target))
}

func buildEventAttr(eventType, event string, values ...map[string]any) template.HTMLAttr {
	var parts []string
	parts = append(parts, fmt.Sprintf(`data-lv-%s="%s"`, eventType, event))

	for _, vals := range values {
		for k, v := range vals {
			parts = append(parts, fmt.Sprintf(`data-lv-value-%s="%v"`, k, v))
		}
	}

	return template.HTMLAttr(strings.Join(parts, " "))
}
