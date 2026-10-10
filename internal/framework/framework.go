package framework

import (
	"github.com/matthewmueller/jig/internal/esbuild"
	"github.com/matthewmueller/jig/internal/pages"
)

// Framework compiles one family of view files. Its plugins are registered
// before jig's own plugins.
type Framework interface {
	// Server plugins load "entry:<page.Entry>" as a module exporting
	// render(propsJSON: string): string, which returns the full HTML document.
	Server(page *pages.Page) []esbuild.Plugin
	// Client plugins load "entry:<entry>" as a module that hydrates the page.
	Client() []esbuild.Plugin
}
