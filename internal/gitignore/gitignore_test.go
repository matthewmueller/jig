package gitignore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
	"github.com/matthewmueller/jig/internal/gitignore"
)

func TestFrom(t *testing.T) {
	is := is.New(t)
	// Nest under a "build" directory to check patterns are relative to dir
	dir := filepath.Join(t.TempDir(), "build", "app")
	is.NoErr(os.MkdirAll(dir, 0755))
	is.NoErr(os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("build/\ndist/\n/secret.txt\n"), 0644))
	ignore := gitignore.From(dir)
	is.True(!ignore(dir))
	is.True(!ignore(filepath.Join(dir, "index.tsx")))
	is.True(!ignore(filepath.Join(dir, "view", "secret.txt")))
	is.True(ignore(filepath.Join(dir, "dist")))
	is.True(ignore(filepath.Join(dir, "dist", "ssr", "index.tsx.js")))
	is.True(ignore(filepath.Join(dir, "secret.txt")))
	is.True(ignore(filepath.Join(dir, "node_modules")))
	is.True(ignore(filepath.Join(dir, "view", "node_modules", "react", "index.js")))
	is.True(ignore(filepath.Join(dir, ".git")))
}

func TestFromMissing(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	ignore := gitignore.From(dir)
	is.True(!ignore(filepath.Join(dir, "index.tsx")))
	is.True(ignore(filepath.Join(dir, "node_modules")))
}
