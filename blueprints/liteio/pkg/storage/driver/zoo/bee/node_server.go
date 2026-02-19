package bee

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

// HTTPNodeServer exposes a single Bee storage node over HTTP.
type HTTPNodeServer struct {
	node *nodeEngine
	mux  *http.ServeMux
}

// NewHTTPNodeServer creates an HTTP-exposed Bee node.
func NewHTTPNodeServer(dataDir, syncMode string, inlineKB int) (*HTTPNodeServer, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("bee: data dir is required")
	}
	if inlineKB <= 0 {
		inlineKB = defaultInlineLimit / 1024
	}

	nodePath := filepath.Join(dataDir, "bee.log")
	n, err := openNodeEngine(0, nodePath, syncMode, int64(inlineKB*1024))
	if err != nil {
		return nil, err
	}

	s := &HTTPNodeServer{node: n, mux: http.NewServeMux()}
	s.routes()
	return s, nil
}

// Handler returns the HTTP handler for this node.
func (s *HTTPNodeServer) Handler() http.Handler { return s.mux }

// Close releases node resources.
func (s *HTTPNodeServer) Close() error { return s.node.close() }

func (s *HTTPNodeServer) routes() {
	s.mux.HandleFunc("/v1/ping", s.handlePing)
	s.mux.HandleFunc("/v1/object", s.handleObject)
	s.mux.HandleFunc("/v1/list", s.handleList)
	s.mux.HandleFunc("/v1/has-bucket", s.handleHasBucket)
	s.mux.HandleFunc("/v1/buckets", s.handleBuckets)
	s.mux.HandleFunc("/v1/bucket", s.handleBucket)
}

func (s *HTTPNodeServer) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *HTTPNodeServer) handleObject(w http.ResponseWriter, r *http.Request) {
	bucket := strings.TrimSpace(r.URL.Query().Get("bucket"))
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if bucket == "" || key == "" {
		http.Error(w, "bucket and key are required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleObjectPut(w, r, bucket, key)
	case http.MethodGet:
		s.handleObjectGet(w, r, bucket, key)
	case http.MethodHead:
		s.handleObjectHead(w, r, bucket, key)
	case http.MethodDelete:
		s.handleObjectDelete(w, r, bucket, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *HTTPNodeServer) handleObjectPut(w http.ResponseWriter, r *http.Request, bucket, key string) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	contentType := r.URL.Query().Get("content_type")
	ts := queryTimestamp(r, time.Now().UnixNano())

	e, err := s.node.write(bucket, key, contentType, data, ts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setEntryHeaders(w.Header(), e)
	w.WriteHeader(http.StatusOK)
}

func (s *HTTPNodeServer) handleObjectGet(w http.ResponseWriter, r *http.Request, bucket, key string) {
	data, e, err := s.node.read(bucket, key)
	if err != nil {
		if errors.Is(err, storage.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setEntryHeaders(w.Header(), e)
	if e.contentType != "" {
		w.Header().Set("Content-Type", e.contentType)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *HTTPNodeServer) handleObjectHead(w http.ResponseWriter, r *http.Request, bucket, key string) {
	e, err := s.node.stat(bucket, key)
	if err != nil {
		if errors.Is(err, storage.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setEntryHeaders(w.Header(), e)
	w.WriteHeader(http.StatusOK)
}

func (s *HTTPNodeServer) handleObjectDelete(w http.ResponseWriter, r *http.Request, bucket, key string) {
	ts := queryTimestamp(r, time.Now().UnixNano())
	_, err := s.node.delete(bucket, key, ts)
	if err != nil {
		if errors.Is(err, storage.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPNodeServer) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bucket := strings.TrimSpace(r.URL.Query().Get("bucket"))
	if bucket == "" {
		http.Error(w, "bucket is required", http.StatusBadRequest)
		return
	}
	prefix := r.URL.Query().Get("prefix")
	recursive := queryBoolFromString(r.URL.Query().Get("recursive"), true)

	items := s.node.list(bucket, prefix, recursive)
	out := make([]listResponseItem, 0, len(items))
	for _, it := range items {
		out = append(out, listResponseItem{
			Key:         it.key,
			Size:        it.entry.size,
			ContentType: it.entry.contentType,
			Created:     it.entry.created,
			Updated:     it.entry.updated,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *HTTPNodeServer) handleHasBucket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bucket := strings.TrimSpace(r.URL.Query().Get("bucket"))
	if bucket == "" {
		http.Error(w, "bucket is required", http.StatusBadRequest)
		return
	}
	if s.node.hasBucket(bucket) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func (s *HTTPNodeServer) handleBuckets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.node.bucketNames())
}

func (s *HTTPNodeServer) handleBucket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bucket := strings.TrimSpace(r.URL.Query().Get("bucket"))
	if bucket == "" {
		http.Error(w, "bucket is required", http.StatusBadRequest)
		return
	}
	ts := queryTimestamp(r, time.Now().UnixNano())
	s.node.deleteBucket(bucket, ts)
	w.WriteHeader(http.StatusNoContent)
}

func setEntryHeaders(h http.Header, e *nodeEntry) {
	h.Set("X-Bee-Size", strconv.FormatInt(e.size, 10))
	h.Set("X-Bee-Content-Type", e.contentType)
	h.Set("X-Bee-Created", strconv.FormatInt(e.created, 10))
	h.Set("X-Bee-Updated", strconv.FormatInt(e.updated, 10))
}

func queryTimestamp(r *http.Request, def int64) int64 {
	v := strings.TrimSpace(r.URL.Query().Get("timestamp"))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func queryBoolFromString(v string, def bool) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
