package flight

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/flight"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	// DefaultChunkSize is the default chunk size for streaming (1MB).
	DefaultChunkSize = 1024 * 1024

	// DefaultMaxMsgSize is the default maximum message size (64MB).
	DefaultMaxMsgSize = 64 * 1024 * 1024

	// DefaultMaxConcurrentStreams is the default max concurrent streams.
	DefaultMaxConcurrentStreams = 100
)

// Config configures the Flight server.
type Config struct {
	// Addr is the listen address (e.g., ":8080").
	Addr string

	// TLS configures TLS. Nil uses plaintext.
	TLS *tls.Config

	// Auth configures authentication. Nil disables auth.
	Auth *AuthConfig

	// MaxRecvMsgSize is the max message size in bytes. Default 64MB.
	MaxRecvMsgSize int

	// MaxSendMsgSize is the max message size in bytes. Default 64MB.
	MaxSendMsgSize int

	// ChunkSize for streaming data. Default 1MB.
	ChunkSize int

	// MaxConcurrentStreams limits parallel streams per connection.
	MaxConcurrentStreams uint32

	// Logger for server events. If nil, slog.Default is used.
	Logger *slog.Logger

	// Allocator for Arrow memory. Default uses Go allocator.
	Allocator memory.Allocator
}

func (c *Config) clone() *Config {
	if c == nil {
		return &Config{
			Addr:                 ":8080",
			MaxRecvMsgSize:       DefaultMaxMsgSize,
			MaxSendMsgSize:       DefaultMaxMsgSize,
			ChunkSize:            DefaultChunkSize,
			MaxConcurrentStreams: DefaultMaxConcurrentStreams,
			Logger:               slog.Default(),
			Allocator:            memory.DefaultAllocator,
		}
	}

	cp := *c
	if cp.Addr == "" {
		cp.Addr = ":8080"
	}
	if cp.MaxRecvMsgSize == 0 {
		cp.MaxRecvMsgSize = DefaultMaxMsgSize
	}
	if cp.MaxSendMsgSize == 0 {
		cp.MaxSendMsgSize = DefaultMaxMsgSize
	}
	if cp.ChunkSize == 0 {
		cp.ChunkSize = DefaultChunkSize
	}
	if cp.MaxConcurrentStreams == 0 {
		cp.MaxConcurrentStreams = DefaultMaxConcurrentStreams
	}
	if cp.Logger == nil {
		cp.Logger = slog.Default()
	}
	if cp.Allocator == nil {
		cp.Allocator = memory.DefaultAllocator
	}
	return &cp
}

// Server is a Flight server backed by storage.Storage.
type Server struct {
	flight.BaseFlightServer
	store      storage.Storage
	cfg        *Config
	flightSrv  flight.Server
	grpcServer *grpc.Server
}

// New creates a new Flight storage server.
func New(store storage.Storage, cfg *Config) *Server {
	if store == nil {
		panic("flight: storage is nil")
	}
	return &Server{
		store: store,
		cfg:   cfg.clone(),
	}
}

// Init initializes the server but does not start serving.
func (s *Server) Init() error {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(s.cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.cfg.MaxSendMsgSize),
		grpc.MaxConcurrentStreams(s.cfg.MaxConcurrentStreams),
	}

	if s.cfg.TLS != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(s.cfg.TLS)))
	}

	if s.cfg.Auth != nil {
		opts = append(opts,
			grpc.ChainUnaryInterceptor(UnaryAuthInterceptor(s.cfg.Auth)),
			grpc.ChainStreamInterceptor(StreamAuthInterceptor(s.cfg.Auth)),
		)
	}

	var middleware []flight.ServerMiddleware
	if s.cfg.Auth != nil {
		middleware = append(middleware, CreateAuthMiddleware(s.cfg.Auth))
	}

	s.flightSrv = flight.NewServerWithMiddleware(middleware, opts...)

	if s.cfg.Auth != nil {
		s.SetAuthHandler(NewServerAuthHandler(s.cfg.Auth))
	}

	s.flightSrv.RegisterFlightService(s)

	return s.flightSrv.Init(s.cfg.Addr)
}

