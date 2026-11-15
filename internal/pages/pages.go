package pages

import (
	"os"
	"path/filepath"
	"strings"
)

type Page struct {
	Entry string
	Error string
}

// Find the pages for entries in dir, in order, each with its nearest error
// view. Layout and error entries aren't pages, so they're skipped.
func Find(dir string, entries ...string) (pages []*Page, err error) {
	// TODO: async
	for _, entry := range entries {
		if IsLayout(entry) || extless(entry) == "error" {
			continue
		}
		page, err := find(dir, filepath.Dir(entry), entry)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	return pages, nil
}

// IsLayout reports whether entry is a layout, which only renders on the server
func IsLayout(entry string) bool {
	return extless(entry) == "layout"
}

func extless(entry string) string {
	base := filepath.Base(entry)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func find(root, dir, entry string) (*Page, error) {
	page := &Page{
		Entry: entry,
	}
	entryExt := filepath.Ext(entry)
next:
	// Read the directory
	des, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		return nil, err
	}
	for _, de := range des {
		name := de.Name()
		if entryExt == filepath.Ext(name) && extless(name) == "error" {
			page.Error = filepath.Join(dir, name)
		}
	}
	// Exit if we've reached the root or found the nearest error view
	if dir == "." || page.Error != "" {
		return page, nil
	}
	// Recurse
	dir = filepath.Dir(dir)
	goto next
}
