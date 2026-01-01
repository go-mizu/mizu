package flight

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"time"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/flight"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-mizu/blueprints/drive/lib/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientConfig configures the Flight client.
type ClientConfig struct {
	// Target is the Flight server address.
	Target string

	// TLS configures TLS. Nil uses plaintext.
	TLS *tls.Config

	// Token for bearer authentication.
	Token string

	// Username for basic auth handshake.
	Username string

	// Password for basic auth handshake.
	Password string

	// MaxRecvMsgSize is the max message size in bytes. Default 64MB.
	MaxRecvMsgSize int

	// MaxSendMsgSize is the max message size in bytes. Default 64MB.
	MaxSendMsgSize int

	// ChunkSize for streaming writes. Default 1MB.
	ChunkSize int

	// DialOptions are additional gRPC dial options.
	DialOptions []grpc.DialOption

	// Allocator for Arrow memory. Default uses Go allocator.
	Allocator memory.Allocator
}

func (c *ClientConfig) clone() *ClientConfig {
	if c == nil {
		c = &ClientConfig{}
	}

	cp := *c
	if cp.MaxRecvMsgSize == 0 {
		cp.MaxRecvMsgSize = DefaultMaxMsgSize
	}
	if cp.MaxSendMsgSize == 0 {
		cp.MaxSendMsgSize = DefaultMaxMsgSize
	}
	if cp.ChunkSize == 0 {
		cp.ChunkSize = DefaultChunkSize
	}
	if cp.Allocator == nil {
		cp.Allocator = memory.DefaultAllocator
	}
	return &cp
}

// ClientOption configures a Client.
type ClientOption func(*ClientConfig)

// WithClientTLS configures TLS for the client connection.
func WithClientTLS(cfg *tls.Config) ClientOption {
	return func(c *ClientConfig) {
		c.TLS = cfg
	}
}

// WithToken sets the bearer authentication token.
func WithToken(token string) ClientOption {
	return func(c *ClientConfig) {
		c.Token = token
	}
}

// WithBasicAuth sets username and password for authentication.
func WithBasicAuth(username, password string) ClientOption {
	return func(c *ClientConfig) {
		c.Username = username
		c.Password = password
	}
}

// WithClientMaxRecvMsgSize sets the maximum receive message size.
func WithClientMaxRecvMsgSize(size int) ClientOption {
	return func(c *ClientConfig) {
		c.MaxRecvMsgSize = size
	}
}

// WithClientMaxSendMsgSize sets the maximum send message size.
func WithClientMaxSendMsgSize(size int) ClientOption {
	return func(c *ClientConfig) {
		c.MaxSendMsgSize = size
	}
}

// WithClientChunkSize sets the chunk size for streaming writes.
func WithClientChunkSize(size int) ClientOption {
	return func(c *ClientConfig) {
		c.ChunkSize = size
	}
}

// WithClientDialOptions adds additional gRPC dial options.
func WithClientDialOptions(opts ...grpc.DialOption) ClientOption {
	return func(c *ClientConfig) {
		c.DialOptions = append(c.DialOptions, opts...)
	}
}

// WithAllocator sets the Arrow memory allocator.
func WithAllocator(alloc memory.Allocator) ClientOption {
	return func(c *ClientConfig) {
		c.Allocator = alloc
	}
}

// Client is a Flight storage client implementing storage.Storage.
type Client struct {
	flightClient flight.Client
	cfg          *ClientConfig
}

