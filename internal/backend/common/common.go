package common

import (
	"context"

	"github.com/julianstephens/warden/internal/storage"
)

type BackendType int

const (
	LocalStorage BackendType = 1 << iota
	S3
	SFTP
)

//go:generate stringer -type=BackendType

var BackendTypeStringMap = func() map[string]BackendType {
	m := make(map[string]BackendType)
	for i := LocalStorage; i <= SFTP; i = i << 1 {
		m[i.String()] = i
	}
	return m
}()

var BackendTypes = func() []string {
	var m []string
	for i := LocalStorage; i <= SFTP; i = i << 1 {
		m = append(m, i.String())
	}
	return m
}()

type Backend interface {
	// Save writes content to the specified backend
	Save(ctx context.Context, event Event, reader IReader) error
	// ListSnapshots retrieves all backup snapshots for a store
	ListSnapshots(ctx context.Context) ([]storage.Snapshot, error)
	// Exists determines whether a specified resource is on the backup medium
	Exists(ctx context.Context, resource_type string, resource_id string) (bool, error)
}

type WardenBackend struct {
	Self    BackendType
	Handler EventHandler
	Name    string
}

var Resources = []string{"masterkey", "config", "blob"}
