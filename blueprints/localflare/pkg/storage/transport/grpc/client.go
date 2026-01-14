// File: lib/storage/transport/grpc/client.go

package grpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	pb "github.com/go-mizu/blueprints/localflare/pkg/storage/transport/grpc/storagepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ClientConfig configures the gRPC client.
type ClientConfig struct {
	// Target is the gRPC server address (e.g., "localhost:9000").
	Target string

	// TLS configures TLS. Nil uses insecure connection.
	TLS *tls.Config

	// Token for authentication.
	Token string

	// MaxRecvMsgSize is the max message size in bytes. Default 16MB.
	MaxRecvMsgSize int

	// MaxSendMsgSize is the max message size in bytes. Default 16MB.
	MaxSendMsgSize int

	// ChunkSize for streaming writes. Default 64KB.
	ChunkSize int

	// DialOptions are additional gRPC dial options.
	DialOptions []grpc.DialOption
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
	return &cp
}

// ClientOption configures a Client.
type ClientOption func(*ClientConfig)

// WithTLS configures TLS for the client connection.
func WithTLS(cfg *tls.Config) ClientOption {
	return func(c *ClientConfig) {
		c.TLS = cfg
	}
}

// WithToken sets the authentication token.
func WithToken(token string) ClientOption {
	return func(c *ClientConfig) {
		c.Token = token
	}
}

// WithMaxRecvMsgSize sets the maximum receive message size.
func WithMaxRecvMsgSize(size int) ClientOption {
	return func(c *ClientConfig) {
		c.MaxRecvMsgSize = size
	}
}

// WithMaxSendMsgSize sets the maximum send message size.
func WithMaxSendMsgSize(size int) ClientOption {
	return func(c *ClientConfig) {
		c.MaxSendMsgSize = size
	}
}

// WithChunkSize sets the chunk size for streaming writes.
func WithChunkSize(size int) ClientOption {
	return func(c *ClientConfig) {
		c.ChunkSize = size
	}
}

// WithDialOptions adds additional gRPC dial options.
func WithDialOptions(opts ...grpc.DialOption) ClientOption {
	return func(c *ClientConfig) {
		c.DialOptions = append(c.DialOptions, opts...)
	}
}

// Client is a gRPC storage client implementing storage.Storage.
type Client struct {
	conn   *grpc.ClientConn
	client pb.StorageServiceClient
	cfg    *ClientConfig
}

// Open connects to a gRPC storage server.
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

	conn, err := grpc.NewClient(cfg.Target, dialOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewStorageServiceClient(conn),
		cfg:    cfg,
	}, nil
}

// Bucket returns a handle. No network IO.
func (c *Client) Bucket(name string) storage.Bucket {
	return &clientBucket{
		client: c.client,
		name:   name,
		cfg:    c.cfg,
	}
}