// Open connects to a Flight storage server.
func Open(ctx context.Context, target string, opts ...ClientOption) (*Client, error) {
	cfg := &ClientConfig{Target: target}
	for _, opt := range opts {
		opt(cfg)
	}
	cfg = cfg.clone()

	dialOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMsgSize),
		),
	}

	if cfg.TLS != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(cfg.TLS)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if cfg.Token != "" {
		dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(&TokenCredentials{
			Token:    cfg.Token,
			Insecure: cfg.TLS == nil,
		}))
	}

	dialOpts = append(dialOpts, cfg.DialOptions...)

	var authHandler flight.ClientAuthHandler
	if cfg.Username != "" || cfg.Token != "" {
		authHandler = &ClientAuthHandler{
			Username: cfg.Username,
			Password: cfg.Password,
			Token:    cfg.Token,
		}
	}

	flightClient, err := flight.NewClientWithMiddleware(cfg.Target, authHandler, nil, dialOpts...)
	if err != nil {
		return nil, err
	}

	// Authenticate if needed
	if authHandler != nil && cfg.Token == "" && cfg.Username != "" {
		if err := flightClient.Authenticate(ctx); err != nil {
			flightClient.Close()
			return nil, mapStatusError(err)
		}
	}

	return &Client{
		flightClient: flightClient,
		cfg:          cfg,
	}, nil
}

// Bucket returns a handle. No network IO.
func (c *Client) Bucket(name string) storage.Bucket {
	return &clientBucket{
		client: c,
		name:   name,
	}
}

// Buckets enumerates buckets.
func (c *Client) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	criteria := &Criteria{
		Limit:   limit,
		Offset:  offset,
		Options: opts,
	}
	criteriaBytes, _ := EncodeCriteria(criteria)

	stream, err := c.flightClient.ListFlights(ctx, &flight.Criteria{Expression: criteriaBytes})
	if err != nil {
		return nil, mapStatusError(err)
	}

	return &bucketIterator{stream: stream}, nil
}

// CreateBucket creates a bucket.
func (c *Client) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	req := CreateBucketRequest{
		Name:    name,
		Options: opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := c.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: reqBytes,
	})
	if err != nil {
		return nil, mapStatusError(err)
	}

	result, err := stream.Recv()
	if err != nil {
		return nil, mapStatusError(err)
	}

	return DecodeBucketInfo(result.Body)
}

// DeleteBucket deletes a bucket.
func (c *Client) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	req := DeleteBucketRequest{
		Name:    name,
		Options: opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := c.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionDeleteBucket,
		Body: reqBytes,
	})
	if err != nil {
		return mapStatusError(err)
	}

	_, err = stream.Recv()
	return mapStatusError(err)
}

// Features reports capability flags.
func (c *Client) Features() storage.Features {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := c.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionGetFeatures,
		Body: []byte("{}"),
	})
	if err != nil {
		return nil
	}

	result, err := stream.Recv()
	if err != nil {
		return nil
	}

	features, _ := DecodeFeatures(result.Body)
	return features
}

// Close releases resources.
func (c *Client) Close() error {
	return c.flightClient.Close()
}

// clientBucket implements storage.Bucket for Flight client.
type clientBucket struct {
	client *Client
	name   string
}

func (b *clientBucket) Name() string {
	return b.name
}

func (b *clientBucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	desc := &flight.FlightDescriptor{
		Type: flight.DescriptorPATH,
		Path: []string{b.name},
	}

	info, err := b.client.flightClient.GetFlightInfo(ctx, desc)
	if err != nil {
		return nil, mapStatusError(err)
	}

	if len(info.Endpoint) > 0 && len(info.Endpoint[0].AppMetadata) > 0 {
		return DecodeBucketInfo(info.Endpoint[0].AppMetadata)
	}

	// Fallback: decode from ticket
	if len(info.Endpoint) > 0 && len(info.Endpoint[0].Ticket.Ticket) > 0 {
		return DecodeBucketInfo(info.Endpoint[0].Ticket.Ticket)
	}

	return &storage.BucketInfo{Name: b.name}, nil
}

func (b *clientBucket) Features() storage.Features {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := GetFeaturesRequest{Bucket: b.name}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionGetFeatures,
		Body: reqBytes,
	})
	if err != nil {
		return nil
	}

	result, err := stream.Recv()
	if err != nil {
		return nil
	}

	features, _ := DecodeFeatures(result.Body)
	return features
}

