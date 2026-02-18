// File: lib/storage/transport/s3/response.go
package s3

import (
	"bytes"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// xmlBufferPool provides pooled buffers for XML response generation.
// This eliminates allocation overhead for common responses.
var xmlBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

// getXMLBuffer gets a buffer from the pool.
func getXMLBuffer() *bytes.Buffer {
	return xmlBufferPool.Get().(*bytes.Buffer)
}

// putXMLBuffer returns a buffer to the pool.
func putXMLBuffer(buf *bytes.Buffer) {
	buf.Reset()
	xmlBufferPool.Put(buf)
}

// writeCopyObjectResultFast writes a CopyObjectResult XML response without reflection.
// This is 5-10x faster than using encoding/xml.
func writeCopyObjectResultFast(c *mizu.Ctx, lastMod time.Time, etag string) error {
	buf := getXMLBuffer()
	defer putXMLBuffer(buf)

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<CopyObjectResult xmlns="`)
	buf.WriteString(s3XMLNS)
	buf.WriteString(`">`)
	buf.WriteString(`<LastModified>`)
	buf.WriteString(lastMod.UTC().Format(time.RFC3339Nano))
	buf.WriteString(`</LastModified>`)
	buf.WriteString(`<ETag>`)
	buf.WriteString(quoteRawETag(etag))
	buf.WriteString(`</ETag>`)
	buf.WriteString(`</CopyObjectResult>`)

	w := c.Writer()
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(buf.Bytes())
	return err
}

// writeDeleteResultFast writes a DeleteResult XML response without reflection.
func writeDeleteResultFast(c *mizu.Ctx, deleted []DeletedObject, errors []DeleteError) error {
	buf := getXMLBuffer()
	defer putXMLBuffer(buf)

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<DeleteResult xmlns="`)
	buf.WriteString(s3XMLNS)
	buf.WriteString(`">`)

	for _, d := range deleted {
		buf.WriteString(`<Deleted><Key>`)
		writeXMLEscaped(buf, d.Key)
		buf.WriteString(`</Key>`)
		if d.VersionId != "" {
			buf.WriteString(`<VersionId>`)
			writeXMLEscaped(buf, d.VersionId)
			buf.WriteString(`</VersionId>`)
		}
		buf.WriteString(`</Deleted>`)
	}

	for _, e := range errors {
		buf.WriteString(`<Error><Key>`)
		writeXMLEscaped(buf, e.Key)
		buf.WriteString(`</Key><Code>`)
		writeXMLEscaped(buf, e.Code)
		buf.WriteString(`</Code><Message>`)
		writeXMLEscaped(buf, e.Message)
		buf.WriteString(`</Message></Error>`)
	}

	buf.WriteString(`</DeleteResult>`)

	w := c.Writer()
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(buf.Bytes())
	return err
}

// writeXMLEscaped writes XML-escaped string to buffer.
func writeXMLEscaped(buf *bytes.Buffer, s string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteByte(s[i])
		}
	}
}

// DeleteError is defined in handle_bucket.go

