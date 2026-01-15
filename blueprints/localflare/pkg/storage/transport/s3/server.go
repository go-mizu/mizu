// File: lib/storage/transport/s3/server.go
package s3

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	"github.com/go-mizu/mizu"
)

const (
	s3XMLNS = "http://s3.amazonaws.com/doc/2006-03-01/"
)

// OriginalPathContextKey is a context key used to store the original request path
// before any path normalization (e.g., stripping trailing slashes).
// This is used by signature verification to validate against the original signed path.
type OriginalPathContextKey struct{}

// Config controls S3 transport behavior.
type Config struct {
	// Region used in Signature V4 scope and some responses.
	Region string

	// Endpoint for building Location headers. If empty, derived from request.
	Endpoint string

	// MaxObjectSize limits PUT object size. Zero or negative means unlimited here.
	MaxObjectSize int64

	// Clock used for timestamps and signature validation.
	// If nil, time.Now is used.
	Clock func() time.Time

	// Credentials resolves access keys for Signature V4.
	// If nil, auth is disabled and all requests are treated as anonymous.
	Credentials CredentialProvider

	// Signer validates AWS Signature V4.
	// If nil, signatures are not checked.
	Signer Signer

	// AllowedSkew is max clock drift when validating signatures.
	// Default 15 minutes if zero.
	AllowedSkew time.Duration

	// Service name used in signing scope. Typically "s3".
	Service string
}

func (c *Config) clone() *Config {
	if c == nil {
		return &Config{
			Region:      "us-east-1",
			Service:     "s3",
			AllowedSkew: 15 * time.Minute,
			Clock:       time.Now,
		}
	}
	cp := *c
	if cp.Region == "" {
		cp.Region = "us-east-1"
	}
	if cp.Service == "" {
		cp.Service = "s3"
	}
	if cp.AllowedSkew == 0 {
		cp.AllowedSkew = 15 * time.Minute
	}
	if cp.Clock == nil {
		cp.Clock = time.Now
	}
	return &cp
}

// Credential holds static credentials used by the signer.
type Credential struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// CredentialProvider looks up credentials by access key id.
type CredentialProvider interface {
	Lookup(accessKeyID string) (*Credential, error)
}

// Signer validates S3 requests.
type Signer interface {
	Verify(r *http.Request, cfg *Config) error
}

// SignerV4 validates AWS Signature Version 4 headers and presigned URLs.
type SignerV4 struct{}

type signatureInputs struct {
	credentialScope string
	accessKey       string
	region          string
	service         string
	signedHeaders   []string
	signature       string
	requestTime     time.Time
	expires         time.Duration
	payloadHash     string
	isPresign       bool
	query           url.Values
	securityToken   string
}

// Verify implements Signer for Signature V4.
func (s *SignerV4) Verify(r *http.Request, cfg *Config) error {
	if cfg == nil || cfg.Credentials == nil {
		return nil
	}

	inputs, err := parseSignatureInputs(r)
	if err != nil {
		return err
	}

	cred, err := cfg.Credentials.Lookup(inputs.accessKey)
	if err != nil {
		return err
	}
	if cred == nil {
		return errors.New("s3: unknown access key")
	}
	if cred.SessionToken != "" && inputs.securityToken != "" && cred.SessionToken != inputs.securityToken {
		return errors.New("s3: invalid session token")
	}

	if inputs.service != cfg.Service {
		return errors.New("s3: invalid service in credential scope")
	}
	if cfg.Region != "" && cfg.Region != "auto" && inputs.region != cfg.Region {
		return errors.New("s3: invalid region in credential scope")
	}

	if err := validateRequestTime(cfg, inputs); err != nil {
		return err
	}

	// Use original path if available (before normalization)
	// The original path may be stored in context by path-normalizing middleware
	path := r.URL.EscapedPath()
	if origPath, ok := r.Context().Value(OriginalPathContextKey{}).(string); ok && origPath != "" {
		path = origPath
	}
	candidates := canonicalPathCandidates(path)

	signingKey := deriveSigningKey(cred.SecretAccessKey, inputs.requestTime, inputs.region, inputs.service)

	for _, canonicalPath := range candidates {
		canonicalReq, err := buildCanonicalRequest(r, inputs, canonicalPath)
		if err != nil {
			return err
		}
		stringToSign := buildStringToSign(inputs, canonicalReq)
		expectedSig := hmacSHA256(signingKey, []byte(stringToSign))

		if hmac.Equal([]byte(inputs.signature), []byte(hex.EncodeToString(expectedSig))) {
			return nil
		}
	}

	return errors.New("s3: signature mismatch")
}

