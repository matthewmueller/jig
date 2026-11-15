package framework

import (
	"github.com/matthewmueller/jig/internal/esbuild"
	"github.com/matthewmueller/jig/internal/pages"
)

// Framework compiles one family of view files. Its plugins are registered
// before jig's own plugins.
type Framework interface {
	// Server plugins load "entry:views" as a module that bundles pages with
	// their error views, along with the views that pages render inside of. It
	// exports views, an object keyed by view, and
	// render(names: string[], props: string[]): string. render takes a stack of
	// views, outermost first, with the JSON props of each, and returns the full
	// HTML document.
	Server(pages []*pages.Page, views ...string) []esbuild.Plugin
	// Client plugins load "entry:<entry>" as a module whose default export
	// renders the page. The page's HTML calls it from an inline module script,
	// which runs again each time navigation swaps the page in. Before a swap,
	// "jig:unload" is dispatched on the document to unmount the page.
	Client() []esbuild.Plugin
}
