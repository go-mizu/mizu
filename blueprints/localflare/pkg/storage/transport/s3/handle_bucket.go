// File: lib/storage/transport/s3/handle_bucket.go
package s3

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	"github.com/go-mizu/mizu"
)

// handleService handles service level operations mounted at basePath,
// for example:
//
//	GET basePath/ -> ListBuckets
func (s *Server) handleService(c *mizu.Ctx) error {
	req, err := s.authAndParse(c)
	if err != nil {
		return writeError(c, err)
	}

	switch req.Op {
	case OpListBuckets:
		return s.handleListBuckets(c, req)
	default:
		return writeError(c, ErrMethodNotAllowed)
	}
}

// handleBucket handles bucket level operations mounted at:
//
//	basePath/:bucket
//
// It covers:
//
//	PUT    basePath/:bucket           -> CreateBucket
//	DELETE basePath/:bucket           -> DeleteBucket
//	HEAD   basePath/:bucket           -> HeadBucket
//	GET    basePath/:bucket?location  -> GetBucketLocation
//	GET    basePath/:bucket[?... ]    -> ListObjectsV2
func (s *Server) handleBucket(c *mizu.Ctx) error {
	req, err := s.authAndParse(c)
	if err != nil {
		return writeError(c, err)
	}

	// Debug logging disabled for performance
	// Enable with build tag: -tags=s3debug
	debugLogRequest(c, req)

	switch req.Op {
	case OpCreateBucket:
		return s.handleCreateBucket(c, req)
	case OpDeleteBucket:
		return s.handleDeleteBucket(c, req)
	case OpHeadBucket:
		return s.handleHeadBucket(c, req)
	case OpListObjects:
		return s.handleListObjects(c, req)
	case OpGetBucketLocation:
		return s.handleGetBucketLocation(c, req)
	case OpListMultipartUploads:
		return s.handleListMultipartUploads(c, req)
	case OpDeleteObjects:
		return s.handleDeleteObjects(c, req)
	default:
		return writeError(c, ErrMethodNotAllowed)
	}
}

// handleListBuckets implements S3 ListBuckets:
//
//	GET basePath/
func (s *Server) handleListBuckets(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)

	iter, err := s.stor.Buckets(ctx, 0, 0, storage.Options{})
	if err != nil {
		return writeError(c, mapError(err))
	}
	defer func() { _ = iter.Close() }()

	var buckets []BucketSummary
	for {
		info, err := iter.Next()
		if err != nil {
			return writeError(c, mapError(err))
		}
		if info == nil {
			break
		}
		buckets = append(buckets, BucketSummary{
			Name:         info.Name,
			CreationDate: info.CreatedAt.UTC(),
		})
	}

	resp := ListBucketsResult{
		Xmlns: s3XMLNS,
		Owner: Owner{
			ID:          "local",
			DisplayName: "local",
		},
		Buckets: BucketsContainer{
			Buckets: buckets,
		},
	}
	return writeXML(c, http.StatusOK, resp)
}

// handleCreateBucket implements:
//
//	PUT basePath/:bucket
func (s *Server) handleCreateBucket(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)

	info, err := s.stor.CreateBucket(ctx, req.Bucket, storage.Options{})
	if err != nil {
		return writeError(c, mapError(err))
	}

	w := c.Writer()
	w.Header().Set("Location", buildBucketLocation(c, s.cfg, info.Name))
	w.WriteHeader(http.StatusOK)
	return nil
}

// handleDeleteBucket implements:
//
//	DELETE basePath/:bucket
func (s *Server) handleDeleteBucket(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}
	// List objects only (not directories) to check if bucket is empty.
	// We use files_only option to skip internal directories like _multipart.
	iter, err := b.List(ctx, "", 1, 0, storage.Options{
		"files_only": true,
	})
	if err != nil {
		return writeError(c, mapError(err))
	}
	defer func() { _ = iter.Close() }()

	// If we find any file, the bucket is not empty
	obj, err := iter.Next()
	if err != nil {
		return writeError(c, mapError(err))
	}
	if obj != nil {
		return writeError(c, ErrBucketNotEmpty)
	}

	if err := s.stor.DeleteBucket(ctx, req.Bucket, storage.Options{"force": true}); err != nil {
		return writeError(c, mapError(err))
	}
	c.Writer().WriteHeader(http.StatusNoContent)
	return nil
}

