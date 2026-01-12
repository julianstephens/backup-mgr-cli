package warden

import "github.com/klauspost/compress/zstd"

var encoder, _ = zstd.NewWriter(nil)
var decoder, _ = zstd.NewReader(nil)

type CompressionType int

const (
	Zstd int = 1 << iota
)

func Compress(src []byte) []byte {
	return encoder.EncodeAll(src, make([]byte, 0, len(src)))
}

func Decompress(src []byte) ([]byte, error) {
	return decoder.DecodeAll(src, nil)
}
