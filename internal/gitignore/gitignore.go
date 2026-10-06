package gitignore

import (
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

var alwaysIgnore = []string{
	"node_modules",
	".git",
	".DS_Store",
}

// From returns a matcher for absolute paths within dir.
//
// TODO: this isn't comprehensive enough, it doesn't handle parent .gitignores
// or the global gitignore in git config.
func From(dir string) (ignore func(path string) bool) {
	lines := alwaysIgnore
	if code, err := os.ReadFile(filepath.Join(dir, ".gitignore")); err == nil {
		lines = append(strings.Split(string(code), "\n"), alwaysIgnore...)
	}
	ignorer := gitignore.CompileIgnoreLines(lines...)
	return func(path string) bool {
		// Patterns are relative to dir
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return false
		}
		// Also try as a directory, so "dist/" matches the dist directory itself
		return ignorer.MatchesPath(rel) || ignorer.MatchesPath(rel+"/")
	}
}
