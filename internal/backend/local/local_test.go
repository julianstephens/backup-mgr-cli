package local_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/rs/zerolog"

	"github.com/julianstephens/warden/internal/backend/common"
	"github.com/julianstephens/warden/internal/backend/local"
	"github.com/julianstephens/warden/internal/crypto"
	"github.com/julianstephens/warden/internal/storage"
	"github.com/julianstephens/warden/internal/warden"
)

var testParams = common.LocalStorageParams{
	Location: "/tmp/local",
}

func TestNewLocalStorage(t *testing.T) {
	_, err := local.NewLocalStorage(common.LocalStorageParams{Location: ""})
	if err == nil {
		t.Fatal("should error on empty localstorage location")
	}

	local, err := local.NewLocalStorage(testParams)
	if err != nil {
		t.Fatal(err)
	}

	if local.Self != common.LocalStorage {
		t.Fatalf("expected backend type %s, got %s", common.LocalStorage.String(), local.Self.String())
	}
	if local.Name != "LocalStorage" {
		t.Fatalf("expected backend named LocalStorage, got %s", local.Name)
	}

	if _, err = os.Stat(testParams.Location); errors.Is(err, os.ErrNotExist) {
		t.Fatalf("should create store %s", testParams.Location)
	}

	os.RemoveAll(testParams.Location)
}

func TestWriteConfig(t *testing.T) {
	warden.SetLog(warden.NewLog(os.Stderr, zerolog.ErrorLevel, time.RFC1123))
	l, err := local.NewLocalStorage(testParams)
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatalf("unable to generate json data: %+v", err)
	}

	var testConf warden.Config
	err = gofakeit.Struct(&testConf)
	if err != nil {
		t.Fatalf("unable to generate test config: %+v", err)
	}
	testConf.Params = crypto.DefaultParams.ToMap()

	data, err := json.Marshal(testConf)
	if err != nil {
		t.Fatal(err)
	}
	reader := common.NewByteReader(data)

	ctx := context.Background()
	k := local.LocationCtxKey("location")
	ctx = context.WithValue(ctx, k, "")

	err = l.Handler.WriteConfig(ctx, reader)
	if err == nil || !errors.Is(err, local.ErrNoStoreLocation) {
		t.Fatal("should error on context with no store location")
	}

	ctx = context.WithValue(ctx, k, testParams.Location)
	err = l.Handler.WriteConfig(ctx, reader)
	if err != nil {
		t.Fatal(err)
	}

	data, err = os.ReadFile(path.Join(testParams.Location, "config.json"))
	if err != nil {
		t.Fatal(err)
	}

	var conf warden.Config
	err = json.Unmarshal(data, &conf)
	if err != nil {
		t.Fatal("unable to unmarshal stored data to warden config")
	}

	if testConf.ID != conf.ID {
		t.Fatalf("expected config id %s, got %s", testConf.ID, conf.ID)
	}

	if eq := reflect.DeepEqual(testConf.Params, conf.Params); !eq {
		t.Fatal("expected stored config params to equal test config params")
	}

	os.RemoveAll(testParams.Location)
}

func TestSave(t *testing.T) {
	warden.SetLog(warden.NewLog(os.Stderr, zerolog.ErrorLevel, time.RFC1123))
	l, err := local.NewLocalStorage(testParams)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = l.Save(ctx, common.Event{}, &common.ByteReader{})
	if err == nil {
		t.Fatal("expected error on invalid event type, got nil")
	}

	err = l.Save(ctx, common.Event{Type: common.Key, Name: nil}, &common.ByteReader{})
	if err == nil {
		t.Fatal("expected error on invalid key name, got nil")
	}
}

func TestListSnapshots(t *testing.T) {
	warden.SetLog(warden.NewLog(os.Stderr, zerolog.ErrorLevel, time.RFC1123))
	ctx := context.Background()

	snapDir := path.Join(testParams.Location, "snapshots")
	err := warden.EnsureDir(snapDir)
	if err != nil {
		t.Fatal(err)
	}
	var testSnaps []storage.Snapshot

	for i := range rand.Intn(20) {
		var s storage.Snapshot
		err = gofakeit.Struct(&s)
		if err != nil {
			t.Fatal(err)
		}
		testSnaps = append(testSnaps, s)
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(path.Join(snapDir, fmt.Sprintf("%s.json", strconv.Itoa(i))), data, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	}

	l, err := local.NewLocalStorage(testParams)
	if err != nil {
		t.Fatal(err)
	}

	snaps, err := l.ListSnapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(snaps) != len(testSnaps) {
		t.Fatalf("expected %d snapshots, got %d", len(testSnaps), len(snaps))
	}

	os.RemoveAll(snapDir)
}

func TestExists(t *testing.T) {
	warden.SetLog(warden.NewLog(os.Stderr, zerolog.ErrorLevel, time.RFC1123))

	l, err := local.NewLocalStorage(testParams)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	k := local.LocationCtxKey("location")
	ctx = context.WithValue(ctx, k, testParams.Location)

	cases := []struct {
		name          string
		resource_type string
		resource_id   string
		expected      bool
		gotError      interface{}
	}{
		{
			name:          "should error on invalid resource type",
			resource_type: "invalid",
			resource_id:   "xxx",
			expected:      false,
			gotError:      &warden.InvalidArgumentError{Expecting: strings.Join(common.Resources, ","), Got: "invalid"},
		},
		{
			name:          "should error on invalid resource id",
			resource_type: "config",
			resource_id:   "",
			expected:      false,
			gotError:      &warden.InvalidArgumentError{Expecting: "resource id", Got: "empty string"},
		},
		{
			name:          "should return true if file exists",
			resource_type: "config",
			resource_id:   "config.json",
			expected:      true,
			gotError:      nil,
		},
		{
			name:          "should return false if file does not exist",
			resource_type: "config",
			resource_id:   "nonexistent",
			expected:      false,
			gotError:      nil,
		},
	}

	var testConf warden.Config
	err = gofakeit.Struct(&testConf)
	if err != nil {
		t.Fatalf("unable to generate test config: %+v", err)
	}
	testConf.Params = crypto.DefaultParams.ToMap()

	data, err := json.Marshal(testConf)
	if err != nil {
		t.Fatal(err)
	}
	reader := common.NewByteReader(data)

	err = l.Handler.WriteConfig(ctx, reader)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			exists, err := l.Exists(ctx, c.resource_type, c.resource_id)

			if c.gotError != nil {
				if !errors.As(err, c.gotError) {
					t.Fatalf("expected error: %+v, got: %+v", c.gotError, err)
					return
				}
			}

			if exists != c.expected {
				t.Fatalf("expected file exists %+v, got %+v", c.expected, exists)
			}
		})
	}

	os.RemoveAll(testParams.Location)
}
