// File: lib/storage/transport/grpc/server.go

// Package grpc provides a gRPC transport layer for storage.Storage backends.
//
// This package implements a gRPC server that exposes any storage.Storage
// implementation over gRPC, enabling high-performance remote storage access.
//
// Path Mapping:
//
//	/                    → list buckets
//	/<bucket>/           → bucket root
//	/<bucket>/<key>      → object
//
// Example:
//
//	store, _ := storage.Open(ctx, "local:///data")
//
//	cfg := &grpc.Config{
//	    Auth: &grpc.AuthConfig{
//	        TokenValidator: validateToken,
//	    },
//	}
//
//	server := grpc.New(store, cfg)
//	grpcServer := grpc.NewServer()
//	server.Register(grpcServer)
//	lis, _ := net.Listen("tcp", ":9000")
//	grpcServer.Serve(lis)
package grpc

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	pb "github.com/go-mizu/blueprints/localflare/pkg/storage/transport/grpc/storagepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	// DefaultChunkSize is the default chunk size for streaming (64KB).
	DefaultChunkSize = 64 * 1024

	// DefaultMaxMsgSize is the default maximum message size (16MB).
	DefaultMaxMsgSize = 16 * 1024 * 1024
)

// Config configures the gRPC server.
type Config struct {
	// MaxRecvMsgSize is the max message size in bytes. Default 16MB.
	MaxRecvMsgSize int

	// MaxSendMsgSize is the max message size in bytes. Default 16MB.
	MaxSendMsgSize int

	// ChunkSize for streaming reads. Default 64KB.
	ChunkSize int

	// Auth configures authentication. Nil disables auth.
	Auth *AuthConfig

	// Logger for server events. If nil, slog.Default is used.
	Logger *slog.Logger
}

func (c *Config) clone() *Config {
	if c == nil {
		return &Config{
			MaxRecvMsgSize: DefaultMaxMsgSize,
			MaxSendMsgSize: DefaultMaxMsgSize,
			ChunkSize:      DefaultChunkSize,
			Logger:         slog.Default(),
		}
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
	if cp.Logger == nil {
		cp.Logger = slog.Default()
	}
	return &cp
}

// Server is a gRPC server backed by storage.Storage.
type Server struct {
	pb.UnimplementedStorageServiceServer
	store storage.Storage
	cfg   *Config
}

// New creates a new gRPC storage server.
func New(store storage.Storage, cfg *Config) *Server {
	if store == nil {
		panic("grpc: storage is nil")
	}
	return &Server{
		store: store,
		cfg:   cfg.clone(),
	}
}

// Register registers the storage service with a gRPC server.
func (s *Server) Register(grpcServer *grpc.Server) {
	pb.RegisterStorageServiceServer(grpcServer, s)
}

// ServerOptions returns recommended gRPC server options.
func (s *Server) ServerOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(s.cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.cfg.MaxSendMsgSize),
	}

	if s.cfg.Auth != nil {
		opts = append(opts,
			grpc.UnaryInterceptor(UnaryAuthInterceptor(s.cfg.Auth)),
			grpc.StreamInterceptor(StreamAuthInterceptor(s.cfg.Auth)),
		)
	}

	return opts
}

// Bucket Operations

func (s *Server) CreateBucket(ctx context.Context, req *pb.CreateBucketRequest) (*pb.BucketInfo, error) {
	opts := optionsFromProto(req.Options)
	info, err := s.store.CreateBucket(ctx, req.Name, opts)
	if err != nil {
		return nil, mapStorageError(err)
	}
	return bucketInfoToProto(info), nil
}

