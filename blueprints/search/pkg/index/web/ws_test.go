package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
	"github.com/gorilla/websocket"
)

// helper: dial a websocket connection to the test server's /ws endpoint.
func dialWS(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected 101, got %d", resp.StatusCode)
	}
	return conn
}

// helper: send a subscribe message for the given job IDs.
func subscribe(t *testing.T, conn *websocket.Conn, jobIDs ...string) {
	t.Helper()
	msg := map[string]any{
		"type":    "subscribe",
		"job_ids": jobIDs,
	}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("subscribe write: %v", err)
	}
}

// helper: send an unsubscribe message for the given job IDs.
func unsubscribe(t *testing.T, conn *websocket.Conn, jobIDs ...string) {
	t.Helper()
	msg := map[string]any{
		"type":    "unsubscribe",
		"job_ids": jobIDs,
	}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("unsubscribe write: %v", err)
	}
}

// helper: read one JSON message with a deadline.
func readJSON(t *testing.T, conn *websocket.Conn, timeout time.Duration) (map[string]any, bool) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	var msg map[string]any
	err := conn.ReadJSON(&msg)
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			return nil, false
		}
		if netErr, ok := err.(*json.SyntaxError); ok {
			t.Fatalf("json syntax error: %v", netErr)
		}
		// Timeout or other error — no message received.
		return nil, false
	}
	return msg, true
}

func TestWSHub_BroadcastToSubscriber(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	conn := dialWS(t, srv)
	defer conn.Close()

	// Subscribe to job "j1".
	subscribe(t, conn, "j1")

	// Give the hub a moment to process the subscription.
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message for "j1".
	hub.Broadcast("j1", map[string]string{"status": "running"})

	// Read the broadcast message.
	msg, ok := readJSON(t, conn, 2*time.Second)
	if !ok {
		t.Fatal("expected to receive broadcast message, got none")
	}
	if msg["status"] != "running" {
		t.Fatalf("expected status=running, got %v", msg["status"])
	}
}

func TestWSHub_UnsubscribedClientDoesNotReceive(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	conn := dialWS(t, srv)
	defer conn.Close()

	// Subscribe to "j2" only.
	subscribe(t, conn, "j2")

	// Give the hub a moment to process the subscription.
	time.Sleep(50 * time.Millisecond)

	// Broadcast for "j1" — this client should NOT receive it.
	hub.Broadcast("j1", map[string]string{"status": "running"})

	// Verify no message is received within a short timeout.
	msg, ok := readJSON(t, conn, 200*time.Millisecond)
	if ok {
		t.Fatalf("expected no message, but received: %v", msg)
	}
}

func TestWSHub_WildcardSubscription(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	conn := dialWS(t, srv)
	defer conn.Close()

	// Subscribe to wildcard "*".
	subscribe(t, conn, "*")

	time.Sleep(50 * time.Millisecond)

	// Broadcast for any job ID — wildcard client should receive it.
	hub.Broadcast("any-job-id", map[string]string{"event": "progress"})

	msg, ok := readJSON(t, conn, 2*time.Second)
	if !ok {
		t.Fatal("wildcard subscriber expected to receive broadcast, got none")
	}
	if msg["event"] != "progress" {
		t.Fatalf("expected event=progress, got %v", msg["event"])
	}
}

func TestWSHub_Unsubscribe(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	conn := dialWS(t, srv)
	defer conn.Close()

	// Subscribe then unsubscribe from "j1".
	subscribe(t, conn, "j1")
	time.Sleep(50 * time.Millisecond)

	unsubscribe(t, conn, "j1")
	time.Sleep(50 * time.Millisecond)

	// Broadcast for "j1" — should not arrive.
	hub.Broadcast("j1", map[string]string{"status": "done"})

	msg, ok := readJSON(t, conn, 200*time.Millisecond)
	if ok {
		t.Fatalf("expected no message after unsubscribe, but received: %v", msg)
	}
}

func TestWSHub_MultipleClients(t *testing.T) {
	hub := pipeline.NewHub()
	defer hub.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Client 1: subscribes to "j1".
	c1 := dialWS(t, srv)
	defer c1.Close()
	subscribe(t, c1, "j1")

	// Client 2: subscribes to "j2".
	c2 := dialWS(t, srv)
	defer c2.Close()
	subscribe(t, c2, "j2")

	time.Sleep(50 * time.Millisecond)

	// Broadcast for "j1" — only c1 should receive.
	hub.Broadcast("j1", map[string]string{"for": "c1"})

	msg1, ok1 := readJSON(t, c1, 2*time.Second)
	if !ok1 {
		t.Fatal("client 1 expected to receive broadcast")
	}
	if msg1["for"] != "c1" {
		t.Fatalf("client 1: expected for=c1, got %v", msg1["for"])
	}

	_, ok2 := readJSON(t, c2, 200*time.Millisecond)
	if ok2 {
		t.Fatal("client 2 should not have received broadcast for j1")
	}
}
