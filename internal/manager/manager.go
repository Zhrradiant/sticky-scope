// Package manager coordinates everything per project: the watcher, debounced
// single-flight rescans, the diff summary cache, and the confirm/version
// operations. It is the only stateful layer the Wails app binds to. It emits
// "changes:updated" and "add:progress" events through the injected EmitFunc.
package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"sticky-scope/internal/baseline"
	"sticky-scope/internal/config"
	"sticky-scope/internal/diff"
	"sticky-scope/internal/model"
	"sticky-scope/internal/scanner"
	"sticky-scope/internal/store"
	"sticky-scope/internal/watcher"
)

// Event names emitted to the frontend.
const (
	EventChanges  = "changes:updated"
	EventAddProg  = "add:progress"
)

// EmitFunc is supplied by the app layer; it forwards events to the frontend
// (guarded so nothing is emitted before the DOM is ready).
type EmitFunc func(event string, data any)

// Manager owns all monitors and the global config.
type Manager struct {
	mu       sync.Mutex
	root     string // global config directory (app data dir)
	cfg      *config.Config
	monitors map[string]*monitor
	emit     EmitFunc
}

// monitor is the per-project runtime state.
type monitor struct {
	mu         sync.Mutex
	meta       config.ProjectMeta
	st         *store.Store
	repo       *baseline.Repo
	hc         *scanner.HashCache
	baseline   *store.Manifest // immutable once published; replaced on confirm
	live       *store.Manifest // last scanned live state
	current    model.ChangeSet // last computed summary
	w          *watcher.Watcher
	stop       chan struct{}
	monitoring bool

	scanning bool // single-flight guard for rescans
	dirty    bool // a change arrived while a scan was in flight
}

// New loads config and reconstructs a monitor for each known project.
func New(emit EmitFunc) (*Manager, error) {
	root, err := config.Root()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	m := &Manager{root: root, cfg: cfg, monitors: map[string]*monitor{}, emit: emit}
	for _, pm := range cfg.Projects {
		if mon, err := m.newMonitor(pm); err == nil {
			m.monitors[pm.ID] = mon
		}
	}
	return m, nil
}

func (m *Manager) newMonitor(pm config.ProjectMeta) (*monitor, error) {
	st, err := store.NewStore(config.ObjectsDir(pm.Path))
	if err != nil {
		return nil, err
	}
	repo := baseline.NewRepo(st, pm.Path,
		config.BaselineFile(pm.Path),
		config.VersionsDir(pm.Path),
		config.VersionIndexFile(pm.Path),
	)
	base, err := repo.LoadBaseline()
	if err != nil {
		base = store.NewManifest()
	}
	return &monitor{
		meta:     pm,
		st:       st,
		repo:     repo,
		hc:       scanner.NewHashCache(),
		baseline: base,
	}, nil
}

