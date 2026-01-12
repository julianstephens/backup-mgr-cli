package warden_test

import (
	"bytes"
	"testing"

	"github.com/brianvoe/gofakeit/v7"

	. "github.com/julianstephens/warden/internal/warden"
)

func TestCompress(t *testing.T) {
	data := gofakeit.Paragraph(50, 5, 12, "\n")

	res := Compress([]byte(data))

	if len(res) >= len([]byte(data)) {
		t.Fatalf("expected output size less than %d, got %d", len([]byte(data)), len(res))
	}
}

func TestDecompress(t *testing.T) {
	data := gofakeit.Paragraph(50, 5, 12, "\n")

	res := Compress([]byte(data))

	raw, err := Decompress(res)
	if err != nil {
		t.Fatal(err)
	}

	if len(raw) != len([]byte(data)) {
		t.Fatalf("expected decompressed data len %d, got %d", len([]byte(data)), len(raw))
	}

	if !bytes.Equal([]byte(data), raw) {
		t.Fatal("expected decompressed data to equal original data")
	}
}