func (b *clientBucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	stream, err := b.client.flightClient.DoPut(ctx)
	if err != nil {
		return nil, mapStatusError(err)
	}

	// Create upload descriptor
	uploadDesc := &UploadDescriptor{
		Bucket:      b.name,
		Key:         key,
		Size:        size,
		ContentType: contentType,
		Options:     opts,
	}
	descBytes, _ := EncodeUploadDescriptor(uploadDesc)

	// Create writer with schema
	schema := ObjectDataSchema()
	writer := flight.NewRecordWriter(stream, ipc.WithSchema(schema))

	// Set descriptor
	writer.SetFlightDescriptor(&flight.FlightDescriptor{
		Type: flight.DescriptorCMD,
		Cmd:  descBytes,
	})

	// Read all data first to avoid streaming issues
	var dataBuf bytes.Buffer
	if _, err := io.Copy(&dataBuf, src); err != nil {
		writer.Close()
		return nil, err
	}

	// Build a single record with all data
	if dataBuf.Len() > 0 {
		builder := NewObjectDataBuilder(b.client.cfg.Allocator)
		rec := builder.Build(dataBuf.Bytes())
		if err := writer.Write(rec); err != nil {
			rec.Release()
			writer.Close()
			return nil, mapStatusError(err)
		}
		rec.Release()
	}

	// Close the writer to signal end of data
	if err := writer.Close(); err != nil {
		return nil, mapStatusError(err)
	}

	// Close the send side of the stream
	if err := stream.CloseSend(); err != nil {
		return nil, mapStatusError(err)
	}

	// Get result
	result, err := stream.Recv()
	if err != nil {
		return nil, mapStatusError(err)
	}

	return DecodeObjectInfo(result.AppMetadata)
}

func (b *clientBucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	ticket := &Ticket{
		Bucket:  b.name,
		Key:     key,
		Offset:  offset,
		Length:  length,
		Options: opts,
	}
	ticketBytes, _ := EncodeTicket(ticket)

	stream, err := b.client.flightClient.DoGet(ctx, &flight.Ticket{Ticket: ticketBytes})
	if err != nil {
		return nil, nil, mapStatusError(err)
	}

	// Read first message to get metadata
	data, err := stream.Recv()
	if err != nil {
		return nil, nil, mapStatusError(err)
	}

	var obj *storage.Object
	if len(data.AppMetadata) > 0 {
		obj, _ = DecodeObjectInfo(data.AppMetadata)
	}

	if obj == nil {
		obj = &storage.Object{
			Bucket: b.name,
			Key:    key,
		}
	}

	reader := &flightStreamReader{
		stream: stream,
		alloc:  b.client.cfg.Allocator,
	}

	return reader, obj, nil
}

func (b *clientBucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	req := StatRequest{
		Bucket:  b.name,
		Key:     key,
		Options: opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionStat,
		Body: reqBytes,
	})
	if err != nil {
		return nil, mapStatusError(err)
	}

	result, err := stream.Recv()
	if err != nil {
		return nil, mapStatusError(err)
	}

	return DecodeObjectInfo(result.Body)
}

func (b *clientBucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	req := DeleteObjectRequest{
		Bucket:  b.name,
		Key:     key,
		Options: opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionDeleteObject,
		Body: reqBytes,
	})
	if err != nil {
		return mapStatusError(err)
	}

	_, err = stream.Recv()
	return mapStatusError(err)
}

func (b *clientBucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	req := CopyObjectRequest{
		SrcBucket: srcBucket,
		SrcKey:    srcKey,
		DstBucket: b.name,
		DstKey:    dstKey,
		Options:   opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCopyObject,
		Body: reqBytes,
	})
	if err != nil {
		return nil, mapStatusError(err)
	}

	result, err := stream.Recv()
	if err != nil {
		return nil, mapStatusError(err)
	}

	return DecodeObjectInfo(result.Body)
}

func (b *clientBucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	req := MoveObjectRequest{
		SrcBucket: srcBucket,
		SrcKey:    srcKey,
		DstBucket: b.name,
		DstKey:    dstKey,
		Options:   opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionMoveObject,
		Body: reqBytes,
	})
	if err != nil {
		return nil, mapStatusError(err)
	}

	result, err := stream.Recv()
	if err != nil {
		return nil, mapStatusError(err)
	}

	return DecodeObjectInfo(result.Body)
}

