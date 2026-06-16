// Package fsutil provides small filesystem helpers shared across the backend.
package fsutil

import (
	"os"
	"path/filepath"
)

// WriteAtomic writes data to path atomically: it writes to a temp file in the
// same directory, fsyncs, then renames over the destination. A torn write can
// therefore never leave a half-written file at path.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Cleanup if we fail before the rename; harmless no-op after a successful rename.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = os.Chmod(tmpName, perm) // best-effort; meaningless on Windows
	return os.Rename(tmpName, path)
}
