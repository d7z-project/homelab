package site

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"homelab/pkg/models"
)

const (
	Magic   = "SITE"
	Version = 1
)

var (
	ErrInvalidMagic   = errors.New("invalid magic number")
	ErrInvalidVersion = errors.New("invalid version")
	ErrCorruptedData  = errors.New("corrupted data")
)

type Header struct {
	Magic      [4]byte
	Version    uint8
	EntryCount uint32
	Checksum   [32]byte
	DictOffset uint64
}

type Entry struct {
	Type       uint8
	Value      string
	TagIndices []uint32
}

type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

func (c *Codec) WritePool(w io.Writer, tags []string, entries []Entry) error {
	h := sha256.New()
	mw := io.MultiWriter(w, h)

	header := Header{
		Version:    Version,
		EntryCount: uint32(len(entries)),
	}
	copy(header.Magic[:], Magic)

	if err := binary.Write(w, binary.LittleEndian, header); err != nil {
		return err
	}

	// 1. Write Dictionary
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

	// 2. Write Payload
	for _, e := range entries {
		if err := binary.Write(mw, binary.LittleEndian, e.Type); err != nil {
			return err
		}
		if err := binary.Write(mw, binary.LittleEndian, uint16(len(e.Value))); err != nil {
			return err
		}
		if _, err := mw.Write([]byte(e.Value)); err != nil {
			return err
		}
		if err := binary.Write(mw, binary.LittleEndian, uint8(len(e.TagIndices))); err != nil {
			return err
		}
		for _, idx := range e.TagIndices {
			if err := binary.Write(mw, binary.LittleEndian, idx); err != nil {
				return err
			}
		}
	}

	checksum := h.Sum(nil)
	copy(header.Checksum[:], checksum)

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

func (r *Reader) Next() (models.SitePoolEntry, error) {
	var ruleType uint8
	if err := binary.Read(r.r, binary.LittleEndian, &ruleType); err != nil {
		return models.SitePoolEntry{}, err
	}

	var valLen uint16
	if err := binary.Read(r.r, binary.LittleEndian, &valLen); err != nil {
		return models.SitePoolEntry{}, err
	}
	valBuf := make([]byte, valLen)
	if _, err := io.ReadFull(r.r, valBuf); err != nil {
		return models.SitePoolEntry{}, err
	}

	var tagCount uint8
	if err := binary.Read(r.r, binary.LittleEndian, &tagCount); err != nil {
		return models.SitePoolEntry{}, err
	}
	tags := make([]string, tagCount)
	for i := uint8(0); i < tagCount; i++ {
		var idx uint32
		if err := binary.Read(r.r, binary.LittleEndian, &idx); err != nil {
			return models.SitePoolEntry{}, err
		}
		if idx >= uint32(len(r.dictionary)) {
			return models.SitePoolEntry{}, ErrCorruptedData
		}
		tags[i] = r.dictionary[idx]
	}

	return models.SitePoolEntry{
		Type:  ruleType,
		Value: string(valBuf),
		Tags:  tags,
	}, nil
}
