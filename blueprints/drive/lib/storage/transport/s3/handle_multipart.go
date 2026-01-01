// File: lib/storage/transport/s3/handle_multipart.go
package s3

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-mizu/blueprints/drive/lib/storage"
	"github.com/go-mizu/mizu"
)

// S3 XML models for multipart responses.

// Initiator represents the initiator of a multipart upload.
type Initiator struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName,omitempty"`
}

type InitiateMultipartUploadResult struct {
	XMLName xml.Name `xml:"InitiateMultipartUploadResult"`
	Xmlns   string   `xml:"xmlns,attr"`

	Bucket   string `xml:"Bucket"`
	Key      string `xml:"Key"`
	UploadID string `xml:"UploadId"`
}

type CompletedPart struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

type CompleteMultipartUploadRequest struct {
	XMLName xml.Name        `xml:"CompleteMultipartUpload"`
	Parts   []CompletedPart `xml:"Part"`
}

type CompleteMultipartUploadResult struct {
	XMLName xml.Name `xml:"CompleteMultipartUploadResult"`
	Xmlns   string   `xml:"xmlns,attr"`

	Location string `xml:"Location"`
	Bucket   string `xml:"Bucket"`
	Key      string `xml:"Key"`
	ETag     string `xml:"ETag"`
}

