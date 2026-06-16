package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"sticky-scope/internal/store"
)

// Options controls a scan.
type Options struct {
	Patterns     []string // all ignore patterns (default + extra), gitignore format
	UseGitignore bool     // also honour .gitignore files in the tree
}

// ProgressFunc is called during a progress-aware scan: current is the number of
// files processed so far, total is the estimated total (0 when unknown).
type ProgressFunc func(current, total int)

// Scan walks the project tree and returns a manifest of the current state.
func Scan(root string, opts Options, hc *HashCache) (*store.Manifest, error) {
	return ScanWithProgress(root, opts, hc, nil)
}

// CountFiles does a fast walk (no hashing) to estimate how many regular files
// the real scan will process.
func CountFiles(root string, opts Options) int {
	count := 0
	rootClean := filepath.Clean(root)
	ig := newIgnorer(opts.Patterns, opts.UseGitignore)
	if opts.UseGitignore {
		if data, err := os.ReadFile(filepath.Join(rootClean, ".gitignore")); err == nil {
			ig.addGitignore(nil, data)
		}
	}
	_ = filepath.WalkDir(rootClean, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == rootClean {
			return nil
		}
		rel, relErr := filepath.Rel(rootClean, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		comps := strings.Split(rel, "/")
		isDir := d.IsDir()
		if ig.match(comps, isDir) {
			if isDir {
				return fs.SkipDir
			}
			return nil
		}
		if isDir {
			if opts.UseGitignore {
				if data, err := os.ReadFile(filepath.Join(path, ".gitignore")); err == nil {
					ig.addGitignore(comps, data)
				}
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		count++
		return nil
	})
	return count
}

// ScanWithProgress is like Scan but fires a callback after each file is hashed.
func ScanWithProgress(root string, opts Options, hc *HashCache, onFile ProgressFunc) (*store.Manifest, error) {
	rootClean := filepath.Clean(root)
	ig := newIgnorer(opts.Patterns, opts.UseGitignore)
	m := store.NewManifest()

	if opts.UseGitignore {
		if data, err := os.ReadFile(filepath.Join(rootClean, ".gitignore")); err == nil {
			ig.addGitignore(nil, data)
		}
	}

	processed := 0

	err := filepath.WalkDir(rootClean, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == rootClean {
			return nil
		}

		rel, relErr := filepath.Rel(rootClean, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		comps := strings.Split(rel, "/")
		isDir := d.IsDir()

		if ig.match(comps, isDir) {
			if isDir {
				return fs.SkipDir
			}
			return nil
		}

		if isDir {
			if opts.UseGitignore {
				if data, err := os.ReadFile(filepath.Join(path, ".gitignore")); err == nil {
					ig.addGitignore(comps, data)
				}
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		mode := info.Mode()

		if mode&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return nil
			}
			m.Files[rel] = store.Entry{
				Symlink: target,
				Mode:    uint32(mode),
				ModTime: info.ModTime().UnixNano(),
			}
			processed++
			if onFile != nil {
				onFile(processed, 0)
			}
			return nil
		}
		if !mode.IsRegular() {
			return nil
		}

		size := info.Size()
		mtime := info.ModTime().UnixNano()
		hash, err := hc.Hash(path, size, mtime)
		if err != nil {
			return nil
		}
		m.Files[rel] = store.Entry{
			Hash:    hash,
			Size:    size,
			Mode:    uint32(mode),
			ModTime: mtime,
		}
		processed++
		if onFile != nil {
			onFile(processed, 0)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}