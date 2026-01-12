package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/julianstephens/warden/internal/crypto"
	"github.com/julianstephens/warden/internal/warden"
)

type headerData struct {
	Type               uint8
	Length             uint32
	UncompressedLength uint32
	ID                 warden.ID
}

const (
	// size of the header length field
	headerLenSize = 4
)

var (
	MaxHeaderSize              = (16 * warden.MiB) + headerLenSize
	UncompressedHeaderDataSize = uint(binary.Size(BlobType(0)) + headerLenSize + warden.IdSize)
	headerDataSize             = UncompressedHeaderDataSize + uint(headerLenSize)
	ErrInvalidReadLen          = errors.New("wrong number of bytes read")
)

func DecodeHeader(key *crypto.Key, sessionKey *crypto.Key, reader io.Reader, size int64) (blobs []Blob, decodedKey *crypto.Key, err error) {
	buf := make([]byte, size)
	n, err := reader.Read(buf)
	if err != nil {
		return
	}
	if n != int(size) {
		err = ErrInvalidReadLen
		return
	}

	metadata := buf[len(buf)-headerLenSize*2:]
	buf = buf[:len(buf)-headerLenSize*2]
	keySize := binary.LittleEndian.Uint32(metadata[:headerLenSize])
	headerLen := binary.LittleEndian.Uint32(metadata[len(metadata)-headerLenSize:])

	encKey := buf[len(buf)-int(keySize):]

	keyData, err := crypto.Decrypt(*key, encKey, nil)
	if err != nil {
		return
	}
	decodedKey, err = crypto.LoadKey(keyData)
	if err != nil {
		return
	}

	encHeader := buf[:len(buf)-int(keySize)]
	if headerLen != uint32(len(encHeader)) {
		err = errors.New("parsed header length does not match data")
		return
	}
	headerData, err := crypto.Decrypt(*key, encHeader, nil)
	if err != nil {
		return
	}

	blobs, err = parseHeaderData(bytes.NewReader(headerData))
	if err != nil {
		return
	}

	return
}

func buildHeaderData(blobs []Blob) (header []byte, err error) {
	header = make([]byte, 0, len(blobs)*int(UncompressedHeaderDataSize))

	for _, b := range blobs {
		err = fmt.Errorf("invalid blob type %d", b.Type)
		switch b.Type {
		case Data:
			if b.UncompressedLength == 0 {
				header = append(header, uint8(Data))
			} else {
				return
			}
		case CompressedData:
			if b.UncompressedLength != 0 {
				header = append(header, uint8(CompressedData))
			} else {
				return
			}
		default:
			return
		}
		err = nil

		var bloblen [4]byte
		binary.LittleEndian.PutUint32(bloblen[:], uint32(b.Length))
		header = append(header, bloblen[:]...)

		if b.UncompressedLength != 0 {
			binary.LittleEndian.PutUint32(bloblen[:], uint32(b.UncompressedLength))
			header = append(header, bloblen[:]...)
		}

		header = append(header, b.ID[:]...)
	}

	return
}

func verifyHeader(key *crypto.Key, sessionKey *crypto.Key, header []byte, blobs []Blob) (err error) {
	if len(header) > MaxHeaderSize {
		err = &warden.InvalidHeaderError{Msg: warden.StringPtr("header is too large")}
		return
	}

	decodedBlobs, decodedKey, err := DecodeHeader(key, sessionKey, bytes.NewReader(header), int64(len(header)))
	if err != nil {
		return
	}

	if !sessionKey.Equals(decodedKey) {
		err = &warden.InvalidHeaderError{Msg: warden.StringPtr("parsed session key does not match actual key")}
		return
	}

	if len(decodedBlobs) != len(blobs) {
		err = &warden.InvalidHeaderError{Msg: warden.StringPtr(fmt.Sprintf("expected %d blobs, got %d", len(blobs), len(decodedBlobs)))}
		return
	}

	for i, b := range blobs {
		if b.ID.String() != decodedBlobs[i].ID.String() {
			err = &warden.InvalidHeaderError{Msg: warden.StringPtr("parsed blob id does not match expected")}
			return
		}
	}

	return
}

func parseHeaderData(rdr *bytes.Reader) (res []Blob, err error) {
	cursor := 0

	for {
		b := Blob{}
		var bType byte
		bType, err = rdr.ReadByte()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}

			return
		}
		b.Type = BlobType(int(bType))

		var bLen uint32
		if err = binary.Read(rdr, binary.LittleEndian, &bLen); err != nil {
			return
		}
		b.Length = uint(bLen)

		if b.Type == CompressedData {
			uncompressedLen := uint32(b.UncompressedLength)
			if err = binary.Read(rdr, binary.LittleEndian, &uncompressedLen); err != nil {
				return
			}
		}

		if _, err = io.ReadFull(rdr, b.ID[:]); err != nil {
			return
		}

		b.Offset = uint(cursor)

		cursor += int(b.Length)

		res = append(res, b)
	}

	return
}
