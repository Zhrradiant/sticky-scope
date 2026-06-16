// Package watcher wraps fsnotify. Filesystem events are treated only as triggers:
// any event (or a buffer overflow) schedules a debounced full rescan, which the
// manager performs as the single source of truth. This makes the watcher immune
// to lost/coalesced/overflowed events — it only has to say "something changed".
package watcher

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	debounceDelay = 300 * time.Millisecond // quiet period before firing
	maxWait       = 2 * time.Second        // fire at least this often during a burst
	winBufferSize = 512 * 1024             // larger ReadDirectoryChangesW buffer (Windows)
)

// Watcher recursively watches a directory tree (minus skipped dirs) and emits a
// debounced trigger whenever anything changes.
type Watcher struct {
	root      string
	skipDir   func(rel string) bool
	fsw       *fsnotify.Watcher
	trigger   chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

// New creates a watcher for root. skipDir receives a forward-slash relative path
// and reports whether that directory should not be watched.
func New(root string, skipDir func(rel string) bool) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		root:    filepath.Clean(root),
		skipDir: skipDir,
		fsw:     fsw,
		trigger: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}, nil
}

// Trigger is the channel that receives a value when a rescan is warranted.
func (w *Watcher) Trigger() <-chan struct{} { return w.trigger }

// Start registers watches across the tree and begins the event loop.
func (w *Watcher) Start() error {
	w.addRecursive(w.root)
	go w.loop()
	return nil
}

func (w *Watcher) addRecursive(dir string) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			if err != nil && d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(w.root, path)
		rel = filepath.ToSlash(rel)
		if rel != "." && w.skipDir(rel) {
			return fs.SkipDir
		}
		// AddWith's buffer-size option matters on Windows and is ignored elsewhere.
		_ = w.fsw.AddWith(path, fsnotify.WithBufferSize(winBufferSize))
		return nil
	})
}

func (w *Watcher) signal() {
	select {
	case w.trigger <- struct{}{}:
	default: // a trigger is already pending; the rescan will catch this change too
	}
}

func (w *Watcher) loop() {
	defer func() { _ = recover() }() // WebView2 may trigger panics in the event loop; guard the goroutine

	var timer *time.Timer
	var timerC <-chan time.Time
	pending := false
	var firstAt time.Time

	schedule := func() {
		now := time.Now()
		if !pending {
			pending = true
			firstAt = now
		}
		d := debounceDelay
		if rem := maxWait - now.Sub(firstAt); rem < d {
			d = rem
		}
		if d < 0 {
			d = 0
		}
		if timer == nil {
			timer = time.NewTimer(d)
		} else {
			timer.Reset(d)
		}
		timerC = timer.C
	}

	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			// New directories must be added to the watch set so their contents
			// generate events too.
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					rel, _ := filepath.Rel(w.root, ev.Name)
					rel = filepath.ToSlash(rel)
					if !w.skipDir(rel) {
						w.addRecursive(ev.Name)
					}
				}
			}
			schedule()
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			if err == fsnotify.ErrEventOverflow {
				w.signal() // dropped events — force an immediate rescan
			}
		case <-timerC:
			pending = false
			timerC = nil
			w.signal()
		}
	}
}

// Close stops the event loop and releases the underlying watcher.
func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.done)
		err = w.fsw.Close()
	})
	return err
}