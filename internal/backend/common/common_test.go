package common_test

import (
	"io"
	"math/rand"
	"testing"

	"github.com/julianstephens/warden/internal/backend/common"
)

func TestByteReader(t *testing.T) {
	data := genBytes(100, 55)

	reader := common.NewByteReader(data)

	if reader.Length() != int64(len(data)) {
		t.Fatalf("expected byte reader len %d, got %d", len(data), reader.Len)
	}

	pos := rand.Intn(len(data))
	_, err := reader.Reader.Seek(int64(pos), io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	first := data[0]
	curr := data[pos]
	res := make([]byte, 1)
	if _, err := reader.Reader.Read(res); err != nil {
		t.Fatal(err)
	}

	if res[0] != curr {
		t.Fatalf("expected data at pos %d to be %d, got %d", pos, curr, res[0])
	}

	err = reader.Reset()
	if err != nil {
		t.Fatal(err)
	}

	res = make([]byte, 1)
	if _, err := reader.Reader.Read(res); err != nil {
		t.Fatal(err)
	}

	if res[0] != first {
		t.Fatalf("expected data at pos %d to be %d, got %d", pos, first, res[0])
	}
}

func genBytes(n int, seed int64) []byte {
	b := make([]byte, n)
	rnd := rand.New(rand.NewSource(seed))
	_, err := rnd.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
