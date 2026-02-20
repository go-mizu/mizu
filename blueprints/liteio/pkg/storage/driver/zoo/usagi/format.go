package usagi

import (
	"encoding/binary"
	"errors"
)

const (
	recordMagic   uint32 = 0x55534147 // "USAG"
	recordVersion uint8  = 1

	recordOpPut    uint8 = 1
	recordOpDelete uint8 = 2
)

const recordHeaderSize = 36

var errCorruptRecord = errors.New("usagi: corrupt record")

type recordHeader struct {
	Magic          uint32
	Version        uint8
	Op             uint8
	Flags          uint16
	KeyLen         uint32
	ContentTypeLen uint16
	Reserved       uint16
	DataLen        uint64
	UpdatedUnixNs  int64
	Checksum       uint32
}

func encodeHeader(h recordHeader, buf []byte) []byte {
	if cap(buf) < recordHeaderSize {
		buf = make([]byte, recordHeaderSize)
	} else {
		buf = buf[:recordHeaderSize]
	}
	binary.LittleEndian.PutUint32(buf[0:4], h.Magic)
	buf[4] = h.Version
	buf[5] = h.Op
	binary.LittleEndian.PutUint16(buf[6:8], h.Flags)
	binary.LittleEndian.PutUint32(buf[8:12], h.KeyLen)
	binary.LittleEndian.PutUint16(buf[12:14], h.ContentTypeLen)
	binary.LittleEndian.PutUint16(buf[14:16], h.Reserved)
	binary.LittleEndian.PutUint64(buf[16:24], h.DataLen)
	binary.LittleEndian.PutUint64(buf[24:32], uint64(h.UpdatedUnixNs))
	binary.LittleEndian.PutUint32(buf[32:36], h.Checksum)
	return buf
}

func decodeHeader(buf []byte) (recordHeader, error) {
	if len(buf) < recordHeaderSize {
		return recordHeader{}, errCorruptRecord
	}
	h := recordHeader{}
	h.Magic = binary.LittleEndian.Uint32(buf[0:4])
	h.Version = buf[4]
	h.Op = buf[5]
	h.Flags = binary.LittleEndian.Uint16(buf[6:8])
	h.KeyLen = binary.LittleEndian.Uint32(buf[8:12])
	h.ContentTypeLen = binary.LittleEndian.Uint16(buf[12:14])
	h.Reserved = binary.LittleEndian.Uint16(buf[14:16])
	h.DataLen = binary.LittleEndian.Uint64(buf[16:24])
	h.UpdatedUnixNs = int64(binary.LittleEndian.Uint64(buf[24:32]))
	h.Checksum = binary.LittleEndian.Uint32(buf[32:36])
	return h, nil
}
