package chunker_test

import (
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/julianstephens/warden/internal/backend/common"
	"github.com/julianstephens/warden/internal/chunker"
)

var normalSize = 1024

func TestMinSizeData(t *testing.T) {
	opts := &chunker.Options{AverageSize: &normalSize}

	cases := []struct {
		name           string
		totalSize      int
		expectedChunks int
		gotError       error
	}{
		{
			name:           "should return single chunk of original size",
			totalSize:      rand.Intn(normalSize / 10),
			expectedChunks: 1,
			gotError:       io.EOF,
		},
		{
			name:           "should return multiple chunks",
			totalSize:      rand.Intn(normalSize * 10),
			expectedChunks: 10,
			gotError:       io.EOF,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := genBytes(c.totalSize, 67)

			cK := chunker.NewChunker(common.NewByteReader(data).Reader, opts)

			res := make([]byte, 0)
			for i := range c.expectedChunks {
				chunk, err := cK.Next()
				if err != nil {
					if c.gotError != nil {
						if !errors.Is(err, c.gotError) {
							t.Fatalf("expected err: %+v, got: %+v", c.gotError, err)
						}
						return
					}

					t.Fatalf("expected no error, got: %+v", err)
				}

				if c.expectedChunks == 1 {
					if i > 1 {
						t.Fatal("expected 1 chunk, got multiple")
					}
				}
				res = append(res, chunk.Data...)
			}

			_, err := cK.Next()
			if !errors.Is(err, io.EOF) {
				t.Fatalf("expected EOF error, got: %+v", err)
			}
			assertSliceEqual(t, data, res)
		})
	}
}

func TestLargeData(t *testing.T) {
	paragraphCount := rand.Intn(100)
	seed := uint64(82342)
	fakeData := []byte(gofakeit.Paragraph(paragraphCount, 5, 12, "\n"))

	cK := chunker.NewChunker(common.NewByteReader(fakeData).Reader, &chunker.Options{AverageSize: &normalSize, Seed: &seed})

	res := make([]byte, 0)
	max := normalSize * 8
	for {
		chunk, err := cK.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		res = append(res, chunk.Data...)

		if chunk.Length > max {
			t.Fatalf("expected chunk length less than %d, got %d", max, chunk.Length)
		}
	}

	assertSliceEqual(t, fakeData, res)
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

func assertSliceEqual[T comparable](t *testing.T, expected []T, actual []T) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("expected (%+v) is not equal to actual (%+v): len(expected)=%d len(actual)=%d",
			expected, actual, len(expected), len(actual))
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("expected[%d] (%+v) is not equal to actual[%d] (%+v)",
				i, expected[i],
				i, actual[i])
		}
	}
}