func (m *Manager) getMonitor(id string) (*monitor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mon, ok := m.monitors[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	return mon, nil
}

// ---- project lifecycle ----

// AddProject validates a directory, snapshots it as the initial baseline, and
// registers it. The current state becomes the baseline, so the board starts with
// zero changes.
func (m *Manager) AddProject(path string) (model.ProjectInfo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return model.ProjectInfo{}, err
	}
	abs = filepath.Clean(abs)
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return model.ProjectInfo{}, fmt.Errorf("not a directory: %s", abs)
	}
	if err := m.validatePath(abs); err != nil {
		return model.ProjectInfo{}, err
	}

	id := config.ProjectID(abs)
	m.mu.Lock()
	mon, exists := m.monitors[id]
	m.mu.Unlock()
	if exists {
		// Project already loaded from config (e.g. sticky-note process).
		// Ensure monitoring is active — StartMonitoring is idempotent.
		if err := m.StartMonitoring(id); err != nil {
			return model.ProjectInfo{}, err
		}
		return m.projectInfo(mon), nil
	}

	pm := config.ProjectMeta{
		ID:              id,
		Name:            filepath.Base(abs),
		Path:            abs,
		CreatedAt:       time.Now().Format(time.RFC3339),
		DefaultPatterns: config.DefaultPreset(),
		Ignore:          []string{},
		UseGitignore:    true,
	}
	mon, err = m.newMonitor(pm)
	if err != nil {
		return model.ProjectInfo{}, err
	}

	// Phase 1: quick count → frontend shows progress bar
	total := scanner.CountFiles(abs, scanOpts(pm))
	m.emit(EventAddProg, model.AddProgress{Message: "scanning", Current: 0, Total: total})

	// Phase 2: full scan with progress
	live, err := scanner.ScanWithProgress(abs, scanOpts(pm), mon.hc,
		func(cur, _ int) {
			m.emit(EventAddProg, model.AddProgress{Message: "scanning", Current: cur, Total: total})
		})
	if err != nil {
		return model.ProjectInfo{}, err
	}
	m.emit(EventAddProg, model.AddProgress{Message: "storing", Current: total, Total: total})
	if err := mon.repo.SetBaseline(live); err != nil {
		return model.ProjectInfo{}, err
	}
	mon.baseline, _ = mon.repo.LoadBaseline()
	mon.live = mon.baseline
	mon.current = emptyChangeSet(id)

	m.mu.Lock()
	m.monitors[id] = mon
	m.cfg.Projects = append(m.cfg.Projects, pm)
	err = config.Save(m.cfg)
	m.mu.Unlock()
	if err != nil {
		return model.ProjectInfo{}, err
	}
	// Auto-start monitoring immediately. Emits initial changes via rescanAndEmit.
	if err := m.StartMonitoring(id); err != nil {
		return model.ProjectInfo{}, err
	}
	m.emit(EventAddProg, model.AddProgress{Message: "done", Current: total, Total: total})
	return m.projectInfo(mon), nil
}

// RemoveProject stops monitoring, forgets the project, and deletes its data dir.
func (m *Manager) RemoveProject(id string) error {
	m.mu.Lock()
	mon, ok := m.monitors[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("project not found")
	}
	delete(m.monitors, id)
	out := m.cfg.Projects[:0]
	for _, pm := range m.cfg.Projects {
		if pm.ID != id {
			out = append(out, pm)
		}
	}
	m.cfg.Projects = out
	err := config.Save(m.cfg)
	m.mu.Unlock()

	m.stopMonitor(mon)
	_ = os.RemoveAll(config.ProjectDir(mon.meta.Path))
	return err
}

// ListProjects returns all projects in config order.
// Also starts monitoring for every loaded project so that the watchers are
// active immediately on cold restart (monitors are created in New but watchers
// are not started until the frontend signals readiness via this call).
func (m *Manager) ListProjects() []model.ProjectInfo {
	m.mu.Lock()
	out := []model.ProjectInfo{}
	ids := make([]string, 0, len(m.cfg.Projects))
	for _, pm := range m.cfg.Projects {
		if mon, ok := m.monitors[pm.ID]; ok {
			out = append(out, m.projectInfo(mon))
			ids = append(ids, pm.ID)
		}
	}
	m.mu.Unlock()

	// Start watchers now that the frontend is ready. Idempotent — no-op for
	// already-running projects (e.g. sticky-note where AddProject already
	// started monitoring).
	for _, id := range ids {
		_ = m.StartMonitoring(id)
	}
	return out
}

// ---- monitoring ----

