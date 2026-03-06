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
// 注意：为了计算 Checksum 并流式写入，建议先写入临时 Buffer 或文件。
func (c *Codec) WritePool(w io.Writer, tags []string, entries []Entry) error {
	// 1. 计算 Checksum (对除了 Header 以外的所有内容)
	h := sha256.New()
	mw := io.MultiWriter(w, h)

	// 我们需要先写 Header，但 Header 包含 Checksum。
	// 方案：先写一个占位 Header，最后 Seek 回去改；或者先 Buffer。
	// 考虑到流式要求，我们假设 w 是可 Seek 的，或者我们先写到一个内存 Buffer。
	// 这里我们定义一个简单的协议：Header 之后是 Dictionary，然后是 Payload。

	header := Header{
		Version:    Version,
		EntryCount: uint32(len(entries)),
	}
	copy(header.Magic[:], Magic)

	// 如果 w 不支持 Seek，这里会报错。但在实际 Service 中，我们会用 temp file。
	if err := binary.Write(w, binary.LittleEndian, header); err != nil {
		return err
	}

	// 开始计算 Checksum 的部分
	// 写字典
	if err := binary.Write(mw, binary.LittleEndian, uint32(len(tags))); err != nil {
		return err
	}
	for _, t := range tags {
		if err := binary.Write(mw, binary.LittleEndian, uint16(len(t))); err != nil {
			return err
		}
		if _, err := mw.Write([]byte(t)); err != nil {
			return err
		}
	}

	// 写 Payload
	for _, e := range entries {
		addr := e.Prefix.Addr()
		family := uint8(4)
		if addr.Is6() {
			family = 6
		}
		if err := binary.Write(mw, binary.LittleEndian, family); err != nil {
			return err
		}
		if family == 4 {
			ip4 := addr.As4()
			if _, err := mw.Write(ip4[:]); err != nil {
				return err
			}
		} else {
			ip16 := addr.As16()
			if _, err := mw.Write(ip16[:]); err != nil {
				return err
			}
		}
		if err := binary.Write(mw, binary.LittleEndian, uint8(e.Prefix.Bits())); err != nil {
			return err
		}
		if err := binary.Write(mw, binary.LittleEndian, uint16(len(e.TagIndices))); err != nil {
			return err
		}
		for _, idx := range e.TagIndices {
			if err := binary.Write(mw, binary.LittleEndian, idx); err != nil {
				return err
			}
		}
	}

	// 计算最终 Checksum 并更新 Header
	checksum := h.Sum(nil)
	copy(header.Checksum[:], checksum)

	// 如果支持 Seek，回到开头重写 Header
	if seeker, ok := w.(io.WriteSeeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := binary.Write(seeker, binary.LittleEndian, header); err != nil {
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
	var header Header
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	if string(header.Magic[:]) != Magic {
		return nil, ErrInvalidMagic
	}
	if header.Version != Version {
		return nil, ErrInvalidVersion
	}

	// 读取字典
	var dictCount uint32
	if err := binary.Read(r, binary.LittleEndian, &dictCount); err != nil {
		return nil, err
	}
	dict := make([]string, dictCount)
	for i := uint32(0); i < dictCount; i++ {
		var l uint16
		if err := binary.Read(r, binary.LittleEndian, &l); err != nil {
			return nil, err
		}
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
	var family uint8
	if err := binary.Read(r.r, binary.LittleEndian, &family); err != nil {
		return netip.Prefix{}, nil, err
	}
	var ipBytes []byte
	if family == 4 {
		ipBytes = make([]byte, 4)
	} else if family == 6 {
		ipBytes = make([]byte, 16)
	} else {
		return netip.Prefix{}, nil, ErrCorruptedData
	}

	if _, err := io.ReadFull(r.r, ipBytes); err != nil {
		return netip.Prefix{}, nil, err
	}
	var mask uint8
	if err := binary.Read(r.r, binary.LittleEndian, &mask); err != nil {
		return netip.Prefix{}, nil, err
	}

	addr, ok := netip.AddrFromSlice(ipBytes)
	if !ok {
		return netip.Prefix{}, nil, ErrCorruptedData
	}
	prefix := netip.PrefixFrom(addr, int(mask))

	var tagCount uint16
	if err := binary.Read(r.r, binary.LittleEndian, &tagCount); err != nil {
		return netip.Prefix{}, nil, err
	}
	tags := make([]string, tagCount)
	for i := uint16(0); i < tagCount; i++ {
		var idx uint32
		if err := binary.Read(r.r, binary.LittleEndian, &idx); err != nil {
			return netip.Prefix{}, nil, err
		}
		if idx >= uint32(len(r.dictionary)) {
			return netip.Prefix{}, nil, ErrCorruptedData
		}
		tags[i] = r.dictionary[idx]
	}

	return prefix, tags, nil
}

func (r *Reader) NextIndices() (netip.Prefix, []uint32, error) {
	var family uint8
	if err := binary.Read(r.r, binary.LittleEndian, &family); err != nil {
		return netip.Prefix{}, nil, err
	}
	var ipBytes []byte
	if family == 4 {
		ipBytes = make([]byte, 4)
	} else if family == 6 {
		ipBytes = make([]byte, 16)
	} else {
		return netip.Prefix{}, nil, ErrCorruptedData
	}

	if _, err := io.ReadFull(r.r, ipBytes); err != nil {
		return netip.Prefix{}, nil, err
	}
	var mask uint8
	if err := binary.Read(r.r, binary.LittleEndian, &mask); err != nil {
		return netip.Prefix{}, nil, err
	}

	addr, ok := netip.AddrFromSlice(ipBytes)
	if !ok {
		return netip.Prefix{}, nil, ErrCorruptedData
	}
	prefix := netip.PrefixFrom(addr, int(mask))

	var tagCount uint16
	if err := binary.Read(r.r, binary.LittleEndian, &tagCount); err != nil {
		return netip.Prefix{}, nil, err
	}
	indices := make([]uint32, tagCount)
	for i := uint16(0); i < tagCount; i++ {
		var idx uint32
		if err := binary.Read(r.r, binary.LittleEndian, &idx); err != nil {
			return netip.Prefix{}, nil, err
		}
		indices[i] = idx
	}

	return prefix, indices, nil
}