// Serve starts the server and blocks until shutdown.
func (s *Server) Serve() error {
	if err := s.Init(); err != nil {
		return err
	}
	return s.flightSrv.Serve()
}

// ServeListener starts the server on the given listener.
func (s *Server) ServeListener(lis net.Listener) error {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(s.cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.cfg.MaxSendMsgSize),
		grpc.MaxConcurrentStreams(s.cfg.MaxConcurrentStreams),
	}

	if s.cfg.TLS != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(s.cfg.TLS)))
	}

	if s.cfg.Auth != nil {
		opts = append(opts,
			grpc.ChainUnaryInterceptor(UnaryAuthInterceptor(s.cfg.Auth)),
			grpc.ChainStreamInterceptor(StreamAuthInterceptor(s.cfg.Auth)),
		)
	}

	s.grpcServer = grpc.NewServer(opts...)
	flight.RegisterFlightServiceServer(s.grpcServer, s)
	return s.grpcServer.Serve(lis)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() {
	if s.flightSrv != nil {
		s.flightSrv.Shutdown()
	}
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// GetFlightInfo returns metadata about a flight (object or bucket).
func (s *Server) GetFlightInfo(ctx context.Context, desc *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	if desc.Type == flight.DescriptorPATH {
		return s.getFlightInfoPath(ctx, desc)
	}
	return nil, status.Error(codes.InvalidArgument, "unsupported descriptor type")
}

func (s *Server) getFlightInfoPath(ctx context.Context, desc *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	path := desc.Path

	switch len(path) {
	case 0:
		// List all buckets - not a single flight info
		return nil, status.Error(codes.InvalidArgument, "empty path; use ListFlights for bucket listing")

	case 1:
		// Bucket info
		bucketName := path[0]
		bucket := s.store.Bucket(bucketName)
		info, err := bucket.Info(ctx)
		if err != nil {
			return nil, mapStorageError(err)
		}

		infoBytes, _ := EncodeBucketInfo(info)
		schema := BucketListSchema()
		schemaBytes := flight.SerializeSchema(schema, s.cfg.Allocator)

		return &flight.FlightInfo{
			Schema:           schemaBytes,
			FlightDescriptor: desc,
			Endpoint: []*flight.FlightEndpoint{{
				Ticket: &flight.Ticket{Ticket: infoBytes},
			}},
			TotalRecords: 1,
			TotalBytes:   -1,
		}, nil

	default:
		// Object info: path[0] = bucket, path[1:] = key parts
		bucketName := path[0]
		key := joinPath(path[1:])

		bucket := s.store.Bucket(bucketName)
		obj, err := bucket.Stat(ctx, key, nil)
		if err != nil {
			return nil, mapStorageError(err)
		}

		ticket := &Ticket{Bucket: bucketName, Key: key}
		ticketBytes, _ := EncodeTicket(ticket)

		schema := ObjectDataSchema()
		schemaBytes := flight.SerializeSchema(schema, s.cfg.Allocator)

		objInfoBytes, _ := EncodeObjectInfo(obj)

		return &flight.FlightInfo{
			Schema:           schemaBytes,
			FlightDescriptor: desc,
			Endpoint: []*flight.FlightEndpoint{{
				Ticket:      &flight.Ticket{Ticket: ticketBytes},
				AppMetadata: objInfoBytes,
			}},
			TotalRecords: 1,
			TotalBytes:   obj.Size,
		}, nil
	}
}

// ListFlights lists available flights (objects or buckets).
func (s *Server) ListFlights(criteria *flight.Criteria, stream flight.FlightService_ListFlightsServer) error {
	ctx := stream.Context()

	c, err := DecodeCriteria(criteria.Expression)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid criteria: %v", err)
	}

	if c.Bucket == "" {
		// List buckets
		return s.listBuckets(ctx, c, stream)
	}

	// List objects in bucket
	return s.listObjects(ctx, c, stream)
}