// Part in ListPartsResult.
type Part struct {
	PartNumber   int    `xml:"PartNumber"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	// StorageClass is optional but common. Keep it fixed to STANDARD.
	StorageClass string `xml:"StorageClass,omitempty"`
}

type ListPartsResult struct {
	XMLName xml.Name `xml:"ListPartsResult"`
	Xmlns   string   `xml:"xmlns,attr"`

	Bucket   string `xml:"Bucket"`
	Key      string `xml:"Key"`
	UploadID string `xml:"UploadId"`

	Initiator            Initiator `xml:"Initiator,omitempty"`
	Owner                Owner     `xml:"Owner,omitempty"`
	StorageClass         string    `xml:"StorageClass,omitempty"`
	PartNumberMarker     int       `xml:"PartNumberMarker"`
	NextPartNumberMarker int       `xml:"NextPartNumberMarker,omitempty"`
	MaxParts             int       `xml:"MaxParts"`
	IsTruncated          bool      `xml:"IsTruncated"`
	Parts                []Part    `xml:"Part"`
}

// Upload represents one multipart upload in a listing.
type Upload struct {
	Key          string    `xml:"Key"`
	UploadID     string    `xml:"UploadId"`
	Initiated    string    `xml:"Initiated"` // ISO8601 timestamp
	StorageClass string    `xml:"StorageClass,omitempty"`
	Owner        Owner     `xml:"Owner,omitempty"`
	Initiator    Initiator `xml:"Initiator,omitempty"`
}

// ListMultipartUploadsResult is bucket level listing of uploads.
type ListMultipartUploadsResult struct {
	XMLName xml.Name `xml:"ListMultipartUploadsResult"`
	Xmlns   string   `xml:"xmlns,attr"`

	Bucket             string   `xml:"Bucket"`
	KeyMarker          string   `xml:"KeyMarker,omitempty"`
	UploadIDMarker     string   `xml:"UploadIdMarker,omitempty"`
	NextKeyMarker      string   `xml:"NextKeyMarker,omitempty"`
	NextUploadIDMarker string   `xml:"NextUploadIdMarker,omitempty"`
	Prefix             string   `xml:"Prefix,omitempty"`
	Delimiter          string   `xml:"Delimiter,omitempty"`
	MaxUploads         int      `xml:"MaxUploads"`
	IsTruncated        bool     `xml:"IsTruncated"`
	Uploads            []Upload `xml:"Upload"`
}

// Public dispatcher helpers (to be wired in server.go / parseRequest):
//
// In bucket-level dispatch:
//
//   case OpListMultipartUploads:
//       return s.handleListMultipartUploads(c, req)
//
// In object-level dispatch:
//
//   case OpCreateMultipartUpload:
//       return s.handleCreateMultipartUpload(c, req)
//   case OpUploadPart:
//       return s.handleUploadPart(c, req)
//   case OpListParts:
//       return s.handleListParts(c, req)
//   case OpCompleteMultipartUpload:
//       return s.handleCompleteMultipartUpload(c, req)
//   case OpAbortMultipartUpload:
//       return s.handleAbortMultipartUpload(c, req)

// handleListMultipartUploads implements:
//
//	GET /:bucket?uploads
//
// Query parameters:
//   - prefix: Limit results to keys that begin with the prefix
//   - delimiter: Character used to group keys
//   - key-marker: Start listing after this key
//   - upload-id-marker: Start listing after this upload ID (when key-marker is specified)
//   - max-uploads: Maximum number of uploads to return (default 1000, max 1000)
//
// Note: This returns a properly structured response but may be empty if the backend
// doesn't support listing multipart uploads. The storage.HasMultipart interface
// doesn't currently include a ListMultipartUploads method.
func (s *Server) handleListMultipartUploads(c *mizu.Ctx, req *Request) error {
	r := c.Request()
	q := r.URL.Query()

	// Parse query parameters
	prefix := q.Get("prefix")
	delimiter := q.Get("delimiter")
	keyMarker := q.Get("key-marker")
	uploadIDMarker := q.Get("upload-id-marker")

	maxUploads := 1000
	if maxStr := q.Get("max-uploads"); maxStr != "" {
		if val, err := strconv.Atoi(maxStr); err == nil && val > 0 {
			maxUploads = val
			if maxUploads > 1000 {
				maxUploads = 1000
			}
		}
	}

	// Build response with proper structure
	// Currently returns empty list as backend may not support listing
	resp := ListMultipartUploadsResult{
		Xmlns:          s3XMLNS,
		Bucket:         req.Bucket,
		Prefix:         prefix,
		Delimiter:      delimiter,
		KeyMarker:      keyMarker,
		UploadIDMarker: uploadIDMarker,
		MaxUploads:     maxUploads,
		IsTruncated:    false,
		Uploads:        []Upload{},
	}

	w := c.Writer()
	w.Header().Set("x-amz-request-id", generateRequestID())
	return writeXML(c, http.StatusOK, resp)
}

// handleCreateMultipartUpload implements:
//
//	POST /:bucket/*key?uploads
//
// Supported headers:
//   - Content-Type: MIME type for the object
//   - x-amz-meta-*: Custom metadata
//   - x-amz-storage-class: Storage class (STANDARD, REDUCED_REDUNDANCY, etc.)
//   - x-amz-server-side-encryption: Encryption algorithm (AES256, aws:kms)
//   - x-amz-server-side-encryption-aws-kms-key-id: KMS key ID
//   - x-amz-tagging: URL-encoded tags
//   - Content-Encoding, Content-Disposition, Content-Language
//   - Cache-Control, Expires
//   - x-amz-acl: Canned ACL
func (s *Server) handleCreateMultipartUpload(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)
	r := c.Request()

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	hm, ok := b.(storage.HasMultipart)
	if !ok {
		return writeError(c, ErrNotImplemented.WithMessage("multipart uploads not supported"))
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "binary/octet-stream"
	}

	opts := storage.Options{}

	// Extract x-amz-meta-* headers
	meta := map[string]string{}
	for name, values := range r.Header {
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "x-amz-meta-") {
			continue
		}
		key := strings.TrimPrefix(lower, "x-amz-meta-")
		if key == "" {
			continue
		}
		meta[key] = values[0]
	}
	if len(meta) > 0 {
		opts["metadata"] = meta
	}

	// Extract standard content headers
	if val := r.Header.Get("Content-Encoding"); val != "" {
		opts["content_encoding"] = val
	}
	if val := r.Header.Get("Content-Disposition"); val != "" {
		opts["content_disposition"] = val
	}
	if val := r.Header.Get("Content-Language"); val != "" {
		opts["content_language"] = val
	}
	if val := r.Header.Get("Cache-Control"); val != "" {
		opts["cache_control"] = val
	}
	if val := r.Header.Get("Expires"); val != "" {
		opts["expires"] = val
	}

	// Extract AWS-specific headers
	if val := r.Header.Get("x-amz-storage-class"); val != "" {
		opts["storage_class"] = val
	}
	if val := r.Header.Get("x-amz-server-side-encryption"); val != "" {
		opts["sse_algorithm"] = val
	}
	if val := r.Header.Get("x-amz-server-side-encryption-aws-kms-key-id"); val != "" {
		opts["sse_kms_key_id"] = val
	}
	if val := r.Header.Get("x-amz-server-side-encryption-context"); val != "" {
		opts["sse_context"] = val
	}
	if val := r.Header.Get("x-amz-acl"); val != "" {
		opts["acl"] = val
	}
	if val := r.Header.Get("x-amz-tagging"); val != "" {
		// Parse URL-encoded tags (key1=value1&key2=value2)
		tags := make(map[string]string)
		pairs := strings.Split(val, "&")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				key, _ := url.QueryUnescape(kv[0])
				value, _ := url.QueryUnescape(kv[1])
				tags[key] = value
			}
		}
		if len(tags) > 0 {
			opts["tags"] = tags
		}
	}

	mu, err := hm.InitMultipart(ctx, req.Key, contentType, opts)
	if err != nil {
		return writeError(c, mapError(err))
	}

	w := c.Writer()
	w.Header().Set("x-amz-request-id", generateRequestID())

	// Echo back encryption headers if they were set
	if sse := r.Header.Get("x-amz-server-side-encryption"); sse != "" {
		w.Header().Set("x-amz-server-side-encryption", sse)
		if kmsKey := r.Header.Get("x-amz-server-side-encryption-aws-kms-key-id"); kmsKey != "" {
			w.Header().Set("x-amz-server-side-encryption-aws-kms-key-id", kmsKey)
		}
	}

	resp := InitiateMultipartUploadResult{
		Xmlns:    s3XMLNS,
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: mu.UploadID,
	}
	return writeXML(c, http.StatusOK, resp)
}

// handleUploadPart implements:
//
//	PUT /:bucket/*key?partNumber=N&uploadId=UPLOADID
//
// Part numbers must be in the range 1-10000 (S3 limit).
func (s *Server) handleUploadPart(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)
	r := c.Request()

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	hm, ok := b.(storage.HasMultipart)
	if !ok {
		return writeError(c, ErrNotImplemented.WithMessage("multipart uploads not supported"))
	}

	q := r.URL.Query()
	partNumStr := q.Get("partNumber")
	uploadID := q.Get("uploadId")
	if partNumStr == "" || uploadID == "" {
		return writeError(c, ErrInvalidRequest.WithMessage("missing partNumber or uploadId"))
	}

	partNum, err := strconv.Atoi(partNumStr)
	if err != nil || partNum < 1 || partNum > 10000 {
		return writeError(c, ErrInvalidRequest.WithMessage("partNumber must be between 1 and 10000"))
	}

	if s.cfg.MaxObjectSize > 0 && r.ContentLength > s.cfg.MaxObjectSize {
		return writeError(c, ErrEntityTooLarge)
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: uploadID,
	}

	// Extract checksum headers if present
	opts := storage.Options{}
	if contentMD5 := r.Header.Get("Content-MD5"); contentMD5 != "" {
		if opts["checksum"] == nil {
			opts["checksum"] = storage.Hashes{}
		}
		opts["checksum"].(storage.Hashes)["md5-base64"] = contentMD5
	}

	partInfo, err := hm.UploadPart(ctx, mu, partNum, r.Body, r.ContentLength, opts)
	if err != nil {
		// Map to NoSuchUpload if the upload doesn't exist
		if err == storage.ErrNotExist {
			return writeError(c, ErrNoSuchUpload.WithInternal(err))
		}
		return writeError(c, mapError(err))
	}

	w := c.Writer()
	w.Header().Set("x-amz-request-id", generateRequestID())
	if partInfo.ETag != "" {
		w.Header().Set("ETag", quoteRawETag(partInfo.ETag))
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

// handleListParts implements:
//
//	GET /:bucket/*key?uploadId=UPLOADID
//
// Query parameters:
//   - uploadId (required): The upload ID for the multipart upload
//   - max-parts: Maximum number of parts to return (default 1000, max 1000)
//   - part-number-marker: Start listing after this part number
func (s *Server) handleListParts(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)
	r := c.Request()
	q := r.URL.Query()

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	hm, ok := b.(storage.HasMultipart)
	if !ok {
		return writeError(c, ErrNotImplemented.WithMessage("multipart uploads not supported"))
	}

	uploadID := q.Get("uploadId")
	if uploadID == "" {
		return writeError(c, ErrInvalidRequest)
	}

	// Parse pagination parameters
	maxParts := 1000
	if maxStr := q.Get("max-parts"); maxStr != "" {
		if val, err := strconv.Atoi(maxStr); err == nil && val > 0 {
			maxParts = val
			if maxParts > 1000 {
				maxParts = 1000
			}
		}
	}

	partNumberMarker := 0
	if markerStr := q.Get("part-number-marker"); markerStr != "" {
		if val, err := strconv.Atoi(markerStr); err == nil && val > 0 {
			partNumberMarker = val
		}
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: uploadID,
	}

	// Request one more than maxParts to determine if truncated
	requestLimit := maxParts + 1
	partInfos, err := hm.ListParts(ctx, mu, requestLimit, partNumberMarker, storage.Options{})
	if err != nil {
		return writeError(c, mapError(err))
	}

	// Check if results are truncated
	isTruncated := len(partInfos) > maxParts
	if isTruncated {
		partInfos = partInfos[:maxParts]
	}

	// Convert to S3 Part format
	parts := make([]Part, 0, len(partInfos))
	for _, pi := range partInfos {
		var lastMod string
		if pi.LastModified != nil {
			// S3 uses ISO8601 style timestamps, for example:
			// 2009-10-12T17:50:30.000Z
			lastMod = pi.LastModified.UTC().Format("2006-01-02T15:04:05.000Z")
		}

		parts = append(parts, Part{
			PartNumber:   pi.Number,
			LastModified: lastMod,
			ETag:         quoteRawETag(pi.ETag),
			Size:         pi.Size,
			StorageClass: "STANDARD",
		})
	}

	// Build response
	resp := ListPartsResult{
		Xmlns:            s3XMLNS,
		Bucket:           req.Bucket,
		Key:              req.Key,
		UploadID:         uploadID,
		StorageClass:     "STANDARD",
		PartNumberMarker: partNumberMarker,
		MaxParts:         maxParts,
		IsTruncated:      isTruncated,
		Parts:            parts,
	}

	// Set NextPartNumberMarker if truncated
	if isTruncated && len(parts) > 0 {
		resp.NextPartNumberMarker = parts[len(parts)-1].PartNumber
	}

	w := c.Writer()
	w.Header().Set("x-amz-request-id", generateRequestID())
	return writeXML(c, http.StatusOK, resp)
}

// handleCompleteMultipartUpload implements:
//
//	POST /:bucket/*key?uploadId=UPLOADID
//
// It reads the CompleteMultipartUpload XML from the body and
// completes the multipart upload. Parts must be provided in ascending
// order by part number.
func (s *Server) handleCompleteMultipartUpload(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)
	r := c.Request()

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	hm, ok := b.(storage.HasMultipart)
	if !ok {
		return writeError(c, ErrNotImplemented.WithMessage("multipart uploads not supported"))
	}

	uploadID := r.URL.Query().Get("uploadId")
	if uploadID == "" {
		return writeError(c, ErrInvalidRequest.WithMessage("missing uploadId"))
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return writeError(c, ErrInternal.WithInternal(err))
	}
	var compReq CompleteMultipartUploadRequest
	if err := xml.Unmarshal(body, &compReq); err != nil {
		return writeError(c, ErrInvalidRequest.WithInternal(err))
	}
	if len(compReq.Parts) == 0 {
		return writeError(c, ErrInvalidRequest.WithMessage("parts list must not be empty"))
	}

	// Validate parts are in ascending order and no duplicates
	seen := make(map[int]bool)
	for i, cp := range compReq.Parts {
		if seen[cp.PartNumber] {
			return writeError(c, ErrInvalidRequest.WithMessage("duplicate part number"))
		}
		seen[cp.PartNumber] = true

		if i > 0 && compReq.Parts[i].PartNumber <= compReq.Parts[i-1].PartNumber {
			return writeError(c, ErrInvalidPartOrder)
		}
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: uploadID,
	}

	// Convert CompletedParts to PartInfos
	parts := make([]*storage.PartInfo, len(compReq.Parts))
	for i, cp := range compReq.Parts {
		parts[i] = &storage.PartInfo{
			Number: cp.PartNumber,
			ETag:   strings.Trim(cp.ETag, "\""),
		}
	}

	obj, err := hm.CompleteMultipart(ctx, mu, parts, storage.Options{})
	if err != nil {
		// Map to NoSuchUpload if the upload doesn't exist
		if err == storage.ErrNotExist {
			return writeError(c, ErrNoSuchUpload.WithInternal(err))
		}
		return writeError(c, mapError(err))
	}

	// URL encode the key for the location header
	location := buildBucketLocation(c, s.cfg, req.Bucket) + "/" + url.PathEscape(req.Key)

	etag := obj.ETag
	if etag == "" && obj.Hash != nil {
		if v := obj.Hash["etag"]; v != "" {
			etag = v
		} else if v := obj.Hash["md5"]; v != "" {
			etag = v
		}
	}

	w := c.Writer()
	w.Header().Set("x-amz-request-id", generateRequestID())
	if etag != "" {
		w.Header().Set("ETag", quoteRawETag(etag))
	}

	resp := CompleteMultipartUploadResult{
		Xmlns:    s3XMLNS,
		Location: location,
		Bucket:   req.Bucket,
		Key:      req.Key,
		ETag:     quoteRawETag(etag),
	}
	return writeXML(c, http.StatusOK, resp)
}

// handleAbortMultipartUpload implements:
//
//	DELETE /:bucket/*key?uploadId=UPLOADID
//
// Aborting a multipart upload that doesn't exist returns success per S3 semantics.
func (s *Server) handleAbortMultipartUpload(c *mizu.Ctx, req *Request) error {
	ctx := contextFromCtx(c)
	r := c.Request()

	b := s.stor.Bucket(req.Bucket)
	if b == nil {
		return writeError(c, ErrNoSuchBucket)
	}

	hm, ok := b.(storage.HasMultipart)
	if !ok {
		return writeError(c, ErrNotImplemented.WithMessage("multipart uploads not supported"))
	}

	uploadID := r.URL.Query().Get("uploadId")
	if uploadID == "" {
		return writeError(c, ErrInvalidRequest.WithMessage("missing uploadId"))
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: uploadID,
	}

	w := c.Writer()
	w.Header().Set("x-amz-request-id", generateRequestID())

	if err := hm.AbortMultipart(ctx, mu, storage.Options{}); err != nil {
		// Treat ErrNotExist as success per S3 semantics
		if err == storage.ErrNotExist {
			w.WriteHeader(http.StatusNoContent)
			return nil
		}
		return writeError(c, mapError(err))
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
