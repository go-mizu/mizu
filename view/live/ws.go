package live

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// WebSocket opcodes.
const (
	opContinuation = 0x0
	opText         = 0x1
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

// WebSocket magic GUID for handshake.
const wsMagicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Errors.
var (
	ErrNotWebSocket    = errors.New("not a websocket upgrade request")
	ErrBadHandshake    = errors.New("websocket handshake failed")
	ErrConnectionClose = errors.New("connection closed")
	ErrFrameTooLarge   = errors.New("frame too large")
)

// wsConn represents a WebSocket connection.
type wsConn struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	isServ bool // true if server-side

	readMu  sync.Mutex
	writeMu sync.Mutex

	closed   bool
	closeMu  sync.Mutex
	closeErr error

	req *http.Request // original HTTP request (server only)
}

// Request returns the original HTTP request (server-side only).
func (c *wsConn) Request() *http.Request {
	return c.req
}

// Close closes the WebSocket connection.
func (c *wsConn) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	// Send close frame.
	c.writeFrame(opClose, []byte{})
	return c.conn.Close()
}

// ReadMessage reads a text message from the connection.
func (c *wsConn) ReadMessage() (string, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	for {
		op, data, err := c.readFrame()
		if err != nil {
			return "", err
		}

		switch op {
		case opText:
			return string(data), nil
		case opPing:
			// Send pong.
			c.writeFrame(opPong, data)
		case opPong:
			// Ignore pongs.
		case opClose:
			c.closeMu.Lock()
			c.closed = true
			c.closeMu.Unlock()
			return "", ErrConnectionClose
		}
	}
}

// WriteMessage writes a text message to the connection.
func (c *wsConn) WriteMessage(data string) error {
	return c.writeFrame(opText, []byte(data))
}

// readFrame reads a single WebSocket frame.
func (c *wsConn) readFrame() (byte, []byte, error) {
	// Read first 2 bytes.
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return 0, nil, err
	}

	fin := header[0]&0x80 != 0
	op := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7F)

	// Extended payload length.
	if length == 126 {
		ext := make([]byte, 2)
		if _, err := io.ReadFull(c.reader, ext); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext))
	} else if length == 127 {
		ext := make([]byte, 8)
		if _, err := io.ReadFull(c.reader, ext); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext)
	}

	// Sanity check.
	if length > 10*1024*1024 { // 10MB max
		return 0, nil, ErrFrameTooLarge
	}

	// Read masking key if present.
	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(c.reader, mask); err != nil {
			return 0, nil, err
		}
	}

	// Read payload.
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(c.reader, payload); err != nil {
			return 0, nil, err
		}
	}

	// Unmask if needed.
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	// Handle continuation frames (simplified: we just read single frames).
	if !fin {
		// For now, we don't support fragmented messages.
		return 0, nil, errors.New("fragmented frames not supported")
	}

	return op, payload, nil
}

// writeFrame writes a WebSocket frame.
func (c *wsConn) writeFrame(op byte, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return ErrConnectionClose
	}
	c.closeMu.Unlock()

	length := len(data)

	// First byte: FIN + opcode.
	frame := []byte{0x80 | op}

	// Second byte: mask bit + payload length.
	// Client frames must be masked, server frames are not masked.
	maskBit := byte(0)
	if !c.isServ {
		maskBit = 0x80
	}

	if length <= 125 {
		frame = append(frame, maskBit|byte(length))
	} else if length <= 65535 {
		frame = append(frame, maskBit|126)
		frame = append(frame, byte(length>>8), byte(length))
	} else {
		frame = append(frame, maskBit|127)
		for i := 7; i >= 0; i-- {
			frame = append(frame, byte(length>>(i*8)))
		}
	}

	// Client must mask data.
	if !c.isServ {
		mask := make([]byte, 4)
		rand.Read(mask)
		frame = append(frame, mask...)

		masked := make([]byte, length)
		for i := range data {
			masked[i] = data[i] ^ mask[i%4]
		}
		frame = append(frame, masked...)
	} else {
		frame = append(frame, data...)
	}

	_, err := c.writer.Write(frame)
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

// upgradeHTTP upgrades an HTTP connection to WebSocket (server-side).
func upgradeHTTP(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	// Validate upgrade request.
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, ErrNotWebSocket
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		return nil, ErrNotWebSocket
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, ErrBadHandshake
	}

	// Compute accept key.
	h := sha1.New()
	h.Write([]byte(key + wsMagicGUID))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Hijack the connection.
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("server doesn't support hijacking")
	}

	conn, brw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	// Send upgrade response.
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	if _, err := brw.WriteString(response); err != nil {
		conn.Close()
		return nil, err
	}
	if err := brw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}

	return &wsConn{
		conn:   conn,
		reader: brw.Reader,
		writer: brw.Writer,
		isServ: true,
		req:    r,
	}, nil
}

// dialWebSocket connects to a WebSocket server (client-side, for testing).
func dialWebSocket(url, origin string) (*wsConn, error) {
	// Parse URL to extract host and path.
	var host, path string
	if strings.HasPrefix(url, "ws://") {
		url = strings.TrimPrefix(url, "ws://")
	} else if strings.HasPrefix(url, "wss://") {
		return nil, errors.New("wss not supported in test client")
	}

	idx := strings.Index(url, "/")
	if idx == -1 {
		host = url
		path = "/"
	} else {
		host = url[:idx]
		path = url[idx:]
	}

	// Add default port if not present.
	if !strings.Contains(host, ":") {
		host = host + ":80"
	}

	// Connect.
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	// Generate key.
	keyBytes := make([]byte, 16)
	rand.Read(keyBytes)
	key := base64.StdEncoding.EncodeToString(keyBytes)

	// Send handshake.
	hostHeader := host
	if idx := strings.LastIndex(hostHeader, ":80"); idx == len(hostHeader)-3 {
		hostHeader = hostHeader[:idx]
	}

	handshake := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + hostHeader + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n"
	if origin != "" {
		handshake += "Origin: " + origin + "\r\n"
	}
	handshake += "\r\n"

	if _, err := conn.Write([]byte(handshake)); err != nil {
		conn.Close()
		return nil, err
	}

	// Read response.
	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}

	if !strings.Contains(statusLine, "101") {
		conn.Close()
		return nil, ErrBadHandshake
	}

	// Read headers until empty line.
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, err
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	return &wsConn{
		conn:   conn,
		reader: reader,
		writer: bufio.NewWriter(conn),
		isServ: false,
	}, nil
}
