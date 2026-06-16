package diff_test

import (
	"os"
	"path/filepath"
	"testing"

	"sticky-scope/internal/diff"
	"sticky-scope/internal/model"
	"sticky-scope/internal/store"
)

func writeFile(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestManifestDiffSummary(t *testing.T) {
	root := t.TempDir()
	st, err := store.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// Baseline content lives in the CAS.
	oldA := []byte("old1\nold2\n")
	hashOldA, _ := st.PutBytes(oldA)
	gone := []byte("g1\n")
	hashGone, _ := st.PutBytes(gone)

	// Live content lives on disk.
	newA := []byte("old1\nold2\nold3\n") // +1 line
	writeFile(t, root, "a.txt", newA)
	addB := []byte("n1\nn2\n") // +2 lines (added file)
	writeFile(t, root, "b.txt", addB)

	base := &store.Manifest{Files: map[string]store.Entry{
		"a.txt":    {Hash: hashOldA, Size: int64(len(oldA))},
		"gone.txt": {Hash: hashGone, Size: int64(len(gone))},
	}}
	live := &store.Manifest{Files: map[string]store.Entry{
		"a.txt": {Hash: store.HashBytes(newA), Size: int64(len(newA))},
		"b.txt": {Hash: store.HashBytes(addB), Size: int64(len(addB))},
	}}

	cs := diff.ManifestDiff("pid", root, base, live, st)
	if cs.TotalFiles != 3 {
		t.Fatalf("TotalFiles = %d, want 3", cs.TotalFiles)
	}
	got := map[string]model.FileChange{}
	for _, f := range cs.Files {
		got[f.Path] = f
	}
	if f := got["a.txt"]; f.Status != model.StatusModified || f.Added != 1 || f.Removed != 0 {
		t.Errorf("a.txt = %+v, want modified +1 -0", f)
	}
	if f := got["b.txt"]; f.Status != model.StatusAdded || f.Added != 2 {
		t.Errorf("b.txt = %+v, want added +2", f)
	}
	if f := got["gone.txt"]; f.Status != model.StatusDeleted || f.Removed != 1 {
		t.Errorf("gone.txt = %+v, want deleted -1", f)
	}
	if cs.TotalAdded != 3 || cs.TotalRemoved != 1 {
		t.Errorf("totals = +%d -%d, want +3 -1", cs.TotalAdded, cs.TotalRemoved)
	}
}

func TestBuildFileDiffHunks(t *testing.T) {
	root := t.TempDir()
	st, err := store.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	oldC := []byte("a\nb\nc\n")
	h, _ := st.PutBytes(oldC)
	newC := []byte("a\nB\nc\nd\n") // change b->B (-1+1), add d (+1)
	writeFile(t, root, "f.txt", newC)

	base := &store.Manifest{Files: map[string]store.Entry{"f.txt": {Hash: h}}}
	live := &store.Manifest{Files: map[string]store.Entry{"f.txt": {Hash: store.HashBytes(newC)}}}

	fd, err := diff.BuildFileDiff(root, "f.txt", base, live, st)
	if err != nil {
		t.Fatal(err)
	}
	if fd.Status != model.StatusModified {
		t.Fatalf("status = %s, want modified", fd.Status)
	}
	if fd.Added != 2 || fd.Removed != 1 {
		t.Errorf("counts = +%d -%d, want +2 -1", fd.Added, fd.Removed)
	}
	if len(fd.Hunks) == 0 {
		t.Error("expected at least one hunk")
	}
}

func TestIsBinary(t *testing.T) {
	if diff.IsBinary([]byte("hello\nworld\n")) {
		t.Error("UTF-8 text wrongly flagged as binary")
	}
	if !diff.IsBinary([]byte{'a', 0x00, 'b'}) {
		t.Error("content with NUL byte not detected as binary")
	}
}