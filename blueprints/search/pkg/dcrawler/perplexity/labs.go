package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// LabsClient interacts with Perplexity Labs via Socket.IO/WebSocket.
// No account required — uses anonymous authentication.
// NOTE: Cloudflare may block WebSocket connections with managed challenges.
// If Labs fails, use SSE search (always works) or the official API instead.
type LabsClient struct {
	httpClient *http.Client
	ws         *websocket.Conn
	sid        string
	history    []labsMessage
	mu         sync.Mutex

	// Response channel
	lastAnswer *LabsResult
	answerMu   sync.Mutex
	answerCh   chan struct{}

	// Connection ready signal (Socket.IO namespace connected)
	readyCh chan struct{}
}

// NewLabsClient creates a new Labs client with Socket.IO handshake.
func NewLabsClient(ctx context.Context) (*LabsClient, error) {
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialTLSChrome(ctx, network, addr)
		},
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}
	lc := &LabsClient{
		httpClient: &http.Client{Timeout: defaultTimeout, Jar: jar, Transport: transport},
		answerCh:   make(chan struct{}, 1),
		readyCh:    make(chan struct{}),
	}

	if err := lc.connect(ctx); err != nil {
		return nil, err
	}

	return lc, nil
}

// connect performs the Engine.IO v4 handshake and WebSocket upgrade.
func (lc *LabsClient) connect(ctx context.Context) error {
	timestamp := fmt.Sprintf("%08x", rand.Uint32())

	// Step 1: HTTP polling — get SID
	pollURL := fmt.Sprintf("%s?EIO=4&transport=polling&t=%s", endpointSocketIO, timestamp)
	req, err := http.NewRequestWithContext(ctx, "GET", pollURL, nil)
	if err != nil {
		return err
	}
	setHeaders(req, defaultHeaders())

	resp, err := lc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("socket.io polling: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Response starts with a length-prefix character, skip it
	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr[0] != '{' {
		bodyStr = bodyStr[1:]
	}

	var handshake socketIOHandshake
	if err := json.Unmarshal([]byte(bodyStr), &handshake); err != nil {
		return fmt.Errorf("parse handshake: %w (body: %s)", err, string(body))
	}
	lc.sid = handshake.SID

	// Step 2: HTTP polling — authenticate
	authURL := fmt.Sprintf("%s?EIO=4&transport=polling&t=%s&sid=%s", endpointSocketIO, timestamp, lc.sid)
	authReq, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(`40{"jwt":"anonymous-ask-user"}`))
	if err != nil {
		return err
	}
	setHeaders(authReq, defaultHeaders())
	authReq.Header.Set("Content-Type", "text/plain")

	// Copy cookies from step 1
	reqURL, _ := url.Parse(endpointSocketIO)
	for _, cookie := range lc.httpClient.Jar.Cookies(reqURL) {
		authReq.AddCookie(cookie)
	}

	authResp, err := lc.httpClient.Do(authReq)
	if err != nil {
		return fmt.Errorf("socket.io auth: %w", err)
	}
	defer authResp.Body.Close()
	authBody, _ := io.ReadAll(authResp.Body)
	if string(authBody) != "OK" {
		if authResp.StatusCode >= 400 {
			return fmt.Errorf("socket.io auth failed: HTTP %d body=%s", authResp.StatusCode, string(authBody))
		}
	}

	// Step 3: WebSocket upgrade
	wsURL := fmt.Sprintf("wss://www.perplexity.ai/socket.io/?EIO=4&transport=websocket&sid=%s", lc.sid)

	wsHeaders := http.Header{
		"User-Agent":      {chromeUA},
		"Origin":          {"https://www.perplexity.ai"},
		"Accept-Language": {"en-US,en;q=0.9"},
	}
	// Forward cookies from polling session
	reqURL, _ = url.Parse("https://www.perplexity.ai/")
	for _, cookie := range lc.httpClient.Jar.Cookies(reqURL) {
		if wsHeaders.Get("Cookie") == "" {
			wsHeaders.Set("Cookie", cookie.Name+"="+cookie.Value)
		} else {
			wsHeaders.Set("Cookie", wsHeaders.Get("Cookie")+"; "+cookie.Name+"="+cookie.Value)
		}
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		Jar:              lc.httpClient.Jar,
		NetDialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialTLSChrome(ctx, network, addr)
		},
	}

	conn, wsResp, err := dialer.DialContext(ctx, wsURL, wsHeaders)
	if err != nil {
		if wsResp != nil {
			wsBody, _ := io.ReadAll(wsResp.Body)
			wsResp.Body.Close()
			return fmt.Errorf("websocket dial: %w (HTTP %d, body: %s)", err, wsResp.StatusCode, truncate(string(wsBody), 200))
		}
		return fmt.Errorf("websocket dial: %w", err)
	}
	lc.ws = conn

	// Send upgrade probe
	if err := conn.WriteMessage(websocket.TextMessage, []byte("2probe")); err != nil {
		return fmt.Errorf("send probe: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte("5")); err != nil {
		return fmt.Errorf("send upgrade: %w", err)
	}

	// Start message reader
	go lc.readLoop()

	// Wait for Socket.IO namespace connect (40 message)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-lc.readyCh:
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for Socket.IO connect")
	}

	return nil
}