func (b *clientBucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	criteria := &Criteria{
		Bucket:  b.name,
		Prefix:  prefix,
		Limit:   limit,
		Offset:  offset,
		Options: opts,
	}
	criteriaBytes, _ := EncodeCriteria(criteria)

	stream, err := b.client.flightClient.ListFlights(ctx, &flight.Criteria{Expression: criteriaBytes})
	if err != nil {
		return nil, mapStatusError(err)
	}

	return &objectIterator{stream: stream}, nil
}

func (b *clientBucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	req := SignedURLRequest{
		Bucket:  b.name,
		Key:     key,
		Method:  method,
		Expires: expires.String(),
		Options: opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionSignedURL,
		Body: reqBytes,
	})
	if err != nil {
		return "", mapStatusError(err)
	}

	result, err := stream.Recv()
	if err != nil {
		return "", mapStatusError(err)
	}

	var resp SignedURLResponse
	if err := json.Unmarshal(result.Body, &resp); err != nil {
		return "", err
	}

	return resp.URL, nil
}

// Multipart operations

// InitMultipart starts a multipart upload.
func (b *clientBucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	req := InitMultipartRequest{
		Bucket:      b.name,
		Key:         key,
		ContentType: contentType,
		Options:     opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionInitMultipart,
		Body: reqBytes,
	})
	if err != nil {
		return nil, mapStatusError(err)
	}

	result, err := stream.Recv()
	if err != nil {
		return nil, mapStatusError(err)
	}

	return DecodeMultipartUpload(result.Body)
}

// UploadPart uploads a single part via DoExchange.
func (b *clientBucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	stream, err := b.client.flightClient.DoExchange(ctx)
	if err != nil {
		return nil, mapStatusError(err)
	}

	// Send upload part message
	msg := UploadPartMessage{
		Bucket:     mu.Bucket,
		Key:        mu.Key,
		UploadID:   mu.UploadID,
		PartNumber: number,
		Size:       size,
		Options:    opts,
	}
	msgBytes, _ := EncodeExchangeMessage(ExchangeUploadPart, msg)

	// Read all data first
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, src); err != nil {
		return nil, err
	}

	// Send message with data
	if err := stream.Send(&flight.FlightData{
		AppMetadata: msgBytes,
		DataBody:    buf.Bytes(),
	}); err != nil {
		return nil, mapStatusError(err)
	}

	// Close send side
	if err := stream.CloseSend(); err != nil {
		return nil, mapStatusError(err)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		return nil, mapStatusError(err)
	}

	respMsg, err := DecodeExchangeMessage(resp.AppMetadata)
	if err != nil {
		return nil, err
	}

	if respMsg.Type == ExchangeError {
		var errStr string
		json.Unmarshal(respMsg.Payload, &errStr)
		return nil, storage.ErrUnsupported
	}

	var partJSON PartInfoJSON
	if err := json.Unmarshal(respMsg.Payload, &partJSON); err != nil {
		return nil, err
	}

	return PartInfoFromJSON(&partJSON), nil
}

// CopyPart copies a range from an existing object as a part.
func (b *clientBucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	return nil, storage.ErrUnsupported
}

// ListParts lists uploaded parts.
func (b *clientBucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	// Not directly exposed via Flight - would need custom action
	return nil, storage.ErrUnsupported
}

// CompleteMultipart completes the multipart upload.
func (b *clientBucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	partsJSON := make([]*PartInfoJSON, len(parts))
	for i, p := range parts {
		partsJSON[i] = PartInfoToJSON(p)
	}

	req := CompleteMultipartRequest{
		Bucket:   mu.Bucket,
		Key:      mu.Key,
		UploadID: mu.UploadID,
		Parts:    partsJSON,
		Options:  opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCompleteMultipart,
		Body: reqBytes,
	})
	if err != nil {
		return nil, mapStatusError(err)
	}

	result, err := stream.Recv()
	if err != nil {
		return nil, mapStatusError(err)
	}

	return DecodeObjectInfo(result.Body)
}

