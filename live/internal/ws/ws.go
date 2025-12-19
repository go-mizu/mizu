// Package ws provides a minimal RFC 6455 WebSocket implementation.
// This is an internal package; the live package exposes higher-level APIs.
package ws

import (
	"bufio"
	"crypto/sha1" //nolint:gosec // G505: SHA1 is required by WebSocket protocol (RFC 6455)
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// WebSocket magic GUID per RFC 6455.
const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Message types (opcodes) per RFC 6455.
const (
	OpContinuation = 0
	OpText         = 1
	OpBinary       = 2
	OpClose        = 8
	OpPing         = 9
	OpPong         = 10
)

// Close codes per RFC 6455.
const (
	CloseNormal        = 1000
	CloseGoingAway     = 1001
	CloseProtocolError = 1002
	CloseUnsupported   = 1003
	CloseTooLarge      = 1009
)

// maxInt is the maximum value of int for overflow checks.
const maxInt = int(^uint(0) >> 1)

// Errors returned by WebSocket operations.
var (
	// ErrProtocolError indicates a WebSocket protocol violation.
	ErrProtocolError = errors.New("ws: protocol error")

	// ErrMessageTooLarge indicates a message exceeds the read limit.
	ErrMessageTooLarge = errors.New("ws: message too large")
)

// Conn represents a WebSocket connection.
type Conn struct {
	conn      net.Conn
	reader    *bufio.Reader
	writer    *bufio.Writer
	mu        sync.Mutex
	readLimit int
}

// Upgrade performs the WebSocket handshake and returns a Conn.
// It validates the request headers and sends the 101 Switching Protocols response.
//
//nolint:cyclop // Handshake requires multiple validation steps
func Upgrade(w http.ResponseWriter, r *http.Request, readLimit int) (*Conn, error) {
	// Check if it's a WebSocket upgrade request
	if !isUpgradeRequest(r) {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return nil, errors.New("ws: not an upgrade request")
	}

	// Validate WebSocket version (RFC 6455 requires version 13)
	version := r.Header.Get("Sec-WebSocket-Version")
	if version != "13" {
		w.Header().Set("Sec-WebSocket-Version", "13")
		http.Error(w, "unsupported WebSocket version", http.StatusUpgradeRequired)
		return nil, errors.New("ws: unsupported version")
	}

	// Get and validate WebSocket key
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, errors.New("ws: missing key")
	}
	if !validateKey(key) {
		http.Error(w, "invalid Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, errors.New("ws: invalid key")
	}

	// Calculate accept key
	acceptKey := computeAcceptKey(key)

	// Hijack connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket: hijack not supported", http.StatusInternalServerError)
		return nil, errors.New("ws: hijack not supported")
	}

	conn, bufrw, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	// Send upgrade response
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	_, _ = bufrw.WriteString(response)
	_ = bufrw.Flush()

	return &Conn{
		conn:      conn,
		reader:    bufrw.Reader,
		writer:    bufrw.Writer,
		readLimit: readLimit,
	}, nil
}

