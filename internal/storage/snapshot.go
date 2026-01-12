package storage

import (
	"fmt"
	"time"

	"github.com/julianstephens/warden/internal/warden"
)

type BackupMetadata struct {
	BackupStart         time.Time `json:"backupStart"`
	BackupEnd           time.Time `json:"backupEnd"`
	FilesAdded          uint      `json:"filesAdded"`
	FilesModified       uint      `json:"filesModified"`
	FilesUnchanged      uint      `json:"filesUnchanged"`
	TotalFilesProcessed uint      `json:"totalFilesProcessed"`
}

type PathMetadata struct {
	Path       string    `json:"path"`
	FileSize   int64     `json:"fileSize"`
	FilePerm   string    `json:"filePermission"`
	ModifiedAt time.Time `json:"modifiedAt"`

	Chunks []string `json:"chunks"`
}

type PackedChunk struct {
	Chunk string `json:"chunk"`
	Pack  string `json:"pack"`

	ChunkStart int64 `json:"chunkStartOffset"`
	ChunkEnd   int64 `json:"chunkEndOffset"`
}

type Snapshot struct {
	BackupVolume string         `json:"backupVolume"`
	Paths        []PathMetadata `json:"paths"`

	PackedChunks []PackedChunk `json:"packedChunks"`

	CreatedAt time.Time `json:"createdAt"`
	Hostname  string    `json:"hostname,omitempty"`
	Username  string    `json:"username,omitempty"`

	BackupSummary BackupMetadata `json:"backupSummary"`
}

func NewSnapshot(backupVolume string) (*Snapshot, error) {
	if backupVolume == "" {
		return nil, fmt.Errorf("expected valid path to backup volume, got empty string")
	}

	if ok, err := warden.PathExists(backupVolume); err != nil || !ok {
		return nil, fmt.Errorf("expected valid path to backup volume, got false or error")
	}

	username, hostname, err := warden.GetSystemInfo()
	if err != nil {
		return nil, err
	}

	s := &Snapshot{
		BackupVolume: backupVolume,
		CreatedAt:    time.Now(),
		Username:     username,
		Hostname:     hostname,
		Paths:        make([]PathMetadata, 0),
		PackedChunks: make([]PackedChunk, 0),
		BackupSummary: BackupMetadata{
			FilesAdded:          0,
			FilesModified:       0,
			FilesUnchanged:      0,
			TotalFilesProcessed: 0,
		},
	}

	return s, nil
}

// GetPack returns the pack information for a chunk if it exists
func (s *Snapshot) GetPack(chunk string) *PackedChunk {
	for _, c := range s.PackedChunks {
		if c.Chunk == chunk {
			return &c
		}
	}
	return nil
}
