package fts_zig

import (
	"bufio"
	"encoding/binary"
	"errors"
	"net"
	"os"
	"sync"
)

// ipcDriver implements Driver using Unix socket IPC.
type ipcDriver struct {
	mu         sync.RWMutex
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	profile    Profile
	socketPath string
	built      bool
	docCount   uint32
	docs       []string // Buffer for building
}

// IPC message types
const (
	msgAddDoc   uint8 = 1
	msgBuild    uint8 = 2
	msgSearch   uint8 = 3
	msgStats    uint8 = 4
	msgClose    uint8 = 5
	msgResponse uint8 = 128
)

func newIPCDriver(cfg Config) (Driver, error) {
	socketPath := cfg.IPCSocketPath
	if socketPath == "" {
		socketPath = "/tmp/fts_zig.sock"
	}

	// Check if socket exists (server running)
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		// Server not running, use in-memory fallback
		return &ipcDriver{
			profile:    cfg.Profile,
			socketPath: socketPath,
			docs:       make([]string, 0),
		}, nil
	}

	// Connect to server
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		// Fallback to in-memory
		return &ipcDriver{
			profile:    cfg.Profile,
			socketPath: socketPath,
			docs:       make([]string, 0),
		}, nil
	}

	return &ipcDriver{
		conn:       conn,
		reader:     bufio.NewReader(conn),
		writer:     bufio.NewWriter(conn),
		profile:    cfg.Profile,
		socketPath: socketPath,
	}, nil
}

func (d *ipcDriver) AddDocument(text string) (uint32, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.built {
		return 0, ErrAlreadyBuilt
	}

	// If no connection, buffer locally
	if d.conn == nil {
		d.docs = append(d.docs, text)
		docID := d.docCount
		d.docCount++
		return docID, nil
	}

	// Send to server
	if err := d.sendMessage(msgAddDoc, []byte(text)); err != nil {
		return 0, err
	}

	// Read response
	_, err := d.readResponse()
	if err != nil {
		return 0, err
	}

	docID := d.docCount
	d.docCount++
	return docID, nil
}

func (d *ipcDriver) AddDocuments(texts []string) error {
	for _, text := range texts {
		if _, err := d.AddDocument(text); err != nil {
			return err
		}
	}
	return nil
}

func (d *ipcDriver) Build() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.built {
		return ErrAlreadyBuilt
	}

	if d.conn == nil {
		// No server, just mark as built
		d.built = true
		return nil
	}

	if err := d.sendMessage(msgBuild, nil); err != nil {
		return err
	}

	_, err := d.readResponse()
	if err != nil {
		return err
	}

	d.built = true
	return nil
}

func (d *ipcDriver) Search(query string, limit int) ([]SearchResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.built {
		return nil, ErrNotBuilt
	}

	if d.conn == nil {
		// No server, do simple in-memory search
		return d.searchLocal(query, limit), nil
	}

	// Encode query + limit
	payload := make([]byte, 4+len(query))
	binary.LittleEndian.PutUint32(payload[:4], uint32(limit))
	copy(payload[4:], query)

	if err := d.sendMessage(msgSearch, payload); err != nil {
		return nil, err
	}

	resp, err := d.readResponse()
	if err != nil {
		return nil, err
	}

	// Decode results
	if len(resp) < 4 {
		return nil, errors.New("invalid response")
	}

	count := binary.LittleEndian.Uint32(resp[:4])
	results := make([]SearchResult, count)

	offset := 4
	for i := uint32(0); i < count && offset+8 <= len(resp); i++ {
		results[i].DocID = binary.LittleEndian.Uint32(resp[offset:])
		results[i].Score = float32(binary.LittleEndian.Uint32(resp[offset+4:]))
		offset += 8
	}

	return results, nil
}

// Simple local search (when no server)
func (d *ipcDriver) searchLocal(query string, limit int) []SearchResult {
	// Very basic substring search
	var results []SearchResult
	for i, doc := range d.docs {
		if len(results) >= limit {
			break
		}
		if containsWord(doc, query) {
			results = append(results, SearchResult{
				DocID: uint32(i),
				Score: 1.0,
			})
		}
	}
	return results
}

func containsWord(doc, word string) bool {
	// Simple word boundary check
	docLower := toLower(doc)
	wordLower := toLower(word)

	for i := 0; i <= len(docLower)-len(wordLower); i++ {
		if docLower[i:i+len(wordLower)] == wordLower {
			// Check word boundaries
			leftOK := i == 0 || !isLetter(docLower[i-1])
			rightOK := i+len(wordLower) == len(docLower) || !isLetter(docLower[i+len(wordLower)])
			if leftOK && rightOK {
				return true
			}
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func (d *ipcDriver) Stats() (Stats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return Stats{DocCount: d.docCount}, nil
}

func (d *ipcDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.conn != nil {
		_ = d.sendMessage(msgClose, nil)
		d.conn.Close()
		d.conn = nil
	}

	d.docs = nil
	return nil
}

func (d *ipcDriver) sendMessage(msgType uint8, payload []byte) error {
	// Message format: [type:1][length:4][payload]
	header := make([]byte, 5)
	header[0] = msgType
	binary.LittleEndian.PutUint32(header[1:], uint32(len(payload)))

	if _, err := d.writer.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := d.writer.Write(payload); err != nil {
			return err
		}
	}
	return d.writer.Flush()
}

func (d *ipcDriver) readResponse() ([]byte, error) {
	header := make([]byte, 5)
	if _, err := d.reader.Read(header); err != nil {
		return nil, err
	}

	if header[0] != msgResponse {
		return nil, errors.New("unexpected message type")
	}

	length := binary.LittleEndian.Uint32(header[1:])
	if length == 0 {
		return nil, nil
	}

	payload := make([]byte, length)
	if _, err := d.reader.Read(payload); err != nil {
		return nil, err
	}

	return payload, nil
}
