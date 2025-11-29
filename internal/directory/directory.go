package directory

import (
	"path/filepath"
)

// Resolve resolves a series of directories into an absolute path.
func Resolve(dir string, dirs ...string) (abs string, err error) {
	abs, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for _, dir := range dirs {
		if filepath.IsAbs(dir) {
			abs = dir
			continue
		}
		abs = filepath.Join(abs, dir)
	}
	return filepath.EvalSymlinks(abs)
}
