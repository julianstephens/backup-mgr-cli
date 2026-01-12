package local

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	pkgerr "github.com/pkg/errors"

	"github.com/julianstephens/warden/internal/backend/common"
	"github.com/julianstephens/warden/internal/storage"
	"github.com/julianstephens/warden/internal/warden"
)

type Local struct {
	common.WardenBackend
	location string
}

type LocationCtxKey string

const (
	name = "LocalStorage"
)

var (
	ErrStoreInitialization = errors.New("unable to initialize new store")
)

func NewLocalStorage(params common.LocalStorageParams) (*Local, error) {
	local := &Local{
		WardenBackend: common.WardenBackend{Self: common.LocalStorage, Name: name, Handler: &LocalHandler{}},
		location:      params.Location,
	}

	if err := scaffold(local.location); err != nil {
		return nil, err
	}

	return local, nil
}

func makeReadonly(filename string) error {
	err := os.Chmod(filename, 0444)
	if err != nil {
		return fmt.Errorf("unable to make file read-only: %+v", err)
	}
	return nil
}

func scaffold(storeLoc string) error {
	if storeLoc == "" {
		return &warden.InvalidStoreError{Msg: "expected valid path to store, got empty string"}
	}

	// create directory
	if _, err := os.Stat(storeLoc); err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(storeLoc, os.ModePerm); err != nil {
				return pkgerr.Wrap(ErrStoreInitialization, err.Error())
			}
		} else {
			return pkgerr.Wrap(ErrStoreInitialization, err.Error())
		}
	}

	return nil
}

func getCtxLocation(ctx context.Context, k LocationCtxKey) any {
	if v := ctx.Value(k); v != nil {
		return v
	}

	return nil
}

func (l *Local) Save(ctx context.Context, event common.Event, reader common.IReader) error {
	k := LocationCtxKey("location")
	ctx = context.WithValue(ctx, k, l.location)

	switch event.Type {
	case common.Key:
		if event.Name == nil {
			return fmt.Errorf("no name provided for key file")
		}
		warden.Log.Debug().Msg("localstorage backend handling key save event...")
		return l.WardenBackend.Handler.WriteKey(ctx, fmt.Sprintf("%s.json", *event.Name), reader)
	case common.Config:
		warden.Log.Debug().Msg("localstorage backend handling config save event...")
		return l.WardenBackend.Handler.WriteConfig(ctx, reader)
	case common.Pack:
		warden.Log.Debug().Msg("localstorage backend handling pack save event...")
		if event.Name == nil {
			return fmt.Errorf("no name provided for pack file")
		}
		// TODO: handle pack save
	default:
		return fmt.Errorf("got invalid event type: %s", event.Type.String())
	}

	return fmt.Errorf("got invalid event type: %s", event.Type.String())
}

func (l *Local) ListSnapshots(ctx context.Context) ([]storage.Snapshot, error) {
	var snaps []storage.Snapshot

	snapDir := path.Join(l.location, "snapshots")
	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		return snaps, nil
	}

	err := filepath.WalkDir(snapDir, func(path string, d fs.DirEntry, dirErr error) error {
		if d.IsDir() {
			return nil
		}

		s, err := warden.LoadJSON[storage.Snapshot](path)
		if err != nil {
			return err
		}

		snaps = append(snaps, s)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return snaps, nil
}

func (l *Local) Exists(ctx context.Context, resource_type string, resource_id string) (bool, error) {
	dirs := map[string]string{
		"masterkey": "keys",
		"config":    ".",
		"blob":      "blobs",
	}
	dir := dirs[resource_type]
	if dir == "" {
		return false, warden.InvalidArgumentError{Expecting: strings.Join(common.Resources, ", "), Got: resource_type}
	}

	if resource_id == "" {
		return false, warden.InvalidArgumentError{Expecting: "resource id", Got: "empty string"}
	}

	p := path.Join(l.location, dirs[resource_type], resource_id)

	if _, err := os.Stat(p); err == nil {
		return true, nil
	} else {
		return false, nil
	}
}
