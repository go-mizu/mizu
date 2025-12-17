package live

import (
	"bufio"
	"crypto/sha1" //nolint:gosec // G505: SHA1 is required by WebSocket protocol (RFC 6455)
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"
	stdsync "sync"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// WebSocket message types
const (
	wsTextMessage   = 1
	wsBinaryMessage = 2
	wsCloseMessage  = 8
	wsPingMessage   = 9
	wsPongMessage   = 10
)

// wsConn represents a WebSocket connection.
type wsConn struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	mu     stdsync.Mutex
}

// handleConn handles a new WebSocket connection.
//
//nolint:cyclop // Connection handling requires multiple steps
func (srv *Server) handleConn(w http.ResponseWriter, r *http.Request) {
	// Check if it's a WebSocket upgrade request
	if !isWebSocketUpgrade(r) {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return
	}

	// Check origin
	if len(srv.opts.Origins) > 0 {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range srv.opts.Origins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
	}

	// Authenticate if OnAuth is set
	var meta Meta
	if srv.opts.OnAuth != nil {
		var err error
		meta, err = srv.opts.OnAuth(r.Context(), r)
		if err != nil {
			http.Error(w, "authentication failed", http.StatusUnauthorized)
			return
		}
	}

	// Get WebSocket key
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	// Calculate accept key
	acceptKey := computeAcceptKey(key)

	// Hijack connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket: hijack not supported", http.StatusInternalServerError)
		return
	}

	conn, bufrw, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send upgrade response
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	_, _ = bufrw.WriteString(response)
	_ = bufrw.Flush()

	// Create WebSocket connection wrapper
	ws := &wsConn{
		conn:   conn,
		reader: bufrw.Reader,
		writer: bufrw.Writer,
	}

	// Create session
	session := newSession(srv.opts.IDGenerator(), meta, srv.opts.QueueSize, srv)
	srv.addSession(session)

	// Start write loop
	go srv.writeLoop(session, ws)

	// Run read loop (blocking)
	readErr := srv.readLoop(r, session, ws)

	// Cleanup
	session.closeWithError(readErr)
	srv.removeSession(session)
	_ = conn.Close()

	// Call OnClose callback
	if srv.opts.OnClose != nil {
		srv.opts.OnClose(session, readErr)
	}
}

// readLoop reads messages from the WebSocket and dispatches to OnMessage.
func (srv *Server) readLoop(r *http.Request, session *Session, ws *wsConn) error {
	ctx := r.Context()

	for {
		msgType, data, err := ws.readMessage()
		if err != nil {
			return err
		}

		// Handle control frames
		switch msgType {
		case wsCloseMessage:
			return nil
		case wsPingMessage:
			_ = ws.writeMessage(wsPongMessage, data)
			continue
		case wsPongMessage:
			continue
		}

		// Only process text/binary messages
		if msgType != wsTextMessage && msgType != wsBinaryMessage {
			continue
		}

		// Decode message
		msg, err := srv.opts.Codec.Decode(data)
		if err != nil {
			continue // Skip invalid messages
		}

		// Dispatch to handler
		if srv.opts.OnMessage != nil {
			srv.opts.OnMessage(ctx, session, msg)
		}
	}
}

// writeLoop sends messages from the session queue to the WebSocket.
func (srv *Server) writeLoop(session *Session, ws *wsConn) {
	for {
		select {
		case msg := <-session.sendCh:
			data, err := srv.opts.Codec.Encode(msg)
			if err != nil {
				continue
			}
			if err := ws.writeMessage(wsTextMessage, data); err != nil {
				session.closeWithError(err)
				return
			}
		case <-session.doneCh:
			// Send close frame
			_ = ws.writeMessage(wsCloseMessage, []byte{0x03, 0xe8}) // 1000 normal closure
			return
		}
	}
}

// isWebSocketUpgrade checks if request is a WebSocket upgrade.
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// computeAcceptKey computes the Sec-WebSocket-Accept header value.
func computeAcceptKey(key string) string {
	h := sha1.New() //nolint:gosec // G401: SHA1 required by WebSocket protocol (RFC 6455)
	h.Write([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// readMessage reads a WebSocket frame.
//
//nolint:cyclop // WebSocket frame parsing requires multiple format checks
func (ws *wsConn) readMessage() (messageType int, data []byte, err error) {
	// Read first byte (FIN + opcode)
	b, err := ws.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	opcode := int(b & 0x0F)

	// Read second byte (MASK + payload length)
	b, err = ws.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	masked := b&0x80 != 0
	length := int(b & 0x7F)

	// Extended payload length
	switch length {
	case 126:
		lenBytes := make([]byte, 2)
		if _, err := io.ReadFull(ws.reader, lenBytes); err != nil {
			return 0, nil, err
		}
		length = int(lenBytes[0])<<8 | int(lenBytes[1])
	case 127:
		lenBytes := make([]byte, 8)
		if _, err := io.ReadFull(ws.reader, lenBytes); err != nil {
			return 0, nil, err
		}
		length = int(lenBytes[4])<<24 | int(lenBytes[5])<<16 | int(lenBytes[6])<<8 | int(lenBytes[7])
	}

	// Read masking key
	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(ws.reader, mask); err != nil {
			return 0, nil, err
		}
	}

	// Read payload
	data = make([]byte, length)
	if _, err := io.ReadFull(ws.reader, data); err != nil {
		return 0, nil, err
	}

	// Unmask data
	if masked {
		for i := range data {
			data[i] ^= mask[i%4]
		}
	}

	return opcode, data, nil
}

// writeMessage writes a WebSocket frame.
func (ws *wsConn) writeMessage(messageType int, data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	// First byte: FIN + opcode
	_ = ws.writer.WriteByte(0x80 | byte(messageType))

	// Second byte: payload length (server never masks)
	length := len(data)
	if length <= 125 {
		_ = ws.writer.WriteByte(byte(length))
	} else if length <= 65535 {
		_ = ws.writer.WriteByte(126)
		_ = ws.writer.WriteByte(byte(length >> 8))
		_ = ws.writer.WriteByte(byte(length))
	} else {
		_ = ws.writer.WriteByte(127)
		for i := 7; i >= 0; i-- {
			_ = ws.writer.WriteByte(byte(length >> (8 * i)))
		}
	}

	// Write payload
	_, _ = ws.writer.Write(data)

	return ws.writer.Flush()
}
