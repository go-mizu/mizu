// File: lib/storage/transport/grpc/convert.go

package grpc

import (
	"encoding/json"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	pb "github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/transport/grpc/storagepb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// bucketInfoToProto converts storage.BucketInfo to protobuf.
func bucketInfoToProto(b *storage.BucketInfo) *pb.BucketInfo {
	if b == nil {
		return nil
	}
	return &pb.BucketInfo{
		Name:      b.Name,
		CreatedAt: timestamppb.New(b.CreatedAt),
		Public:    b.Public,
		Metadata:  b.Metadata,
	}
}

// bucketInfoFromProto converts protobuf to storage.BucketInfo.
func bucketInfoFromProto(b *pb.BucketInfo) *storage.BucketInfo {
	if b == nil {
		return nil
	}
	info := &storage.BucketInfo{
		Name:     b.Name,
		Public:   b.Public,
		Metadata: b.Metadata,
	}
	if b.CreatedAt != nil {
		info.CreatedAt = b.CreatedAt.AsTime()
	}
	return info
}

// objectInfoToProto converts storage.Object to protobuf.
func objectInfoToProto(o *storage.Object) *pb.ObjectInfo {
	if o == nil {
		return nil
	}
	return &pb.ObjectInfo{
		Bucket:      o.Bucket,
		Key:         o.Key,
		Size:        o.Size,
		ContentType: o.ContentType,
		Etag:        o.ETag,
		Version:     o.Version,
		Created:     timestamppb.New(o.Created),
		Updated:     timestamppb.New(o.Updated),
		Hash:        storage.Hashes(o.Hash),
		Metadata:    o.Metadata,
		IsDir:       o.IsDir,
	}
}

// objectInfoFromProto converts protobuf to storage.Object.
func objectInfoFromProto(o *pb.ObjectInfo) *storage.Object {
	if o == nil {
		return nil
	}
	obj := &storage.Object{
		Bucket:      o.Bucket,
		Key:         o.Key,
		Size:        o.Size,
		ContentType: o.ContentType,
		ETag:        o.Etag,
		Version:     o.Version,
		Hash:        storage.Hashes(o.Hash),
		Metadata:    o.Metadata,
		IsDir:       o.IsDir,
	}
	if o.Created != nil {
		obj.Created = o.Created.AsTime()
	}
	if o.Updated != nil {
		obj.Updated = o.Updated.AsTime()
	}
	return obj
}

// multipartUploadToProto converts storage.MultipartUpload to protobuf.
func multipartUploadToProto(m *storage.MultipartUpload) *pb.MultipartUpload {
	if m == nil {
		return nil
	}
	return &pb.MultipartUpload{
		Bucket:   m.Bucket,
		Key:      m.Key,
		UploadId: m.UploadID,
		Metadata: m.Metadata,
	}
}

// multipartUploadFromProto converts protobuf to storage.MultipartUpload.
func multipartUploadFromProto(m *pb.MultipartUpload) *storage.MultipartUpload {
	if m == nil {
		return nil
	}
	return &storage.MultipartUpload{
		Bucket:   m.Bucket,
		Key:      m.Key,
		UploadID: m.UploadId,
		Metadata: m.Metadata,
	}
}

// partInfoToProto converts storage.PartInfo to protobuf.
func partInfoToProto(p *storage.PartInfo) *pb.PartInfo {
	if p == nil {
		return nil
	}
	info := &pb.PartInfo{
		Number: int32(p.Number),
		Size:   p.Size,
		Etag:   p.ETag,
		Hash:   storage.Hashes(p.Hash),
	}
	if p.LastModified != nil {
		info.LastModified = timestamppb.New(*p.LastModified)
	}
	return info
}

// partInfoFromProto converts protobuf to storage.PartInfo.
func partInfoFromProto(p *pb.PartInfo) *storage.PartInfo {
	if p == nil {
		return nil
	}
	info := &storage.PartInfo{
		Number: int(p.Number),
		Size:   p.Size,
		ETag:   p.Etag,
		Hash:   storage.Hashes(p.Hash),
	}
	if p.LastModified != nil {
		t := p.LastModified.AsTime()
		info.LastModified = &t
	}
	return info
}

// featuresToProto converts storage.Features to protobuf.
func featuresToProto(f storage.Features) *pb.Features {
	return &pb.Features{
		Features: f,
	}
}

// featuresFromProto converts protobuf to storage.Features.
func featuresFromProto(f *pb.Features) storage.Features {
	if f == nil {
		return nil
	}
	return storage.Features(f.Features)
}

// optionsToProto converts storage.Options to protobuf map[string][]byte.
func optionsToProto(opts storage.Options) map[string][]byte {
	if opts == nil {
		return nil
	}
	result := make(map[string][]byte, len(opts))
	for k, v := range opts {
		data, _ := json.Marshal(v)
		result[k] = data
	}
	return result
}

// optionsFromProto converts protobuf map[string][]byte to storage.Options.
func optionsFromProto(opts map[string][]byte) storage.Options {
	if opts == nil {
		return nil
	}
	result := make(storage.Options, len(opts))
	for k, v := range opts {
		var val any
		if err := json.Unmarshal(v, &val); err == nil {
			result[k] = val
		} else {
			result[k] = string(v)
		}
	}
	return result
}