// StartMonitoring begins watching a project and performs an initial rescan.
func (m *Manager) StartMonitoring(id string) error {
	mon, err := m.getMonitor(id)
	if err != nil {
		return err
	}
	mon.mu.Lock()
	if mon.monitoring {
		mon.mu.Unlock()
		return nil
	}
	if !dirExists(mon.meta.Path) {
		mon.mu.Unlock()
		return fmt.Errorf("project directory unavailable")
	}
	w, err := watcher.New(mon.meta.Path, buildSkipDir(mon.meta.DefaultPatterns, mon.meta.Ignore))
	if err != nil {
		mon.mu.Unlock()
		return err
	}
	mon.w = w
	mon.stop = make(chan struct{})
	mon.monitoring = true
	stop := mon.stop
	mon.mu.Unlock()

	if err := w.Start(); err != nil {
		// Roll back so a future StartMonitoring call can retry.
		mon.mu.Lock()
		mon.monitoring = false
		mon.w = nil
		mon.stop = nil
		mon.mu.Unlock()
		return err
	}
	go m.watchLoop(mon, w, stop)
	go m.rescanAndEmit(mon)
	return nil
}

// StopMonitoring stops watching a project (its baseline/versions are untouched).
func (m *Manager) StopMonitoring(id string) error {
	mon, err := m.getMonitor(id)
	if err != nil {
		return err
	}
	m.stopMonitor(mon)
	return nil
}

func (m *Manager) stopMonitor(mon *monitor) {
	mon.mu.Lock()
	if mon.w != nil {
		_ = mon.w.Close()
		mon.w = nil
	}
	if mon.stop != nil {
		close(mon.stop)
		mon.stop = nil
	}
	mon.monitoring = false
	mon.mu.Unlock()
}

func (m *Manager) watchLoop(mon *monitor, w *watcher.Watcher, stop chan struct{}) {
	defer func() { _ = recover() }()
	for {
		select {
		case <-stop:
			return
		case <-w.Trigger():
			m.rescanAndEmit(mon)
		}
	}
}

// rescanAndEmit runs a rescan with single-flight semantics: if one is already in
// progress, it marks the monitor dirty and the running scan loops once more when
// it finishes, so no trigger is ever lost and scans never overlap.
func (m *Manager) rescanAndEmit(mon *monitor) {
	mon.mu.Lock()
	if mon.scanning {
		mon.dirty = true
		mon.mu.Unlock()
		return
	}
	mon.scanning = true
	mon.mu.Unlock()

	for {
		cs, err := m.doRescan(mon)

		mon.mu.Lock()
		if err == nil {
			mon.current = cs
		}
		again := mon.dirty
		mon.dirty = false
		if !again {
			mon.scanning = false
		}
		mon.mu.Unlock()

		if err == nil {
			m.emit(EventChanges, cs)
		}
		if !again {
			return
		}
	}
}

func (m *Manager) doRescan(mon *monitor) (model.ChangeSet, error) {
	mon.mu.Lock()
	meta := mon.meta
	base := mon.baseline
	hc := mon.hc
	st := mon.st
	mon.mu.Unlock()

	if !dirExists(meta.Path) {
		return model.ChangeSet{}, fmt.Errorf("unavailable")
	}
	live, err := scanner.Scan(meta.Path, scanOpts(meta), hc)
	if err != nil {
		return model.ChangeSet{}, err
	}
	cs := diff.ManifestDiff(meta.ID, meta.Path, base, live, st)

	mon.mu.Lock()
	mon.live = live
	mon.mu.Unlock()
	return cs, nil
}

// ---- queries ----

// GetChanges returns the cached summary, computing one if none exists yet.
func (m *Manager) GetChanges(id string) (model.ChangeSet, error) {
	mon, err := m.getMonitor(id)
	if err != nil {
		return model.ChangeSet{}, err
	}
	mon.mu.Lock()
	cur := mon.current
	hasLive := mon.live != nil
	mon.mu.Unlock()
	if hasLive {
		return cur, nil
	}
	cs, err := m.doRescan(mon)
	if err != nil {
		return emptyChangeSet(id), nil
	}
	mon.mu.Lock()
	mon.current = cs
	mon.mu.Unlock()
	return cs, nil
}

