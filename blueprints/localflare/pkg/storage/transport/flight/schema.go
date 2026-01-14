package flight

import (
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-mizu/blueprints/drive/lib/storage"
)

// ObjectDataSchema returns the schema for object data streams.
// The data is transferred as a single binary column with metadata in schema.
func ObjectDataSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{
		{Name: "data", Type: arrow.BinaryTypes.Binary, Nullable: true},
	}, nil)
}

// ObjectListSchema returns the schema for object listings.
func ObjectListSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{
		{Name: "bucket", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "key", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "size", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "content_type", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "etag", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "version", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "created", Type: arrow.FixedWidthTypes.Timestamp_ns, Nullable: true},
		{Name: "updated", Type: arrow.FixedWidthTypes.Timestamp_ns, Nullable: true},
		{Name: "is_dir", Type: arrow.FixedWidthTypes.Boolean, Nullable: true},
	}, nil)
}

// BucketListSchema returns the schema for bucket listings.
func BucketListSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{
		{Name: "name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "created_at", Type: arrow.FixedWidthTypes.Timestamp_ns, Nullable: true},
		{Name: "public", Type: arrow.FixedWidthTypes.Boolean, Nullable: true},
	}, nil)
}

// PartListSchema returns the schema for multipart part listings.
func PartListSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{
		{Name: "number", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "size", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "etag", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "last_modified", Type: arrow.FixedWidthTypes.Timestamp_ns, Nullable: true},
	}, nil)
}

// ObjectDataBuilder builds record batches for object data.
type ObjectDataBuilder struct {
	alloc  memory.Allocator
	schema *arrow.Schema
}

// NewObjectDataBuilder creates a new object data builder.
func NewObjectDataBuilder(alloc memory.Allocator) *ObjectDataBuilder {
	if alloc == nil {
		alloc = memory.DefaultAllocator
	}
	return &ObjectDataBuilder{
		alloc:  alloc,
		schema: ObjectDataSchema(),
	}
}

// Schema returns the schema for object data.
func (b *ObjectDataBuilder) Schema() *arrow.Schema {
	return b.schema
}

// Build creates a record batch containing the given data chunk.
func (b *ObjectDataBuilder) Build(data []byte) arrow.Record {
	bldr := array.NewBinaryBuilder(b.alloc, arrow.BinaryTypes.Binary)
	defer bldr.Release()

	bldr.Append(data)

	arr := bldr.NewArray()
	defer arr.Release()

	return array.NewRecord(b.schema, []arrow.Array{arr}, 1)
}

// ObjectListBuilder builds record batches for object listings.
type ObjectListBuilder struct {
	alloc   memory.Allocator
	schema  *arrow.Schema
	bucket  *array.StringBuilder
	key     *array.StringBuilder
	size    *array.Int64Builder
	ctype   *array.StringBuilder
	etag    *array.StringBuilder
	version *array.StringBuilder
	created *array.TimestampBuilder
	updated *array.TimestampBuilder
	isDir   *array.BooleanBuilder
	count   int64
}

// NewObjectListBuilder creates a new object list builder.
func NewObjectListBuilder(alloc memory.Allocator) *ObjectListBuilder {
	if alloc == nil {
		alloc = memory.DefaultAllocator
	}
	schema := ObjectListSchema()
	return &ObjectListBuilder{
		alloc:   alloc,
		schema:  schema,
		bucket:  array.NewStringBuilder(alloc),
		key:     array.NewStringBuilder(alloc),
		size:    array.NewInt64Builder(alloc),
		ctype:   array.NewStringBuilder(alloc),
		etag:    array.NewStringBuilder(alloc),
		version: array.NewStringBuilder(alloc),
		created: array.NewTimestampBuilder(alloc, &arrow.TimestampType{Unit: arrow.Nanosecond}),
		updated: array.NewTimestampBuilder(alloc, &arrow.TimestampType{Unit: arrow.Nanosecond}),
		isDir:   array.NewBooleanBuilder(alloc),
	}
}

// Schema returns the schema for object listings.
func (b *ObjectListBuilder) Schema() *arrow.Schema {
	return b.schema
}

// Append adds an object to the batch.
func (b *ObjectListBuilder) Append(obj *storage.Object) {
	b.bucket.Append(obj.Bucket)
	b.key.Append(obj.Key)
	b.size.Append(obj.Size)

	if obj.ContentType != "" {
		b.ctype.Append(obj.ContentType)
	} else {
		b.ctype.AppendNull()
	}

	if obj.ETag != "" {
		b.etag.Append(obj.ETag)
	} else {
		b.etag.AppendNull()
	}

	if obj.Version != "" {
		b.version.Append(obj.Version)
	} else {
		b.version.AppendNull()
	}

	if !obj.Created.IsZero() {
		b.created.Append(arrow.Timestamp(obj.Created.UnixNano()))
	} else {
		b.created.AppendNull()
	}

	if !obj.Updated.IsZero() {
		b.updated.Append(arrow.Timestamp(obj.Updated.UnixNano()))
	} else {
		b.updated.AppendNull()
	}

	b.isDir.Append(obj.IsDir)
	b.count++
}

