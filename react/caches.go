package react

import (
	"sync"

	"github.com/matthewmueller/views/internal/goja"
)

type renderCache struct {
	sm sync.Map
}

func (c *renderCache) Get(key string) (*goja.Program, bool) {
	value, ok := c.sm.Load(key)
	if !ok {
		return nil, false
	}
	return value.(*goja.Program), true
}

func (c *renderCache) Set(key string, value *goja.Program) {
	c.sm.Store(key, value)
}

type serveCache struct {
	sm sync.Map
}

func (c *serveCache) Get(key string) ([]byte, bool) {
	value, ok := c.sm.Load(key)
	if !ok {
		return nil, false
	}
	return value.([]byte), true
}

func (c *serveCache) Set(key string, value []byte) {
	c.sm.Store(key, value)
}
