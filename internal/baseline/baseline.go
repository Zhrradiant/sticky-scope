// Package baseline manages a single project's persisted state: the baseline
// manifest (the "last accepted" snapshot), saved named versions, blob
// persistence into the CAS, and garbage collection of unreferenced blobs.
package baseline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"sticky-scope/internal/fsutil"
	"sticky-scope/internal/model"
	"sticky-scope/internal/store"
)

// Repo bundles a project's CAS store with the file locations of its baseline and
// version data.
type Repo struct {
	st           *store.Store
	root         string // the monitored project directory
	baselineFile string
	versionsDir  string
	indexFile    string
}

func NewRepo(st *store.Store, root, baselineFile, versionsDir, indexFile string) *Repo {
	return &Repo{st: st, root: root, baselineFile: baselineFile, versionsDir: versionsDir, indexFile: indexFile}
}

// LoadBaseline returns the persisted baseline manifest (empty if none yet).
func (r *Repo) LoadBaseline() (*store.Manifest, error) {
	return store.LoadManifest(r.baselineFile)
}

// persistBlobs ensures a blob exists in the CAS for every file entry in m and
// returns a manifest whose entries are guaranteed consistent with stored blobs.
//
// Fast path: if an entry already has a hash whose blob is present, it is trusted
// as-is (the blob content matches that hash by construction, so it does not
// matter if the live file has since changed). Only genuinely new content is read
// from disk and stored — which is what makes Confirm/SaveVersion O(changed
// files). Entries whose files vanished are dropped; symlinks are kept verbatim.
func (r *Repo) persistBlobs(m *store.Manifest) (*store.Manifest, error) {
	out := store.NewManifest()
	for p, e := range m.Files {
		if e.Symlink != "" {
			out.Files[p] = e
			continue
		}
		if e.Hash != "" && r.st.Has(e.Hash) {
			out.Files[p] = e
			continue
		}
		full := filepath.Join(r.root, filepath.FromSlash(p))
		hash, size, err := r.st.PutFile(full)
		if err != nil {
			continue // file gone or unreadable — skip
		}
		e.Hash = hash
		e.Size = size
		out.Files[p] = e
	}
	return out, nil
}

// SetBaseline persists blobs for m and writes it as the new baseline. The
// manifest write is atomic and happens only after all blobs are durable.
func (r *Repo) SetBaseline(m *store.Manifest) error {
	persisted, err := r.persistBlobs(m)
	if err != nil {
		return err
	}
	return store.SaveManifest(r.baselineFile, persisted)
}

type versionIndex struct {
	Versions []model.Version `json:"versions"`
}

func (r *Repo) loadIndex() (*versionIndex, error) {
	data, err := os.ReadFile(r.indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &versionIndex{}, nil
		}
		return nil, err
	}
	var vi versionIndex
	if err := json.Unmarshal(data, &vi); err != nil {
		return nil, err
	}
	return &vi, nil
}

func (r *Repo) saveIndex(vi *versionIndex) error {
	data, err := json.MarshalIndent(vi, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(r.indexFile, data, 0o644)
}

// SaveVersion snapshots m as a named version and appends it to the index.
func (r *Repo) SaveVersion(m *store.Manifest, name, message string, auto bool, added, removed int) (model.Version, error) {
	persisted, err := r.persistBlobs(m)
	if err != nil {
		return model.Version{}, err
	}
	id := uuid.NewString()
	if err := store.SaveManifest(filepath.Join(r.versionsDir, id+".json"), persisted); err != nil {
		return model.Version{}, err
	}
	v := model.Version{
		ID:        id,
		Name:      name,
		Message:   message,
		CreatedAt: time.Now().Format(time.RFC3339),
		Auto:      auto,
		FileCount: len(persisted.Files),
		Added:     added,
		Removed:   removed,
	}
	vi, err := r.loadIndex()
	if err != nil {
		return model.Version{}, err
	}
	vi.Versions = append(vi.Versions, v)
	if err := r.saveIndex(vi); err != nil {
		return model.Version{}, err
	}
	return v, nil
}

// ListVersions returns saved versions, newest first.
func (r *Repo) ListVersions() ([]model.Version, error) {
	vi, err := r.loadIndex()
	if err != nil {
		return nil, err
	}
	sort.Slice(vi.Versions, func(i, j int) bool {
		return vi.Versions[i].CreatedAt > vi.Versions[j].CreatedAt
	})
	if vi.Versions == nil {
		return []model.Version{}, nil
	}
	return vi.Versions, nil
}

// LoadVersionManifest loads the manifest snapshot for a version id.
func (r *Repo) LoadVersionManifest(vid string) (*store.Manifest, error) {
	return store.LoadManifest(filepath.Join(r.versionsDir, vid+".json"))
}

// DeleteVersion removes a version from the index and deletes its manifest.
func (r *Repo) DeleteVersion(vid string) error {
	vi, err := r.loadIndex()
	if err != nil {
		return err
	}
	out := vi.Versions[:0]
	for _, v := range vi.Versions {
		if v.ID != vid {
			out = append(out, v)
		}
	}
	vi.Versions = out
	if err := r.saveIndex(vi); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(r.versionsDir, vid+".json"))
	return nil
}

// GC deletes blobs not referenced by the baseline or any saved version.
func (r *Repo) GC() error {
	referenced := map[string]struct{}{}
	addRefs := func(m *store.Manifest) {
		for _, e := range m.Files {
			if e.Hash != "" {
				referenced[e.Hash] = struct{}{}
			}
		}
	}
	if base, err := r.LoadBaseline(); err == nil {
		addRefs(base)
	}
	if vi, err := r.loadIndex(); err == nil {
		for _, v := range vi.Versions {
			if m, err := r.LoadVersionManifest(v.ID); err == nil {
				addRefs(m)
			}
		}
	}
	_, err := r.st.GC(referenced)
	return err
}