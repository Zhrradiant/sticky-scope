package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sticky-scope/internal/manager"
	"sticky-scope/internal/model"
)

// App is the struct Wails binds to the frontend. Every exported method becomes a
// callable from JS/TS. It owns the manager and gates event emission until the
// DOM is ready (the frontend registers its listeners on mount).
type App struct {
	ctx             context.Context
	mgr             *manager.Manager
	mu              sync.Mutex
	ready           bool
	readyCh         chan struct{} // closed once startup finished (mgr set or failed)
	stickyProjectID string        // project ID loaded via --project-path CLI arg
}

// NewApp creates the app shell. The manager is built in startup.
func NewApp() *App { return &App{readyCh: make(chan struct{})} }

// emit forwards a manager event to the frontend, but only once the DOM is ready.
// This avoids the documented EventsEmit/EventsOn data race and dropped events.
func (a *App) emit(event string, data any) {
	a.mu.Lock()
	ready, ctx := a.ready, a.ctx
	a.mu.Unlock()
	if !ready || ctx == nil {
		return
	}
	runtime.EventsEmit(ctx, event, data)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	defer close(a.readyCh)

	mgr, err := manager.New(a.emit)
	if err != nil {
		runtime.LogError(ctx, "failed to initialise manager: "+err.Error())
		return
	}
	a.mgr = mgr

	// Handle sticky-note spawning: position window and auto-load project.
	if stickyPath != "" {
		if stickyX > 0 || stickyY > 0 {
			runtime.WindowSetPosition(ctx, stickyX, stickyY)
		}
		info, err := a.mgr.AddProject(stickyPath)
		if err == nil {
			a.stickyProjectID = info.ID
		}
	}
}

func (a *App) domReady(_ context.Context) {
	a.mu.Lock()
	a.ready = true
	a.mu.Unlock()
}

func (a *App) shutdown(_ context.Context) {
	if a.mgr != nil {
		a.mgr.Shutdown()
	}
}

var errNotReady = errors.New("backend not initialised")

// waitForMgr blocks until the manager has finished initialising in startup, or
// until a timeout elapses. Wails runs OnStartup concurrently with the frontend:
// the DOM can become ready (and call bound methods like ListProjects) before
// manager.New has returned for all projects. Without this wait the very first
// ListProjects call would see a.mgr == nil and return an empty list, causing
// the UI to flash the "add a project" empty state on cold start. The manager
// typically becomes ready within tens of milliseconds, so this wait is
// imperceptible in practice.
func (a *App) waitForMgr() *manager.Manager {
	select {
	case <-a.readyCh:
	case <-time.After(5 * time.Second):
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mgr
}

// ---- bound methods (callable from the frontend) ----

// SelectDirectory opens the native directory picker. Dialogs are unavailable in
// the JS runtime, so this must be a bound Go method.
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择要监控的项目目录 / Select a project directory to monitor",
	})
}

func (a *App) AddProject(path string) (model.ProjectInfo, error) {
	mgr := a.waitForMgr()
	if mgr == nil {
		return model.ProjectInfo{}, errNotReady
	}
	return mgr.AddProject(path)
}

func (a *App) ListProjects() []model.ProjectInfo {
	mgr := a.waitForMgr()
	if mgr == nil {
		return []model.ProjectInfo{}
	}
	return mgr.ListProjects()
}

// StartAllMonitoring launches background watchers for every known project.
// It returns immediately; each watcher registers in its own goroutine. The
// frontend calls this right after ListProjects so the project list renders
// instantly instead of waiting for fsnotify registration across large trees.
func (a *App) StartAllMonitoring() {
	mgr := a.waitForMgr()
	if mgr == nil {
		return
	}
	mgr.StartAllMonitoring()
}

func (a *App) RemoveProject(id string) error {
	if a.mgr == nil {
		return errNotReady
	}
	return a.mgr.RemoveProject(id)
}

func (a *App) GetChanges(id string) (model.ChangeSet, error) {
	mgr := a.waitForMgr()
	if mgr == nil {
		return model.ChangeSet{}, errNotReady
	}
	return mgr.GetChanges(id)
}

