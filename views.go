package views

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"
)

type Renderer interface {
	Render(w io.Writer, path string, props any) error
}

func New(log *slog.Logger) *View {
	return &View{
		log,
		sync.RWMutex{},
		map[string]Renderer{},
	}
}

type View struct {
	log *slog.Logger

	mu        sync.RWMutex
	renderers map[string]Renderer
}

func (v *View) Set(ext string, renderer Renderer) *View {
	v.mu.Lock()
	v.renderers[ext] = renderer
	v.mu.Unlock()
	return v
}

func (v *View) Render(w io.Writer, path string, props any) error {
	v.mu.RLock()
	renderer := v.renderers[filepath.Ext(path)]
	if renderer == nil {
		v.mu.RUnlock()
		return fmt.Errorf("views: no renderer for %q", filepath.Ext(path))
	}
	v.mu.RUnlock()
	return renderer.Render(w, path, props)
}

func (v *View) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
