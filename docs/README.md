<div align="center">

<img src="appicon.png" alt="Sticky Scope" width="112" height="112" />

# Sticky Scope

**An always-on-top desktop sticky note that tracks what changed in any folder — no Git required.**

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev) [![Wails](https://img.shields.io/badge/Wails-v2.12-DF0000?logo=wails&logoColor=white)](https://wails.io) [![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white)](https://vuejs.org) [![Platform](https://img.shields.io/badge/Windows-10%2F11-0078D6?logo=windows&logoColor=white)](#prerequisites) [![Version](https://img.shields.io/badge/version-1.0.1-333333)]()

**English** · [简体中文](README.zh-CN.md)

</div>

Sticky Scope watches a directory you choose and shows you, live, every file that has been **added, modified, or deleted** since a baseline snapshot — with line counts and a full line-by-line diff. It floats on your desktop as a small, pinned note so you always know the shape of your in-progress work, even in folders that aren't a Git repository.

Think of it as a lightweight, version-control-independent "what did I change?" overlay for any folder: drafts, config directories, asset libraries, generated output, or a codebase mid-experiment.

## Why I built this

This started as a small tool to scratch my own itch. While coding with AI assistants, I kept running into cases where the tool's diff panel simply gave up — changes weren't faithfully captured — which left me worrying it had quietly touched files somewhere I wasn't looking. Before tackling each new task I'd end up manually comparing against the previous version, again and again: tedious, and never quite reassuring.

So Sticky Scope took that chore off my hands — a small, always-on window that shows the changes in real time.

And it isn't only for code. Git answers "what changed since my last commit," but plenty of work lives outside a repo. Sticky Scope gives you the same _diff_ feedback for **any** directory, in a window that stays in the corner of your screen instead of a terminal you have to keep re-running.

It's a small, focused tool. If you're the kind of person who only feels at ease once you can *see* the diff, it's for you.

> [!IMPORTANT]
> Sticky Scope is read-only: it only reads your files to compute diffs and **never modifies your project's code**. Its own snapshots live in a separate `.sticky-scope/` directory, and it watches quietly from the side — so it **won't conflict with any AI coding tool**. Run them together with peace of mind.

## Features

- **Baseline snapshots.** Adding a project captures its current state as the baseline, so you start at zero changes and watch the delta grow as you work.
- **Live monitoring.** A recursive filesystem watcher triggers a debounced rescan on any change — create, edit, rename, or delete — and the note updates itself.
- **Added / modified / deleted, at a glance.** Per-file `+`/`−` line counts in a compact list, with a side panel for the full line-level diff.
- **Update the baseline when you're happy.** "Sync" accepts the current state as the new baseline and resets the counter to zero.
- **Pin one note per project.** Open an independent always-on-top note for each folder you're tracking and arrange them around your desktop.
- **Smart ignores.** Each project ships with an editable, gitignore-syntax preset (VCS, `node_modules`, build output, IDE folders, Python/JS caches…), plus your own extra patterns and optional parsing of the project's real `.gitignore` (including nested ones).
- **Binary- and symlink-aware.** Binary files are detected and skip the line diff; symlink targets are tracked; oversized files and huge change sets are capped so the UI stays responsive.
- **Content-addressable storage.** File contents are stored as deduplicated, SHA-256-addressed blobs, with garbage collection of anything no longer referenced.
- **Bilingual UI.** English and 简体中文, switchable at runtime.

## How it works

```
 Add project ──► full scan ──► manifest (path → hash, size, mtime)
                                   │
                                   ├─► contents stored in content-addressable store (deduped)
                                   └─► published as the baseline

 file changes ──► fsnotify watcher ──► debounced trigger ──► single-flight rescan
                                                                  │
                                          baseline ⇄ live manifest diff
                                                                  │
                                          ChangeSet  ──Wails event──►  Vue UI
```

- **Manifest + CAS.** Every scan produces a manifest mapping each file to a content hash, size, mode, and mtime. Contents are written once into a sharded `objects/` store keyed by SHA-256; identical content is never stored twice. The baseline is just a published manifest.
- **Events are only triggers.** The watcher never trusts individual filesystem events. Any event — or a buffer overflow — schedules a debounced (300 ms quiet, 2 s max) full rescan, which is the single source of truth. That makes it immune to lost, coalesced, or overflowed events.
- **Single-flight rescans.** Only one rescan runs per project at a time; changes that arrive mid-scan mark it dirty so it loops once more, so nothing is missed and scans never overlap.
- **Cheap re-scans.** A hash cache keyed by `(path, size, mtime)` skips re-hashing unchanged files. **Deep rescan** clears the cache and recomputes everything from scratch.
- **Lazy diffs.** The change summary carries counts only; the full line diff for a file is computed on demand (via [`go-udiff`](https://github.com/aymanbagabas/go-udiff)) when you open it.

## Tech stack

| Layer        | Tech                                                                 |
| ------------ | ------------------------------------------------------------------- |
| App shell    | [Wails v2](https://wails.io) (Go ↔ WebView)                          |
| Backend      | Go 1.25                                                              |
| Diffing      | [`aymanbagabas/go-udiff`](https://github.com/aymanbagabas/go-udiff) |
| File watching| [`fsnotify`](https://github.com/fsnotify/fsnotify)                  |
| Ignore rules | [`go-git`](https://github.com/go-git/go-git) (gitignore parsing)    |
| Frontend     | Vue 3 · Pinia · vue-i18n · TypeScript · Vite                        |

## Prerequisites

- **[Go](https://go.dev/dl/) 1.25+**
- **[Node.js](https://nodejs.org/) 18+** (with npm) for the frontend
- **[Wails CLI v2](https://wails.io/docs/gettingstarted/installation)**:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

## Getting started

```bash
# from the sticky-scope/ directory

# 1. Run in development mode (hot reload for Go + Vue)
wails dev

# 2. Build a production binary
wails build
# → build/bin/Sticky Scope.exe
```

## Usage

1. **Launch** Sticky Scope. With no project yet, the note shows a welcome card.
2. **Add a project** and pick a directory. Its current contents become the baseline and monitoring starts immediately — the note now reads "No changes."
3. **Work as usual.** As you create, edit, or delete files, the changed-files list updates live with per-file `+`/`−` counts.
4. **Inspect a change.** Click any file to expand the note and view its line-by-line diff.
5. **Sync** to accept the current state as the new baseline and reset to zero.
6. **Pin a note** (the ↗ action on a project) to open an independent always-on-top note — handy for tracking several folders at once.
7. **Tune ignores** in Settings: toggle `.gitignore` parsing, edit the default preset, or add your own patterns. Switch the UI language here too.

> [!TIP]
> Use **Deep rescan** (⟳) if you ever suspect the view is stale — for example after a bulk operation by another tool. It rebuilds the change set from scratch.

### Window & launch flags

The "Pin as new note" action relaunches the executable with a project preloaded; the same flags work from the command line:

| Flag                  | Description                                      |
| --------------------- | ------------------------------------------------ |
| `--project-path=<dir>`| Open the note with this directory preloaded.     |
| `--x=<n>` / `--y=<n>` | Initial window position (screen coordinates).    |

## Configuration & data

Sticky Scope keeps two kinds of state:

- **Global config** — the list of tracked projects and your settings — in
  `…/StickyScope/config.json` under your OS config directory (e.g. `%APPDATA%\StickyScope` on Windows).
- **Per-project data** in a `.sticky-scope/` directory at the root of each tracked folder (like `.git` or `.claude`):
  ```
  <project>/.sticky-scope/
  ├── objects/        # content-addressable blobs (deduped, SHA-256)
  ├── baseline.json   # the current baseline manifest
  └── versions/       # saved snapshots + index.json
  ```

> [!TIP]
> Add `.sticky-scope/` to your project's `.gitignore`. Sticky Scope already ignores its own data directory when scanning, but you typically don't want to commit it.

> [!WARNING]
> Removing a project from Sticky Scope deletes its `.sticky-scope/` directory, including all snapshots and version history. You cannot monitor the app's own data directory or any of its ancestors.

## Project structure

```
sticky-scope/
├── main.go              # Wails entry point, window options, CLI flag parsing
├── app.go               # bound methods exposed to the frontend
├── internal/
│   ├── manager/         # per-project orchestration: watch, rescan, confirm, versions
│   ├── scanner/         # tree walk, ignore matching, hash cache
│   ├── watcher/         # fsnotify wrapper → debounced rescan triggers
│   ├── store/           # content-addressable blob store, manifests, GC
│   ├── baseline/        # baseline + version persistence
│   ├── diff/            # change-set summaries and line-level diffs
│   ├── config/          # on-disk layout, global config, ignore preset
│   ├── model/           # DTOs shared with the frontend (Wails generates TS from these)
│   └── fsutil/          # atomic writes and helpers
└── frontend/            # Vue 3 + Pinia + Vite UI
    └── src/
        ├── components/  # StickyHeader, Compact/Expanded views, DiffViewer, …
        ├── stores/      # projects, changes, ui (Pinia)
        ├── composables/ # Wails event bridge
        └── i18n/        # en / zh
```

> [!TIP]
> The TypeScript types under `frontend/wailsjs/` are **generated** from the Go `model` package and `app.go` — re-run `wails dev`/`wails build` after changing bound methods or DTOs rather than editing them by hand.

## Acknowledgments

This project includes code from the following open-source libraries: Wails, go-udiff, fsnotify, go-git, google/uuid, Vue 3, Pinia, vue-i18n.
