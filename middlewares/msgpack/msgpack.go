// Package msgpack provides MessagePack serialization middleware for Mizu.
// This is a lightweight implementation without external dependencies.
package msgpack

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strings"

	"github.com/go-mizu/mizu"
)

// ContentType is the MIME type for MessagePack.
const ContentType = "application/msgpack"

// Options configures the msgpack middleware.
type Options struct {
	// ContentTypes are content types to recognize as MessagePack.
	// Default: ["application/msgpack", "application/x-msgpack"].
	ContentTypes []string
}

// contextKey is a private type for context keys.
type contextKey struct{}

// bodyKey stores the raw msgpack body.
var bodyKey = contextKey{}

// New creates msgpack middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates msgpack middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if len(opts.ContentTypes) == 0 {
		opts.ContentTypes = []string{"application/msgpack", "application/x-msgpack"}
	}

	contentTypes := make(map[string]bool)
	for _, ct := range opts.ContentTypes {
		contentTypes[ct] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			contentType := c.Request().Header.Get("Content-Type")

			// Check if content type matches msgpack
			for ct := range contentTypes {
				if strings.Contains(contentType, ct) {
					body, err := io.ReadAll(c.Request().Body)
					_ = c.Request().Body.Close()
					if err == nil {
						ctx := context.WithValue(c.Context(), bodyKey, body)
						req := c.Request().WithContext(ctx)
						*c.Request() = *req
						c.Request().Body = io.NopCloser(bytes.NewReader(body))
					}
					break
				}
			}

			return next(c)
		}
	}
}

// Body returns the raw msgpack body.
func Body(c *mizu.Ctx) []byte {
	if body, ok := c.Context().Value(bodyKey).([]byte); ok {
		return body
	}
	return nil
}

// Errors
var (
	ErrUnsupportedType = errors.New("msgpack: unsupported type")
	ErrInvalidFormat   = errors.New("msgpack: invalid format")
	ErrBufferTooSmall  = errors.New("msgpack: buffer too small")
)

// Encoder encodes values to MessagePack format.
type Encoder struct {
	buf *bytes.Buffer
}

// NewEncoder creates a new encoder.
func NewEncoder() *Encoder {
	return &Encoder{buf: &bytes.Buffer{}}
}

// Bytes returns the encoded bytes.
func (e *Encoder) Bytes() []byte {
	return e.buf.Bytes()
}

// Encode encodes a value.
func (e *Encoder) Encode(v any) error {
	return e.encodeValue(v)
}

//nolint:cyclop // Type switch for all MessagePack types is inherently complex
func (e *Encoder) encodeValue(v any) error {
	switch val := v.(type) {
	case nil:
		e.buf.WriteByte(0xc0) // nil
	case bool:
		if val {
			e.buf.WriteByte(0xc3) // true
		} else {
			e.buf.WriteByte(0xc2) // false
		}
	case int:
		return e.encodeInt(int64(val))
	case int8:
		return e.encodeInt(int64(val))
	case int16:
		return e.encodeInt(int64(val))
	case int32:
		return e.encodeInt(int64(val))
	case int64:
		return e.encodeInt(val)
	case uint:
		return e.encodeUint(uint64(val))
	case uint8:
		return e.encodeUint(uint64(val))
	case uint16:
		return e.encodeUint(uint64(val))
	case uint32:
		return e.encodeUint(uint64(val))
	case uint64:
		return e.encodeUint(val)
	case float32:
		e.buf.WriteByte(0xca) // float 32
		_ = binary.Write(e.buf, binary.BigEndian, val)
	case float64:
		e.buf.WriteByte(0xcb) // float 64
		_ = binary.Write(e.buf, binary.BigEndian, val)
	case string:
		return e.encodeString(val)
	case []byte:
		return e.encodeBinary(val)
	case []any:
		return e.encodeArray(val)
	case map[string]any:
		return e.encodeMap(val)
	default:
		return ErrUnsupportedType
	}
	return nil
}

//nolint:cyclop,gosec // G115: MessagePack integer encoding requires type conversions within checked ranges
func (e *Encoder) encodeInt(v int64) error {
	switch {
	case v >= 0 && v <= 127:
		e.buf.WriteByte(byte(v)) // positive fixint
	case v >= -32 && v < 0:
		e.buf.WriteByte(byte(v)) // negative fixint
	case v >= -128 && v <= 127:
		e.buf.WriteByte(0xd0) // int 8
		e.buf.WriteByte(byte(v))
	case v >= -32768 && v <= 32767:
		e.buf.WriteByte(0xd1) // int 16
		_ = binary.Write(e.buf, binary.BigEndian, int16(v))
	case v >= -2147483648 && v <= 2147483647:
		e.buf.WriteByte(0xd2) // int 32
		_ = binary.Write(e.buf, binary.BigEndian, int32(v))
	default:
		e.buf.WriteByte(0xd3) // int 64
		_ = binary.Write(e.buf, binary.BigEndian, v)
	}
	return nil
}