// handleHeadBucket implements:
//
//	HEAD basePath/:bucket
func (s *Server) handleHeadBucket(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}
	if _, err := b.Info(ctx); err != nil {
		return writeError(c, mapError(err))
	}
	c.Writer().WriteHeader(http.StatusOK)
	return nil
}

// handleGetBucketLocation implements:
//
//	GET basePath/:bucket?location
//
// Warp calls this during server preparation.
// AWS returns empty LocationConstraint for us-east-1, so we mimic that.
func (s *Server) handleGetBucketLocation(c *mizu.Ctx, req *Request) error {
	region := "us-east-1"
	if s.cfg != nil && s.cfg.Region != "" {
		region = s.cfg.Region
	}

	loc := region
	if region == "us-east-1" {
		loc = ""
	}

	c.Logger().Info("s3 handleGetBucketLocation",
		"bucket", req.Bucket,
		"region", region,
		"location_constraint", loc,
	)

	resp := GetBucketLocationResult{
		Xmlns:              s3XMLNS,
		LocationConstraint: loc,
	}
	return writeXML(c, http.StatusOK, resp)
}

// handleListObjects implements a ListObjectsV2 compatible response:
//
//	GET basePath/:bucket
//	    ?list-type=2
//	    [&prefix=...]
//	    [&delimiter=...]
//	    [&max-keys=N]
//	    [&start-after=K]
//	    [&continuation-token=T]
//	    [&encoding-type=url]
func (s *Server) handleListObjects(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	r := c.Request()
	q := r.URL.Query()

	prefix := q.Get("prefix")
	delimiter := q.Get("delimiter")
	encodingType := q.Get("encoding-type")

	// MaxKeys default is 1000, upper bound 1000.
	maxKeys := 1000
	if v := q.Get("max-keys"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			maxKeys = n
		}
	}

	continuationToken := q.Get("continuation-token")
	startAfter := q.Get("start-after")
	// Treat continuation token as a startAfter hint to approximate S3 pagination.
	if continuationToken != "" && startAfter == "" {
		startAfter = continuationToken
	}

	// For now we implement a flat listing and handle prefix and pagination
	// in this layer. Delimiter is accepted but not used to build CommonPrefixes.
	iter, err := b.List(ctx, prefix, 0, 0, storage.Options{
		"recursive":        true,
		"include_metadata": false,
		"delimiter":        delimiter,
	})
	if err != nil {
		return writeError(c, mapError(err))
	}
	defer func() { _ = iter.Close() }()

	type objectView struct {
		Key          string
		LastModified time.Time
		ETag         string
		Size         int64
	}

	var objs []objectView
	now := s.cfg.Clock().UTC()

	for {
		obj, err := iter.Next()
		if err != nil {
			return writeError(c, mapError(err))
		}
		if obj == nil {
			break
		}
		// Only return real objects, not directory markers.
		if obj.IsDir {
			continue
		}
		mod := obj.Updated
		if mod.IsZero() {
			mod = now
		}
		objs = append(objs, objectView{
			Key:          obj.Key,
			LastModified: mod.UTC(),
			ETag:         obj.ETag,
			Size:         obj.Size,
		})
	}

	// S3 ListObjectsV2 is lexicographically ordered by key.
	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })

	// Apply StartAfter and continuation semantics: only keys strictly greater
	// than startAfter are returned.
	if startAfter != "" {
		filtered := objs[:0]
		for _, o := range objs {
			if o.Key > startAfter {
				filtered = append(filtered, o)
			}
		}
		objs = filtered
	}

	isTruncated := false
	nextToken := ""

	if len(objs) > maxKeys {
		isTruncated = true
		nextToken = objs[maxKeys-1].Key
		objs = objs[:maxKeys]
	}

	entries := make([]ListEntry, 0, len(objs))
	for _, o := range objs {
		key := encodeKeyForResponse(o.Key, encodingType)
		entries = append(entries, ListEntry{
			Key:          key,
			LastModified: o.LastModified,
			ETag:         quoteRawETag(o.ETag),
			Size:         o.Size,
			StorageClass: "STANDARD",
		})
	}

	resp := ListBucketResultV2{
		Xmlns:    s3XMLNS,
		Name:     req.Bucket,
		Prefix:   encodeKeyForResponse(prefix, encodingType),
		MaxKeys:  maxKeys,
		KeyCount: len(entries),

		IsTruncated:           isTruncated,
		ContinuationToken:     continuationToken,
		NextContinuationToken: nextToken,

		Contents: entries,
	}
	return writeXML(c, http.StatusOK, resp)
}

