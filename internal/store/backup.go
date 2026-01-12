package store

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"time"

	"github.com/julianstephens/warden/internal/storage"
	"github.com/julianstephens/warden/internal/warden"
)

func (s *Store) Backup(ctx context.Context, backupDir string) (*storage.Snapshot, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if backupDir == s.Location {
		return nil, fmt.Errorf("cannot backup warden store")
	}
	snapshot, err := backup(s, ctx, backupDir)
	if err != nil {
		return nil, fmt.Errorf("unable to backup dir %s: %+v", backupDir, err)
	}

	return snapshot, nil
}

func backup(store *Store, ctx context.Context, backupPath string) (snapshot *storage.Snapshot, err error) {
	latestSnapshot, err := getLastestSnapshot(store, ctx, backupPath)
	if err != nil {
		err = fmt.Errorf("unable to retrieve latest snapshot for backup path %s: %+v", backupPath, err)
		return
	}

	snapshot, err = storage.NewSnapshot(backupPath)
	if err != nil {
		return
	}

	snapshot.BackupSummary.BackupStart = time.Now()
	backupPaths, err := getBackupPaths(latestSnapshot, snapshot)
	if err != nil {
		return
	}

	fmt.Println(backupPaths)
	fmt.Println(snapshot.Paths)

	// fmt.Printf("%-64s  %s\n", "HASH", "CHUNK SIZE")
	// for _, p := range backupPaths {
	// 	chunkAndHash(store, p)
	// }
	snapshot.BackupSummary.BackupEnd = time.Now()

	return
}

func getLastestSnapshot(store *Store, ctx context.Context, backupDir string) (snap *storage.Snapshot, err error) {
	snapshots, err := store.backend.ListSnapshots(ctx)
	if err != nil {
		return
	}
	fmt.Print(len(snapshots))

	backupSnaps := warden.Filter(snapshots, func(t storage.Snapshot) bool {
		return t.BackupVolume == backupDir
	})

	if len(backupSnaps) == 0 {
		return
	}

	sort.Slice(backupSnaps, func(i, j int) bool {
		return backupSnaps[i].CreatedAt.After(backupSnaps[j].CreatedAt)
	})
	snap = &backupSnaps[0]

	return
}

func getBackupPaths(latestSnapshot *storage.Snapshot, currentSnapshot *storage.Snapshot) (backupPaths []string, err error) {
	if currentSnapshot == nil {
		err = fmt.Errorf("current backup snapshot must be provided, got nil")
		return
	}
	err = filepath.WalkDir(currentSnapshot.BackupVolume, func(path string, entry fs.DirEntry, err error) error {
		if entry.IsDir() {
			return nil
		}

		if latestSnapshot != nil {
			for _, p := range latestSnapshot.Paths {
				if p.Path == path {
					entryInfo, err := entry.Info()
					if err != nil || p.FileSize != entryInfo.Size() || !p.ModifiedAt.Equal(entryInfo.ModTime()) {
						backupPaths = append(backupPaths, path)
						currentSnapshot.BackupSummary.FilesModified += 1
						currentSnapshot.BackupSummary.TotalFilesProcessed += 1
						return nil
					} else {
						// copy path data from latest snapshot to current snapshot
						pathData := storage.PathMetadata{
							Path:       p.Path,
							FileSize:   p.FileSize,
							FilePerm:   p.FilePerm,
							ModifiedAt: p.ModifiedAt,
						}
						for _, c := range p.Chunks {
							pack := latestSnapshot.GetPack(c)
							if pack != nil {
								currentSnapshot.PackedChunks = append(currentSnapshot.PackedChunks, *pack)
							}
							pathData.Chunks = append(pathData.Chunks, c)
						}
						currentSnapshot.Paths = append(currentSnapshot.Paths, pathData)
						currentSnapshot.BackupSummary.FilesUnchanged += 1
						currentSnapshot.BackupSummary.TotalFilesProcessed += 1
						return nil
					}
				}
			}
		}

		backupPaths = append(backupPaths, path)
		currentSnapshot.BackupSummary.FilesAdded += 1
		currentSnapshot.BackupSummary.TotalFilesProcessed += 1
		return nil
	})

	return
}

// func chunkAndHash(store *Store, filepath string, snapshot *storage.Snapshot) (err error) {
// 	warden.Log.Debug().Msgf("checking file %s exists...", filepath)

// 	if _, err = os.Stat(filepath); err == nil {
// 		var file *os.File
// 		file, err = os.Open(filepath)
// 		if err != nil {
// 			return
// 		}
// 		defer file.Close()

// 		warden.Log.Debug().Msg("chunking and hashing file...")
// 		cKr := chunker.NewChunker(file)

// 	// 	for {
// 	// 		var chunk chunker.Chunk
// 	// 		chunk, err = cKr.Next()
// 	// 		if err == io.EOF {
// 	// 			break
// 	// 		}
// 	// 		if err != nil {
// 	// 			return
// 	// 		}

// 	// 		hashedChunk := crypto.SecureHash(chunk.Data, store.master.user.Data)
// 	// 		fmt.Println(hashedChunk)

// 	// 		newSnapshot := storage.Snapshot{
// 	// 			Paths: make([]storage.PathMetadata, 0),
// 	// 		}

// 	// 		if snapshot != nil {
// 	// 			foundChunk := warden.Filter[storage.PathMetadata](snapshot.Paths, func(p storage.PathMetadata) bool {
// 	// 				filteredChunk := warden.Filter[string](p.Chunks, func(chunk string) bool {
// 	// 					if chunk == hashedChunk {
// 	// 						return true
// 	// 					}

// 	// 					return false
// 	// 				})
// 	// 				if len(filteredChunk) > 0 {
// 	// 					return true
// 	// 				}

// 	// 				return false
// 	// 			})

// 	// 			// chunkLocations := getChunkLocations(snapshot)

// 	// 			// if len(foundChunk) > 0 {
// 	// 			// 	fmt.Println("found!")
// 	// 			// 	newSnapshot.Paths = append(newSnapshot.Paths, foundChunk...)
// 	// 			// } else {
// 	// 			// 	fmt.Println("not found!")
// 	// 			// 	exists, err := store.backend.Exists("blob", chunkLocations[hashedChunk])

// 	// 			// 	if err != nil {
// 	// 			// 		break
// 	// 			// 	}
// 	// 			// }
// 	// 		}
// 	// 	}
// } else {
// 	warden.Log.Info().Msgf("file %s does not exist. skipping...", filepath)
// }

// return
// }

// func getChunkLocations(allChunks []string, packedChunks []storage.PackedChunk) map[string]string {
// 	var locs map[string]string

// 	packedChunkMap := func(chunks []storage.PackedChunk) map[string]storage.PackedChunk {
// 		res := map[string]storage.PackedChunk{}
// 		for i, c := range chunks {
// 			res[c.Chunk] = chunks[i]
// 		}
// 		return res
// 	}(packedChunks)

// 	for _, chunk := range allChunks {
// 		pC, ok := packedChunkMap[chunk]
// 		if !ok {
// 			locs[chunk] = fmt.Sprintf("%s.chunk", chunk)
// 		} else {
// 			locs[chunk] = fmt.Sprintf("%s.pack", pC.Pack)
// 		}
// 	}

// 	return locs
// }
