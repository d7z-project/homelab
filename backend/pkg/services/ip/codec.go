package ip

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"net/netip"
)

const (
	Magic   = "IPPL"
	Version = 1
)

var (
	ErrInvalidMagic   = errors.New("invalid magic number")
	ErrInvalidVersion = errors.New("invalid version")
	ErrCorruptedData  = errors.New("corrupted data")
)

// Header 存储二进制文件的头部信息
type Header struct {
	Magic      [4]byte
	Version    uint8
	EntryCount uint32
	Checksum   [32]byte
	DictOffset uint64
}

// Entry 代表二进制文件中的一条 IP/CIDR 记录
type Entry struct {
	Prefix     netip.Prefix
	TagIndices []uint32
}

// Codec 提供二进制编解码功能
type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

// WritePool 将数据流式写入 Writer。
func (c *Codec) WritePool(w io.Writer, tags []string, entries []Entry) error {
	h := sha256.New()
	mw := io.MultiWriter(w, h)

	header := Header{
		Version:    Version,
		EntryCount: uint32(len(entries)),
	}
	copy(header.Magic[:], Magic)

	// 先写占位 Header
	// Header 结构体：Magic(4), Version(1), EntryCount(4), Checksum(32), DictOffset(8)
	// 注意 Struct Padding。我们使用手动写入以确保紧凑。

	writeHeader := func(writer io.Writer, hdr Header) error {
		buf := make([]byte, 4+1+4+32+8)
		copy(buf[0:4], hdr.Magic[:])
		buf[4] = hdr.Version
		binary.LittleEndian.PutUint32(buf[5:9], hdr.EntryCount)
		copy(buf[9:41], hdr.Checksum[:])
		binary.LittleEndian.PutUint64(buf[41:49], hdr.DictOffset)
		_, err := writer.Write(buf)
		return err
	}

	if err := writeHeader(w, header); err != nil {
		return err
	}

	// 写字典
	dictCountBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(dictCountBuf, uint32(len(tags)))
	if _, err := mw.Write(dictCountBuf); err != nil {
		return err
	}

	for _, t := range tags {
		lenBuf := make([]byte, 2)
		binary.LittleEndian.PutUint16(lenBuf, uint16(len(t)))
		if _, err := mw.Write(lenBuf); err != nil {
			return err
		}
		if _, err := mw.Write([]byte(t)); err != nil {
			return err
		}
	}

	// 写 Payload
	buf := make([]byte, 1+16+1+2) // family(1) + ip(16) + mask(1) + tagCount(2)
	for _, e := range entries {
		addr := e.Prefix.Addr()
		family := uint8(4)
		ipLen := 4
		if addr.Is6() {
			family = 6
			ipLen = 16
		}

		buf[0] = family
		asSlice := addr.AsSlice()
		copy(buf[1:1+ipLen], asSlice)
		buf[1+ipLen] = uint8(e.Prefix.Bits())
		binary.LittleEndian.PutUint16(buf[2+ipLen:4+ipLen], uint16(len(e.TagIndices)))

		if _, err := mw.Write(buf[:4+ipLen]); err != nil {
			return err
		}

		// 写 Tag Indices
		idxBuf := make([]byte, 4)
		for _, idx := range e.TagIndices {
			binary.LittleEndian.PutUint32(idxBuf, idx)
			if _, err := mw.Write(idxBuf); err != nil {
				return err
			}
		}
	}

	// 计算最终 Checksum 并更新 Header
	checksum := h.Sum(nil)
	copy(header.Checksum[:], checksum)

	if seeker, ok := w.(io.WriteSeeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := writeHeader(seeker, header); err != nil {
			return err
		}
	}

	return nil
}

// Reader 提供流式读取和标签解析
type Reader struct {
	r          io.Reader
	header     Header
	dictionary []string
}

func NewReader(r io.Reader) (*Reader, error) {
	headerBuf := make([]byte, 4+1+4+32+8)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return nil, err
	}

	var header Header
	copy(header.Magic[:], headerBuf[0:4])
	header.Version = headerBuf[4]
	header.EntryCount = binary.LittleEndian.Uint32(headerBuf[5:9])
	copy(header.Checksum[:], headerBuf[9:41])
	header.DictOffset = binary.LittleEndian.Uint64(headerBuf[41:49])

	if string(header.Magic[:]) != Magic {
		return nil, ErrInvalidMagic
	}
	if header.Version != Version {
		return nil, ErrInvalidVersion
	}

	// 读取字典
	dictCountBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, dictCountBuf); err != nil {
		return nil, err
	}
	dictCount := binary.LittleEndian.Uint32(dictCountBuf)

	dict := make([]string, dictCount)
	for i := uint32(0); i < dictCount; i++ {
		lenBuf := make([]byte, 2)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return nil, err
		}
		l := binary.LittleEndian.Uint16(lenBuf)
		buf := make([]byte, l)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		dict[i] = string(buf)
	}

	return &Reader{
		r:          r,
		header:     header,
		dictionary: dict,
	}, nil
}

func (r *Reader) EntryCount() uint32 {
	return r.header.EntryCount
}

func (r *Reader) Tags() []string {
	return r.dictionary
}

func (r *Reader) Next() (netip.Prefix, []string, error) {
	prefix, indices, err := r.NextIndices()
	if err != nil {
		return netip.Prefix{}, nil, err
	}
	tags := make([]string, len(indices))
	for i, idx := range indices {
		if idx >= uint32(len(r.dictionary)) {
			return netip.Prefix{}, nil, ErrCorruptedData
		}
		tags[i] = r.dictionary[idx]
	}
	return prefix, tags, nil
}

func (r *Reader) NextIndices() (netip.Prefix, []uint32, error) {
	familyBuf := make([]byte, 1)
	if _, err := io.ReadFull(r.r, familyBuf); err != nil {
		return netip.Prefix{}, nil, err
	}
	family := familyBuf[0]

	var ipLen int
	if family == 4 {
		ipLen = 4
	} else if family == 6 {
		ipLen = 16
	} else {
		return netip.Prefix{}, nil, ErrCorruptedData
	}

	ipBytes := make([]byte, ipLen)
	if _, err := io.ReadFull(r.r, ipBytes); err != nil {
		return netip.Prefix{}, nil, err
	}

	maskBuf := make([]byte, 1)
	if _, err := io.ReadFull(r.r, maskBuf); err != nil {
		return netip.Prefix{}, nil, err
	}
	mask := maskBuf[0]

	tagCountBuf := make([]byte, 2)
	if _, err := io.ReadFull(r.r, tagCountBuf); err != nil {
		return netip.Prefix{}, nil, err
	}
	tagCount := binary.LittleEndian.Uint16(tagCountBuf)

	addr, ok := netip.AddrFromSlice(ipBytes)
	if !ok {
		return netip.Prefix{}, nil, ErrCorruptedData
	}
	prefix := netip.PrefixFrom(addr, int(mask))

	indices := make([]uint32, tagCount)
	idxBuf := make([]byte, 4)
	for i := uint16(0); i < tagCount; i++ {
		if _, err := io.ReadFull(r.r, idxBuf); err != nil {
			return netip.Prefix{}, nil, err
		}
		indices[i] = binary.LittleEndian.Uint32(idxBuf)
	}

	return prefix, indices, nil
}
