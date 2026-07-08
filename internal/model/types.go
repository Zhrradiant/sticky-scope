// Package model holds the data-transfer objects shared between the Go backend
// and the Vue frontend. Wails generates TypeScript definitions from these types,
// so every field that the UI needs must be exported and carry a json tag.
package model

// ChangeStatus is the kind of change a file has relative to the baseline.
type ChangeStatus string

const (
	StatusAdded    ChangeStatus = "added"
	StatusModified ChangeStatus = "modified"
	StatusDeleted  ChangeStatus = "deleted"
)

// ProjectInfo describes a monitored project (safe to send to the frontend).
type ProjectInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	CreatedAt    string   `json:"createdAt"`
	Available    bool     `json:"available"` // project dir currently exists
	Ignore       []string `json:"ignore"`    // extra user ignore patterns
	UseGitignore bool     `json:"useGitignore"`
}

// SettingsInfo is the global, project-independent configuration surfaced to
// the frontend. It carries the shared default ignore patterns.
type SettingsInfo struct {
	Language        string   `json:"language"`
	DefaultPatterns []string `json:"defaultPatterns"` // global shared default ignore patterns (gitignore format)
}

// FileChange is one entry in a ChangeSet summary (no line content — counts only).
type FileChange struct {
	Path    string       `json:"path"` // project-relative, forward slashes
	Status  ChangeStatus `json:"status"`
	Added   int          `json:"added"`   // added line count
	Removed int          `json:"removed"` // removed line count
	Binary  bool         `json:"binary"`
	OldSize int64        `json:"oldSize"`
	NewSize int64        `json:"newSize"`
}

// ChangeSet is the summary of all changes vs the baseline for a project.
// It is intentionally lightweight (no per-line content) so it can be emitted
// on every rescan without overwhelming the UI.
type ChangeSet struct {
	ProjectID    string       `json:"projectId"`
	Files        []FileChange `json:"files"`
	TotalAdded   int          `json:"totalAdded"`
	TotalRemoved int          `json:"totalRemoved"`
	TotalFiles   int          `json:"totalFiles"`
	Truncated    bool         `json:"truncated"` // file list capped
	GeneratedAt  string       `json:"generatedAt"`
}

// DiffLine is a single line within a hunk.
type DiffLine struct {
	Kind    string `json:"kind"` // "context" | "add" | "del"
	Content string `json:"content"`
	OldLine int    `json:"oldLine"` // 1-based; 0 when not applicable
	NewLine int    `json:"newLine"`
}

// Hunk is a contiguous block of changed/context lines.
type Hunk struct {
	OldStart int        `json:"oldStart"`
	OldLines int        `json:"oldLines"`
	NewStart int        `json:"newStart"`
	NewLines int        `json:"newLines"`
	Header   string     `json:"header"`
	Lines    []DiffLine `json:"lines"`
}

// FileDiff is the full line-level diff for a single file (fetched lazily).
type FileDiff struct {
	Path      string       `json:"path"`
	Status    ChangeStatus `json:"status"`
	Binary    bool         `json:"binary"`
	Truncated bool         `json:"truncated"` // diff capped; only partial returned
	Added     int          `json:"added"`
	Removed   int          `json:"removed"`
	OldSize   int64        `json:"oldSize"`
	NewSize   int64        `json:"newSize"`
	Message   string       `json:"message"` // e.g. "binary file", "diff too large"
	Hunks     []Hunk       `json:"hunks"`
}

// Version is a saved historical snapshot's metadata.
type Version struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
	Auto      bool   `json:"auto"` // automatically created safety snapshot
	FileCount int    `json:"fileCount"`
	Added     int    `json:"added"`   // +lines vs baseline at creation
	Removed   int    `json:"removed"` // -lines vs baseline at creation
}

// AddProgress is emitted during project addition to let the frontend display a
// progress bar while the initial scan runs.
type AddProgress struct {
	Message string `json:"message"`
	Current int    `json:"current"`
	Total   int    `json:"total"`
}