func parseSignatureInputs(r *http.Request) (*signatureInputs, error) {
	if algo := r.URL.Query().Get("X-Amz-Algorithm"); algo != "" {
		return parsePresignInputs(r)
	}
	return parseHeaderSignatureInputs(r)
}

func parseHeaderSignatureInputs(r *http.Request) (*signatureInputs, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, errors.New("s3: missing Authorization header")
	}
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") {
		return nil, errors.New("s3: unsupported auth scheme")
	}
	rest := strings.TrimPrefix(auth, "AWS4-HMAC-SHA256 ")
	parts := strings.Split(rest, ",")
	var credentialStr, signedHeadersStr, signatureHex string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "Credential":
			credentialStr = strings.TrimSpace(kv[1])
		case "SignedHeaders":
			signedHeadersStr = strings.TrimSpace(kv[1])
		case "Signature":
			signatureHex = strings.TrimSpace(kv[1])
		}
	}
	if credentialStr == "" || signedHeadersStr == "" || signatureHex == "" {
		return nil, errors.New("s3: malformed Authorization header")
	}

	credParts := strings.Split(credentialStr, "/")
	if len(credParts) != 5 {
		return nil, errors.New("s3: malformed Credential scope")
	}
	signedHeaders := splitSignedHeaders(signedHeadersStr)
	if len(signedHeaders) == 0 {
		return nil, errors.New("s3: no signed headers")
	}
	payloadHash := r.Header.Get("x-amz-content-sha256")
	reqTime, err := readRequestTime(r)
	if err != nil {
		return nil, err
	}
	shortDate := reqTime.UTC().Format("20060102")
	if credParts[1] != shortDate {
		return nil, errors.New("s3: credential date mismatch")
	}
	return &signatureInputs{
		credentialScope: strings.Join(credParts[1:], "/"),
		accessKey:       credParts[0],
		region:          credParts[2],
		service:         credParts[3],
		signedHeaders:   signedHeaders,
		signature:       signatureHex,
		requestTime:     reqTime,
		payloadHash:     payloadHash,
		securityToken:   r.Header.Get("X-Amz-Security-Token"),
		query:           r.URL.Query(),
	}, nil
}

func parsePresignInputs(r *http.Request) (*signatureInputs, error) {
	q := r.URL.Query()
	if algo := q.Get("X-Amz-Algorithm"); algo != "AWS4-HMAC-SHA256" {
		return nil, errors.New("s3: unsupported presign algorithm")
	}
	credentialStr := q.Get("X-Amz-Credential")
	signedHeadersStr := q.Get("X-Amz-SignedHeaders")
	signatureHex := q.Get("X-Amz-Signature")
	expiresStr := q.Get("X-Amz-Expires")
	if credentialStr == "" || signedHeadersStr == "" || signatureHex == "" || expiresStr == "" {
		return nil, errors.New("s3: malformed presigned URL")
	}
	credParts := strings.Split(credentialStr, "/")
	if len(credParts) != 5 {
		return nil, errors.New("s3: malformed Credential scope")
	}
	expiresSec, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil || expiresSec <= 0 {
		return nil, errors.New("s3: invalid presign expiration")
	}
	reqTime, err := time.Parse("20060102T150405Z", q.Get("X-Amz-Date"))
	if err != nil {
		return nil, errors.New("s3: invalid presign date")
	}
	shortDate := reqTime.UTC().Format("20060102")
	if credParts[1] != shortDate {
		return nil, errors.New("s3: credential date mismatch")
	}
	return &signatureInputs{
		credentialScope: strings.Join(credParts[1:], "/"),
		accessKey:       credParts[0],
		region:          credParts[2],
		service:         credParts[3],
		signedHeaders:   splitSignedHeaders(signedHeadersStr),
		signature:       signatureHex,
		requestTime:     reqTime,
		expires:         time.Duration(expiresSec) * time.Second,
		payloadHash:     q.Get("X-Amz-Content-Sha256"),
		isPresign:       true,
		query:           q,
		securityToken:   q.Get("X-Amz-Security-Token"),
	}, nil
}

