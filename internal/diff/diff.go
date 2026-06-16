package diff

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	udiff "github.com/aymanbagabas/go-udiff"

	"sticky-scope/internal/model"
	"sticky-scope/internal/store"
)

const (
	// maxFilesInChangeSet caps the file list in a summary so a repo-wide change
	// can't produce an unbounded payload. TotalFiles still reports the real count.
	maxFilesInChangeSet = 5000
	// maxCountSize skips +/- line counting for very large files (counts shown 0).
	maxCountSize = 5 << 20 // 5 MB
)

// ManifestDiff builds the ChangeSet summary between a baseline manifest and the
// current live manifest. Old content is read from the CAS; new content from the
// project tree. Only counts (not line content) are produced here.
func ManifestDiff(projectID, root string, base, live *store.Manifest, st *store.Store) model.ChangeSet {
	cs := model.ChangeSet{
		ProjectID:   projectID,
		Files:       []model.FileChange{},
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	seen := make(map[string]struct{}, len(live.Files))
	paths := make([]string, 0, len(live.Files)+len(base.Files))
	for p := range live.Files {
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	for p := range base.Files {
		if _, ok := seen[p]; !ok {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)

	for _, p := range paths {
		be, inBase := base.Files[p]
		le, inLive := live.Files[p]
		switch {
		case inLive && !inBase:
			cs.Files = append(cs.Files, classifyAdded(root, p, le))
		case inBase && !inLive:
			cs.Files = append(cs.Files, classifyDeleted(p, be, st))
		case inBase && inLive:
			if sameEntry(be, le) {
				continue
			}
			cs.Files = append(cs.Files, classifyModified(root, p, be, le, st))
		}
	}

	cs.TotalFiles = len(cs.Files)
	for _, f := range cs.Files {
		cs.TotalAdded += f.Added
		cs.TotalRemoved += f.Removed
	}
	if len(cs.Files) > maxFilesInChangeSet {
		cs.Files = cs.Files[:maxFilesInChangeSet]
		cs.Truncated = true
	}
	return cs
}

func sameEntry(a, b store.Entry) bool {
	if a.Symlink != "" || b.Symlink != "" {
		return a.Symlink == b.Symlink
	}
	return a.Hash == b.Hash
}

func classifyAdded(root, p string, le store.Entry) model.FileChange {
	fc := model.FileChange{Path: p, Status: model.StatusAdded, NewSize: le.Size}
	if le.Symlink != "" {
		return fc
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	if err != nil {
		return fc
	}
	if IsBinary(content) {
		fc.Binary = true
		return fc
	}
	fc.Added = countLines(content)
	return fc
}

func classifyDeleted(p string, be store.Entry, st *store.Store) model.FileChange {
	fc := model.FileChange{Path: p, Status: model.StatusDeleted, OldSize: be.Size}
	if be.Symlink != "" {
		return fc
	}
	content, err := st.GetBytes(be.Hash)
	if err != nil {
		return fc
	}
	if IsBinary(content) {
		fc.Binary = true
		return fc
	}
	fc.Removed = countLines(content)
	return fc
}

func classifyModified(root, p string, be, le store.Entry, st *store.Store) model.FileChange {
	fc := model.FileChange{Path: p, Status: model.StatusModified, OldSize: be.Size, NewSize: le.Size}
	if be.Symlink != "" || le.Symlink != "" {
		return fc // symlink target change; no line counts
	}
	if be.Size > maxCountSize || le.Size > maxCountSize {
		return fc
	}
	oldContent, err1 := st.GetBytes(be.Hash)
	newContent, err2 := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	if err1 != nil || err2 != nil {
		return fc
	}
	if IsBinary(oldContent) || IsBinary(newContent) {
		fc.Binary = true
		return fc
	}
	fc.Added, fc.Removed = countDiff(string(oldContent), string(newContent))
	return fc
}

// countDiff returns the number of added and removed lines between two strings.
func countDiff(oldS, newS string) (added, removed int) {
	edits := udiff.Lines(oldS, newS)
	u, err := udiff.ToUnifiedDiff("a", "b", oldS, edits, 0)
	if err != nil {
		return 0, 0
	}
	for _, h := range u.Hunks {
		for _, l := range h.Lines {
			switch l.Kind {
			case udiff.Insert:
				added++
			case udiff.Delete:
				removed++
			}
		}
	}
	return added, removed
}