// NetConn returns the underlying net.Conn.
func (c *Conn) NetConn() net.Conn {
	return c.conn
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// ReadMessage reads a WebSocket frame.
// It enforces RFC 6455 requirements:
// - Client frames must be masked
// - Control frames must have payload <= 125 bytes and FIN=1
// - Fragmentation is rejected (FIN=0 or opcode=0 continuation)
// - Message size is limited by readLimit
//
//nolint:cyclop // Frame parsing requires multiple format checks
func (c *Conn) ReadMessage() (opcode int, data []byte, err error) {
	// Read first byte (FIN + RSV + opcode)
	b, err := c.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	fin := b&0x80 != 0
	opcode = int(b & 0x0F)

	// Reject continuation frames (we don't support fragmentation)
	if opcode == OpContinuation {
		return 0, nil, ErrProtocolError
	}

	// Reject fragmented data frames (FIN=0)
	// Control frames (8-10) must always have FIN=1
	if !fin {
		return 0, nil, ErrProtocolError
	}

	// Read second byte (MASK + payload length)
	b, err = c.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	masked := b&0x80 != 0
	length := int(b & 0x7F)

	// RFC 6455: Client frames MUST be masked
	if !masked {
		return 0, nil, ErrProtocolError
	}

	// Control frames must have payload length <= 125
	isControlFrame := opcode >= OpClose
	if isControlFrame && length > 125 {
		return 0, nil, ErrProtocolError
	}

	// Extended payload length
	switch length {
	case 126:
		lenBytes := make([]byte, 2)
		if _, err := io.ReadFull(c.reader, lenBytes); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint16(lenBytes))
	case 127:
		lenBytes := make([]byte, 8)
		if _, err := io.ReadFull(c.reader, lenBytes); err != nil {
			return 0, nil, err
		}
		length64 := binary.BigEndian.Uint64(lenBytes)
		// Check for overflow and read limit
		if c.readLimit > 0 && length64 > uint64(c.readLimit) {
			return 0, nil, ErrMessageTooLarge
		}
		if length64 > uint64(maxInt) {
			return 0, nil, ErrMessageTooLarge
		}
		length = int(length64)
	}

	// Check read limit for all payload lengths
	if c.readLimit > 0 && length > c.readLimit {
		return 0, nil, ErrMessageTooLarge
	}

	// Read masking key (always present since we require masked frames)
	mask := make([]byte, 4)
	if _, err := io.ReadFull(c.reader, mask); err != nil {
		return 0, nil, err
	}

	// Read payload
	data = make([]byte, length)
	if _, err := io.ReadFull(c.reader, data); err != nil {
		return 0, nil, err
	}

	// Unmask data
	for i := range data {
		data[i] ^= mask[i%4]
	}

	return opcode, data, nil
}

// WriteMessage writes a WebSocket frame.
func (c *Conn) WriteMessage(opcode int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// First byte: FIN + opcode
	_ = c.writer.WriteByte(0x80 | byte(opcode))

	// Second byte: payload length (server never masks)
	length := len(data)
	if length <= 125 {
		_ = c.writer.WriteByte(byte(length))
	} else if length <= 65535 {
		_ = c.writer.WriteByte(126)
		_ = c.writer.WriteByte(byte(length >> 8))
		_ = c.writer.WriteByte(byte(length))
	} else {
		_ = c.writer.WriteByte(127)
		for i := 7; i >= 0; i-- {
			_ = c.writer.WriteByte(byte(length >> (8 * i)))
		}
	}

	// Write payload
	_, _ = c.writer.Write(data)

	return c.writer.Flush()
}

// WriteClose sends a close frame with the given code.
func (c *Conn) WriteClose(code int) error {
	data := []byte{byte(code >> 8), byte(code)}
	return c.WriteMessage(OpClose, data)
}

// isUpgradeRequest checks if request is a WebSocket upgrade.
func isUpgradeRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// validateKey validates that the Sec-WebSocket-Key is valid base64
// and decodes to exactly 16 bytes (per RFC 6455).
func validateKey(key string) bool {
	decoded, err := base64.StdEncoding.DecodeString(key)
	return err == nil && len(decoded) == 16
}

// computeAcceptKey computes the Sec-WebSocket-Accept header value.
func computeAcceptKey(key string) string {
	h := sha1.New() //nolint:gosec // G401: SHA1 required by WebSocket protocol (RFC 6455)
	h.Write([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// IsUpgradeRequest checks if a request is a WebSocket upgrade request.
// This is exported for use by the live package for pre-upgrade validation.
func IsUpgradeRequest(r *http.Request) bool {
	return isUpgradeRequest(r)
}

// ValidateKey validates a Sec-WebSocket-Key header value.
// This is exported for use by the live package for pre-upgrade validation.
func ValidateKey(key string) bool {
	return validateKey(key)
}
