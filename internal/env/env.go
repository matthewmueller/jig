package env

import (
	"os"
	"path/filepath"
	"strings"
)

// IsGoRun reports whether the current process is running via "go run".
func IsGoRun() (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	if strings.Contains(exe, string(filepath.Separator)+"go-build") {
		return true, nil
	}
	return false, nil
}
