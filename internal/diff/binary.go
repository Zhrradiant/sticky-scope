// Package diff turns manifests and file content into the structures the UI
// renders: a lightweight ChangeSet summary (counts only) and, on demand, a full
// line-level FileDiff. Line diffing uses go-udiff (the gopls diff algorithm).
package diff

import "bytes"

// IsBinary reports whether content looks binary. We use the simple, reliable
// "NUL byte in the first 8KB" heuristic (as Mercurial does): UTF-8 source never
// contains NUL, so this avoids the false positives that http.DetectContentType
// produces on many code files.
func IsBinary(content []byte) bool {
	n := len(content)
	if n > 8000 {
		n = 8000
	}
	return bytes.IndexByte(content[:n], 0) >= 0
}

// countLines counts the lines in content (a trailing newline does not add an
// extra empty line).
func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	n := bytes.Count(content, []byte{'\n'})
	if content[len(content)-1] != '\n' {
		n++
	}
	return n
}
