package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// Store is a content-addressable blob store rooted at a single objects/ dir.
type Store struct {
	objectsDir string
}

// NewStore creates (if needed) and returns a Store for the given objects dir.
func NewStore(objectsDir string) (*Store, error) {
	if err := os.MkdirAll(objectsDir, 0o755); err != nil {
		return nil, err
	}
	return &Store{objectsDir: objectsDir}, nil
}

func (s *Store) blobPath(hash string) string {
	return filepath.Join(s.objectsDir, hash[:2], hash[2:])
}

// Has reports whether a blob with the given hash already exists.
func (s *Store) Has(hash string) bool {
	if len(hash) < 3 {
		return false
	}
	_, err := os.Stat(s.blobPath(hash))
	return err == nil
}

// Put streams content from r into the store and returns its sha256 (hex) and
// size. Storage is atomic and deduplicated: existing blobs are not rewritten.
func (s *Store) Put(r io.Reader) (string, int64, error) {
	tmp, err := os.CreateTemp(s.objectsDir, ".put-*")
	if err != nil {
		return "", 0, err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), r)
	if err != nil {
		_ = tmp.Close()
		return "", 0, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", 0, err
	}
	if err := tmp.Close(); err != nil {
		return "", 0, err
	}

	hash := hex.EncodeToString(h.Sum(nil))
	dst := s.blobPath(hash)
	if _, err := os.Stat(dst); err == nil {
		return hash, n, nil // already stored — dedup
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", 0, err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		// A concurrent writer may have created it between our Stat and Rename.
		if _, statErr := os.Stat(dst); statErr == nil {
			return hash, n, nil
		}
		return "", 0, err
	}
	return hash, n, nil
}

// PutBytes stores a byte slice, returning its hash.
func (s *Store) PutBytes(b []byte) (string, error) {
	hash := HashBytes(b)
	if s.Has(hash) {
		return hash, nil
	}
	h, _, err := s.Put(bytes.NewReader(b))
	return h, err
}

// PutFile streams a file from disk into the store.
func (s *Store) PutFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	return s.Put(f)
}

// Get opens a blob for reading.
func (s *Store) Get(hash string) (io.ReadCloser, error) {
	return os.Open(s.blobPath(hash))
}

// GetBytes reads an entire blob into memory.
func (s *Store) GetBytes(hash string) ([]byte, error) {
	if len(hash) < 3 {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(s.blobPath(hash))
}

// HashBytes returns the hex sha256 of b.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}