func (s *Server) listBuckets(ctx context.Context, c *Criteria, stream flight.FlightService_ListFlightsServer) error {
	iter, err := s.store.Buckets(ctx, c.Limit, c.Offset, c.Options)
	if err != nil {
		return mapStorageError(err)
	}
	defer iter.Close()

	schema := BucketListSchema()
	schemaBytes := flight.SerializeSchema(schema, s.cfg.Allocator)

	for {
		info, err := iter.Next()
		if err != nil {
			return mapStorageError(err)
		}
		if info == nil {
			break
		}

		infoBytes, _ := EncodeBucketInfo(info)

		fi := &flight.FlightInfo{
			Schema: schemaBytes,
			FlightDescriptor: &flight.FlightDescriptor{
				Type: flight.DescriptorPATH,
				Path: []string{info.Name},
			},
			Endpoint: []*flight.FlightEndpoint{{
				Ticket:      &flight.Ticket{Ticket: infoBytes},
				AppMetadata: infoBytes,
			}},
			TotalRecords: 1,
			TotalBytes:   -1,
		}

		if err := stream.Send(fi); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) listObjects(ctx context.Context, c *Criteria, stream flight.FlightService_ListFlightsServer) error {
	bucket := s.store.Bucket(c.Bucket)
	opts := c.Options
	if opts == nil {
		opts = storage.Options{}
	}
	if c.Recursive {
		opts["recursive"] = true
	}

	iter, err := bucket.List(ctx, c.Prefix, c.Limit, c.Offset, opts)
	if err != nil {
		return mapStorageError(err)
	}
	defer iter.Close()

	schema := ObjectDataSchema()
	schemaBytes := flight.SerializeSchema(schema, s.cfg.Allocator)

	for {
		obj, err := iter.Next()
		if err != nil {
			return mapStorageError(err)
		}
		if obj == nil {
			break
		}

		ticket := &Ticket{Bucket: c.Bucket, Key: obj.Key}
		ticketBytes, _ := EncodeTicket(ticket)
		objInfoBytes, _ := EncodeObjectInfo(obj)

		fi := &flight.FlightInfo{
			Schema: schemaBytes,
			FlightDescriptor: &flight.FlightDescriptor{
				Type: flight.DescriptorPATH,
				Path: []string{c.Bucket, obj.Key},
			},
			Endpoint: []*flight.FlightEndpoint{{
				Ticket:      &flight.Ticket{Ticket: ticketBytes},
				AppMetadata: objInfoBytes,
			}},
			TotalRecords: 1,
			TotalBytes:   obj.Size,
		}

		if err := stream.Send(fi); err != nil {
			return err
		}
	}

	return nil
}

// GetSchema returns the schema for a flight.
func (s *Server) GetSchema(ctx context.Context, desc *flight.FlightDescriptor) (*flight.SchemaResult, error) {
	schema := ObjectDataSchema()
	return &flight.SchemaResult{Schema: flight.SerializeSchema(schema, s.cfg.Allocator)}, nil
}

// DoGet streams object data.
func (s *Server) DoGet(ticket *flight.Ticket, stream flight.FlightService_DoGetServer) error {
	ctx := stream.Context()

	t, err := DecodeTicket(ticket.Ticket)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid ticket: %v", err)
	}

	bucket := s.store.Bucket(t.Bucket)
	rc, obj, err := bucket.Open(ctx, t.Key, t.Offset, t.Length, t.Options)
	if err != nil {
		return mapStorageError(err)
	}
	defer rc.Close()

	// Create writer
	writer := flight.NewRecordWriter(stream, ipc.WithSchema(ObjectDataSchema()))
	defer writer.Close()

	// Set object metadata
	objInfoBytes, _ := EncodeObjectInfo(obj)
	writer.SetFlightDescriptor(&flight.FlightDescriptor{
		Type: flight.DescriptorPATH,
		Path: []string{t.Bucket, t.Key},
	})

	// Send first message with metadata
	if err := stream.Send(&flight.FlightData{
		FlightDescriptor: &flight.FlightDescriptor{
			Type: flight.DescriptorPATH,
			Path: []string{t.Bucket, t.Key},
		},
		AppMetadata: objInfoBytes,
	}); err != nil {
		return err
	}

	// Stream data in chunks
	builder := NewObjectDataBuilder(s.cfg.Allocator)
	buf := make([]byte, s.cfg.ChunkSize)

	for {
		n, err := rc.Read(buf)
		if n > 0 {
			rec := builder.Build(buf[:n])
			if err := writer.Write(rec); err != nil {
				rec.Release()
				return err
			}
			rec.Release()
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

// DoPut handles object uploads.
func (s *Server) DoPut(stream flight.FlightService_DoPutServer) error {
	ctx := stream.Context()

	// Read first message with descriptor
	reader, err := flight.NewRecordReader(stream)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to create reader: %v", err)
	}
	defer reader.Release()

	desc := reader.LatestFlightDescriptor()
	if desc == nil {
		return status.Error(codes.InvalidArgument, "missing flight descriptor")
	}

	// Decode upload descriptor from CMD
	var uploadDesc *UploadDescriptor
	if desc.Type == flight.DescriptorCMD {
		uploadDesc, err = DecodeUploadDescriptor(desc.Cmd)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid upload descriptor: %v", err)
		}
	} else if desc.Type == flight.DescriptorPATH && len(desc.Path) >= 2 {
		// Use path as bucket/key with metadata from app_metadata
		uploadDesc = &UploadDescriptor{
			Bucket: desc.Path[0],
			Key:    joinPath(desc.Path[1:]),
			Size:   -1,
		}
		// Try to get additional metadata from app_metadata
		if appMeta := reader.LatestAppMetadata(); len(appMeta) > 0 {
			var extra UploadDescriptor
			if json.Unmarshal(appMeta, &extra) == nil {
				if extra.ContentType != "" {
					uploadDesc.ContentType = extra.ContentType
				}
				if extra.Size > 0 {
					uploadDesc.Size = extra.Size
				}
				if extra.Options != nil {
					uploadDesc.Options = extra.Options
				}
			}
		}
	} else {
		return status.Error(codes.InvalidArgument, "invalid descriptor format")
	}

	// Collect all data
	var buf bytes.Buffer
	for reader.Next() {
		rec := reader.Record()
		if rec.NumCols() > 0 {
			col := rec.Column(0)
			if binArr, ok := col.(*array.Binary); ok {
				for i := 0; i < binArr.Len(); i++ {
					if !binArr.IsNull(i) {
						buf.Write(binArr.Value(i))
					}
				}
			}
		}
	}

	if err := reader.Err(); err != nil {
		return mapStorageError(err)
	}

	// Write to storage
	bucket := s.store.Bucket(uploadDesc.Bucket)
	size := int64(buf.Len())
	if uploadDesc.Size >= 0 && uploadDesc.Size != size {
		return status.Errorf(codes.InvalidArgument, "size mismatch: expected %d, got %d", uploadDesc.Size, size)
	}

	obj, err := bucket.Write(ctx, uploadDesc.Key, &buf, size, uploadDesc.ContentType, uploadDesc.Options)
	if err != nil {
		return mapStorageError(err)
	}

	// Send result
	objInfoBytes, _ := EncodeObjectInfo(obj)
	return stream.Send(&flight.PutResult{AppMetadata: objInfoBytes})
}

// DoExchange handles bidirectional streaming for multipart uploads.
func (s *Server) DoExchange(stream flight.FlightService_DoExchangeServer) error {
	ctx := stream.Context()

	for {
		// Read request
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Decode exchange message from app_metadata
		if len(data.AppMetadata) == 0 {
			continue
		}

		msg, err := DecodeExchangeMessage(data.AppMetadata)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid exchange message: %v", err)
		}

		switch msg.Type {
		case ExchangeInitMultipart:
			if err := s.handleExchangeInitMultipart(ctx, msg, stream); err != nil {
				return err
			}

		case ExchangeUploadPart:
			if err := s.handleExchangeUploadPart(ctx, msg, data, stream); err != nil {
				return err
			}

		case ExchangeCompleteMultipart:
			if err := s.handleExchangeCompleteMultipart(ctx, msg, stream); err != nil {
				return err
			}

		case ExchangeAbortMultipart:
			if err := s.handleExchangeAbortMultipart(ctx, msg, stream); err != nil {
				return err
			}

		default:
			return status.Errorf(codes.InvalidArgument, "unknown exchange message type: %s", msg.Type)
		}
	}
}