// GetFileDiff returns the full line-level diff for one file (lazy fetch).
func (m *Manager) GetFileDiff(id, path string) (model.FileDiff, error) {
	mon, err := m.getMonitor(id)
	if err != nil {
		return model.FileDiff{}, err
	}
	mon.mu.Lock()
	base, live, root, st := mon.baseline, mon.live, mon.meta.Path, mon.st
	mon.mu.Unlock()
	if live == nil {
		if _, err := m.doRescan(mon); err != nil {
			return model.FileDiff{}, err
		}
		mon.mu.Lock()
		live = mon.live
		mon.mu.Unlock()
	}
	if live == nil {
		return model.FileDiff{}, fmt.Errorf("no live state")
	}
	return diff.BuildFileDiff(root, path, base, live, st)
}

// DeepRescan clears the hash cache and recomputes everything from scratch.
func (m *Manager) DeepRescan(id string) (model.ChangeSet, error) {
	mon, err := m.getMonitor(id)
	if err != nil {
		return model.ChangeSet{}, err
	}
	mon.hc.Clear()
	cs, err := m.doRescan(mon)
	if err != nil {
		return model.ChangeSet{}, err
	}
	mon.mu.Lock()
	mon.current = cs
	mon.mu.Unlock()
	m.emit(EventChanges, cs)
	return cs, nil
}

// ---- confirm / versions ----

// ConfirmAll accepts every change: the live state becomes the new baseline.
func (m *Manager) ConfirmAll(id string) error {
	mon, err := m.getMonitor(id)
	if err != nil {
		return err
	}
	live, err := m.scanLive(mon)
	if err != nil {
		return err
	}
	if err := mon.repo.SetBaseline(live); err != nil {
		return err
	}
	m.reloadBaseline(mon)
	_ = mon.repo.GC()
	m.rescanAndEmit(mon)
	return nil
}

// SaveVersion snapshots the current live state as a named version.
func (m *Manager) SaveVersion(id, name, message string) (model.Version, error) {
	mon, err := m.getMonitor(id)
	if err != nil {
		return model.Version{}, err
	}
	live, err := m.scanLive(mon)
	if err != nil {
		return model.Version{}, err
	}
	mon.mu.Lock()
	base, root, st := mon.baseline, mon.meta.Path, mon.st
	mon.mu.Unlock()
	cs := diff.ManifestDiff(id, root, base, live, st)
	if strings.TrimSpace(name) == "" {
		name = "版本 / Version " + time.Now().Format("2006-01-02 15:04:05")
	}
	return mon.repo.SaveVersion(live, name, message, false, cs.TotalAdded, cs.TotalRemoved)
}

// ListVersions returns saved versions, newest first.
func (m *Manager) ListVersions(id string) ([]model.Version, error) {
	mon, err := m.getMonitor(id)
	if err != nil {
		return nil, err
	}
	return mon.repo.ListVersions()
}

// DeleteVersion removes a saved version and GCs unreferenced blobs.
func (m *Manager) DeleteVersion(id, vid string) error {
	mon, err := m.getMonitor(id)
	if err != nil {
		return err
	}
	if err := mon.repo.DeleteVersion(vid); err != nil {
		return err
	}
	_ = mon.repo.GC()
	return nil
}

// UpdateIgnore changes a project's ignore configuration and rescans.
func (m *Manager) UpdateIgnore(id string, defaultPatterns []string, extraPatterns []string, useGitignore bool) error {
	mon, err := m.getMonitor(id)
	if err != nil {
		return err
	}
	mon.mu.Lock()
	mon.meta.DefaultPatterns = defaultPatterns
	mon.meta.Ignore = extraPatterns
	mon.meta.UseGitignore = useGitignore
	mon.mu.Unlock()

	m.mu.Lock()
	for i := range m.cfg.Projects {
		if m.cfg.Projects[i].ID == id {
			m.cfg.Projects[i].DefaultPatterns = defaultPatterns
			m.cfg.Projects[i].Ignore = extraPatterns
			m.cfg.Projects[i].UseGitignore = useGitignore
		}
	}
	err = config.Save(m.cfg)
	m.mu.Unlock()

	mon.hc.Clear()
	// Restart watcher so the skipDir predicate picks up the new patterns.
	// Otherwise directories newly removed from the ignore list would not be
	// watched and changes inside them would be invisible until a deep rescan.
	m.stopMonitor(mon)
	_ = m.StartMonitoring(id)
	// StartMonitoring kicks off an async rescan; also trigger one synchronously
	// so the frontend receives an immediate change event.
	m.rescanAndEmit(mon)
	return err
}

