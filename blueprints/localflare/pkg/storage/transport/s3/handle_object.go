// File: lib/storage/transport/s3/handle_object.go
package s3

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	"github.com/go-mizu/mizu"
)

// s3ResponseBufferSize is the buffer size for HTTP response streaming.
// Using 512KB for optimal balance between memory and throughput.
const s3ResponseBufferSize = 512 * 1024

// s3BufferPool provides pooled buffers for HTTP response streaming.
var s3BufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, s3ResponseBufferSize)
		return &buf
	},
}

func getS3Buffer() []byte {
	return *s3BufferPool.Get().(*[]byte)
}

func putS3Buffer(buf []byte) {
	if cap(buf) >= s3ResponseBufferSize {
		s3BufferPool.Put(&buf)
	}
}

// handleObject handles object level operations mounted at:
//
//	basePath/:bucket/*key
//
// It covers:
//
//	GET    basePath/:bucket/*key  -> GetObject
//	PUT    basePath/:bucket/*key  -> PutObject or CopyObject (x-amz-copy-source)
//	DELETE basePath/:bucket/*key  -> DeleteObject
//	HEAD   basePath/:bucket/*key  -> HeadObject
func (s *Server) handleObject(c *mizu.Ctx) error {
	req, err := s.authAndParse(c)
	if err != nil {
		return writeError(c, err)
	}

	switch req.Op {
	case OpGetObject:
		return s.handleGetObject(c, req)
	case OpPutObject:
		return s.handlePutObject(c, req)
	case OpCopyObject:
		return s.handleCopyObject(c, req)
	case OpDeleteObject:
		return s.handleDeleteObject(c, req)
	case OpHeadObject:
		return s.handleHeadObject(c, req)
	// Multipart upload operations
	case OpCreateMultipartUpload:
		return s.handleCreateMultipartUpload(c, req)
	case OpUploadPart:
		return s.handleUploadPart(c, req)
	case OpListParts:
		return s.handleListParts(c, req)
	case OpCompleteMultipartUpload:
		return s.handleCompleteMultipartUpload(c, req)
	case OpAbortMultipartUpload:
		return s.handleAbortMultipartUpload(c, req)
	default:
		return writeError(c, ErrMethodNotAllowed)
	}
}

// handleGetObject implements:
//
//	GET basePath/:bucket/*key
//
// It supports single-range requests via the Range header:
//   - Range: bytes=start-end
//   - Range: bytes=start-
//   - Range: bytes=-suffix
func (s *Server) handleGetObject(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	// First fetch object metadata to know size / type / etag.
	meta, err := b.Stat(ctx, req.Key, storage.Options{})
	if err != nil {
		return writeError(c, mapError(err))
	}

	size := meta.Size
	if size < 0 {
		// For safety, treat unknown size as full-body only.
		size = 0
	}

	r := c.Request()
	w := c.Writer()

	// Always advertise byte range support.
	w.Header().Set("Accept-Ranges", "bytes")

	rangeHeader := r.Header.Get("Range")
	var (
		start      int64
		end        int64
		length     int64
		isPartial  bool
		openOffset int64
		openLimit  int64
	)

	if rangeHeader != "" && strings.HasPrefix(rangeHeader, "bytes=") && size > 0 {
		spec := strings.TrimPrefix(rangeHeader, "bytes=")
		parts := strings.SplitN(spec, "-", 2)

		if len(parts) == 2 {
			var parseErr error

			switch {
			// Suffix range: bytes=-N
			case parts[0] == "" && parts[1] != "":
				suffixLen, errParse := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				if errParse == nil && suffixLen > 0 {
					if suffixLen > size {
						suffixLen = size
					}
					start = size - suffixLen
					end = size - 1
					isPartial = true
				}

			// Open-ended range: bytes=start-
			case parts[0] != "" && parts[1] == "":
				start, parseErr = strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
				if parseErr == nil && start >= 0 && start < size {
					end = size - 1
					isPartial = true
				}

			// Explicit range: bytes=start-end
			case parts[0] != "" && parts[1] != "":
				start, parseErr = strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
				if parseErr == nil && start >= 0 && start < size {
					end, parseErr = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
					if parseErr == nil && end >= start {
						if end >= size {
							end = size - 1
						}
						isPartial = true
					}
				}
			}
		}
	}

	if isPartial {
		length = end - start + 1
		openOffset = start
		openLimit = length
	} else {
		// Full object.
		openOffset = 0
		openLimit = 0
		length = size
	}

	// Use storage backend range support if available via Open(offset, limit).
	rc, obj, err := b.Open(ctx, req.Key, openOffset, openLimit, storage.Options{})
	if err != nil {
		return writeError(c, mapError(err))
	}
	defer func() {
		_ = rc.Close()
	}()

	// Base headers from object metadata.
	contentType := obj.ContentType
	if contentType == "" {
		contentType = "binary/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	if obj.ETag != "" {
		w.Header().Set("ETag", quoteRawETag(obj.ETag))
	}
	if !obj.Updated.IsZero() {
		w.Header().Set("Last-Modified", obj.Updated.UTC().Format(http.TimeFormat))
	}
	if length > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	}

	if isPartial {
		w.Header().Set("Content-Range",
			"bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(size, 10),
		)
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// For GET we stream the body; for HEAD the router dispatches to handleHeadObject
	// so we do not send a body here when Method == HEAD.
	if r.Method != http.MethodHead {
		// Use pooled buffer for optimal streaming performance.
		buf := getS3Buffer()
		defer putS3Buffer(buf)
		_, _ = io.CopyBuffer(w, rc, buf)
	}
	return nil
}