func (s *Server) handleExchangeInitMultipart(ctx context.Context, msg *ExchangeMessage, stream flight.FlightService_DoExchangeServer) error {
	var req InitMultipartRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid init multipart request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	upload, err := mp.InitMultipart(ctx, req.Key, req.ContentType, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	respBytes, _ := EncodeExchangeMessage(ExchangeMultipartInfo, MultipartUploadToJSON(upload))
	return stream.Send(&flight.FlightData{AppMetadata: respBytes})
}

func (s *Server) handleExchangeUploadPart(ctx context.Context, msg *ExchangeMessage, data *flight.FlightData, stream flight.FlightService_DoExchangeServer) error {
	var req UploadPartMessage
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid upload part request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: req.UploadID,
	}

	// Collect part data from subsequent messages
	var buf bytes.Buffer

	// Check if data is in the DataBody
	if len(data.DataBody) > 0 {
		buf.Write(data.DataBody)
	}

	// Read more data if needed
	for buf.Len() < int(req.Size) {
		partData, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Check for next command (means end of part data)
		if len(partData.AppMetadata) > 0 {
			nextMsg, _ := DecodeExchangeMessage(partData.AppMetadata)
			if nextMsg != nil && nextMsg.Type != ExchangePartData {
				// Push back - handle this message
				// For now, we complete current part and process would need buffering
				break
			}
		}

		if len(partData.DataBody) > 0 {
			buf.Write(partData.DataBody)
		}
	}

	partInfo, err := mp.UploadPart(ctx, mu, req.PartNumber, &buf, int64(buf.Len()), req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	respBytes, _ := EncodeExchangeMessage(ExchangePartInfo, PartInfoToJSON(partInfo))
	return stream.Send(&flight.FlightData{AppMetadata: respBytes})
}

func (s *Server) handleExchangeCompleteMultipart(ctx context.Context, msg *ExchangeMessage, stream flight.FlightService_DoExchangeServer) error {
	var req CompleteMultipartRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid complete multipart request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: req.UploadID,
	}

	parts := make([]*storage.PartInfo, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = PartInfoFromJSON(p)
	}

	obj, err := mp.CompleteMultipart(ctx, mu, parts, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	respBytes, _ := EncodeExchangeMessage(ExchangeObjectInfo, ObjectToJSON(obj))
	return stream.Send(&flight.FlightData{AppMetadata: respBytes})
}

func (s *Server) handleExchangeAbortMultipart(ctx context.Context, msg *ExchangeMessage, stream flight.FlightService_DoExchangeServer) error {
	var req AbortMultipartRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid abort multipart request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: req.UploadID,
	}

	if err := mp.AbortMultipart(ctx, mu, req.Options); err != nil {
		return mapStorageError(err)
	}

	respBytes, _ := EncodeExchangeMessage(ExchangeObjectInfo, nil)
	return stream.Send(&flight.FlightData{AppMetadata: respBytes})
}

