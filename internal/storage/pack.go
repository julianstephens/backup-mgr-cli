package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/julianstephens/warden/internal/crypto"
	"github.com/julianstephens/warden/internal/warden"
)

type BlobType int

const (
	Data BlobType = 1 << iota
	CompressedData
)

var (
	minPackSize = UncompressedHeaderDataSize + crypto.KeySize + (headerLenSize * 2)
)

type Blob struct {
	ID                 warden.ID
	Data               []byte
	Length             uint
	UncompressedLength uint
	Offset             uint
	Type               BlobType
}

type Header struct {
	Key  []byte
	Data []byte
}

type Pack struct {
	data []Blob

	numBytes uint
	key      *crypto.Key
	writer   io.Writer

	m sync.Mutex
}

func NewBlob(data []byte, dataType BlobType, uncompressedLength uint, offset uint) *Blob {
	return &Blob{
		ID:                 crypto.Hash(data),
		Data:               data,
		Length:             uint(len(data)),
		UncompressedLength: uncompressedLength,
		Offset:             offset,
		Type:               dataType,
	}
}

func NewPack(key *crypto.Key, writer io.Writer) *Pack {
	return &Pack{key: key, writer: writer}
}

func (p *Pack) Append(blob Blob) (int, error) {
	p.m.Lock()
	defer p.m.Unlock()

	n, err := p.writer.Write(blob.Data)
	blob.Length = uint(n)
	blob.Offset = p.numBytes

	p.numBytes += uint(n)
	p.data = append(p.data, blob)
	n += int(blob.Length)

	return n, err
}

func (p *Pack) Size() uint {
	p.m.Lock()
	defer p.m.Unlock()

	return p.numBytes
}

func (p *Pack) Blobs() []Blob {
	p.m.Lock()
	defer p.m.Unlock()

	return p.data
}

func (p *Pack) Finalize(sessionKey *crypto.Key) error {
	p.m.Lock()
	defer p.m.Unlock()

	headerData, err := buildHeaderData(p.data)
	if err != nil {
		return err
	}

	encHeaderData, err := crypto.Encrypt(*p.key, headerData, nil)
	if err != nil {
		return err
	}

	headerDataLen := len(encHeaderData)

	encSessionKey, err := crypto.Encrypt(*p.key, sessionKey.Data, nil)
	if err != nil {
		return err
	}

	encHeaderData = append(encHeaderData, encSessionKey[:]...)

	var sessionLen [headerLenSize]byte
	binary.LittleEndian.PutUint32(sessionLen[:], uint32(len(encSessionKey)))
	var headerLen [headerLenSize]byte
	binary.LittleEndian.PutUint32(headerLen[:], uint32(headerDataLen))

	encHeaderData = append(encHeaderData, sessionLen[:]...)
	encHeaderData = append(encHeaderData, headerLen[:]...)

	err = verifyHeader(p.key, sessionKey, encHeaderData, p.data)
	if err != nil {
		return err
	}

	n, err := p.writer.Write(encHeaderData)
	if err != nil {
		return err
	}

	if n != len(encHeaderData) {
		return fmt.Errorf("expected to write %d bytes but wrote %d", len(encHeaderData), n)
	}
	p.numBytes += uint(n)

	if p.numBytes < minPackSize {
		return errors.New("final pack is below minimum size")
	}

	return nil
}