// handlePutObject implements:
//
//	PUT basePath/:bucket/*key
//
// when x-amz-copy-source is not set.
func (s *Server) handlePutObject(c *mizu.Ctx, req *Request) error {
	r := c.Request()

	if s.cfg.MaxObjectSize > 0 && r.ContentLength > s.cfg.MaxObjectSize {
		return writeError(c, ErrEntityTooLarge)
	}

	ctx := contextFromCtx(c)

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "binary/octet-stream"
	}

	// Extract x-amz-meta-* headers into metadata.
	meta := map[string]string{}
	for name, values := range r.Header {
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "x-amz-meta-") {
			continue
		}
		key := strings.TrimPrefix(lower, "x-amz-meta-")
		if key == "" || len(values) == 0 {
			continue
		}
		meta[key] = values[0]
	}

	opts := storage.Options{}
	if len(meta) > 0 {
		opts["metadata"] = meta
	}

	obj, err := b.Write(ctx, req.Key, r.Body, r.ContentLength, contentType, opts)
	if err != nil {
		return writeError(c, mapError(err))
	}

	etag := obj.ETag
	if etag == "" && obj.Hash != nil {
		if v := obj.Hash["etag"]; v != "" {
			etag = v
		} else if v := obj.Hash["md5"]; v != "" {
			etag = v
		}
	}

	w := c.Writer()
	if etag != "" {
		w.Header().Set("ETag", quoteRawETag(etag))
	}
	// S3 returns 200 for simple PUT Object.
	w.WriteHeader(http.StatusOK)
	return nil
}

// handleCopyObject implements:
//
//	PUT basePath/:bucket/*key with header x-amz-copy-source
//
// This is a minimal CopyObject compatible with most SDKs.
func (s *Server) handleCopyObject(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)
	r := c.Request()

	dstBucket := req.Bucket
	dstKey := req.Key

	src := r.Header.Get("x-amz-copy-source") // format: /bucket/key or bucket/key
	src = strings.TrimSpace(src)
	src = strings.TrimPrefix(src, "/")
	parts := strings.SplitN(src, "/", 2)
	if len(parts) != 2 {
		return writeError(c, ErrInvalidRequest)
	}
	srcBucket := parts[0]
	srcKey := parts[1]

	db := s.stor.Bucket(dstBucket)
	if db == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	obj, err := db.Copy(ctx, dstKey, srcBucket, srcKey, storage.Options{})
	if err != nil {
		return writeError(c, mapError(err))
	}

	etag := obj.ETag
	if etag == "" && obj.Hash != nil {
		if v := obj.Hash["etag"]; v != "" {
			etag = v
		}
	}

	// Minimal CopyObjectResult XML.
	type copyObjectResult struct {
		XMLName      xml.Name  `xml:"CopyObjectResult"`
		LastModified time.Time `xml:"LastModified"`
		ETag         string    `xml:"ETag"`
	}

	mod := obj.Updated
	if mod.IsZero() {
		mod = s.cfg.Clock().UTC()
	}

	resp := copyObjectResult{
		LastModified: mod.UTC(),
		ETag:         quoteRawETag(etag),
	}

	return writeXML(c, http.StatusOK, resp)
}

// handleDeleteObject implements:
//
//	DELETE basePath/:bucket/*key
func (s *Server) handleDeleteObject(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	err := b.Delete(ctx, req.Key, storage.Options{})
	// S3 returns 204 for a successful delete (even if the key did not exist).
	// Ignore ErrNotExist per S3 semantics.
	if err != nil && !errors.Is(err, storage.ErrNotExist) {
		return writeError(c, mapError(err))
	}
	c.Writer().WriteHeader(http.StatusNoContent)
	return nil
}

// handleHeadObject implements:
//
//	HEAD basePath/:bucket/*key
func (s *Server) handleHeadObject(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	obj, err := b.Stat(ctx, req.Key, storage.Options{})
	if err != nil {
		return writeError(c, mapError(err))
	}

	w := c.Writer()
	if obj.ContentType != "" {
		w.Header().Set("Content-Type", obj.ContentType)
	}
	if obj.ETag != "" {
		w.Header().Set("ETag", quoteRawETag(obj.ETag))
	}
	if !obj.Updated.IsZero() {
		w.Header().Set("Last-Modified", obj.Updated.UTC().Format(http.TimeFormat))
	}
	if obj.Size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(obj.Size, 10))
	}
	// S3 returns 200 for a successful HEAD Object.
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)
	return nil
}