// Build creates a record batch from the appended objects.
func (b *ObjectListBuilder) Build() arrow.Record {
	bucketArr := b.bucket.NewArray()
	keyArr := b.key.NewArray()
	sizeArr := b.size.NewArray()
	ctypeArr := b.ctype.NewArray()
	etagArr := b.etag.NewArray()
	versionArr := b.version.NewArray()
	createdArr := b.created.NewArray()
	updatedArr := b.updated.NewArray()
	isDirArr := b.isDir.NewArray()

	defer func() {
		bucketArr.Release()
		keyArr.Release()
		sizeArr.Release()
		ctypeArr.Release()
		etagArr.Release()
		versionArr.Release()
		createdArr.Release()
		updatedArr.Release()
		isDirArr.Release()
	}()

	count := b.count
	b.count = 0

	return array.NewRecord(b.schema, []arrow.Array{
		bucketArr, keyArr, sizeArr, ctypeArr, etagArr, versionArr, createdArr, updatedArr, isDirArr,
	}, count)
}

// Release releases resources.
func (b *ObjectListBuilder) Release() {
	b.bucket.Release()
	b.key.Release()
	b.size.Release()
	b.ctype.Release()
	b.etag.Release()
	b.version.Release()
	b.created.Release()
	b.updated.Release()
	b.isDir.Release()
}

// BucketListBuilder builds record batches for bucket listings.
type BucketListBuilder struct {
	alloc     memory.Allocator
	schema    *arrow.Schema
	name      *array.StringBuilder
	createdAt *array.TimestampBuilder
	public    *array.BooleanBuilder
	count     int64
}

// NewBucketListBuilder creates a new bucket list builder.
func NewBucketListBuilder(alloc memory.Allocator) *BucketListBuilder {
	if alloc == nil {
		alloc = memory.DefaultAllocator
	}
	schema := BucketListSchema()
	return &BucketListBuilder{
		alloc:     alloc,
		schema:    schema,
		name:      array.NewStringBuilder(alloc),
		createdAt: array.NewTimestampBuilder(alloc, &arrow.TimestampType{Unit: arrow.Nanosecond}),
		public:    array.NewBooleanBuilder(alloc),
	}
}

// Schema returns the schema for bucket listings.
func (b *BucketListBuilder) Schema() *arrow.Schema {
	return b.schema
}

// Append adds a bucket to the batch.
func (b *BucketListBuilder) Append(info *storage.BucketInfo) {
	b.name.Append(info.Name)

	if !info.CreatedAt.IsZero() {
		b.createdAt.Append(arrow.Timestamp(info.CreatedAt.UnixNano()))
	} else {
		b.createdAt.AppendNull()
	}

	b.public.Append(info.Public)
	b.count++
}

// Build creates a record batch from the appended buckets.
func (b *BucketListBuilder) Build() arrow.Record {
	nameArr := b.name.NewArray()
	createdAtArr := b.createdAt.NewArray()
	publicArr := b.public.NewArray()

	defer func() {
		nameArr.Release()
		createdAtArr.Release()
		publicArr.Release()
	}()

	count := b.count
	b.count = 0

	return array.NewRecord(b.schema, []arrow.Array{
		nameArr, createdAtArr, publicArr,
	}, count)
}

// Release releases resources.
func (b *BucketListBuilder) Release() {
	b.name.Release()
	b.createdAt.Release()
	b.public.Release()
}

// PartListBuilder builds record batches for part listings.
type PartListBuilder struct {
	alloc        memory.Allocator
	schema       *arrow.Schema
	number       *array.Int32Builder
	size         *array.Int64Builder
	etag         *array.StringBuilder
	lastModified *array.TimestampBuilder
	count        int64
}

// NewPartListBuilder creates a new part list builder.
func NewPartListBuilder(alloc memory.Allocator) *PartListBuilder {
	if alloc == nil {
		alloc = memory.DefaultAllocator
	}
	schema := PartListSchema()
	return &PartListBuilder{
		alloc:        alloc,
		schema:       schema,
		number:       array.NewInt32Builder(alloc),
		size:         array.NewInt64Builder(alloc),
		etag:         array.NewStringBuilder(alloc),
		lastModified: array.NewTimestampBuilder(alloc, &arrow.TimestampType{Unit: arrow.Nanosecond}),
	}
}

// Schema returns the schema for part listings.
func (b *PartListBuilder) Schema() *arrow.Schema {
	return b.schema
}

