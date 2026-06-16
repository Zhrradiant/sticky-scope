// Package store implements a content-addressable blob store (CAS) plus manifest
// persistence. Blobs are keyed by the sha256 of their content and stored under a
// sharded directory (objects/<aa>/<rest>); identical content is therefore stored
// only once and shared across the baseline and every saved version.
package store

import (
	"encoding/json"
	"os"

	"sticky-scope/internal/fsutil"
)

// Entry describes one tracked path in a manifest. For a regular file Hash/Size
// are set; for a symlink Symlink holds the link target and Hash is empty.
type Entry struct {
	Hash    string `json:"hash,omitempty"`
	Size    int64  `json:"size"`
	Mode    uint32 `json:"mode"`
	ModTime int64  `json:"mtime"` // unix nanoseconds
	Symlink string `json:"symlink,omitempty"`
}

// Manifest is a snapshot of a project tree: project-relative path -> Entry.
// Paths always use forward slashes.
type Manifest struct {
	Files map[string]Entry `json:"files"`
}

// NewManifest returns an empty, ready-to-use manifest.
func NewManifest() *Manifest {
	return &Manifest{Files: map[string]Entry{}}
}

// LoadManifest reads a manifest from disk. A missing file yields an empty
// manifest rather than an error, which keeps "no baseline yet" simple to handle.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewManifest(), nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Files == nil {
		m.Files = map[string]Entry{}
	}
	return &m, nil
}

// SaveManifest writes a manifest atomically. This is the "commit point" for a
// baseline or version: blobs must already be durably stored before it is called.
func SaveManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(path, data, 0o644)
}