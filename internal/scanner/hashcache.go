package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"
)

// HashCache avoids re-hashing files whose (size, mtime) are unchanged since the
// last scan. This makes a full rescan mostly stat syscalls plus hashing of only
// the files that actually changed.
type HashCache struct {
	mu sync.Mutex
	m  map[string]hcEntry
}

type hcEntry struct {
	size  int64
	mtime int64
	hash  string
}

// NewHashCache returns an empty cache.
func NewHashCache() *HashCache {
	return &HashCache{m: map[string]hcEntry{}}
}

// Clear drops all cached hashes, forcing the next scan to re-hash everything.
// Used by the "deep rescan" escape hatch and after we mutate the tree ourselves.
func (hc *HashCache) Clear() {
	hc.mu.Lock()
	hc.m = map[string]hcEntry{}
	hc.mu.Unlock()
}

// Hash returns the sha256 (hex) of the file at path, reusing the cached value
// when size and mtime match the previous observation.
func (hc *HashCache) Hash(path string, size, mtime int64) (string, error) {
	hc.mu.Lock()
	e, ok := hc.m[path]
	hc.mu.Unlock()
	if ok && e.size == size && e.mtime == mtime {
		return e.hash, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	hash := hex.EncodeToString(h.Sum(nil))

	hc.mu.Lock()
	hc.m[path] = hcEntry{size: size, mtime: mtime, hash: hash}
	hc.mu.Unlock()
	return hash, nil
}