func splitSignedHeaders(v string) []string {
	parts := strings.Split(v, ";")
	for i := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(parts[i]))
	}
	sort.Strings(parts)
	return parts
}

func validateRequestTime(cfg *Config, in *signatureInputs) error {
	now := cfg.Clock()
	if in.isPresign {
		if now.Before(in.requestTime) {
			return errors.New("s3: presigned request not yet valid")
		}
		if now.After(in.requestTime.Add(in.expires)) {
			return errors.New("s3: presigned request expired")
		}
		return nil
	}
	if d := now.Sub(in.requestTime); d > cfg.AllowedSkew || d < -cfg.AllowedSkew {
		return errors.New("s3: request time skew too large")
	}
	return nil
}

func buildCanonicalRequest(r *http.Request, in *signatureInputs, canonicalPath string) (string, error) {
	method := r.Method
	path := canonicalPath
	if path == "" {
		path = "/"
	}
	query := canonicalQueryString(r, in)
	canonicalHeaders, err := canonicalHeaders(r, in.signedHeaders)
	if err != nil {
		return "", err
	}
	signedHeaders := strings.Join(in.signedHeaders, ";")
	payloadHash := in.payloadHash
	if payloadHash == "" {
		payloadHash = "UNSIGNED-PAYLOAD"
	}

	var b strings.Builder
	b.WriteString(method)
	b.WriteString("\n")
	b.WriteString(path)
	b.WriteString("\n")
	b.WriteString(query)
	b.WriteString("\n")
	b.WriteString(canonicalHeaders)
	b.WriteString("\n")
	b.WriteString(signedHeaders)
	b.WriteString("\n")
	b.WriteString(payloadHash)
	return b.String(), nil
}

func canonicalQueryString(r *http.Request, in *signatureInputs) string {
	qs := in.query
	var pairs []string
	for key, values := range qs {
		if in.isPresign && strings.EqualFold(key, "X-Amz-Signature") {
			continue
		}
		for _, v := range values {
			pairs = append(pairs, canonicalEncode(key, true)+"="+canonicalEncode(v, true))
		}
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "&")
}

func canonicalHeaders(r *http.Request, signedHeaders []string) (string, error) {
	headers := make(map[string][]string)
	for k, v := range r.Header {
		headers[strings.ToLower(k)] = append([]string(nil), v...)
	}
	if _, ok := headers["host"]; !ok {
		headers["host"] = []string{r.Host}
	}
	var lines []string
	for _, h := range signedHeaders {
		values := headers[h]
		if len(values) == 0 {
			return "", errors.New("s3: signed header missing: " + h)
		}
		for i := range values {
			values[i] = strings.Join(strings.Fields(values[i]), " ")
		}
		lines = append(lines, h+":"+strings.Join(values, ","))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func canonicalEncode(v string, encodeSlash bool) string {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~':
			b.WriteByte(c)
		case c == '/':
			if encodeSlash {
				b.WriteString("%2F")
			} else {
				b.WriteByte(c)
			}
		case c == ' ':
			b.WriteString("%20")
		default:
			b.WriteString("%")
			b.WriteString(strings.ToUpper(hex.EncodeToString([]byte{c})))
		}
	}
	return b.String()
}