func (s *Server) DeleteBucket(ctx context.Context, req *pb.DeleteBucketRequest) (*emptypb.Empty, error) {
	opts := optionsFromProto(req.Options)
	if err := s.store.DeleteBucket(ctx, req.Name, opts); err != nil {
		return nil, mapStorageError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetBucket(ctx context.Context, req *pb.GetBucketRequest) (*pb.BucketInfo, error) {
	bucket := s.store.Bucket(req.Name)
	info, err := bucket.Info(ctx)
	if err != nil {
		return nil, mapStorageError(err)
	}
	return bucketInfoToProto(info), nil
}

func (s *Server) ListBuckets(req *pb.ListBucketsRequest, stream pb.StorageService_ListBucketsServer) error {
	ctx := stream.Context()
	opts := optionsFromProto(req.Options)

	iter, err := s.store.Buckets(ctx, int(req.Limit), int(req.Offset), opts)
	if err != nil {
		return mapStorageError(err)
	}
	defer iter.Close()

	for {
		info, err := iter.Next()
		if err != nil {
			return mapStorageError(err)
		}
		if info == nil {
			break
		}
		if err := stream.Send(bucketInfoToProto(info)); err != nil {
			return err
		}
	}
	return nil
}

// Object Operations

func (s *Server) StatObject(ctx context.Context, req *pb.StatObjectRequest) (*pb.ObjectInfo, error) {
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	obj, err := bucket.Stat(ctx, req.Key, opts)
	if err != nil {
		return nil, mapStorageError(err)
	}
	return objectInfoToProto(obj), nil
}

func (s *Server) DeleteObject(ctx context.Context, req *pb.DeleteObjectRequest) (*emptypb.Empty, error) {
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	if err := bucket.Delete(ctx, req.Key, opts); err != nil {
		return nil, mapStorageError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) CopyObject(ctx context.Context, req *pb.CopyObjectRequest) (*pb.ObjectInfo, error) {
	bucket := s.store.Bucket(req.DstBucket)
	opts := optionsFromProto(req.Options)

	obj, err := bucket.Copy(ctx, req.DstKey, req.SrcBucket, req.SrcKey, opts)
	if err != nil {
		return nil, mapStorageError(err)
	}
	return objectInfoToProto(obj), nil
}

func (s *Server) MoveObject(ctx context.Context, req *pb.MoveObjectRequest) (*pb.ObjectInfo, error) {
	bucket := s.store.Bucket(req.DstBucket)
	opts := optionsFromProto(req.Options)

	obj, err := bucket.Move(ctx, req.DstKey, req.SrcBucket, req.SrcKey, opts)
	if err != nil {
		return nil, mapStorageError(err)
	}
	return objectInfoToProto(obj), nil
}

func (s *Server) ListObjects(req *pb.ListObjectsRequest, stream pb.StorageService_ListObjectsServer) error {
	ctx := stream.Context()
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	iter, err := bucket.List(ctx, req.Prefix, int(req.Limit), int(req.Offset), opts)
	if err != nil {
		return mapStorageError(err)
	}
	defer iter.Close()

	for {
		obj, err := iter.Next()
		if err != nil {
			return mapStorageError(err)
		}
		if obj == nil {
			break
		}
		if err := stream.Send(objectInfoToProto(obj)); err != nil {
			return err
		}
	}
	return nil
}

// Streaming Data Operations

func (s *Server) ReadObject(req *pb.ReadObjectRequest, stream pb.StorageService_ReadObjectServer) error {
	ctx := stream.Context()
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	rc, obj, err := bucket.Open(ctx, req.Key, req.Offset, req.Length, opts)
	if err != nil {
		return mapStorageError(err)
	}
	defer rc.Close()

	// First message: metadata
	if err := stream.Send(&pb.ReadObjectResponse{
		Payload: &pb.ReadObjectResponse_Metadata{
			Metadata: objectInfoToProto(obj),
		},
	}); err != nil {
		return err
	}

	// Subsequent messages: data chunks
	buf := make([]byte, s.cfg.ChunkSize)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			if err := stream.Send(&pb.ReadObjectResponse{
				Payload: &pb.ReadObjectResponse_Data{
					Data: buf[:n],
				},
			}); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return mapStorageError(err)
		}
	}
	return nil
}

func (s *Server) WriteObject(stream pb.StorageService_WriteObjectServer) error {
	ctx := stream.Context()

	// First message must be metadata
	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	meta := msg.GetMetadata()
	if meta == nil {
		return status.Error(codes.InvalidArgument, "first message must be metadata")
	}

	bucket := s.store.Bucket(meta.Bucket)
	opts := optionsFromProto(meta.Options)

	// Collect all data into a buffer
	var buf bytes.Buffer
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		data := msg.GetData()
		if data != nil {
			buf.Write(data)
		}
	}

	// Write to storage
	size := int64(buf.Len())
	if meta.Size >= 0 && meta.Size != size {
		return status.Errorf(codes.InvalidArgument, "size mismatch: expected %d, got %d", meta.Size, size)
	}

	obj, err := bucket.Write(ctx, meta.Key, &buf, size, meta.ContentType, opts)
	if err != nil {
		return mapStorageError(err)
	}

	return stream.SendAndClose(objectInfoToProto(obj))
}

