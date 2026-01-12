package warden

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

const IdSize = sha256.Size

type ID [IdSize]byte

func NewID() ID {
	id := ID{}
	_, err := io.ReadFull(rand.Reader, id[:])
	if err != nil {
		panic(err)
	}
	return id
}

func ParseID(data string) (ID, error) {
	if len(data) != hex.EncodedLen(IdSize) {
		return ID{}, fmt.Errorf("invalid id %q of length %d", data, len(data))
	}

	buf, err := hex.DecodeString(data)
	if err != nil {
		return ID{}, fmt.Errorf("invalid id %q: %+v", data, err)
	}

	i := ID{}

	copy(i[:], buf)

	return i, nil
}

func (id ID) String() string {
	return hex.EncodeToString(id[:])
}
