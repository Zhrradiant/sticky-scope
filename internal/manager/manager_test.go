package manager_test

import (
	"os"
	"path/filepath"
	"testing"

	"sticky-scope/internal/config"
	"sticky-scope/internal/manager"
)

// This exercises the full audit workflow against a temp app-data dir (APPDATA is
// redirected so the real user config is never touched), with monitoring off so
// every step is synchronous and deterministic.
func TestManagerWorkflow(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)

	// Guard: ensure the app-data root actually resolved into our temp dir, so we
	// never pollute the real config during tests.
	root, err := config.Root()
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(appData, root); err != nil || rel == ".." || filepath.IsAbs(rel) {
		t.Skipf("APPDATA override did not take effect (root=%s); skipping", root)
	}

	proj := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(proj, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(proj, filepath.FromSlash(rel)))
		if err != nil {
			return "<missing>"
		}
		return string(b)
	}

	write("keep.txt", "hello\n")
	write("edit.txt", "a\nb\nc\n")
	write("remove.txt", "bye\n")

	mgr, err := manager.New(func(string, any) {})
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Shutdown()

	info, err := mgr.AddProject(proj)
	if err != nil {
		t.Fatal(err)
	}
	id := info.ID

	// Fresh baseline => no changes.
	if cs, _ := mgr.GetChanges(id); cs.TotalFiles != 0 {
		t.Fatalf("new project should be clean, got %d files", cs.TotalFiles)
	}

	// Simulate AI edits: modify, add, delete.
	write("edit.txt", "a\nB\nc\nd\n") // -1 +2
	write("new.txt", "x\ny\n")        // +2 (added)
	os.Remove(filepath.Join(proj, "remove.txt"))

	cs, err := mgr.DeepRescan(id)
	if err != nil {
		t.Fatal(err)
	}
	if cs.TotalFiles != 3 {
		t.Fatalf("expected 3 changed files, got %d", cs.TotalFiles)
	}
	byPath := map[string]string{}
	for _, f := range cs.Files {
		byPath[f.Path] = string(f.Status)
	}
	if byPath["edit.txt"] != "modified" || byPath["new.txt"] != "added" || byPath["remove.txt"] != "deleted" {
		t.Fatalf("unexpected statuses: %+v", byPath)
	}

	// Per-file diff for the modified file.
	fd, err := mgr.GetFileDiff(id, "edit.txt")
	if err != nil {
		t.Fatal(err)
	}
	if fd.Added != 2 || fd.Removed != 1 || len(fd.Hunks) == 0 {
		t.Fatalf("edit.txt diff = +%d -%d hunks=%d, want +2 -1 with hunks", fd.Added, fd.Removed, len(fd.Hunks))
	}

	// Save a version of this changed state.
	_, err = mgr.SaveVersion(id, "checkpoint", "after AI edits")
	if err != nil {
		t.Fatal(err)
	}
	if vs, _ := mgr.ListVersions(id); len(vs) != 1 {
		t.Fatalf("expected 1 version, got %d", len(vs))
	}

	// Confirm-all advances the baseline without touching project files.
	if err := mgr.ConfirmAll(id); err != nil {
		t.Fatal(err)
	}
	if got := read("edit.txt"); got != "a\nB\nc\nd\n" {
		t.Fatalf("confirm must not modify project files: %q", got)
	}
}