func (e *Encoder) encodeUint(v uint64) error {
	switch {
	case v <= 127:
		e.buf.WriteByte(byte(v)) // positive fixint
	case v <= 255:
		e.buf.WriteByte(0xcc) // uint 8
		e.buf.WriteByte(byte(v))
	case v <= 65535:
		e.buf.WriteByte(0xcd) // uint 16
		_ = binary.Write(e.buf, binary.BigEndian, uint16(v))
	case v <= 4294967295:
		e.buf.WriteByte(0xce) // uint 32
		_ = binary.Write(e.buf, binary.BigEndian, uint32(v))
	default:
		e.buf.WriteByte(0xcf) // uint 64
		_ = binary.Write(e.buf, binary.BigEndian, v)
	}
	return nil
}

//nolint:gosec // G115: Length conversions are within valid ranges due to switch guards
func (e *Encoder) encodeString(s string) error {
	l := len(s)
	switch {
	case l <= 31:
		e.buf.WriteByte(0xa0 | byte(l)) // fixstr
	case l <= 255:
		e.buf.WriteByte(0xd9) // str 8
		e.buf.WriteByte(byte(l))
	case l <= 65535:
		e.buf.WriteByte(0xda) // str 16
		_ = binary.Write(e.buf, binary.BigEndian, uint16(l))
	default:
		e.buf.WriteByte(0xdb) // str 32
		_ = binary.Write(e.buf, binary.BigEndian, uint32(l))
	}
	e.buf.WriteString(s)
	return nil
}

//nolint:gosec // G115: Length conversions are within valid ranges due to switch guards
func (e *Encoder) encodeBinary(b []byte) error {
	l := len(b)
	switch {
	case l <= 255:
		e.buf.WriteByte(0xc4) // bin 8
		e.buf.WriteByte(byte(l))
	case l <= 65535:
		e.buf.WriteByte(0xc5) // bin 16
		_ = binary.Write(e.buf, binary.BigEndian, uint16(l))
	default:
		e.buf.WriteByte(0xc6) // bin 32
		_ = binary.Write(e.buf, binary.BigEndian, uint32(l))
	}
	e.buf.Write(b)
	return nil
}

//nolint:gosec // G115: Length conversions are within valid ranges due to switch guards
func (e *Encoder) encodeArray(arr []any) error {
	l := len(arr)
	switch {
	case l <= 15:
		e.buf.WriteByte(0x90 | byte(l)) // fixarray
	case l <= 65535:
		e.buf.WriteByte(0xdc) // array 16
		_ = binary.Write(e.buf, binary.BigEndian, uint16(l))
	default:
		e.buf.WriteByte(0xdd) // array 32
		_ = binary.Write(e.buf, binary.BigEndian, uint32(l))
	}
	for _, v := range arr {
		if err := e.encodeValue(v); err != nil {
			return err
		}
	}
	return nil
}

//nolint:gosec // G115: Length conversions are within valid ranges due to switch guards
func (e *Encoder) encodeMap(m map[string]any) error {
	l := len(m)
	switch {
	case l <= 15:
		e.buf.WriteByte(0x80 | byte(l)) // fixmap
	case l <= 65535:
		e.buf.WriteByte(0xde) // map 16
		_ = binary.Write(e.buf, binary.BigEndian, uint16(l))
	default:
		e.buf.WriteByte(0xdf) // map 32
		_ = binary.Write(e.buf, binary.BigEndian, uint32(l))
	}
	for k, v := range m {
		if err := e.encodeString(k); err != nil {
			return err
		}
		if err := e.encodeValue(v); err != nil {
			return err
		}
	}
	return nil
}

// Decoder decodes MessagePack data.
type Decoder struct {
	data []byte
	pos  int
}

// NewDecoder creates a new decoder.
func NewDecoder(data []byte) *Decoder {
	return &Decoder{data: data}
}

// Decode decodes the next value.
func (d *Decoder) Decode() (any, error) {
	if d.pos >= len(d.data) {
		return nil, io.EOF
	}
	return d.decodeValue()
}

//nolint:cyclop,gosec // G115: MessagePack decoder uses type switches and safe conversions
func (d *Decoder) decodeValue() (any, error) {
	if d.pos >= len(d.data) {
		return nil, ErrBufferTooSmall
	}

	b := d.data[d.pos]
	d.pos++

	switch {
	case b <= 0x7f:
		return int64(b), nil // positive fixint
	case b >= 0xe0:
		return int64(int8(b)), nil // negative fixint
	case b >= 0xa0 && b <= 0xbf:
		return d.readString(int(b & 0x1f))
	case b >= 0x90 && b <= 0x9f:
		return d.readArray(int(b & 0x0f))
	case b >= 0x80 && b <= 0x8f:
		return d.readMap(int(b & 0x0f))
	case b == 0xc0:
		return nil, nil // nil
	case b == 0xc2:
		return false, nil // false
	case b == 0xc3:
		return true, nil // true
	case b == 0xc4:
		return d.readBinary(d.readUint8())
	case b == 0xc5:
		return d.readBinary(d.readUint16())
	case b == 0xc6:
		return d.readBinary(d.readUint32())
	case b == 0xca:
		return d.readFloat32()
	case b == 0xcb:
		return d.readFloat64()
	case b == 0xcc:
		return uint64(d.readUint8()), nil
	case b == 0xcd:
		return uint64(d.readUint16()), nil
	case b == 0xce:
		return uint64(d.readUint32()), nil
	case b == 0xcf:
		return d.readUint64(), nil
	case b == 0xd0:
		return int64(int8(d.readUint8())), nil
	case b == 0xd1:
		return int64(int16(d.readUint16())), nil
	case b == 0xd2:
		return int64(int32(d.readUint32())), nil
	case b == 0xd3:
		return d.readInt64()
	case b == 0xd9:
		return d.readString(d.readUint8())
	case b == 0xda:
		return d.readString(d.readUint16())
	case b == 0xdb:
		return d.readString(d.readUint32())
	case b == 0xdc:
		return d.readArray(d.readUint16())
	case b == 0xdd:
		return d.readArray(d.readUint32())
	case b == 0xde:
		return d.readMap(d.readUint16())
	case b == 0xdf:
		return d.readMap(d.readUint32())
	}

	return nil, ErrInvalidFormat
}