// DoAction handles custom actions.
func (s *Server) DoAction(action *flight.Action, stream flight.FlightService_DoActionServer) error {
	ctx := stream.Context()

	switch action.Type {
	case ActionCreateBucket:
		return s.handleCreateBucket(ctx, action.Body, stream)
	case ActionDeleteBucket:
		return s.handleDeleteBucket(ctx, action.Body, stream)
	case ActionDeleteObject:
		return s.handleDeleteObject(ctx, action.Body, stream)
	case ActionCopyObject:
		return s.handleCopyObject(ctx, action.Body, stream)
	case ActionMoveObject:
		return s.handleMoveObject(ctx, action.Body, stream)
	case ActionInitMultipart:
		return s.handleInitMultipart(ctx, action.Body, stream)
	case ActionCompleteMultipart:
		return s.handleCompleteMultipart(ctx, action.Body, stream)
	case ActionAbortMultipart:
		return s.handleAbortMultipart(ctx, action.Body, stream)
	case ActionSignedURL:
		return s.handleSignedURL(ctx, action.Body, stream)
	case ActionGetFeatures:
		return s.handleGetFeatures(ctx, action.Body, stream)
	case ActionStat:
		return s.handleStat(ctx, action.Body, stream)
	default:
		return status.Errorf(codes.InvalidArgument, "unknown action type: %s", action.Type)
	}
}