// writeListBucketResultFast writes a ListBucketResult (ListObjectsV2) XML response
// without reflection. This is 5-10x faster than encoding/xml for large object lists.
func writeListBucketResultFast(c *mizu.Ctx, resp *ListBucketResultV2) error {
	// Estimate buffer size: ~200 bytes per entry + overhead
	buf := getXMLBuffer()
	defer putXMLBuffer(buf)
	buf.Grow(256 + len(resp.Contents)*256)

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<ListBucketResult xmlns="`)
	buf.WriteString(s3XMLNS)
	buf.WriteString(`">`)

	buf.WriteString(`<Name>`)
	writeXMLEscaped(buf, resp.Name)
	buf.WriteString(`</Name>`)

	buf.WriteString(`<Prefix>`)
	writeXMLEscaped(buf, resp.Prefix)
	buf.WriteString(`</Prefix>`)

	buf.WriteString(`<MaxKeys>`)
	writeInt(buf, resp.MaxKeys)
	buf.WriteString(`</MaxKeys>`)

	buf.WriteString(`<KeyCount>`)
	writeInt(buf, resp.KeyCount)
	buf.WriteString(`</KeyCount>`)

	if resp.IsTruncated {
		buf.WriteString(`<IsTruncated>true</IsTruncated>`)
	} else {
		buf.WriteString(`<IsTruncated>false</IsTruncated>`)
	}

	if resp.ContinuationToken != "" {
		buf.WriteString(`<ContinuationToken>`)
		writeXMLEscaped(buf, resp.ContinuationToken)
		buf.WriteString(`</ContinuationToken>`)
	}
	if resp.NextContinuationToken != "" {
		buf.WriteString(`<NextContinuationToken>`)
		writeXMLEscaped(buf, resp.NextContinuationToken)
		buf.WriteString(`</NextContinuationToken>`)
	}

	for i := range resp.Contents {
		e := &resp.Contents[i]
		buf.WriteString(`<Contents>`)
		buf.WriteString(`<Key>`)
		writeXMLEscaped(buf, e.Key)
		buf.WriteString(`</Key>`)
		buf.WriteString(`<LastModified>`)
		buf.WriteString(e.LastModified.UTC().Format(time.RFC3339Nano))
		buf.WriteString(`</LastModified>`)
		buf.WriteString(`<ETag>`)
		writeXMLEscaped(buf, e.ETag)
		buf.WriteString(`</ETag>`)
		buf.WriteString(`<Size>`)
		writeInt64(buf, e.Size)
		buf.WriteString(`</Size>`)
		buf.WriteString(`<StorageClass>`)
		writeXMLEscaped(buf, e.StorageClass)
		buf.WriteString(`</StorageClass>`)
		buf.WriteString(`</Contents>`)
	}

	buf.WriteString(`</ListBucketResult>`)

	w := c.Writer()
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(buf.Bytes())
	return err
}

// writeListBucketsResultFast writes a ListAllMyBucketsResult XML response
// without reflection.
func writeListBucketsResultFast(c *mizu.Ctx, resp *ListBucketsResult) error {
	buf := getXMLBuffer()
	defer putXMLBuffer(buf)

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<ListAllMyBucketsResult xmlns="`)
	buf.WriteString(s3XMLNS)
	buf.WriteString(`">`)
	buf.WriteString(`<Owner><ID>`)
	writeXMLEscaped(buf, resp.Owner.ID)
	buf.WriteString(`</ID><DisplayName>`)
	writeXMLEscaped(buf, resp.Owner.DisplayName)
	buf.WriteString(`</DisplayName></Owner>`)
	buf.WriteString(`<Buckets>`)
	for i := range resp.Buckets.Buckets {
		b := &resp.Buckets.Buckets[i]
		buf.WriteString(`<Bucket><Name>`)
		writeXMLEscaped(buf, b.Name)
		buf.WriteString(`</Name><CreationDate>`)
		buf.WriteString(b.CreationDate.UTC().Format(time.RFC3339Nano))
		buf.WriteString(`</CreationDate></Bucket>`)
	}
	buf.WriteString(`</Buckets>`)
	buf.WriteString(`</ListAllMyBucketsResult>`)

	w := c.Writer()
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(buf.Bytes())
	return err
}

// writeErrorFast writes an S3 Error XML response without reflection.
func writeErrorFast(c *mizu.Ctx, e *Error) error {
	buf := getXMLBuffer()
	defer putXMLBuffer(buf)

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<Error><Code>`)
	writeXMLEscaped(buf, e.Code)
	buf.WriteString(`</Code><Message>`)
	writeXMLEscaped(buf, e.Message)
	buf.WriteString(`</Message></Error>`)

	w := c.Writer()
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(e.HTTPStatus)
	_, err := w.Write(buf.Bytes())
	return err
}

// writeInt writes an integer to a buffer without allocating.
func writeInt(buf *bytes.Buffer, n int) {
	var scratch [20]byte
	b := strconv.AppendInt(scratch[:0], int64(n), 10)
	buf.Write(b)
}

// writeInt64 writes an int64 to a buffer without allocating.
func writeInt64(buf *bytes.Buffer, n int64) {
	var scratch [20]byte
	b := strconv.AppendInt(scratch[:0], n, 10)
	buf.Write(b)
}
