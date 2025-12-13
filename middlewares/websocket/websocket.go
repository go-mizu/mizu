// Package websocket provides WebSocket upgrade middleware for Mizu.
package websocket

import (
	"bufio"
	"crypto/sha1" //nolint:gosec // G505: SHA1 is required by WebSocket protocol (RFC 6455)
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/go-mizu/mizu"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Conn represents a WebSocket connection.
type Conn struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

// Handler is a WebSocket handler function.
type Handler func(c *mizu.Ctx, ws *Conn) error

// Options configures the WebSocket middleware.
type Options struct {
	// Origins is a list of allowed origins.
	// Default: all origins allowed.
	Origins []string

	// Subprotocols is a list of supported subprotocols.
	Subprotocols []string

	// CheckOrigin validates the request origin.
	// Default: allows all origins.
	CheckOrigin func(r *http.Request) bool
}

// New creates WebSocket middleware with handler.
func New(handler Handler) mizu.Middleware {
	return WithOptions(handler, Options{})
}

// WithOptions creates WebSocket middleware with custom options.
//
//nolint:cyclop // WebSocket upgrade handling requires multiple checks
func WithOptions(handler Handler, opts Options) mizu.Middleware {
	if opts.CheckOrigin == nil {
		if len(opts.Origins) > 0 {
			opts.CheckOrigin = func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				for _, allowed := range opts.Origins {
					if allowed == "*" || allowed == origin {
						return true
					}
				}
				return false
			}
		} else {
			opts.CheckOrigin = func(r *http.Request) bool {
				return true
			}
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()

			// Check if it's a WebSocket upgrade request
			if !IsWebSocketUpgrade(r) {
				return next(c)
			}

			// Check origin
			if !opts.CheckOrigin(r) {
				return c.Text(http.StatusForbidden, "forbidden origin")
			}

			// Get key
			key := r.Header.Get("Sec-WebSocket-Key")
			if key == "" {
				return c.Text(http.StatusBadRequest, "missing Sec-WebSocket-Key")
			}

			// Calculate accept key
			acceptKey := computeAcceptKey(key)

			// Select subprotocol
			var protocol string
			if len(opts.Subprotocols) > 0 {
				requested := r.Header.Get("Sec-WebSocket-Protocol")
				if requested != "" {
					for _, req := range strings.Split(requested, ",") {
						req = strings.TrimSpace(req)
						for _, supported := range opts.Subprotocols {
							if req == supported {
								protocol = req
								break
							}
						}
						if protocol != "" {
							break
						}
					}
				}
			}

			// Hijack connection
			hijacker, ok := c.Writer().(http.Hijacker)
			if !ok {
				return c.Text(http.StatusInternalServerError, "websocket: hijack not supported")
			}

			conn, bufrw, err := hijacker.Hijack()
			if err != nil {
				return err
			}

			// Send upgrade response
			response := "HTTP/1.1 101 Switching Protocols\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: " + acceptKey + "\r\n"

			if protocol != "" {
				response += "Sec-WebSocket-Protocol: " + protocol + "\r\n"
			}
			response += "\r\n"

			_, _ = bufrw.WriteString(response)
			_ = bufrw.Flush()

			// Create WebSocket connection
			ws := &Conn{
				conn:   conn,
				reader: bufrw.Reader,
				writer: bufrw.Writer,
			}

			// Call handler
			err = handler(c, ws)

			// Close connection
			_ = conn.Close()

			return err
		}
	}
}

// IsWebSocketUpgrade checks if request is a WebSocket upgrade.
func IsWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func computeAcceptKey(key string) string {
	h := sha1.New() //nolint:gosec // G401: SHA1 required by WebSocket protocol (RFC 6455)
	h.Write([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Message types
const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

// ReadMessage reads a message from the WebSocket connection.
//
//nolint:cyclop // WebSocket frame parsing requires multiple format checks
func (c *Conn) ReadMessage() (messageType int, data []byte, err error) {
	// Read first byte (FIN + opcode)
	b, err := c.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	// final := b&0x80 != 0
	opcode := int(b & 0x0F)

	// Read second byte (MASK + payload length)
	b, err = c.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	masked := b&0x80 != 0
	length := int(b & 0x7F)

	// Extended payload length
	switch length {
	case 126:
		lenBytes := make([]byte, 2)
		if _, err := io.ReadFull(c.reader, lenBytes); err != nil {
			return 0, nil, err
		}
		length = int(lenBytes[0])<<8 | int(lenBytes[1])
	case 127:
		lenBytes := make([]byte, 8)
		if _, err := io.ReadFull(c.reader, lenBytes); err != nil {
			return 0, nil, err
		}
		length = int(lenBytes[4])<<24 | int(lenBytes[5])<<16 | int(lenBytes[6])<<8 | int(lenBytes[7])
	}

	// Read masking key
	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(c.reader, mask); err != nil {
			return 0, nil, err
		}
	}

	// Read payload
	data = make([]byte, length)
	if _, err := io.ReadFull(c.reader, data); err != nil {
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

// WriteMessage writes a message to the WebSocket connection.
func (c *Conn) WriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// First byte: FIN + opcode
	_ = c.writer.WriteByte(0x80 | byte(messageType))

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

// WriteText writes a text message.
func (c *Conn) WriteText(text string) error {
	return c.WriteMessage(TextMessage, []byte(text))
}

// WriteBinary writes a binary message.
func (c *Conn) WriteBinary(data []byte) error {
	return c.WriteMessage(BinaryMessage, data)
}

// Close closes the WebSocket connection.
func (c *Conn) Close() error {
	_ = c.WriteMessage(CloseMessage, []byte{0x03, 0xe8}) // 1000 normal closure
	return c.conn.Close()
}

// Ping sends a ping message.
func (c *Conn) Ping(data []byte) error {
	return c.WriteMessage(PingMessage, data)
}

// Pong sends a pong message.
func (c *Conn) Pong(data []byte) error {
	return c.WriteMessage(PongMessage, data)
}

// Error types
var (
	ErrNotWebSocket = errors.New("not a websocket request")
	ErrClosed       = errors.New("connection closed")
)
