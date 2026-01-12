package store_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	mp "github.com/agiledragon/gomonkey/v2"
	"github.com/rs/zerolog"

	"github.com/julianstephens/warden/internal/backend"
	"github.com/julianstephens/warden/internal/backend/common"
	"github.com/julianstephens/warden/internal/crypto"
	"github.com/julianstephens/warden/internal/store"
	"github.com/julianstephens/warden/internal/warden"
)

const (
	testDir    = "/tmp/test"
	testVolume = "/tmp/notes"
	testPwd    = "testsecurepassword123"
)

func resetStore(t *testing.T) {
	os.RemoveAll(testDir)
	err := warden.EnsureDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
}

func createAndInitStore(ctx context.Context, t *testing.T) *store.Store {
	warden.SetLog(warden.NewLog(os.Stderr, zerolog.ErrorLevel, time.RFC1123))
	be, err := backend.NewBackend(common.LocalStorage, common.LocalStorageParams{Location: testDir})
	if err != nil {
		t.Fatal(err)
	}
	store := store.NewStore(be, testDir)

	err = store.Init(ctx, crypto.DefaultParams, testPwd)
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func TestInit(t *testing.T) {
	resetStore(t)

	ctx := context.Background()
	createAndInitStore(ctx, t)

	conf, err := warden.LoadJSON[warden.Config](path.Join(testDir, "config.json"))
	if err != nil {
		t.Fatalf("expected config at %s, got err: %+v", path.Join(testDir, "config.json"), err)
	}

	if conf.ID == "" {
		t.Fatal("expected config id, got empty string")
	}

	if conf.Params == nil {
		t.Fatal("expected config params, got nil")
	}

	if _, err := os.Stat(path.Join(testDir, "keys")); os.IsNotExist(err) {
		t.Fatal("key directory not found")
	}

	keys, _ := os.ReadDir(path.Join(testDir, "keys"))
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	os.RemoveAll(testDir)
}

func TestOpen(t *testing.T) {
	resetStore(t)

	ctx := context.Background()
	original := createAndInitStore(ctx, t)

	patches := mp.ApplyFuncReturn(crypto.ReadPassword, testPwd, nil)
	defer patches.Reset()

	opened, err := store.OpenStore(ctx, testDir)
	if err != nil {
		t.Fatal(err)
	}

	originalKey := original.Key()
	openedKey := opened.Key()

	if !openedKey.Valid() {
		t.Fatal("expected valid key, got invalid")
	}

	if originalKey.Hostname != openedKey.Hostname {
		t.Fatalf("expected hostname %s, got %s", originalKey.Hostname, openedKey.Hostname)
	}

	if originalKey.Username != openedKey.Username {
		t.Fatalf("expected user %s, got %s", originalKey.Username, openedKey.Username)
	}

	if !openedKey.CreatedAt.After(originalKey.CreatedAt) {
		t.Fatalf("expected new key creation stamp after original: %s, %s", openedKey.CreatedAt, originalKey.CreatedAt)
	}

	if string(original.Key().Decrypt().Data) == string(opened.Key().Decrypt().Data) {
		t.Fatalf("expected decrypted key %s, got %s", string(original.Key().Decrypt().Data), string(opened.Key().Decrypt().Data))
	}
	os.RemoveAll(testDir)
}

func TestKey(t *testing.T) {
	var key *store.Key = nil

	repr := key.String()
	testRepr := "<Key | nil>"

	if repr != testRepr {
		t.Fatalf("expected key %s, got %s", testRepr, repr)
	}

	testUser := "testuser"
	testHost := "testhost"
	testCreatedAt := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
	key = &store.Key{
		Username:  testUser,
		Hostname:  testHost,
		CreatedAt: testCreatedAt,
	}

	repr = key.String()
	testRepr = fmt.Sprintf("<Key | user: %s, host: %s, created: %s>", testUser, testHost, testCreatedAt)

	if repr != testRepr {
		t.Fatalf("expected key %s, got %s", testRepr, repr)
	}
}

// func TestBackup(t *testing.T) {
// 	initTestVolume(t, 2)

// 	warden.SetLog(warden.NewLog(os.Stderr, zerolog.ErrorLevel, time.RFC1123))

// 	ctx := context.Background()

// 	patches := mp.NewPatches()
// 	patches.ApplyFuncReturn(crypto.ReadPassword, testPwd, nil)

// 	defer patches.Reset()

// 	s, err := store.OpenStore(ctx, testDir)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = s.Backup(ctx, testDir)
// 	if err == nil {
// 		t.Fatal("should error on backup dir equals warden store dir")
// 	}

// 	_, err = s.Backup(ctx, testVolume)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	os.RemoveAll(testVolume)
// }

// func initTestVolume(t *testing.T, n int) {
// 	os.RemoveAll(testVolume)

// 	if err := os.Mkdir(testVolume, os.ModePerm); err != nil {
// 		t.Fatalf("unable to create test volume: %+v", err)
// 	}

// 	for range n {
// 		data := []byte(gofakeit.Paragraph(10, 5, 12, "\n"))
// 		os.WriteFile(path.Join(testVolume, fmt.Sprintf("%s.txt", crypto.Hash(data).String())), data, os.ModePerm)
// 	}
// }