// Append adds a part to the batch.
func (b *PartListBuilder) Append(part *storage.PartInfo) {
	b.number.Append(int32(part.Number))
	b.size.Append(part.Size)

	if part.ETag != "" {
		b.etag.Append(part.ETag)
	} else {
		b.etag.AppendNull()
	}

	if part.LastModified != nil {
		b.lastModified.Append(arrow.Timestamp(part.LastModified.UnixNano()))
	} else {
		b.lastModified.AppendNull()
	}
	b.count++
}

// Build creates a record batch from the appended parts.
func (b *PartListBuilder) Build() arrow.Record {
	numberArr := b.number.NewArray()
	sizeArr := b.size.NewArray()
	etagArr := b.etag.NewArray()
	lastModifiedArr := b.lastModified.NewArray()

	defer func() {
		numberArr.Release()
		sizeArr.Release()
		etagArr.Release()
		lastModifiedArr.Release()
	}()

	count := b.count
	b.count = 0

	return array.NewRecord(b.schema, []arrow.Array{
		numberArr, sizeArr, etagArr, lastModifiedArr,
	}, count)
}

// Release releases resources.
func (b *PartListBuilder) Release() {
	b.number.Release()
	b.size.Release()
	b.etag.Release()
	b.lastModified.Release()
}

// ObjectFromRecord extracts an object from a record batch row.
func ObjectFromRecord(rec arrow.Record, row int) *storage.Object {
	if row < 0 || int64(row) >= rec.NumRows() {
		return nil
	}

	obj := &storage.Object{}

	if col := rec.Column(rec.Schema().FieldIndices("bucket")[0]); col != nil {
		if arr, ok := col.(*array.String); ok && !arr.IsNull(row) {
			obj.Bucket = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("key")[0]); col != nil {
		if arr, ok := col.(*array.String); ok && !arr.IsNull(row) {
			obj.Key = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("size")[0]); col != nil {
		if arr, ok := col.(*array.Int64); ok && !arr.IsNull(row) {
			obj.Size = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("content_type")[0]); col != nil {
		if arr, ok := col.(*array.String); ok && !arr.IsNull(row) {
			obj.ContentType = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("etag")[0]); col != nil {
		if arr, ok := col.(*array.String); ok && !arr.IsNull(row) {
			obj.ETag = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("version")[0]); col != nil {
		if arr, ok := col.(*array.String); ok && !arr.IsNull(row) {
			obj.Version = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("created")[0]); col != nil {
		if arr, ok := col.(*array.Timestamp); ok && !arr.IsNull(row) {
			obj.Created = time.Unix(0, int64(arr.Value(row)))
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("updated")[0]); col != nil {
		if arr, ok := col.(*array.Timestamp); ok && !arr.IsNull(row) {
			obj.Updated = time.Unix(0, int64(arr.Value(row)))
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("is_dir")[0]); col != nil {
		if arr, ok := col.(*array.Boolean); ok && !arr.IsNull(row) {
			obj.IsDir = arr.Value(row)
		}
	}

	return obj
}

// BucketInfoFromRecord extracts a bucket info from a record batch row.
func BucketInfoFromRecord(rec arrow.Record, row int) *storage.BucketInfo {
	if row < 0 || int64(row) >= rec.NumRows() {
		return nil
	}

	info := &storage.BucketInfo{}

	if col := rec.Column(rec.Schema().FieldIndices("name")[0]); col != nil {
		if arr, ok := col.(*array.String); ok && !arr.IsNull(row) {
			info.Name = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("created_at")[0]); col != nil {
		if arr, ok := col.(*array.Timestamp); ok && !arr.IsNull(row) {
			info.CreatedAt = time.Unix(0, int64(arr.Value(row)))
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("public")[0]); col != nil {
		if arr, ok := col.(*array.Boolean); ok && !arr.IsNull(row) {
			info.Public = arr.Value(row)
		}
	}

	return info
}

// PartInfoFromRecord extracts a part info from a record batch row.
func PartInfoFromRecord(rec arrow.Record, row int) *storage.PartInfo {
	if row < 0 || int64(row) >= rec.NumRows() {
		return nil
	}

	part := &storage.PartInfo{}

	if col := rec.Column(rec.Schema().FieldIndices("number")[0]); col != nil {
		if arr, ok := col.(*array.Int32); ok && !arr.IsNull(row) {
			part.Number = int(arr.Value(row))
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("size")[0]); col != nil {
		if arr, ok := col.(*array.Int64); ok && !arr.IsNull(row) {
			part.Size = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("etag")[0]); col != nil {
		if arr, ok := col.(*array.String); ok && !arr.IsNull(row) {
			part.ETag = arr.Value(row)
		}
	}

	if col := rec.Column(rec.Schema().FieldIndices("last_modified")[0]); col != nil {
		if arr, ok := col.(*array.Timestamp); ok && !arr.IsNull(row) {
			t := time.Unix(0, int64(arr.Value(row)))
			part.LastModified = &t
		}
	}

	return part
}
