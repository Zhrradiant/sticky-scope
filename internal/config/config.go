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
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	CreatedAt       string   `json:"createdAt"`
	DefaultPatterns []string `json:"defaultPatterns"` // pre-written default ignore patterns (gitignore format)
	Ignore          []string `json:"ignore"`          // extra user ignore patterns
	UseGitignore    bool     `json:"useGitignore"`
}

// Settings holds global, project-independent preferences.
type Settings struct {
	Language string `json:"language"` // "zh" | "en"; UI also persists this itself
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
// It also backfills DefaultPatterns for projects created before this field existed.
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
	// Migration: backfill default patterns for older projects.
	for i := range c.Projects {
		if len(c.Projects[i].DefaultPatterns) == 0 {
			c.Projects[i].DefaultPatterns = DefaultPreset()
		}
	}
	return &c, nil
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
// format. These are pre-populated into every new project's DefaultPatterns and
// are fully visible and editable in the settings panel.
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