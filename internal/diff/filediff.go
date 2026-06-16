package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	udiff "github.com/aymanbagabas/go-udiff"

	"sticky-scope/internal/model"
	"sticky-scope/internal/store"
)

// maxDiffLines caps the number of rendered lines in a single file diff so a
// huge/generated file can't freeze the UI. Beyond it the diff is truncated.
const maxDiffLines = 6000

// BuildFileDiff produces the full line-level diff for one file, reading old
// content from the CAS and new content from the project tree. This is the lazy
// per-file payload fetched when the user expands a file.
func BuildFileDiff(root, relPath string, base, live *store.Manifest, st *store.Store) (model.FileDiff, error) {
	be, inBase := base.Files[relPath]
	le, inLive := live.Files[relPath]

	fd := model.FileDiff{Path: relPath, OldSize: be.Size, NewSize: le.Size, Hunks: []model.Hunk{}}
	switch {
	case inLive && !inBase:
		fd.Status = model.StatusAdded
	case inBase && !inLive:
		fd.Status = model.StatusDeleted
	case inBase && inLive:
		fd.Status = model.StatusModified
	default:
		return fd, fmt.Errorf("file is not changed: %s", relPath)
	}

	if be.Symlink != "" || le.Symlink != "" {
		fd.Message = fmt.Sprintf("symlink: %q → %q", be.Symlink, le.Symlink)
		return fd, nil
	}

	var oldContent, newContent []byte
	if inBase {
		oldContent, _ = st.GetBytes(be.Hash)
	}
	if inLive {
		newContent, _ = os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	}

	if IsBinary(oldContent) || IsBinary(newContent) {
		fd.Binary = true
		fd.Message = "binary file"
		return fd, nil
	}

	oldS := string(oldContent)
	newS := string(newContent)
	edits := udiff.Lines(oldS, newS)
	u, err := udiff.ToUnifiedDiff(relPath, relPath, oldS, edits, udiff.DefaultContextLines)
	if err != nil {
		return fd, err
	}

	budget := maxDiffLines
	for _, h := range u.Hunks {
		hunk := model.Hunk{OldStart: h.FromLine, NewStart: h.ToLine}
		oldLn, newLn := h.FromLine, h.ToLine
		for _, l := range h.Lines {
			content := strings.TrimSuffix(l.Content, "\n")
			switch l.Kind {
			case udiff.Delete:
				hunk.Lines = append(hunk.Lines, model.DiffLine{Kind: "del", Content: content, OldLine: oldLn})
				oldLn++
				hunk.OldLines++
				fd.Removed++
			case udiff.Insert:
				hunk.Lines = append(hunk.Lines, model.DiffLine{Kind: "add", Content: content, NewLine: newLn})
				newLn++
				hunk.NewLines++
				fd.Added++
			case udiff.Equal:
				hunk.Lines = append(hunk.Lines, model.DiffLine{Kind: "context", Content: content, OldLine: oldLn, NewLine: newLn})
				oldLn++
				newLn++
				hunk.OldLines++
				hunk.NewLines++
			}
			budget--
			if budget <= 0 {
				break
			}
		}
		hunk.Header = fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
		fd.Hunks = append(fd.Hunks, hunk)
		if budget <= 0 {
			fd.Truncated = true
			fd.Message = fmt.Sprintf("diff too large — showing first %d lines", maxDiffLines)
			break
		}
	}
	return fd, nil
}