// Shutdown stops all watchers; called from the Wails OnShutdown hook.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	mons := make([]*monitor, 0, len(m.monitors))
	for _, mon := range m.monitors {
		mons = append(mons, mon)
	}
	m.mu.Unlock()
	for _, mon := range mons {
		m.stopMonitor(mon)
	}
}

// ---- helpers ----

func (m *Manager) reloadBaseline(mon *monitor) {
	nb, err := mon.repo.LoadBaseline()
	if err != nil {
		return
	}
	mon.mu.Lock()
	mon.baseline = nb
	mon.mu.Unlock()
}

func (m *Manager) scanLive(mon *monitor) (*store.Manifest, error) {
	mon.mu.Lock()
	meta := mon.meta
	hc := mon.hc
	mon.mu.Unlock()
	return scanner.Scan(meta.Path, scanOpts(meta), hc)
}

func (m *Manager) projectInfo(mon *monitor) model.ProjectInfo {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	return model.ProjectInfo{
		ID:              mon.meta.ID,
		Name:            mon.meta.Name,
		Path:            mon.meta.Path,
		CreatedAt:       mon.meta.CreatedAt,
		Available:       dirExists(mon.meta.Path),
		DefaultPatterns: mon.meta.DefaultPatterns,
		Ignore:          mon.meta.Ignore,
		UseGitignore:    mon.meta.UseGitignore,
	}
}

func (m *Manager) validatePath(abs string) error {
	a, c := abs, filepath.Clean(m.root)
	if runtime.GOOS == "windows" {
		a, c = strings.ToLower(a), strings.ToLower(c)
	}
	if isUnder(a, c) || isUnder(c, a) {
		return errors.New("不能监控应用数据目录或其上级目录 / cannot monitor the app data directory or its ancestors")
	}
	return nil
}

func scanOpts(pm config.ProjectMeta) scanner.Options {
	// Merge default + extra patterns. Defaults go first so extra patterns can
	// override them (later patterns have higher priority in gitignore semantics).
	merged := make([]string, 0, len(pm.DefaultPatterns)+len(pm.Ignore))
	merged = append(merged, pm.DefaultPatterns...)
	merged = append(merged, pm.Ignore...)
	return scanner.Options{Patterns: merged, UseGitignore: pm.UseGitignore}
}

func emptyChangeSet(id string) model.ChangeSet {
	return model.ChangeSet{ProjectID: id, Files: []model.FileChange{}, GeneratedAt: time.Now().Format(time.RFC3339)}
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}

// buildSkipDir returns the predicate the watcher uses to avoid registering
// watches inside ignored directories (the rescan applies the full ignore rules;
// this just keeps the watch set small and cheap).
func buildSkipDir(defaultPatterns, extraPatterns []string) func(rel string) bool {
	dirset := map[string]struct{}{}
	for _, p := range defaultPatterns {
		p = strings.TrimSpace(strings.TrimSuffix(p, "/"))
		if p == "" || strings.ContainsAny(p, "*?/[") {
			continue
		}
		dirset[p] = struct{}{}
	}
	for _, p := range extraPatterns {
		p = strings.TrimSpace(strings.TrimSuffix(p, "/"))
		if p == "" || strings.ContainsAny(p, "*?/[") {
			continue
		}
		dirset[p] = struct{}{}
	}
	// Always skip .sticky-scope — this is a mandatory, non-removable ignore.
	dirset[config.ProjectDirName] = struct{}{}
	return func(rel string) bool {
		for _, comp := range strings.Split(rel, "/") {
			if _, ok := dirset[comp]; ok {
				return true
			}
		}
		return false
	}
}