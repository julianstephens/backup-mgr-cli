package storage_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	mp "github.com/agiledragon/gomonkey/v2"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/rs/zerolog"

	"github.com/julianstephens/warden/internal/backend"
	"github.com/julianstephens/warden/internal/backend/common"
	"github.com/julianstephens/warden/internal/crypto"
	. "github.com/julianstephens/warden/internal/storage"
	"github.com/julianstephens/warden/internal/store"
	"github.com/julianstephens/warden/internal/warden"
)

const (
	testStore  = "/tmp/test"
	testVolume = "/tmp/notes"
	testPwd    = "testsecurepassword123"
)

func initTestVolume(t *testing.T) {
	os.RemoveAll(testVolume)
	if err := os.Mkdir(testVolume, os.ModePerm); err != nil {
		t.Fatalf("unable to create test volume: %+v", err)
	}
}

func TestSnapshot(t *testing.T) {
	initTestVolume(t)

	testTime := time.Now().Add(-(10 * time.Minute))
	patches := mp.NewPatches()
	patches.ApplyFuncReturn(time.Now, testTime)
	defer patches.Reset()

	_, err := NewSnapshot("")
	if err == nil {
		t.Fatal("should error on empty backup :volume")
	}

	_, err = NewSnapshot("/path/to/nowhere")
	if err == nil {
		t.Fatal("should error on nonexistent backup volume")
	}

	snapshot, err := NewSnapshot(testVolume)
	if err != nil || snapshot == nil {
		t.Fatal("should create a new snapshot instead nil or error")
	}

	if snapshot.BackupVolume != testVolume {
		t.Fatalf("expected backup volume %s, got %s", testVolume, snapshot.BackupVolume)
	}
	if !snapshot.CreatedAt.Equal(testTime) {
		t.Fatalf("expected created at %s, got %s", testTime.String(), snapshot.CreatedAt.String())
	}
	if len(snapshot.Paths) != 0 {
		t.Fatalf("expected empty paths slice, got len %d", len(snapshot.Paths))
	}
	if len(snapshot.PackedChunks) != 0 {
		t.Fatalf("expected empty packed chunk slice, got len %d", len(snapshot.PackedChunks))
	}

	os.RemoveAll(testVolume)
}

func assertChunksEqual(t *testing.T, chunk1, chunk2 PackedChunk) {
	if chunk1.Chunk != chunk2.Chunk {
		t.Fatalf("chunk %s does not equal %s", chunk1.Chunk, chunk2.Chunk)
	}
	if chunk1.Pack != chunk2.Pack {
		t.Fatalf("pack %s does not equal %s", chunk1.Pack, chunk2.Pack)
	}
	if chunk1.ChunkStart != chunk2.ChunkStart {
		t.Fatalf("chunk start %d does not equal %d", chunk1.ChunkStart, chunk2.ChunkStart)
	}
	if chunk1.ChunkEnd != chunk2.ChunkEnd {
		t.Fatalf("chunk end %d does not equal %d", chunk1.ChunkEnd, chunk2.ChunkEnd)
	}
}

func TestPack(t *testing.T) {
	resetStore(t)

	store := createAndInitStore(context.Background(), t)
	k := store.Key().Decrypt()

	sessionKey, err := crypto.NewSessionKey()
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	p := NewPack(k, &buf)

	testData := []byte(gofakeit.Paragraph(10, 5, 12, "\n"))
	encData, err := crypto.Encrypt(*sessionKey, testData, nil)
	if err != nil {
		t.Fatal(err)
	}

	b1 := NewBlob(encData, Data, 0, 0)
	_, err = p.Append(*b1)
	if err != nil {
		t.Fatal(err)
	}

	if p.Size() != uint(b1.Length) {
		t.Fatalf("expected pack size %d, got %d", b1.Length, p.Size())
	}

	compressedTestData := warden.Compress(testData)
	encCompressed, err := crypto.Encrypt(*sessionKey, compressedTestData, nil)
	if err != nil {
		t.Fatal(err)
	}

	b2 := NewBlob(encCompressed, CompressedData, uint(len(testData)), p.Size())
	_, err = p.Append(*b2)
	if err != nil {
		t.Fatal(err)
	}
	if p.Size() != uint(b1.Length+b2.Length) {
		t.Fatalf("expected pack size %d, got %d", b1.Length+b2.Length, p.Size())
	}

	blobs := p.Blobs()

	if len(blobs) != 2 {
		t.Fatalf("expected 2 blobs in pack, got %d", len(blobs))
	}

	if blobs[0].ID.String() != b1.ID.String() {
		t.Fatal("first blob IDs do not match")
	}

	if blobs[1].ID.String() != b2.ID.String() {
		t.Fatal("second blob IDs do not match")
	}

	err = p.Finalize(sessionKey)
	if err != nil {
		t.Fatal(err)
	}

	os.RemoveAll(testStore)
}

func TestGetPack(t *testing.T) {
	initTestVolume(t)

	snapshot, err := NewSnapshot(testVolume)
	if err != nil || snapshot == nil {
		t.Fatal("should create a new snapshot instead nil or error")
	}

	testChunkHash := warden.NewID().String()
	testPackHash := warden.NewID().String()
	testChunkStart := 0
	testChunkEnd := 1024
	testChunk := PackedChunk{Chunk: testChunkHash, Pack: testPackHash, ChunkStart: int64(testChunkStart), ChunkEnd: int64(testChunkEnd)}

	snapshot.PackedChunks = append(snapshot.PackedChunks,
		testChunk,
		PackedChunk{Chunk: "xxx", Pack: "xxx", ChunkStart: int64(testChunkStart + 1024), ChunkEnd: int64(testChunkEnd + 1024)},
	)

	res := snapshot.GetPack(testChunkHash)
	assertChunksEqual(t, testChunk, *res)

	res = snapshot.GetPack("nonexistent")
	if res != nil {
		t.Fatalf("expected nil, got %+v", res)
	}

	os.RemoveAll(testVolume)
}

func resetStore(t *testing.T) {
	os.RemoveAll(testStore)
	err := warden.EnsureDir(testStore)
	if err != nil {
		t.Fatal(err)
	}
}

func createAndInitStore(ctx context.Context, t *testing.T) *store.Store {
	warden.SetLog(warden.NewLog(os.Stderr, zerolog.ErrorLevel, time.RFC1123))
	be, err := backend.NewBackend(common.LocalStorage, common.LocalStorageParams{Location: testStore})
	if err != nil {
		t.Fatal(err)
	}
	store := store.NewStore(be, testStore)

	err = store.Init(ctx, crypto.DefaultParams, testPwd)
	if err != nil {
		t.Fatal(err)
	}

	return store
}