// AbortMultipart aborts the multipart upload.
func (b *clientBucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	req := AbortMultipartRequest{
		Bucket:   mu.Bucket,
		Key:      mu.Key,
		UploadID: mu.UploadID,
		Options:  opts,
	}
	reqBytes, _ := json.Marshal(req)

	stream, err := b.client.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionAbortMultipart,
		Body: reqBytes,
	})
	if err != nil {
		return mapStatusError(err)
	}

	_, err = stream.Recv()
	return mapStatusError(err)
}

// Helper types

// bucketIterator implements storage.BucketIter.
type bucketIterator struct {
	stream flight.FlightService_ListFlightsClient
}

func (i *bucketIterator) Next() (*storage.BucketInfo, error) {
	info, err := i.stream.Recv()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, mapStatusError(err)
	}

	// Decode from app_metadata
	if len(info.Endpoint) > 0 && len(info.Endpoint[0].AppMetadata) > 0 {
		return DecodeBucketInfo(info.Endpoint[0].AppMetadata)
	}

	// Fallback: use path
	if len(info.FlightDescriptor.Path) > 0 {
		return &storage.BucketInfo{Name: info.FlightDescriptor.Path[0]}, nil
	}

	return nil, nil
}

func (i *bucketIterator) Close() error {
	return nil
}

// objectIterator implements storage.ObjectIter.
type objectIterator struct {
	stream flight.FlightService_ListFlightsClient
}

func (i *objectIterator) Next() (*storage.Object, error) {
	info, err := i.stream.Recv()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, mapStatusError(err)
	}

	// Decode from app_metadata
	if len(info.Endpoint) > 0 && len(info.Endpoint[0].AppMetadata) > 0 {
		return DecodeObjectInfo(info.Endpoint[0].AppMetadata)
	}

	// Fallback: construct from descriptor
	if len(info.FlightDescriptor.Path) >= 2 {
		return &storage.Object{
			Bucket: info.FlightDescriptor.Path[0],
			Key:    joinPath(info.FlightDescriptor.Path[1:]),
			Size:   info.TotalBytes,
		}, nil
	}

	return nil, nil
}

func (i *objectIterator) Close() error {
	return nil
}

// flightStreamReader implements io.ReadCloser for DoGet streams.
type flightStreamReader struct {
	stream flight.FlightService_DoGetClient
	alloc  memory.Allocator
	reader *flight.Reader
	buf    bytes.Buffer
}

func (r *flightStreamReader) Read(p []byte) (int, error) {
	// First try to read from buffer
	if r.buf.Len() > 0 {
		return r.buf.Read(p)
	}

	// Initialize reader if needed
	if r.reader == nil {
		reader, err := flight.NewRecordReader(r.stream)
		if err != nil {
			return 0, mapStatusError(err)
		}
		r.reader = reader
	}

	// Read next record
	if !r.reader.Next() {
		if err := r.reader.Err(); err != nil {
			return 0, mapStatusError(err)
		}
		return 0, io.EOF
	}

	rec := r.reader.Record()
	if rec.NumCols() == 0 || rec.NumRows() == 0 {
		return r.Read(p) // Try next record
	}

	// Extract binary data from record
	col := rec.Column(0)
	if binArr, ok := col.(*array.Binary); ok {
		for i := 0; i < binArr.Len(); i++ {
			if !binArr.IsNull(i) {
				data := binArr.Value(i)
				if len(data) <= len(p)-r.buf.Len() {
					// Fits in buffer
					r.buf.Write(data)
				} else {
					// Write what fits, buffer the rest
					r.buf.Write(data)
				}
			}
		}
	}

	return r.buf.Read(p)
}

func (r *flightStreamReader) Close() error {
	if r.reader != nil {
		r.reader.Release()
	}
	return nil
}

// Verify interfaces
var (
	_ storage.Storage      = (*Client)(nil)
	_ storage.Bucket       = (*clientBucket)(nil)
	_ storage.HasMultipart = (*clientBucket)(nil)
)
