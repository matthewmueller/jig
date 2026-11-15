package react

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/matthewmueller/jig/internal/esbuild"
	"github.com/matthewmueller/jig/internal/framework"
	"github.com/matthewmueller/jig/internal/gotext"
	"github.com/matthewmueller/jig/internal/pages"
)

//go:embed ssr_entry.tsx.gotext
var ssrEntryCode string

var ssrEntryTemplate = gotext.MustCompile("ssr_entry.tsx.gotext", ssrEntryCode)

//go:embed ssr_runtime.tsx
var ssrRuntimeCode string

//go:embed dom_entry.jsx.gotext
var domEntryCode string

var domEntryTemplate = gotext.MustCompile("dom_entry.jsx.gotext", domEntryCode)

//go:embed dom_runtime.tsx
var domRuntimeCode string

// New React framework for the views in dir
func New(dir string) *React {
	return &React{dir}
}

// React renders .jsx and .tsx views
type React struct {
	dir string
}

var _ framework.Framework = (*React)(nil)

type ssrEntryData struct {
	Pages []*pages.Page
	Views []string
}

// Server plugins render stacks of views to HTML with react-dom/server
func (r *React) Server(pages []*pages.Page, views ...string) []esbuild.Plugin {
	return []esbuild.Plugin{
		esbuild.Virtual(`^entry\:`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			contents, err := ssrEntryTemplate.Render(ssrEntryData{
				Pages: pages,
				Views: views,
			})
			if err != nil {
				return esbuild.OnLoadResult{}, fmt.Errorf("react: failed to render ssr entry template: %w", err)
			}
			return esbuild.OnLoadResult{
				Loader:     esbuild.LoaderTSX,
				Contents:   &contents,
				ResolveDir: r.dir,
			}, nil
		}),
		esbuild.Virtual(`^runtime:ssr_runtime.tsx$`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			return esbuild.OnLoadResult{
				Loader:     esbuild.LoaderTSX,
				Contents:   &ssrRuntimeCode,
				ResolveDir: r.dir,
			}, nil
		}),
		esbuild.JSX("runtime:ssr_runtime.tsx"),
	}
}

type domEntryData struct {
	Entry string
}

// Client plugins hydrate pages with react-dom/client
func (r *React) Client() []esbuild.Plugin {
	return []esbuild.Plugin{
		esbuild.Virtual(`^entry\:`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			contents, err := domEntryTemplate.Render(domEntryData{
				Entry: strings.TrimPrefix(args.Path, "entry:"),
			})
			if err != nil {
				return esbuild.OnLoadResult{}, fmt.Errorf("react: failed to render dom entry template: %w", err)
			}
			return esbuild.OnLoadResult{
				Loader:     esbuild.LoaderTSX,
				Contents:   &contents,
				ResolveDir: r.dir,
			}, nil
		}),
		esbuild.Virtual(`^runtime:dom_runtime.tsx$`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			return esbuild.OnLoadResult{
				Loader:     esbuild.LoaderTSX,
				Contents:   &domRuntimeCode,
				ResolveDir: r.dir,
			}, nil
		}),
		esbuild.JSX("runtime:dom_runtime.tsx"),
	}
}
