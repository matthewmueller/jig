package pages

import (
	"os"
	"path/filepath"
	"strings"
)

type Page struct {
	EntryPath  string
	LayoutPath string
	ErrorPath  string
}

func Find(dir string, entries ...string) (pages []*Page, err error) {
	// TODO: async
	for _, entry := range entries {
		switch extless(filepath.Base(entry)) {
		case "layout", "error":
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

func extless(base string) string {
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func find(root, dir, entry string) (*Page, error) {
	// Verify that the entry exists
	if _, err := os.Stat(filepath.Join(root, entry)); err != nil {
		return nil, err
	}
	page := &Page{
		EntryPath: entry,
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
		if entryExt != filepath.Ext(name) {
			continue
		}
		switch extless(name) {
		case "layout":
			page.LayoutPath = filepath.Join(dir, name)
		case "error":
			page.ErrorPath = filepath.Join(dir, name)
		}
	}
	// Exit if we've reached the root or filled both layout and error
	if dir == "." || (page.ErrorPath != "" && page.LayoutPath != "") {
		return page, nil
	}
	// Recurse
	dir = filepath.Dir(dir)
	goto next
}