func (a *App) GetFileDiff(id, path string) (model.FileDiff, error) {
	if a.mgr == nil {
		return model.FileDiff{}, errNotReady
	}
	return a.mgr.GetFileDiff(id, path)
}

func (a *App) DeepRescan(id string) (model.ChangeSet, error) {
	if a.mgr == nil {
		return model.ChangeSet{}, errNotReady
	}
	return a.mgr.DeepRescan(id)
}

func (a *App) ConfirmAll(id string) error {
	if a.mgr == nil {
		return errNotReady
	}
	return a.mgr.ConfirmAll(id)
}

func (a *App) UpdateIgnore(id string, extraPatterns []string, useGitignore bool) error {
	if a.mgr == nil {
		return errNotReady
	}
	return a.mgr.UpdateIgnore(id, extraPatterns, useGitignore)
}

// GetSettings returns the global, project-independent settings (including the
// shared default ignore patterns).
func (a *App) GetSettings() (model.SettingsInfo, error) {
	if a.mgr == nil {
		return model.SettingsInfo{}, errNotReady
	}
	return a.mgr.Settings(), nil
}

// UpdateDefaultPatterns replaces the global shared default ignore patterns and
// rescan every monitored project.
func (a *App) UpdateDefaultPatterns(patterns []string) error {
	if a.mgr == nil {
		return errNotReady
	}
	return a.mgr.UpdateDefaultPatterns(patterns)
}

// ResetDefaultPatterns restores the global shared default ignore patterns to the
// factory preset and rescan every monitored project.
func (a *App) ResetDefaultPatterns() error {
	if a.mgr == nil {
		return errNotReady
	}
	return a.mgr.ResetDefaultPatterns()
}

// SetCompactMode toggles between compact note (330×480) and expanded (1100×680).
// If the compact window has been manually resized larger than the expanded default,
// entering expanded mode leaves the window size unchanged but still enforces the
// expanded minimum size. Minimum sizes are updated dynamically for each mode.
func (a *App) SetCompactMode(expand bool) {
	if expand {
		w, h := runtime.WindowGetSize(a.ctx)
		if w < 1100 || h < 680 {
			runtime.WindowSetSize(a.ctx, 1100, 680)
		}
		runtime.WindowSetMinSize(a.ctx, 1100, 680)
	} else {
		runtime.WindowSetMinSize(a.ctx, 330, 480)
		runtime.WindowSetSize(a.ctx, 330, 480)
	}
}

// SetCollapsedMode locks the window to a minimal tray size (header + footer only)
// or unlocks it so the normal compact/expanded sizing takes over again.
func (a *App) SetCollapsedMode(collapse bool) {
	if collapse {
		runtime.WindowSetMinSize(a.ctx, 220, 78)
		runtime.WindowSetMaxSize(a.ctx, 220, 78)
		runtime.WindowSetSize(a.ctx, 220, 78)
	} else {
		// Remove the max-size clamp so resize is allowed again.
		// A zero max means "no constraint" in Wails/WebView2.
		runtime.WindowSetMaxSize(a.ctx, 0, 0)
	}
}

// StickyProjectID returns the project ID that was preloaded via --project-path,
// or an empty string when this is the main (non-sticky) process. The frontend
// uses this to select the correct project on startup instead of defaulting to
// the first one in the list. It waits for startup to finish so that a sticky
// note process (whose startup runs a full-tree AddProject) does not race the
// frontend's ListProjects/activeId selection.
func (a *App) StickyProjectID() string {
	a.waitForMgr()
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stickyProjectID
}

// SpawnStickyNote launches a new process for the same project as an independent
// sticky note window. Wails v2 does not support in-process multi-window, so we
// clone the process via os.Executable() and pass the project path + position
// offset as CLI args.
func (a *App) SpawnStickyNote(projectPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	x, y := runtime.WindowGetPosition(a.ctx)
	return exec.Command(exe,
		"--project-path="+projectPath,
		"--x="+strconv.Itoa(x+40),
		"--y="+strconv.Itoa(y+40),
	).Start()
}

// OpenFileLocation opens the system file manager with the given file selected.
func (a *App) OpenFileLocation(id, path string) error {
	if a.mgr == nil {
		return errNotReady
	}
	return a.mgr.OpenFileLocation(id, path)
}