func buildStringToSign(in *signatureInputs, canonicalReq string) string {
	shortDate := in.requestTime.UTC().Format("20060102")
	dateTime := in.requestTime.UTC().Format("20060102T150405Z")
	hash := sha256.Sum256([]byte(canonicalReq))
	return strings.Join([]string{
		"AWS4-HMAC-SHA256",
		dateTime,
		shortDate + "/" + in.region + "/" + in.service + "/aws4_request",
		hex.EncodeToString(hash[:]),
	}, "\n")
}

// signingKeyCache caches derived signing keys to avoid repeated HMAC computations.
// Keys are scoped by access key + date + region + service, so they're valid for an entire day.
type signingKeyCache struct {
	mu    sync.RWMutex
	cache map[string][]byte
}

var globalSigningKeyCache = &signingKeyCache{
	cache: make(map[string][]byte),
}

func (c *signingKeyCache) getOrDerive(secret string, t time.Time, region, service string) []byte {
	shortDate := t.UTC().Format("20060102")
	cacheKey := secret[:min(8, len(secret))] + ":" + shortDate + ":" + region + ":" + service

	// Fast path: check cache with read lock
	c.mu.RLock()
	if cached, ok := c.cache[cacheKey]; ok {
		c.mu.RUnlock()
		return cached
	}
	c.mu.RUnlock()

	// Slow path: derive and cache
	derived := deriveSigningKeyDirect(secret, t, region, service)

	c.mu.Lock()
	// Cleanup old entries if cache grows too large (keep last 100)
	if len(c.cache) > 100 {
		c.cache = make(map[string][]byte)
	}
	c.cache[cacheKey] = derived
	c.mu.Unlock()

	return derived
}

func deriveSigningKey(secret string, t time.Time, region, service string) []byte {
	return globalSigningKeyCache.getOrDerive(secret, t, region, service)
}

func deriveSigningKeyDirect(secret string, t time.Time, region, service string) []byte {
	shortDate := t.UTC().Format("20060102")
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(shortDate))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

// readRequestTime parses x-amz-date or Date.
func readRequestTime(r *http.Request) (time.Time, error) {
	if v := r.Header.Get("x-amz-date"); v != "" {
		return time.Parse("20060102T150405Z", v)
	}
	if v := r.Header.Get("Date"); v != "" {
		return time.Parse(time.RFC1123, v)
	}
	return time.Time{}, errors.New("s3: missing x-amz-date or Date")
}

func hmacSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

// stripFirstPathSegment turns:
//
//	"/s3/bench/..." -> "/bench/..."
//	"/s3/"          -> "/"
//	"/bucket"       -> "/"
func stripFirstPathSegment(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	idx := strings.Index(p[1:], "/")
	if idx == -1 {
		return "/"
	}
	return p[idx+1:]
}

