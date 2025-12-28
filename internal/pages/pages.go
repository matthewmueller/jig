package pages

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Index struct {
	mu    sync.RWMutex
	pages map[string]*Page
}

func (i *Index) Get(entry string) (*Page, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if page, ok := i.pages[entry]; ok {
		return page, true
	}
	return nil, false
}

func (i *Index) Add(entry string, page *Page) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.pages[entry] = page
}

func (i *Index) List() []*Page {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var pages []*Page
	for _, page := range i.pages {
		pages = append(pages, page)
	}
	return pages
}

type Page struct {
	Entry  string
	Layout string
	Error  string
}

func Find(dir string, entries ...string) (index *Index, err error) {
	index = &Index{
		pages: make(map[string]*Page),
	}
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
		index.Add(entry, page)
	}
	return index, nil
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
		if entryExt != filepath.Ext(name) {
			continue
		}
		switch extless(name) {
		case "layout":
			page.Layout = filepath.Join(dir, name)
		case "error":
			page.Error = filepath.Join(dir, name)
		}
	}
	// Exit if we've reached the root or filled both layout and error
	if dir == "." || (page.Error != "" && page.Layout != "") {
		return page, nil
	}
	// Recurse
	dir = filepath.Dir(dir)
	goto next
}
