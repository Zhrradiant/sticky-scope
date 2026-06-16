package store

import (
	"os"
	"path/filepath"
)

// GC removes every blob whose hash is not present in referenced. Callers build
// the referenced set from the union of the baseline manifest and all saved
// version manifests, then call GC to reclaim space from superseded content.
func (s *Store) GC(referenced map[string]struct{}) (int, error) {
	deleted := 0
	shards, err := os.ReadDir(s.objectsDir)
	if err != nil {
		return 0, err
	}
	for _, shard := range shards {
		if !shard.IsDir() || len(shard.Name()) != 2 {
			continue
		}
		shardDir := filepath.Join(s.objectsDir, shard.Name())
		blobs, err := os.ReadDir(shardDir)
		if err != nil {
			continue
		}
		for _, blob := range blobs {
			hash := shard.Name() + blob.Name()
			if _, ok := referenced[hash]; ok {
				continue
			}
			if err := os.Remove(filepath.Join(shardDir, blob.Name())); err == nil {
				deleted++
			}
		}
		if entries, _ := os.ReadDir(shardDir); len(entries) == 0 {
			_ = os.Remove(shardDir)
		}
	}
	return deleted, nil
}