// canonicalPathCandidates returns a small set of plausible canonical paths
// for SigV4 verification, to account for differences in trailing slashes
// and path prefixes used by clients and servers.
func canonicalPathCandidates(p string) []string {
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	var out []string
	seen := map[string]struct{}{}

	add := func(s string) {
		if s == "" {
			s = "/"
		}
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
		if s != "/" && strings.HasPrefix(s, "//") {
			s = "/" + strings.TrimLeft(s, "/")
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	add(p)
	if p != "/" {
		add(strings.TrimRight(p, "/"))
	}

	stripped := stripFirstPathSegment(p)
	add(stripped)
	if stripped != "/" {
		add(strings.TrimRight(stripped, "/"))
	}

	return out
}

// Error is a typed S3 error.
type Error struct {
	Code       string
	Message    string
	HTTPStatus int
	internal   error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.internal != nil {
		return e.Code + ": " + e.internal.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *Error) WithInternal(err error) *Error {
	if err == nil {
		return e
	}
	cp := *e
	cp.internal = err
	return &cp
}

func (e *Error) WithMessage(msg string) *Error {
	cp := *e
	cp.Message = msg
	return &cp
}

var (
	ErrAccessDenied     = &Error{Code: "AccessDenied", Message: "Access Denied", HTTPStatus: http.StatusForbidden}
	ErrNoSuchBucket     = &Error{Code: "NoSuchBucket", Message: "The specified bucket does not exist", HTTPStatus: http.StatusNotFound}
	ErrNoSuchKey        = &Error{Code: "NoSuchKey", Message: "The specified key does not exist", HTTPStatus: http.StatusNotFound}
	ErrBucketNotEmpty   = &Error{Code: "BucketNotEmpty", Message: "The bucket you tried to delete is not empty", HTTPStatus: http.StatusConflict}
	ErrInvalidRequest   = &Error{Code: "InvalidRequest", Message: "The request is invalid", HTTPStatus: http.StatusBadRequest}
	ErrMethodNotAllowed = &Error{Code: "MethodNotAllowed", Message: "The specified method is not allowed", HTTPStatus: http.StatusMethodNotAllowed}
	ErrEntityTooLarge   = &Error{Code: "EntityTooLarge", Message: "Your proposed upload exceeds the maximum allowed size", HTTPStatus: http.StatusRequestEntityTooLarge}
	ErrInternal         = &Error{Code: "InternalError", Message: "We encountered an internal error. Please try again.", HTTPStatus: http.StatusInternalServerError}
	ErrNotImplemented   = &Error{Code: "NotImplemented", Message: "The requested functionality is not implemented", HTTPStatus: http.StatusNotImplemented}
	ErrInvalidPart      = &Error{Code: "InvalidPart", Message: "One or more of the specified parts could not be found", HTTPStatus: http.StatusBadRequest}
	ErrInvalidPartOrder = &Error{Code: "InvalidPartOrder", Message: "The list of parts was not in ascending order", HTTPStatus: http.StatusBadRequest}
	ErrNoSuchUpload     = &Error{Code: "NoSuchUpload", Message: "The specified multipart upload does not exist", HTTPStatus: http.StatusNotFound}
)

func mapError(err error) *Error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, storage.ErrNotExist):
		return ErrNoSuchKey.WithInternal(err)
	case errors.Is(err, storage.ErrPermission):
		return ErrAccessDenied.WithInternal(err)
	case errors.Is(err, storage.ErrUnsupported):
		return ErrNotImplemented.WithInternal(err)
	default:
		return ErrInternal.WithInternal(err)
	}
}

// ErrorResponse is S3 XML error.
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource,omitempty"`
	RequestID string   `xml:"RequestId,omitempty"`
	HostID    string   `xml:"HostId,omitempty"`
}

func writeError(c *mizu.Ctx, e *Error) error {
	if e == nil {
		e = ErrInternal
	}

	// Extra debug logging
	r := c.Request()
	if r != nil {
		c.Logger().Error("s3 writeError",
			"code", e.Code,
			"message", e.Message,
			"http_status", e.HTTPStatus,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
		)
	} else {
		c.Logger().Error("s3 writeError",
			"code", e.Code,
			"message", e.Message,
			"http_status", e.HTTPStatus,
		)
	}

	resp := ErrorResponse{
		Code:    e.Code,
		Message: e.Message,
	}
	return writeXML(c, e.HTTPStatus, resp)
}

// XML models for S3 responses.

type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type BucketSummary struct {
	Name         string    `xml:"Name"`
	CreationDate time.Time `xml:"CreationDate"`
}

type BucketsContainer struct {
	Buckets []BucketSummary `xml:"Bucket"`
}

