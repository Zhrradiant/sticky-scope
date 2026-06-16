package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"sticky-scope/internal/scanner"
)

func TestScanRespectsIgnoreRules(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("a.txt", "1\n")
	write("sub/b.txt", "2\n")
	write("node_modules/c.txt", "3\n") // ignored via extra pattern "node_modules/"
	write(".gitignore", "ignored.txt\nlogs/\n")
	write("ignored.txt", "x\n") // .gitignore file rule
	write("logs/d.txt", "y\n")  // .gitignore dir rule
	write("keep.log", "z\n")    // not ignored (no *.log pattern)

	m, err := scanner.Scan(root, scanner.Options{Patterns: []string{"node_modules/"}, UseGitignore: true}, scanner.NewHashCache())
	if err != nil {
		t.Fatal(err)
	}
	check := func(p string, want bool) {
		t.Helper()
		_, ok := m.Files[p]
		if ok != want {
			t.Errorf("%s present=%v, want %v", p, ok, want)
		}
	}
	check("a.txt", true)
	check("sub/b.txt", true)
	check(".gitignore", true)
	check("keep.log", true)
	check("node_modules/c.txt", false)
	check("ignored.txt", false)
	check("logs/d.txt", false)
}

func TestScanWithGitignoreDisabled(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, ".gitignore")
	_ = os.WriteFile(p, []byte("secret.txt\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "secret.txt"), []byte("s\n"), 0o644)

	m, err := scanner.Scan(root, scanner.Options{UseGitignore: false}, scanner.NewHashCache())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["secret.txt"]; !ok {
		t.Error("secret.txt should be tracked when .gitignore parsing is disabled")
	}
}