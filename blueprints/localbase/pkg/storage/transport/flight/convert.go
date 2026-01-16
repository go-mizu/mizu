package flight

import (
	"encoding/json"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
)

// Action types for DoAction.
const (
	ActionCreateBucket      = "CreateBucket"
	ActionDeleteBucket      = "DeleteBucket"
	ActionDeleteObject      = "DeleteObject"
	ActionCopyObject        = "CopyObject"
	ActionMoveObject        = "MoveObject"
	ActionInitMultipart     = "InitMultipart"
	ActionCompleteMultipart = "CompleteMultipart"
	ActionAbortMultipart    = "AbortMultipart"
	ActionSignedURL         = "SignedURL"
	ActionGetFeatures       = "GetFeatures"
	ActionStat              = "Stat"
)

// Ticket represents a DoGet ticket.
type Ticket struct {
	Bucket  string          `json:"bucket"`
	Key     string          `json:"key"`
	Offset  int64           `json:"offset,omitempty"`
	Length  int64           `json:"length,omitempty"`
	Options storage.Options `json:"options,omitempty"`
}

// EncodeTicket encodes a ticket to bytes.
func EncodeTicket(t *Ticket) ([]byte, error) {
	return json.Marshal(t)
}

// DecodeTicket decodes a ticket from bytes.
func DecodeTicket(data []byte) (*Ticket, error) {
	var t Ticket
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// Criteria represents ListFlights criteria.
type Criteria struct {
	Bucket    string          `json:"bucket,omitempty"`
	Prefix    string          `json:"prefix,omitempty"`
	Limit     int             `json:"limit,omitempty"`
	Offset    int             `json:"offset,omitempty"`
	Recursive bool            `json:"recursive,omitempty"`
	Options   storage.Options `json:"options,omitempty"`
}

// EncodeCriteria encodes criteria to bytes.
func EncodeCriteria(c *Criteria) ([]byte, error) {
	return json.Marshal(c)
}

// DecodeCriteria decodes criteria from bytes.
func DecodeCriteria(data []byte) (*Criteria, error) {
	if len(data) == 0 {
		return &Criteria{}, nil
	}
	var c Criteria
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// UploadDescriptor represents DoPut descriptor data.
type UploadDescriptor struct {
	Bucket      string          `json:"bucket"`
	Key         string          `json:"key"`
	Size        int64           `json:"size"`
	ContentType string          `json:"content_type,omitempty"`
	Options     storage.Options `json:"options,omitempty"`
}

// EncodeUploadDescriptor encodes an upload descriptor to bytes.
func EncodeUploadDescriptor(d *UploadDescriptor) ([]byte, error) {
	return json.Marshal(d)
}

// DecodeUploadDescriptor decodes an upload descriptor from bytes.
func DecodeUploadDescriptor(data []byte) (*UploadDescriptor, error) {
	var d UploadDescriptor
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// ObjectInfoJSON is the JSON representation of Object metadata.
type ObjectInfoJSON struct {
	Bucket      string            `json:"bucket,omitempty"`
	Key         string            `json:"key,omitempty"`
	Size        int64             `json:"size,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	ETag        string            `json:"etag,omitempty"`
	Version     string            `json:"version,omitempty"`
	Created     *time.Time        `json:"created,omitempty"`
	Updated     *time.Time        `json:"updated,omitempty"`
	Hash        storage.Hashes    `json:"hash,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	IsDir       bool              `json:"is_dir,omitempty"`
}

// ObjectToJSON converts a storage.Object to JSON representation.
func ObjectToJSON(obj *storage.Object) *ObjectInfoJSON {
	if obj == nil {
		return nil
	}
	j := &ObjectInfoJSON{
		Bucket:      obj.Bucket,
		Key:         obj.Key,
		Size:        obj.Size,
		ContentType: obj.ContentType,
		ETag:        obj.ETag,
		Version:     obj.Version,
		Hash:        obj.Hash,
		Metadata:    obj.Metadata,
		IsDir:       obj.IsDir,
	}
	if !obj.Created.IsZero() {
		j.Created = &obj.Created
	}
	if !obj.Updated.IsZero() {
		j.Updated = &obj.Updated
	}
	return j
}

// ObjectFromJSON converts JSON representation to storage.Object.
func ObjectFromJSON(j *ObjectInfoJSON) *storage.Object {
	if j == nil {
		return nil
	}
	obj := &storage.Object{
		Bucket:      j.Bucket,
		Key:         j.Key,
		Size:        j.Size,
		ContentType: j.ContentType,
		ETag:        j.ETag,
		Version:     j.Version,
		Hash:        j.Hash,
		Metadata:    j.Metadata,
		IsDir:       j.IsDir,
	}
	if j.Created != nil {
		obj.Created = *j.Created
	}
	if j.Updated != nil {
		obj.Updated = *j.Updated
	}
	return obj
}

// EncodeObjectInfo encodes object info to JSON bytes.
func EncodeObjectInfo(obj *storage.Object) ([]byte, error) {
	return json.Marshal(ObjectToJSON(obj))
}

// DecodeObjectInfo decodes object info from JSON bytes.
func DecodeObjectInfo(data []byte) (*storage.Object, error) {
	var j ObjectInfoJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	return ObjectFromJSON(&j), nil
}

// BucketInfoJSON is the JSON representation of BucketInfo.
type BucketInfoJSON struct {
	Name      string            `json:"name,omitempty"`
	CreatedAt *time.Time        `json:"created_at,omitempty"`
	Public    bool              `json:"public,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// BucketInfoToJSON converts a storage.BucketInfo to JSON representation.
func BucketInfoToJSON(info *storage.BucketInfo) *BucketInfoJSON {
	if info == nil {
		return nil
	}
	j := &BucketInfoJSON{
		Name:     info.Name,
		Public:   info.Public,
		Metadata: info.Metadata,
	}
	if !info.CreatedAt.IsZero() {
		j.CreatedAt = &info.CreatedAt
	}
	return j
}

// BucketInfoFromJSON converts JSON representation to storage.BucketInfo.
func BucketInfoFromJSON(j *BucketInfoJSON) *storage.BucketInfo {
	if j == nil {
		return nil
	}
	info := &storage.BucketInfo{
		Name:     j.Name,
		Public:   j.Public,
		Metadata: j.Metadata,
	}
	if j.CreatedAt != nil {
		info.CreatedAt = *j.CreatedAt
	}
	return info
}

// EncodeBucketInfo encodes bucket info to JSON bytes.
func EncodeBucketInfo(info *storage.BucketInfo) ([]byte, error) {
	return json.Marshal(BucketInfoToJSON(info))
}

// DecodeBucketInfo decodes bucket info from JSON bytes.
func DecodeBucketInfo(data []byte) (*storage.BucketInfo, error) {
	var j BucketInfoJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	return BucketInfoFromJSON(&j), nil
}

// MultipartUploadJSON is the JSON representation of MultipartUpload.
type MultipartUploadJSON struct {
	Bucket   string            `json:"bucket,omitempty"`
	Key      string            `json:"key,omitempty"`
	UploadID string            `json:"upload_id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// MultipartUploadToJSON converts a storage.MultipartUpload to JSON representation.
func MultipartUploadToJSON(mu *storage.MultipartUpload) *MultipartUploadJSON {
	if mu == nil {
		return nil
	}
	return &MultipartUploadJSON{
		Bucket:   mu.Bucket,
		Key:      mu.Key,
		UploadID: mu.UploadID,
		Metadata: mu.Metadata,
	}
}

// MultipartUploadFromJSON converts JSON representation to storage.MultipartUpload.
func MultipartUploadFromJSON(j *MultipartUploadJSON) *storage.MultipartUpload {
	if j == nil {
		return nil
	}
	return &storage.MultipartUpload{
		Bucket:   j.Bucket,
		Key:      j.Key,
		UploadID: j.UploadID,
		Metadata: j.Metadata,
	}
}

// EncodeMultipartUpload encodes multipart upload to JSON bytes.
func EncodeMultipartUpload(mu *storage.MultipartUpload) ([]byte, error) {
	return json.Marshal(MultipartUploadToJSON(mu))
}

// DecodeMultipartUpload decodes multipart upload from JSON bytes.
func DecodeMultipartUpload(data []byte) (*storage.MultipartUpload, error) {
	var j MultipartUploadJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	return MultipartUploadFromJSON(&j), nil
}

// PartInfoJSON is the JSON representation of PartInfo.
type PartInfoJSON struct {
	Number       int            `json:"number,omitempty"`
	Size         int64          `json:"size,omitempty"`
	ETag         string         `json:"etag,omitempty"`
	Hash         storage.Hashes `json:"hash,omitempty"`
	LastModified *time.Time     `json:"last_modified,omitempty"`
}

// PartInfoToJSON converts a storage.PartInfo to JSON representation.
func PartInfoToJSON(p *storage.PartInfo) *PartInfoJSON {
	if p == nil {
		return nil
	}
	return &PartInfoJSON{
		Number:       p.Number,
		Size:         p.Size,
		ETag:         p.ETag,
		Hash:         p.Hash,
		LastModified: p.LastModified,
	}
}

// PartInfoFromJSON converts JSON representation to storage.PartInfo.
func PartInfoFromJSON(j *PartInfoJSON) *storage.PartInfo {
	if j == nil {
		return nil
	}
	return &storage.PartInfo{
		Number:       j.Number,
		Size:         j.Size,
		ETag:         j.ETag,
		Hash:         j.Hash,
		LastModified: j.LastModified,
	}
}

// EncodePartInfo encodes part info to JSON bytes.
func EncodePartInfo(p *storage.PartInfo) ([]byte, error) {
	return json.Marshal(PartInfoToJSON(p))
}

// DecodePartInfo decodes part info from JSON bytes.
func DecodePartInfo(data []byte) (*storage.PartInfo, error) {
	var j PartInfoJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	return PartInfoFromJSON(&j), nil
}

// FeaturesJSON is the JSON representation of Features.
type FeaturesJSON struct {
	Features storage.Features `json:"features,omitempty"`
}

// EncodeFeatures encodes features to JSON bytes.
func EncodeFeatures(f storage.Features) ([]byte, error) {
	return json.Marshal(FeaturesJSON{Features: f})
}

// DecodeFeatures decodes features from JSON bytes.
func DecodeFeatures(data []byte) (storage.Features, error) {
	var j FeaturesJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	return j.Features, nil
}

// Action request/response types

// CreateBucketRequest is the request for CreateBucket action.
type CreateBucketRequest struct {
	Name    string          `json:"name"`
	Options storage.Options `json:"options,omitempty"`
}

// DeleteBucketRequest is the request for DeleteBucket action.
type DeleteBucketRequest struct {
	Name    string          `json:"name"`
	Options storage.Options `json:"options,omitempty"`
}

// DeleteObjectRequest is the request for DeleteObject action.
type DeleteObjectRequest struct {
	Bucket  string          `json:"bucket"`
	Key     string          `json:"key"`
	Options storage.Options `json:"options,omitempty"`
}

// CopyObjectRequest is the request for CopyObject action.
type CopyObjectRequest struct {
	SrcBucket string          `json:"src_bucket"`
	SrcKey    string          `json:"src_key"`
	DstBucket string          `json:"dst_bucket"`
	DstKey    string          `json:"dst_key"`
	Options   storage.Options `json:"options,omitempty"`
}

// MoveObjectRequest is the request for MoveObject action.
type MoveObjectRequest struct {
	SrcBucket string          `json:"src_bucket"`
	SrcKey    string          `json:"src_key"`
	DstBucket string          `json:"dst_bucket"`
	DstKey    string          `json:"dst_key"`
	Options   storage.Options `json:"options,omitempty"`
}

// InitMultipartRequest is the request for InitMultipart action.
type InitMultipartRequest struct {
	Bucket      string          `json:"bucket"`
	Key         string          `json:"key"`
	ContentType string          `json:"content_type,omitempty"`
	Options     storage.Options `json:"options,omitempty"`
}

// CompleteMultipartRequest is the request for CompleteMultipart action.
type CompleteMultipartRequest struct {
	Bucket   string          `json:"bucket"`
	Key      string          `json:"key"`
	UploadID string          `json:"upload_id"`
	Parts    []*PartInfoJSON `json:"parts"`
	Options  storage.Options `json:"options,omitempty"`
}

// AbortMultipartRequest is the request for AbortMultipart action.
type AbortMultipartRequest struct {
	Bucket   string          `json:"bucket"`
	Key      string          `json:"key"`
	UploadID string          `json:"upload_id"`
	Options  storage.Options `json:"options,omitempty"`
}

// SignedURLRequest is the request for SignedURL action.
type SignedURLRequest struct {
	Bucket  string          `json:"bucket"`
	Key     string          `json:"key"`
	Method  string          `json:"method"`
	Expires string          `json:"expires"` // Duration string
	Options storage.Options `json:"options,omitempty"`
}

// SignedURLResponse is the response for SignedURL action.
type SignedURLResponse struct {
	URL string `json:"url"`
}

// GetFeaturesRequest is the request for GetFeatures action.
type GetFeaturesRequest struct {
	Bucket string `json:"bucket,omitempty"`
}

// StatRequest is the request for Stat action.
type StatRequest struct {
	Bucket  string          `json:"bucket"`
	Key     string          `json:"key"`
	Options storage.Options `json:"options,omitempty"`
}

// ExchangeMessage types for DoExchange

// ExchangeMessageType identifies the type of exchange message.
type ExchangeMessageType string

const (
	ExchangeInitMultipart     ExchangeMessageType = "init_multipart"
	ExchangeUploadPart        ExchangeMessageType = "upload_part"
	ExchangeCompleteMultipart ExchangeMessageType = "complete_multipart"
	ExchangeAbortMultipart    ExchangeMessageType = "abort_multipart"
	ExchangePartData          ExchangeMessageType = "part_data"
	ExchangePartInfo          ExchangeMessageType = "part_info"
	ExchangeMultipartInfo     ExchangeMessageType = "multipart_info"
	ExchangeObjectInfo        ExchangeMessageType = "object_info"
	ExchangeError             ExchangeMessageType = "error"
)

// ExchangeMessage is the wrapper for DoExchange messages.
type ExchangeMessage struct {
	Type    ExchangeMessageType `json:"type"`
	Payload json.RawMessage     `json:"payload,omitempty"`
}

// UploadPartMessage is the message for uploading a part.
type UploadPartMessage struct {
	Bucket     string          `json:"bucket"`
	Key        string          `json:"key"`
	UploadID   string          `json:"upload_id"`
	PartNumber int             `json:"part_number"`
	Size       int64           `json:"size"`
	Options    storage.Options `json:"options,omitempty"`
}

// EncodeExchangeMessage encodes an exchange message.
func EncodeExchangeMessage(msgType ExchangeMessageType, payload any) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ExchangeMessage{
		Type:    msgType,
		Payload: payloadBytes,
	})
}

// DecodeExchangeMessage decodes an exchange message.
func DecodeExchangeMessage(data []byte) (*ExchangeMessage, error) {
	var msg ExchangeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