// Buckets enumerates buckets.
func (c *Client) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	stream, err := c.client.ListBuckets(ctx, &pb.ListBucketsRequest{
		Limit:   int32(limit),
		Offset:  int32(offset),
		Options: optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &bucketIterator{stream: stream}, nil
}

// CreateBucket creates a bucket.
func (c *Client) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	resp, err := c.client.CreateBucket(ctx, &pb.CreateBucketRequest{
		Name:    name,
		Options: optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return bucketInfoFromProto(resp), nil
}

// DeleteBucket deletes a bucket.
func (c *Client) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_, err := c.client.DeleteBucket(ctx, &pb.DeleteBucketRequest{
		Name:    name,
		Options: optionsToProto(opts),
	})
	return mapGRPCError(err)
}

// Features reports capability flags.
func (c *Client) Features() storage.Features {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.GetFeatures(ctx, &pb.GetFeaturesRequest{})
	if err != nil {
		return nil
	}
	return featuresFromProto(resp)
}

// Close releases resources.
func (c *Client) Close() error {
	return c.conn.Close()
}

// clientBucket implements storage.Bucket for gRPC client.
type clientBucket struct {
	client pb.StorageServiceClient
	name   string
	cfg    *ClientConfig
}

func (b *clientBucket) Name() string {
	return b.name
}

func (b *clientBucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	resp, err := b.client.GetBucket(ctx, &pb.GetBucketRequest{Name: b.name})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return bucketInfoFromProto(resp), nil
}

func (b *clientBucket) Features() storage.Features {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := b.client.GetFeatures(ctx, &pb.GetFeaturesRequest{Bucket: b.name})
	if err != nil {
		return nil
	}
	return featuresFromProto(resp)
}

func (b *clientBucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	stream, err := b.client.WriteObject(ctx)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	// Send metadata first
	if err := stream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Metadata{
			Metadata: &pb.WriteObjectMetadata{
				Bucket:      b.name,
				Key:         key,
				Size:        size,
				ContentType: contentType,
				Options:     optionsToProto(opts),
			},
		},
	}); err != nil {
		return nil, mapGRPCError(err)
	}

	// Stream data chunks
	buf := make([]byte, b.cfg.ChunkSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if err := stream.Send(&pb.WriteObjectRequest{
				Payload: &pb.WriteObjectRequest_Data{
					Data: buf[:n],
				},
			}); err != nil {
				return nil, mapGRPCError(err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return objectInfoFromProto(resp), nil
}

func (b *clientBucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	stream, err := b.client.ReadObject(ctx, &pb.ReadObjectRequest{
		Bucket:  b.name,
		Key:     key,
		Offset:  offset,
		Length:  length,
		Options: optionsToProto(opts),
	})
	if err != nil {
		return nil, nil, mapGRPCError(err)
	}

	// First message should be metadata
	msg, err := stream.Recv()
	if err != nil {
		return nil, nil, mapGRPCError(err)
	}

	meta := msg.GetMetadata()
	if meta == nil {
		return nil, nil, storage.ErrNotExist
	}

	reader := &streamReader{stream: stream}
	return reader, objectInfoFromProto(meta), nil
}

func (b *clientBucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	resp, err := b.client.StatObject(ctx, &pb.StatObjectRequest{
		Bucket:  b.name,
		Key:     key,
		Options: optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return objectInfoFromProto(resp), nil
}

func (b *clientBucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_, err := b.client.DeleteObject(ctx, &pb.DeleteObjectRequest{
		Bucket:  b.name,
		Key:     key,
		Options: optionsToProto(opts),
	})
	return mapGRPCError(err)
}

func (b *clientBucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	resp, err := b.client.CopyObject(ctx, &pb.CopyObjectRequest{
		SrcBucket: srcBucket,
		SrcKey:    srcKey,
		DstBucket: b.name,
		DstKey:    dstKey,
		Options:   optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return objectInfoFromProto(resp), nil
}

func (b *clientBucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	resp, err := b.client.MoveObject(ctx, &pb.MoveObjectRequest{
		SrcBucket: srcBucket,
		SrcKey:    srcKey,
		DstBucket: b.name,
		DstKey:    dstKey,
		Options:   optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return objectInfoFromProto(resp), nil
}

func (b *clientBucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	stream, err := b.client.ListObjects(ctx, &pb.ListObjectsRequest{
		Bucket:  b.name,
		Prefix:  prefix,
		Limit:   int32(limit),
		Offset:  int32(offset),
		Options: optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &objectIterator{stream: stream}, nil
}

func (b *clientBucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	resp, err := b.client.SignedURL(ctx, &pb.SignedURLRequest{
		Bucket:  b.name,
		Key:     key,
		Method:  method,
		Expires: durationToProto(expires),
		Options: optionsToProto(opts),
	})
	if err != nil {
		return "", mapGRPCError(err)
	}
	return resp.Url, nil
}

// clientBucketMultipart implements storage.HasMultipart for gRPC client.
type clientBucketMultipart struct {
	*clientBucket
}

// InitMultipart starts a multipart upload.
func (b *clientBucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	resp, err := b.client.InitMultipart(ctx, &pb.InitMultipartRequest{
		Bucket:      b.name,
		Key:         key,
		ContentType: contentType,
		Options:     optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return multipartUploadFromProto(resp), nil
}

// UploadPart uploads a single part.
func (b *clientBucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	stream, err := b.client.UploadPart(ctx)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	// Send metadata first
	if err := stream.Send(&pb.UploadPartRequest{
		Payload: &pb.UploadPartRequest_Metadata{
			Metadata: &pb.UploadPartMetadata{
				Bucket:     mu.Bucket,
				Key:        mu.Key,
				UploadId:   mu.UploadID,
				PartNumber: int32(number),
				Size:       size,
				Options:    optionsToProto(opts),
			},
		},
	}); err != nil {
		return nil, mapGRPCError(err)
	}

	// Stream data chunks
	buf := make([]byte, b.cfg.ChunkSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if err := stream.Send(&pb.UploadPartRequest{
				Payload: &pb.UploadPartRequest_Data{
					Data: buf[:n],
				},
			}); err != nil {
				return nil, mapGRPCError(err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return partInfoFromProto(resp), nil
}

// CopyPart copies a range from an existing object as a part.
func (b *clientBucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	// CopyPart is not directly exposed via streaming, would need a separate RPC
	// For now, return unsupported
	return nil, storage.ErrUnsupported
}

// ListParts lists uploaded parts.
func (b *clientBucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	stream, err := b.client.ListParts(ctx, &pb.ListPartsRequest{
		Bucket:   mu.Bucket,
		Key:      mu.Key,
		UploadId: mu.UploadID,
		Limit:    int32(limit),
		Offset:   int32(offset),
		Options:  optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	var parts []*storage.PartInfo
	for {
		part, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, mapGRPCError(err)
		}
		parts = append(parts, partInfoFromProto(part))
	}
	return parts, nil
}

// CompleteMultipart completes the multipart upload.
func (b *clientBucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	pbParts := make([]*pb.PartInfo, len(parts))
	for i, p := range parts {
		pbParts[i] = partInfoToProto(p)
	}

	resp, err := b.client.CompleteMultipart(ctx, &pb.CompleteMultipartRequest{
		Bucket:   mu.Bucket,
		Key:      mu.Key,
		UploadId: mu.UploadID,
		Parts:    pbParts,
		Options:  optionsToProto(opts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return objectInfoFromProto(resp), nil
}

// AbortMultipart aborts the multipart upload.
func (b *clientBucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	_, err := b.client.AbortMultipart(ctx, &pb.AbortMultipartRequest{
		Bucket:   mu.Bucket,
		Key:      mu.Key,
		UploadId: mu.UploadID,
		Options:  optionsToProto(opts),
	})
	return mapGRPCError(err)
}

// Helper types

// bucketIterator implements storage.BucketIter.
type bucketIterator struct {
	stream pb.StorageService_ListBucketsClient
}

func (i *bucketIterator) Next() (*storage.BucketInfo, error) {
	info, err := i.stream.Recv()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return bucketInfoFromProto(info), nil
}

func (i *bucketIterator) Close() error {
	return nil
}

// objectIterator implements storage.ObjectIter.
type objectIterator struct {
	stream pb.StorageService_ListObjectsClient
}

func (i *objectIterator) Next() (*storage.Object, error) {
	obj, err := i.stream.Recv()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return objectInfoFromProto(obj), nil
}

func (i *objectIterator) Close() error {
	return nil
}

// streamReader implements io.ReadCloser for streaming reads.
type streamReader struct {
	stream pb.StorageService_ReadObjectClient
	buf    bytes.Buffer
}

func (r *streamReader) Read(p []byte) (int, error) {
	// First try to read from buffer
	if r.buf.Len() > 0 {
		return r.buf.Read(p)
	}

	// Get next chunk from stream
	msg, err := r.stream.Recv()
	if err == io.EOF {
		return 0, io.EOF
	}
	if err != nil {
		return 0, mapGRPCError(err)
	}

	data := msg.GetData()
	if data == nil {
		return 0, io.EOF
	}

	// If the chunk fits in p, return directly
	if len(data) <= len(p) {
		copy(p, data)
		return len(data), nil
	}

	// Buffer the excess
	copy(p, data[:len(p)])
	r.buf.Write(data[len(p):])
	return len(p), nil
}

func (r *streamReader) Close() error {
	return nil
}

// durationToProto converts time.Duration to protobuf Duration.
func durationToProto(d time.Duration) *durationpb.Duration {
	return durationpb.New(d)
}