func (s *Server) handleCreateBucket(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req CreateBucketRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	info, err := s.store.CreateBucket(ctx, req.Name, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	infoBytes, _ := EncodeBucketInfo(info)
	return stream.Send(&flight.Result{Body: infoBytes})
}

func (s *Server) handleDeleteBucket(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req DeleteBucketRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	if err := s.store.DeleteBucket(ctx, req.Name, req.Options); err != nil {
		return mapStorageError(err)
	}

	return stream.Send(&flight.Result{Body: []byte("{}")})
}

func (s *Server) handleDeleteObject(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req DeleteObjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	if err := bucket.Delete(ctx, req.Key, req.Options); err != nil {
		return mapStorageError(err)
	}

	return stream.Send(&flight.Result{Body: []byte("{}")})
}

func (s *Server) handleCopyObject(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req CopyObjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	bucket := s.store.Bucket(req.DstBucket)
	obj, err := bucket.Copy(ctx, req.DstKey, req.SrcBucket, req.SrcKey, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	objBytes, _ := EncodeObjectInfo(obj)
	return stream.Send(&flight.Result{Body: objBytes})
}

func (s *Server) handleMoveObject(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req MoveObjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	bucket := s.store.Bucket(req.DstBucket)
	obj, err := bucket.Move(ctx, req.DstKey, req.SrcBucket, req.SrcKey, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	objBytes, _ := EncodeObjectInfo(obj)
	return stream.Send(&flight.Result{Body: objBytes})
}

func (s *Server) handleInitMultipart(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req InitMultipartRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	upload, err := mp.InitMultipart(ctx, req.Key, req.ContentType, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	uploadBytes, _ := EncodeMultipartUpload(upload)
	return stream.Send(&flight.Result{Body: uploadBytes})
}

func (s *Server) handleCompleteMultipart(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req CompleteMultipartRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: req.UploadID,
	}

	parts := make([]*storage.PartInfo, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = PartInfoFromJSON(p)
	}

	obj, err := mp.CompleteMultipart(ctx, mu, parts, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	objBytes, _ := EncodeObjectInfo(obj)
	return stream.Send(&flight.Result{Body: objBytes})
}

func (s *Server) handleAbortMultipart(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req AbortMultipartRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		return status.Error(codes.Unimplemented, "multipart upload not supported")
	}

	mu := &storage.MultipartUpload{
		Bucket:   req.Bucket,
		Key:      req.Key,
		UploadID: req.UploadID,
	}

	if err := mp.AbortMultipart(ctx, mu, req.Options); err != nil {
		return mapStorageError(err)
	}

	return stream.Send(&flight.Result{Body: []byte("{}")})
}

func (s *Server) handleSignedURL(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req SignedURLRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	expires, err := time.ParseDuration(req.Expires)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid expires duration: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	url, err := bucket.SignedURL(ctx, req.Key, req.Method, expires, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	resp := SignedURLResponse{URL: url}
	respBytes, _ := json.Marshal(resp)
	return stream.Send(&flight.Result{Body: respBytes})
}

func (s *Server) handleGetFeatures(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req GetFeaturesRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
		}
	}

	var features storage.Features
	if req.Bucket != "" {
		bucket := s.store.Bucket(req.Bucket)
		features = bucket.Features()
	} else {
		features = s.store.Features()
	}

	featuresBytes, _ := EncodeFeatures(features)
	return stream.Send(&flight.Result{Body: featuresBytes})
}

func (s *Server) handleStat(ctx context.Context, body []byte, stream flight.FlightService_DoActionServer) error {
	var req StatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	bucket := s.store.Bucket(req.Bucket)
	obj, err := bucket.Stat(ctx, req.Key, req.Options)
	if err != nil {
		return mapStorageError(err)
	}

	objBytes, _ := EncodeObjectInfo(obj)
	return stream.Send(&flight.Result{Body: objBytes})
}

// ListActions returns all supported actions.
func (s *Server) ListActions(_ *flight.Empty, stream flight.FlightService_ListActionsServer) error {
	actions := []*flight.ActionType{
		{Type: ActionCreateBucket, Description: "Create a new bucket"},
		{Type: ActionDeleteBucket, Description: "Delete a bucket"},
		{Type: ActionDeleteObject, Description: "Delete an object"},
		{Type: ActionCopyObject, Description: "Copy an object"},
		{Type: ActionMoveObject, Description: "Move an object"},
		{Type: ActionInitMultipart, Description: "Initialize a multipart upload"},
		{Type: ActionCompleteMultipart, Description: "Complete a multipart upload"},
		{Type: ActionAbortMultipart, Description: "Abort a multipart upload"},
		{Type: ActionSignedURL, Description: "Generate a signed URL"},
		{Type: ActionGetFeatures, Description: "Get storage features"},
		{Type: ActionStat, Description: "Get object metadata"},
	}
	for _, action := range actions {
		if err := stream.Send(action); err != nil {
			return err
		}
	}
	return nil
}

// joinPath joins path components with '/'.
func joinPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result = result + "/" + p
	}
	return result
}