// Multipart Upload Operations

func (s *Server) InitMultipart(ctx context.Context, req *pb.InitMultipartRequest) (*pb.MultipartUpload, error) {
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	upload, err := mp.InitMultipart(ctx, req.Key, req.ContentType, opts)
	if err != nil {
		return nil, mapStorageError(err)
	}
	return multipartUploadToProto(upload), nil
}

func (s *Server) UploadPart(stream pb.StorageService_UploadPartServer) error {
	ctx := stream.Context()

	// First message must be metadata
	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	meta := msg.GetMetadata()
	if meta == nil {
		return status.Error(codes.InvalidArgument, "first message must be metadata")
	}

	bucket := s.store.Bucket(meta.Bucket)
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   meta.Bucket,
		Key:      meta.Key,
		UploadID: meta.UploadId,
	}
	opts := optionsFromProto(meta.Options)

	// Collect all data
	var buf bytes.Buffer
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		data := msg.GetData()
		if data != nil {
			buf.Write(data)
		}
	}

	size := int64(buf.Len())
	if meta.Size >= 0 && meta.Size != size {
		return status.Errorf(codes.InvalidArgument, "size mismatch: expected %d, got %d", meta.Size, size)
	}

	partInfo, err := mp.UploadPart(ctx, mu, int(meta.PartNumber), &buf, size, opts)
	if err != nil {
		return mapStorageError(err)
	}

	return stream.SendAndClose(partInfoToProto(partInfo))
}

func (s *Server) ListParts(req *pb.ListPartsRequest, stream pb.StorageService_ListPartsServer) error {
	ctx := stream.Context()
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: req.UploadId,
	}

	parts, err := mp.ListParts(ctx, mu, int(req.Limit), int(req.Offset), opts)
	if err != nil {
		return mapStorageError(err)
	}

	for _, part := range parts {
		if err := stream.Send(partInfoToProto(part)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) CompleteMultipart(ctx context.Context, req *pb.CompleteMultipartRequest) (*pb.ObjectInfo, error) {
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: req.UploadId,
	}

	parts := make([]*storage.PartInfo, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = partInfoFromProto(p)
	}

	obj, err := mp.CompleteMultipart(ctx, mu, parts, opts)
	if err != nil {
		return nil, mapStorageError(err)
	}
	return objectInfoToProto(obj), nil
}

func (s *Server) AbortMultipart(ctx context.Context, req *pb.AbortMultipartRequest) (*emptypb.Empty, error) {
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: req.UploadId,
	}

	if err := mp.AbortMultipart(ctx, mu, opts); err != nil {
		return nil, mapStorageError(err)
	}
	return &emptypb.Empty{}, nil
}

// Signed URL

func (s *Server) SignedURL(ctx context.Context, req *pb.SignedURLRequest) (*pb.SignedURLResponse, error) {
	bucket := s.store.Bucket(req.Bucket)
	opts := optionsFromProto(req.Options)

	expires := req.Expires.AsDuration()
	url, err := bucket.SignedURL(ctx, req.Key, req.Method, expires, opts)
	if err != nil {
		return nil, mapStorageError(err)
	}
	return &pb.SignedURLResponse{Url: url}, nil
}

// Features

func (s *Server) GetFeatures(ctx context.Context, req *pb.GetFeaturesRequest) (*pb.Features, error) {
	var features storage.Features
	if req.Bucket != "" {
		bucket := s.store.Bucket(req.Bucket)
		features = bucket.Features()
	} else {
		features = s.store.Features()
	}
	return featuresToProto(features), nil
}
