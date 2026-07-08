// Package config owns the on-disk layout of the application data directory and
// global config, plus per-project storage directories. The global config.json
// lives under os.UserConfigDir()/StickyScope (e.g. %APPDATA% on Windows); per-
// project data (CAS blobs, baselines, versions) lives in a .sticky-scope
// directory at the root of each monitored project — like .git or .claude.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"sticky-scope/internal/fsutil"
)

const appDirName = "StickyScope"

// ProjectDirName is the directory name inside a monitored project that holds
// all of this tool's per-project data (baseline, CAS blobs, versions).
const ProjectDirName = ".sticky-scope"

// ProjectMeta is the persisted description of a monitored project.
type ProjectMeta struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	CreatedAt    string   `json:"createdAt"`
	Ignore       []string `json:"ignore"`       // extra user ignore patterns
	UseGitignore bool     `json:"useGitignore"`
}

// Settings holds global, project-independent preferences.
type Settings struct {
	Language        string   `json:"language"`        // "zh" | "en"; UI also persists this itself
	DefaultPatterns []string `json:"defaultPatterns"` // global shared default ignore patterns (gitignore format)
}

// Config is the root document persisted to config.json.
type Config struct {
	Projects []ProjectMeta `json:"projects"`
	Settings Settings      `json:"settings"`
}

// Root returns the application data directory, creating nothing.
func Root() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName), nil
}

func configFile(root string) string { return filepath.Join(root, "config.json") }

// Per-project path helpers. Each monitored project stores its data inside a
// .sticky-scope directory at its root (project-level storage, like .git or
// .claude). The project root is the monitored directory, not the app data dir.
func ProjectDir(projectRoot string) string  { return filepath.Join(projectRoot, ProjectDirName) }
func ObjectsDir(projectRoot string) string  { return filepath.Join(ProjectDir(projectRoot), "objects") }
func BaselineFile(projectRoot string) string { return filepath.Join(ProjectDir(projectRoot), "baseline.json") }
func VersionsDir(projectRoot string) string  { return filepath.Join(ProjectDir(projectRoot), "versions") }
func VersionIndexFile(projectRoot string) string {
	return filepath.Join(VersionsDir(projectRoot), "index.json")
}

// Load reads config.json, returning an empty config if it does not exist yet.
// It also collapses legacy per-project DefaultPatterns into the single shared
// global Settings.DefaultPatterns (a one-time migration).
func Load() (*Config, error) {
	root, err := Root()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(configFile(root))
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	// Migration: collapse legacy per-project DefaultPatterns into the new
	// single global list. The field no longer exists on ProjectMeta, so read
	// it from the raw JSON; merge (order-preserving, deduped) into
	// Settings.DefaultPatterns so existing edits are preserved.
	var raw struct {
		Projects []struct {
			DefaultPatterns []string `json:"defaultPatterns"`
		} `json:"projects"`
	}
	_ = json.Unmarshal(data, &raw) // best-effort; shape mismatch is fine
	legacy := []string{}
	for _, p := range raw.Projects {
		legacy = mergePatterns(legacy, p.DefaultPatterns)
	}
	// If the global defaults are already set (from settings.defaultPatterns in
	// the JSON), keep them as the single source of truth. Otherwise seed them
	// from the merged legacy per-project patterns, falling back to the factory
	// preset for a fresh install.
	if len(c.Settings.DefaultPatterns) == 0 {
		if len(legacy) > 0 {
			c.Settings.DefaultPatterns = legacy
		} else {
			c.Settings.DefaultPatterns = DefaultPreset()
		}
	}
	return &c, nil
}

// mergePatterns appends items from src to dst, skipping exact duplicates that
// already appear in dst (case-sensitive). Order is preserved.
func mergePatterns(dst, src []string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, p := range dst {
		seen[strings.TrimSpace(p)] = struct{}{}
	}
	for _, p := range src {
		k := strings.TrimSpace(p)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		dst = append(dst, p)
	}
	return dst
}

// Save writes config.json atomically.
func Save(c *Config) error {
	root, err := Root()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(configFile(root), data, 0o644)
}

// ProjectID derives a stable id from an absolute path. The path is cleaned and,
// on Windows, lower-cased so that the same directory always maps to the same id.
func ProjectID(absPath string) string {
	key := filepath.Clean(absPath)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:16]
}

// DefaultPreset returns the pre-written default ignore patterns in gitignore
// format. These back the global shared Settings.DefaultPatterns and are fully
// visible and editable in the settings panel; the "reset to default" action
// restores them.
func DefaultPreset() []string {
	return []string{
		// VCS
		".git/", ".hg/", ".svn/",
		// Dependency dirs
		"node_modules/", "bower_components/",
		// Build output
		"dist/", "build/", "out/", "target/", "bin/", "obj/",
		// IDE / editor
		".idea/", ".vscode/",
		// Python
		"__pycache__/", ".venv/", "venv/", ".tox/",
		".pytest_cache/", ".mypy_cache/",
		// Frontend
		".next/", ".nuxt/", ".svelte-kit/", ".cache/", ".parcel-cache/",
		// Java / Kotlin
		".gradle/", ".mvn/",
		// This tool's own data
		ProjectDirName + "/",
		// Editor swap / scratch files
		"*.swp", "*.swo", "*~", ".#*", "4913", "___jb_tmp___", "*.tmp",
		// OS metadata
		".DS_Store", "Thumbs.db", "desktop.ini",
		// Compiled bytecode
		"*.pyc", "*.pyo", "*.class",
	}
}