// Logger returns the configured logger.
func (s *Server) Logger() *slog.Logger {
	return s.cfg.Logger
}

// Addr returns the configured address.
func (s *Server) Addr() string {
	return s.cfg.Addr
}

// Config returns a copy of the server configuration.
func (s *Server) Config() Config {
	return *s.cfg
}

// GetGRPCServer returns the underlying gRPC server for testing.
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.grpcServer
}

// RegisterOnGRPC registers the Flight service on an existing gRPC server.
func (s *Server) RegisterOnGRPC(grpcServer *grpc.Server) {
	flight.RegisterFlightServiceServer(grpcServer, s)
}

// Handshake handles the authentication handshake.
func (s *Server) Handshake(stream flight.FlightService_HandshakeServer) error {
	if s.cfg.Auth == nil {
		return status.Error(codes.Unimplemented, "authentication not configured")
	}

	authHandler := NewServerAuthHandler(s.cfg.Auth)

	// Wrap stream in AuthConn
	conn := &authConnWrapper{stream: stream}
	if err := authHandler.Authenticate(conn); err != nil {
		return err
	}

	return nil
}

// authConnWrapper wraps a handshake stream as flight.AuthConn.
type authConnWrapper struct {
	stream flight.FlightService_HandshakeServer
}

func (w *authConnWrapper) Read() ([]byte, error) {
	req, err := w.stream.Recv()
	if err != nil {
		return nil, err
	}
	return req.Payload, nil
}

func (w *authConnWrapper) Send(data []byte) error {
	return w.stream.Send(&flight.HandshakeResponse{Payload: data})
}

// Verify interfaces
var (
	_ flight.FlightServer = (*Server)(nil)
)

// GetStorageFromContext is a helper to get storage-related info from context.
// This is useful for middleware that needs access to storage operations.
func GetStorageFromContext(ctx context.Context) map[string]any {
	return Claims(ctx)
}

// formatError formats an error for logging.
func formatError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