func (d *Decoder) readUint8() int {
	if d.pos >= len(d.data) {
		return 0
	}
	v := d.data[d.pos]
	d.pos++
	return int(v)
}

func (d *Decoder) readUint16() int {
	if d.pos+2 > len(d.data) {
		return 0
	}
	v := binary.BigEndian.Uint16(d.data[d.pos:])
	d.pos += 2
	return int(v)
}

func (d *Decoder) readUint32() int {
	if d.pos+4 > len(d.data) {
		return 0
	}
	v := binary.BigEndian.Uint32(d.data[d.pos:])
	d.pos += 4
	return int(v)
}

func (d *Decoder) readUint64() uint64 {
	if d.pos+8 > len(d.data) {
		return 0
	}
	v := binary.BigEndian.Uint64(d.data[d.pos:])
	d.pos += 8
	return v
}

//nolint:gosec // G115: uint64 to int64 conversion is intentional for MessagePack protocol
func (d *Decoder) readInt64() (int64, error) {
	if d.pos+8 > len(d.data) {
		return 0, ErrBufferTooSmall
	}
	v := binary.BigEndian.Uint64(d.data[d.pos:])
	d.pos += 8
	return int64(v), nil
}

func (d *Decoder) readFloat32() (float32, error) {
	if d.pos+4 > len(d.data) {
		return 0, ErrBufferTooSmall
	}
	bits := binary.BigEndian.Uint32(d.data[d.pos:])
	d.pos += 4
	return math.Float32frombits(bits), nil
}

func (d *Decoder) readFloat64() (float64, error) {
	if d.pos+8 > len(d.data) {
		return 0, ErrBufferTooSmall
	}
	bits := binary.BigEndian.Uint64(d.data[d.pos:])
	d.pos += 8
	return math.Float64frombits(bits), nil
}

func (d *Decoder) readString(length int) (string, error) {
	if d.pos+length > len(d.data) {
		return "", ErrBufferTooSmall
	}
	s := string(d.data[d.pos : d.pos+length])
	d.pos += length
	return s, nil
}

func (d *Decoder) readBinary(length int) ([]byte, error) {
	if d.pos+length > len(d.data) {
		return nil, ErrBufferTooSmall
	}
	b := make([]byte, length)
	copy(b, d.data[d.pos:d.pos+length])
	d.pos += length
	return b, nil
}

func (d *Decoder) readArray(length int) ([]any, error) {
	arr := make([]any, length)
	for i := 0; i < length; i++ {
		v, err := d.decodeValue()
		if err != nil {
			return nil, err
		}
		arr[i] = v
	}
	return arr, nil
}

func (d *Decoder) readMap(length int) (map[string]any, error) {
	m := make(map[string]any, length)
	for i := 0; i < length; i++ {
		k, err := d.decodeValue()
		if err != nil {
			return nil, err
		}
		v, err := d.decodeValue()
		if err != nil {
			return nil, err
		}
		if key, ok := k.(string); ok {
			m[key] = v
		}
	}
	return m, nil
}

// Marshal encodes a value to MessagePack.
func Marshal(v any) ([]byte, error) {
	enc := NewEncoder()
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return enc.Bytes(), nil
}

// Unmarshal decodes MessagePack data.
func Unmarshal(data []byte) (any, error) {
	dec := NewDecoder(data)
	return dec.Decode()
}

// Response sends a MessagePack response.
func Response(c *mizu.Ctx, status int, v any) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}

	c.Header().Set("Content-Type", ContentType)
	c.Writer().WriteHeader(status)
	_, err = c.Writer().Write(data)
	return err
}

// Bind decodes a MessagePack request body.
func Bind(c *mizu.Ctx) (any, error) {
	body := Body(c)
	if body != nil {
		return Unmarshal(body)
	}

	data, err := io.ReadAll(c.Request().Body)
	_ = c.Request().Body.Close()
	if err != nil {
		return nil, err
	}
	c.Request().Body = io.NopCloser(bytes.NewReader(data))
	return Unmarshal(data)
}
