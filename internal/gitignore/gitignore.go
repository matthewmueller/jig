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

var defaultIgnore = gitignore.CompileIgnoreLines(alwaysIgnore...).MatchesPath

// TODO: this isn't comprehensive enough, it doesn't handle parent .gitignores
// or the global gitignore in git config.
func From(dir string) (ignore func(path string) bool) {
	code, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return defaultIgnore
	}
	lines := strings.Split(string(code), "\n")
	lines = append(lines, alwaysIgnore...)
	ignorer := gitignore.CompileIgnoreLines(lines...)
	return ignorer.MatchesPath
}
