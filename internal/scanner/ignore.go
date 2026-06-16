package scanner

import (
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	"sticky-scope/internal/config"
)

// ignorer wraps a go-git gitignore matcher. Patterns are layered: the mandatory
// .sticky-scope self-ignore, then the project's stored patterns (default + extra),
// then any .gitignore files discovered while walking. Later patterns have higher
// priority, matching git semantics.
type ignorer struct {
	patterns     []gitignore.Pattern
	matcher      gitignore.Matcher
	useGitignore bool
}

func newIgnorer(patterns []string, useGitignore bool) *ignorer {
	var ps []gitignore.Pattern

	// Always ignore our own data directory — this cannot be removed by the user.
	ps = append(ps, gitignore.ParsePattern(config.ProjectDirName+"/", nil))

	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		ps = append(ps, gitignore.ParsePattern(p, nil))
	}
	ig := &ignorer{patterns: ps, useGitignore: useGitignore}
	ig.matcher = gitignore.NewMatcher(ps)
	return ig
}

// addGitignore parses a .gitignore located at the directory identified by
// domain (nil for the project root) and folds its patterns into the matcher.
func (ig *ignorer) addGitignore(domain []string, content []byte) {
	if !ig.useGitignore {
		return
	}
	changed := false
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimRight(raw, "\r")
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		ig.patterns = append(ig.patterns, gitignore.ParsePattern(line, domain))
		changed = true
	}
	if changed {
		ig.matcher = gitignore.NewMatcher(ig.patterns)
	}
}

func (ig *ignorer) match(comps []string, isDir bool) bool {
	return ig.matcher.Match(comps, isDir)
}