// readLoop reads WebSocket messages and dispatches them.
func (lc *LabsClient) readLoop() {
	for {
		_, message, err := lc.ws.ReadMessage()
		if err != nil {
			return // connection closed
		}

		msg := string(message)

		// Heartbeat: respond to PING with PONG
		if msg == "2" {
			lc.ws.WriteMessage(websocket.TextMessage, []byte("3"))
			continue
		}

		// Socket.IO CONNECT acknowledgment
		if strings.HasPrefix(msg, "40") {
			select {
			case <-lc.readyCh:
			default:
				close(lc.readyCh)
			}
			continue
		}

		// Socket.IO EVENT message (42) or ACK response (43)
		if strings.HasPrefix(msg, "42") || strings.HasPrefix(msg, "43") {
			// Strip the prefix (42 or 43) and any ACK ID digits
			payload := msg[2:]
			for len(payload) > 0 && payload[0] >= '0' && payload[0] <= '9' {
				payload = payload[1:]
			}
			if len(payload) > 0 {
				lc.handleEvent(payload)
			}
		}
	}
}

// handleEvent processes a Socket.IO event message.
func (lc *LabsClient) handleEvent(payload string) {
	var arr []json.RawMessage
	if err := json.Unmarshal([]byte(payload), &arr); err != nil {
		return
	}
	if len(arr) < 2 {
		return
	}

	var response map[string]any
	if err := json.Unmarshal(arr[1], &response); err != nil {
		return
	}

	// Check if this is a final response
	if _, hasFinal := response["final"]; hasFinal {
		lc.answerMu.Lock()
		lc.lastAnswer = &LabsResult{
			Output: getStr(response, "output"),
			Final:  response["final"] == true,
		}
		lc.answerMu.Unlock()

		// Signal that answer is ready
		select {
		case lc.answerCh <- struct{}{}:
		default:
		}
	}
}

// labsQueryTimeout is the maximum time to wait for a Labs response.
const labsQueryTimeout = 60 * time.Second

// Ask sends a query to Labs and waits for the complete response.
func (lc *LabsClient) Ask(ctx context.Context, query, model string) (*LabsResult, error) {
	lc.mu.Lock()

	// Validate model
	validModel := false
	for _, m := range LabsModels {
		if m == model {
			validModel = true
			break
		}
	}
	if !validModel {
		lc.mu.Unlock()
		return nil, fmt.Errorf("invalid labs model: %s (valid: %s)", model, strings.Join(LabsModels, ", "))
	}

	// Reset answer
	lc.answerMu.Lock()
	lc.lastAnswer = nil
	lc.answerMu.Unlock()

	// Drain answer channel
	select {
	case <-lc.answerCh:
	default:
	}

	// Add to history
	lc.history = append(lc.history, labsMessage{Role: "user", Content: query})

	// Build payload
	payload := labsPayload{
		Messages: lc.history,
		Model:    model,
		Source:   "default",
		Version:  apiVersion,
	}

	data, err := json.Marshal([]any{"perplexity_labs", payload})
	if err != nil {
		lc.mu.Unlock()
		return nil, err
	}

	// Send via WebSocket: "42" + JSON
	msg := "42" + string(data)
	if err := lc.ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		lc.mu.Unlock()
		return nil, fmt.Errorf("send query: %w", err)
	}

	lc.mu.Unlock()

	// Wait for response with timeout
	timer := time.NewTimer(labsQueryTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for Labs response (Cloudflare may be blocking queries)")
	case <-lc.answerCh:
	}

	lc.answerMu.Lock()
	answer := lc.lastAnswer
	lc.lastAnswer = nil
	lc.answerMu.Unlock()

	if answer == nil {
		return nil, fmt.Errorf("no response received")
	}

	// Add assistant response to history
	lc.mu.Lock()
	lc.history = append(lc.history, labsMessage{
		Role:    "assistant",
		Content: answer.Output,
	})
	lc.mu.Unlock()

	answer.Model = model
	return answer, nil
}

// Close closes the WebSocket connection.
func (lc *LabsClient) Close() error {
	if lc.ws != nil {
		return lc.ws.Close()
	}
	return nil
}

// ClearHistory resets the conversation history.
func (lc *LabsClient) ClearHistory() {
	lc.mu.Lock()
	lc.history = nil
	lc.mu.Unlock()
}