// encodeKeyForResponse applies S3 ListObjectsV2 encoding type semantics.
// For encoding-type=url it percent-encodes special characters but preserves slashes.
// AWS S3 URL encoding keeps forward slashes (/) unescaped.
func encodeKeyForResponse(key, encodingType string) string {
	if encodingType == "url" {
		// PathEscape encodes spaces, special chars, BUT also encodes /
		// S3's URL encoding preserves slashes, so we use a custom approach
		var b strings.Builder
		for _, r := range key {
			switch {
			case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
				b.WriteRune(r)
			case r == '-' || r == '_' || r == '.' || r == '~' || r == '/':
				b.WriteRune(r)
			default:
				// Percent-encode the character
				b.WriteString(url.QueryEscape(string(r)))
			}
		}
		return b.String()
	}
	return key
}

// DeleteObjectsRequest is the XML body for DeleteObjects.
type DeleteObjectsRequest struct {
	XMLName xml.Name           `xml:"Delete"`
	Quiet   bool               `xml:"Quiet"`
	Objects []ObjectIdentifier `xml:"Object"`
}

// ObjectIdentifier identifies an object to delete.
type ObjectIdentifier struct {
	Key       string `xml:"Key"`
	VersionId string `xml:"VersionId,omitempty"`
}

// DeletedObject represents a successfully deleted object.
type DeletedObject struct {
	Key                   string `xml:"Key"`
	VersionId             string `xml:"VersionId,omitempty"`
	DeleteMarker          bool   `xml:"DeleteMarker,omitempty"`
	DeleteMarkerVersionId string `xml:"DeleteMarkerVersionId,omitempty"`
}

// DeleteError represents an error deleting an object.
type DeleteError struct {
	Key       string `xml:"Key"`
	VersionId string `xml:"VersionId,omitempty"`
	Code      string `xml:"Code"`
	Message   string `xml:"Message"`
}

// DeleteObjectsResult is the response for DeleteObjects.
type DeleteObjectsResult struct {
	XMLName xml.Name        `xml:"DeleteResult"`
	Xmlns   string          `xml:"xmlns,attr"`
	Deleted []DeletedObject `xml:"Deleted,omitempty"`
	Errors  []DeleteError   `xml:"Error,omitempty"`
}

// handleDeleteObjects implements S3 DeleteObjects (batch delete):
//
//	POST /{bucket}?delete
//
// Request body contains XML with list of keys to delete.
func (s *Server) handleDeleteObjects(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)
	r := c.Request()

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	// Read and parse the XML body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return writeError(c, ErrInternal.WithInternal(err))
	}

	var delReq DeleteObjectsRequest
	if err := xml.Unmarshal(body, &delReq); err != nil {
		return writeError(c, ErrInvalidRequest.WithInternal(err))
	}

	if len(delReq.Objects) == 0 {
		return writeError(c, ErrInvalidRequest.WithMessage("no objects specified for deletion"))
	}

	// Limit to 1000 objects per request per S3 spec
	if len(delReq.Objects) > 1000 {
		return writeError(c, ErrInvalidRequest.WithMessage("too many objects specified, max 1000"))
	}

	var deleted []DeletedObject
	var deleteErrors []DeleteError

	for _, obj := range delReq.Objects {
		// URL-decode the key if needed (AWS CLI sometimes sends URL-encoded keys)
		key := obj.Key
		if decodedKey, decodeErr := url.QueryUnescape(key); decodeErr == nil {
			key = decodedKey
		}

		delErr := b.Delete(ctx, key, storage.Options{})
		if delErr != nil && !errors.Is(delErr, storage.ErrNotExist) {
			// Real error - record it
			deleteErrors = append(deleteErrors, DeleteError{
				Key:     obj.Key,
				Code:    "InternalError",
				Message: delErr.Error(),
			})
		} else {
			// Success or key didn't exist (S3 treats both as success)
			deleted = append(deleted, DeletedObject{
				Key: obj.Key,
			})
		}
	}

	resp := DeleteObjectsResult{
		Xmlns:   s3XMLNS,
		Deleted: deleted,
		Errors:  deleteErrors,
	}

	// In quiet mode, only return errors
	if delReq.Quiet {
		resp.Deleted = nil
	}

	w := c.Writer()
	w.Header().Set("x-amz-request-id", generateRequestID())

	return writeXML(c, http.StatusOK, resp)
}