type ListBucketsResult struct {
	XMLName xml.Name         `xml:"ListAllMyBucketsResult"`
	Xmlns   string           `xml:"xmlns,attr"`
	Owner   Owner            `xml:"Owner"`
	Buckets BucketsContainer `xml:"Buckets"`
}

// GetBucketLocationResult produces:
//
// <LocationConstraint xmlns="...">us-east-1</LocationConstraint>
type GetBucketLocationResult struct {
	XMLName            xml.Name `xml:"LocationConstraint"`
	Xmlns              string   `xml:"xmlns,attr"`
	LocationConstraint string   `xml:",chardata"`
}

type ListEntry struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
	ETag         string    `xml:"ETag"`
	Size         int64     `xml:"Size"`
	StorageClass string    `xml:"StorageClass"`
}

type ListBucketResultV2 struct {
	XMLName xml.Name `xml:"ListBucketResult"`
	Xmlns   string   `xml:"xmlns,attr"`

	Name     string `xml:"Name"`
	Prefix   string `xml:"Prefix"`
	MaxKeys  int    `xml:"MaxKeys"`
	KeyCount int    `xml:"KeyCount"`

	IsTruncated bool `xml:"IsTruncated"`

	ContinuationToken     string `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string `xml:"NextContinuationToken,omitempty"`

	Contents []ListEntry `xml:"Contents"`
}

func writeXML(c *mizu.Ctx, status int, v any) error {
	w := c.Writer()
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	return xml.NewEncoder(w).Encode(v)
}

// generateRequestID creates a unique request ID for x-amz-request-id headers.
func generateRequestID() string {
	return fmt.Sprintf("%016X", time.Now().UnixNano())
}

// requestScope identifies what the request targets.
type requestScope int

const (
	scopeUnknown requestScope = iota
	scopeService
	scopeBucket
	scopeObject
)

// Operation describes a logical S3 operation.
type Operation string

const (
	OpListBuckets Operation = "ListBuckets"

	OpCreateBucket      Operation = "CreateBucket"
	OpDeleteBucket      Operation = "DeleteBucket"
	OpHeadBucket        Operation = "HeadBucket"
	OpListObjects       Operation = "ListObjectsV2"
	OpGetBucketLocation Operation = "GetBucketLocation"
	OpGetObject         Operation = "GetObject"
	OpPutObject         Operation = "PutObject"
	OpDeleteObject      Operation = "DeleteObject"
	OpHeadObject        Operation = "HeadObject"
	OpCopyObject        Operation = "CopyObject"

	// Batch operations
	OpDeleteObjects Operation = "DeleteObjects"

	// Multipart upload operations
	OpListMultipartUploads    Operation = "ListMultipartUploads"
	OpCreateMultipartUpload   Operation = "CreateMultipartUpload"
	OpUploadPart              Operation = "UploadPart"
	OpListParts               Operation = "ListParts"
	OpCompleteMultipartUpload Operation = "CompleteMultipartUpload"
	OpAbortMultipartUpload    Operation = "AbortMultipartUpload"
)

// Request carries parsed routing information.
type Request struct {
	Scope  requestScope
	Op     Operation
	Bucket string
	Key    string

	Prefix       string
	Delimiter    string
	MaxKeys      int
	Continuation string
}

// parseRequest builds a Request from mizu.Ctx.
func parseRequest(c *mizu.Ctx) (*Request, *Error) {
	r := c.Request()
	q := r.URL.Query()

	bucket := strings.TrimSpace(c.Param("bucket"))
	key := strings.TrimPrefix(c.Param("key"), "/")

	req := &Request{
		Bucket: bucket,
		Key:    key,
	}

	// Service level: no bucket in route path.
	if bucket == "" && key == "" {
		req.Scope = scopeService
		if r.Method == http.MethodGet {
			req.Op = OpListBuckets
			return req, nil
		}
		return nil, ErrMethodNotAllowed
	}

	if key == "" {
		req.Scope = scopeBucket
		return detectBucketOp(r, q, req)
	}

	req.Scope = scopeObject
	return detectObjectOp(r, req)
}

func detectBucketOp(r *http.Request, q url.Values, req *Request) (*Request, *Error) {
	switch r.Method {
	case http.MethodPut:
		req.Op = OpCreateBucket
	case http.MethodDelete:
		req.Op = OpDeleteBucket
	case http.MethodHead:
		req.Op = OpHeadBucket
	case http.MethodPost:
		// POST /{bucket}?delete -> DeleteObjects (batch delete)
		if _, ok := q["delete"]; ok {
			req.Op = OpDeleteObjects
			return req, nil
		}
		// Unknown POST operation
		return nil, ErrNotImplemented.WithMessage("bucket POST operation not implemented")
	case http.MethodGet:
		// GET /{bucket}?location -> GetBucketLocation
		if _, ok := q["location"]; ok {
			req.Op = OpGetBucketLocation
			return req, nil
		}

		// GET /{bucket}?uploads -> ListMultipartUploads
		if _, ok := q["uploads"]; ok {
			req.Op = OpListMultipartUploads
			return req, nil
		}

		// Default: ListObjectsV2
		req.Op = OpListObjects
		req.Prefix = q.Get("prefix")
		req.Delimiter = q.Get("delimiter")
		req.Continuation = q.Get("continuation-token")
	default:
		// Unknown bucket method: report as NotImplemented instead of MethodNotAllowed
		return nil, ErrNotImplemented.WithMessage("bucket operation not implemented for method " + r.Method)
	}
	return req, nil
}

func detectObjectOp(r *http.Request, req *Request) (*Request, *Error) {
	q := r.URL.Query()

	// Check for multipart upload operations based on query parameters
	if _, hasUploads := q["uploads"]; hasUploads {
		// POST /:bucket/:key?uploads - CreateMultipartUpload
		if r.Method == http.MethodPost {
			req.Op = OpCreateMultipartUpload
			return req, nil
		}
		return nil, ErrMethodNotAllowed
	}

	uploadID := q.Get("uploadId")
	if uploadID != "" {
		// Operations with uploadId query parameter
		partNumber := q.Get("partNumber")

		if partNumber != "" {
			// PUT /:bucket/:key?partNumber=N&uploadId=ID - UploadPart
			if r.Method == http.MethodPut {
				req.Op = OpUploadPart
				return req, nil
			}
			return nil, ErrMethodNotAllowed
		}

		// Operations with just uploadId (no partNumber)
		switch r.Method {
		case http.MethodGet:
			// GET /:bucket/:key?uploadId=ID - ListParts
			req.Op = OpListParts
			return req, nil
		case http.MethodPost:
			// POST /:bucket/:key?uploadId=ID - CompleteMultipartUpload
			req.Op = OpCompleteMultipartUpload
			return req, nil
		case http.MethodDelete:
			// DELETE /:bucket/:key?uploadId=ID - AbortMultipartUpload
			req.Op = OpAbortMultipartUpload
			return req, nil
		default:
			return nil, ErrMethodNotAllowed
		}
	}

	// Standard object operations (no multipart query parameters)
	switch r.Method {
	case http.MethodGet:
		req.Op = OpGetObject
	case http.MethodPut:
		if r.Header.Get("x-amz-copy-source") != "" {
			req.Op = OpCopyObject
		} else {
			req.Op = OpPutObject
		}
	case http.MethodDelete:
		req.Op = OpDeleteObject
	case http.MethodHead:
		req.Op = OpHeadObject
	default:
		// Unknown object method: report as NotImplemented instead of MethodNotAllowed
		return nil, ErrNotImplemented.WithMessage("object operation not implemented for method " + r.Method)
	}
	return req, nil
}

// Server exposes a minimal S3 compatible API on top of storage.Storage.
type Server struct {
	stor storage.Storage
	cfg  *Config
}

// New creates a Server.
func New(stor storage.Storage, cfg *Config) *Server {
	if stor == nil {
		panic("s3: storage is nil")
	}
	return &Server{
		stor: stor,
		cfg:  cfg.clone(),
	}
}

// Register mounts the S3 API under basePath using mizu.
func Register(app *mizu.App, basePath string, stor storage.Storage, cfg *Config) *Server {
	s := New(stor, cfg)

	// Normalize basePath:
	//   ""   -> "/"
	//   "/"  -> "/"
	//   "s3" -> "/s3"
	//   "/s3" or "/s3/" -> "/s3"
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		basePath = "/"
	}
	isRoot := basePath == "/"
	if !isRoot {
		basePath = "/" + strings.Trim(basePath, "/")
	}

	var servicePath, bucketPath, objectPath string
	if isRoot {
		// Root mount: service-level route at "/"
		servicePath = "/"
		bucketPath = "/{bucket}"
		objectPath = "/{bucket}/{key...}"
	} else {
		// Non root mount:
		// Service ListBuckets lives at "/s3/" (with trailing slash).
		servicePath = basePath + "/"
		bucketPath = basePath + "/{bucket}"
		objectPath = basePath + "/{bucket}/{key...}"
	}

	// Register service-level routes for ListBuckets
	// For root mount, we need special handling to avoid conflicts with bucket routes
	if isRoot {
		// Register explicit GET / for ListBuckets at root
		app.Get(servicePath, s.handleService)
	} else if servicePath != "" {
		registerAllMethods(app, servicePath, s.handleService)
	}

	registerAllMethods(app, bucketPath, s.handleBucket)
	registerAllMethods(app, objectPath, s.handleObject)

	return s
}

func registerAllMethods(app *mizu.App, path string, h func(*mizu.Ctx) error) {
	app.Get(path, h)
	app.Put(path, h)
	app.Post(path, h)
	app.Delete(path, h)
	// HEAD is handled via headMethodWrapper middleware
}

// authAndParse is used from handleService, handleBucket, handleObject.
func (s *Server) authAndParse(c *mizu.Ctx) (*Request, *Error) {
	r := c.Request()

	// Signature V4 validation if configured.
	if s.cfg.Signer != nil && s.cfg.Credentials != nil {
		if err := s.cfg.Signer.Verify(r, s.cfg); err != nil {
			c.Logger().Error("s3 sigv4 verify failed",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"error", err,
			)
			return nil, ErrAccessDenied.WithInternal(err)
		}
	}

	req, err := parseRequest(c)
	if err != nil {
		// parseRequest only returns *Error, but log it anyway
		c.Logger().Error("s3 parseRequest failed",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"error", err,
		)
		return nil, err
	}

	// Debug style logging, but using Info since slog is LevelInfo.
	c.Logger().Info("s3 request parsed",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"scope", req.Scope,
		"op", req.Op,
		"bucket", req.Bucket,
		"key", req.Key,
		"prefix", req.Prefix,
		"delimiter", req.Delimiter,
		"continuation", req.Continuation,
	)

	return req, nil
}

// Helper to get context from mizu.Ctx.
func contextFromCtx(c *mizu.Ctx) context.Context {
	if r := c.Request(); r != nil {
		return r.Context()
	}
	return context.Background()
}

func buildBucketLocation(c *mizu.Ctx, cfg *Config, bucket string) string {
	if cfg != nil && cfg.Endpoint != "" {
		return strings.TrimRight(cfg.Endpoint, "/") + "/" + bucket
	}
	r := c.Request()
	scheme := "http"
	if r != nil && r.TLS != nil {
		scheme = "https"
	}
	host := ""
	if r != nil {
		host = r.Host
	}
	return scheme + "://" + host + "/" + bucket
}

// quoteRawETag wraps an etag in quotes, stripping existing ones.
func quoteRawETag(etag string) string {
	if etag == "" {
		return ""
	}
	etag = strings.Trim(etag, `"`)
	return `"` + etag + `"`
}
