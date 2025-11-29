package react

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	gopath "path"
	"path/filepath"
	"strings"

	"github.com/matthewmueller/views/internal/esbuild"
	"github.com/matthewmueller/views/internal/goja"
	"github.com/matthewmueller/views/internal/gotext"
	"golang.org/x/sync/errgroup"
)

func New(log *slog.Logger) *Viewer {
	return &Viewer{
		log,
		goja.New(log),
		&renderCache{},
		&serveCache{},
		".",
		nil,
		!isGoRun(),
		"dist",
		false,
		"/.live",
		true,
	}
}

type Viewer struct {
	log         *slog.Logger
	pool        *goja.Pool
	renderCache *renderCache
	serveCache  *serveCache
	Dir         string
	FS          fs.FS
	Cache       bool
	OutDir      string
	Minify      bool
	LiveRoute   string
	Hydrate     bool
}

//go:embed inject.js
var injectJs string

//go:embed ssr_entry.tsx.gotext
var ssrEntryCode string

var ssrEntryTemplate = gotext.MustCompile("ssr_entry.tsx.gotext", ssrEntryCode)

//go:embed ssr_runtime.tsx
var ssrRuntimeCode string

type ssrEntryData struct {
	Content     string
	Layout      string
	Error       string
	Frames      []string
	LiveRoute   string
	ClientRoute string
}

func (v *Viewer) TransformSSR(paths ...string) error {
	eg := new(errgroup.Group)
	for _, path := range paths {
		eg.Go(func() error {
			return v.transformSSR(path)
		})
	}
	return eg.Wait()
}

func (v *Viewer) transformSSR(path string) error {
	absDir, err := filepath.Abs(v.Dir)
	if err != nil {
		return fmt.Errorf("react: unable to get absolute path: %w", err)
	}
	plugins := []esbuild.Plugin{
		esbuild.Virtual(`^entry\:`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			path := strings.TrimPrefix(args.Path, "entry:")
			contents, err := ssrEntryTemplate.Render(ssrEntryData{
				EntryPath: path,
				LiveRoute: v.LiveRoute,
				Hydrate:   v.Hydrate,
			})
			if err != nil {
				return esbuild.OnLoadResult{}, fmt.Errorf("react: failed to render ssr entry template: %w", err)
			}
			return esbuild.OnLoadResult{
				Loader:     esbuild.LoaderTSX,
				Contents:   &contents,
				ResolveDir: absDir,
			}, nil

		}),
		esbuild.Virtual(`^inject:inject.js$`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			return esbuild.OnLoadResult{
				Contents:   &injectJs,
				ResolveDir: absDir,
			}, nil
		}),
		esbuild.Virtual(`^runtime:ssr_runtime.tsx$`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			return esbuild.OnLoadResult{
				Loader:     esbuild.LoaderTSX,
				Contents:   &ssrRuntimeCode,
				ResolveDir: absDir,
			}, nil
		}),
	}

	// Build the file
	result, err := esbuild.Build(esbuild.BuildOptions{
		EntryPointsAdvanced: []esbuild.EntryPoint{
			{
				InputPath:  "entry:" + path,
				OutputPath: path,
			},
		},
		AbsWorkingDir:     absDir,
		Bundle:            true,
		Splitting:         false,
		Outdir:            v.OutDir,
		Write:             true,
		Format:            esbuild.FormatIIFE,
		Target:            esbuild.ES2015,
		MinifyWhitespace:  v.Minify,
		MinifyIdentifiers: v.Minify,
		MinifySyntax:      v.Minify,
		Platform:          esbuild.PlatformBrowser,
		GlobalName:        fmt.Sprintf("__go_react__[%q]", path),
		Define: map[string]string{
			"process.env.NODE_ENV": "\"production\"",
		},
		JSX:     esbuild.JSXAutomatic,
		Inject:  []string{"inject:inject.js"},
		Plugins: plugins,
	})
	if err != nil {
		return fmt.Errorf("react: failed to bundle ssr: %w", err)
	} else if len(result.OutputFiles) == 0 {
		return fmt.Errorf("react: no output files after ssr bundle")
	}
	for _, warning := range result.Warnings {
		v.log.Warn("esbuild warning", slog.String("message", warning.Text))
	}
	return nil
}

//go:embed dom_entry.jsx.gotext
var domEntryCode string

var domEntryTemplate = gotext.MustCompile("dom_entry.jsx.gotext", domEntryCode)

type domEntryData struct {
	EntryPath string
}

func (v *Viewer) TransformDOM(paths ...string) error {
	absDir, err := filepath.Abs(v.Dir)
	if err != nil {
		return fmt.Errorf("react: unable to get absolute path: %w", err)
	}
	plugins := []esbuild.Plugin{
		esbuild.Virtual(`^entry\:`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			path := strings.TrimPrefix(args.Path, "entry:")
			contents, err := domEntryTemplate.Render(domEntryData{
				EntryPath: path,
			})
			if err != nil {
				return esbuild.OnLoadResult{}, fmt.Errorf("react: failed to render ssr entry template: %w", err)
			}
			return esbuild.OnLoadResult{
				Loader:     esbuild.LoaderTSX,
				Contents:   &contents,
				ResolveDir: absDir,
			}, nil

		}),
		esbuild.Virtual(`^inject:inject.js$`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			return esbuild.OnLoadResult{
				Contents:   &injectJs,
				ResolveDir: absDir,
			}, nil
		}),
	}

	// Prepare the entrypoints
	entrypoints := make([]esbuild.EntryPoint, len(paths))
	for i, path := range paths {
		entrypoints[i] = esbuild.EntryPoint{
			InputPath:  "entry:" + path,
			OutputPath: path,
		}
	}

	// Build the file
	result, err := esbuild.Build(esbuild.BuildOptions{
		EntryPointsAdvanced: entrypoints,
		AbsWorkingDir:       absDir,
		Bundle:              true,
		Splitting:           true,
		Outdir:              v.OutDir,
		Write:               true,
		Format:              esbuild.FormatESModule,
		Target:              esbuild.ES2015,
		MinifyWhitespace:    v.Minify,
		MinifyIdentifiers:   v.Minify,
		MinifySyntax:        v.Minify,
		Platform:            esbuild.PlatformBrowser,
		// GlobalName:          `__go_react__`,
		Define: map[string]string{
			"process.env.NODE_ENV": "\"production\"",
		},
		JSX:     esbuild.JSXAutomatic,
		Inject:  []string{"inject:inject.js"},
		Plugins: plugins,
	})
	if err != nil {
		return fmt.Errorf("react: failed to bundle ssr: %w", err)
	} else if len(result.OutputFiles) == 0 {
		return fmt.Errorf("react: no output files after ssr bundle")
	}
	for _, warning := range result.Warnings {
		v.log.Warn("esbuild warning", slog.String("message", warning.Text))
	}
	return nil
}

func getRenderFunction(vm *goja.Runtime, path string) (goja.Callable, error) {
	reactObj := vm.Get("__go_react__")
	if reactObj == nil {
		return nil, fmt.Errorf("react: unable to load __go_react__ object")
	}
	pathObj := reactObj.ToObject(vm).Get(path)
	if pathObj == nil {
		return nil, fmt.Errorf("react: unable to load __go_react__[%q]", path)
	}
	obj := pathObj.ToObject(vm)
	// Get the render function
	render, ok := goja.AssertFunction(obj.Get("render"))
	if !ok {
		return nil, fmt.Errorf("react: unable to load __go_react__[%q].render function", path)
	}
	return render, nil
}

func (v *Viewer) Build(paths ...string) error {
	if err := v.TransformSSR(paths...); err != nil {
		return fmt.Errorf("react: failed to build: %w", err)
	}
	if err := v.TransformDOM(paths...); err != nil {
		return fmt.Errorf("react: failed to build: %w", err)
	}
	return nil
}

func (v *Viewer) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the live route
			if r.URL.Path == v.LiveRoute {
				v.serveLiveRoute(w, r)
				return
			}
			// Forward along non-JS requests
			if !strings.HasSuffix(r.URL.Path, ".js") {
				next.ServeHTTP(w, r)
				return
			}

			path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), ".js")
			code, ok := v.serveCache.Get(path)
			if !ok {
				distfs := v.FS
				// Transform the live path when we're not caching or when there's no
				// embedded filesystem present. Often a static filesystem will be passed in
				// during development but we still want the current version of the code.
				if !v.Cache || distfs == nil {
					if err := v.TransformDOM(path); err != nil {
						if errors.Is(err, fs.ErrNotExist) {
							next.ServeHTTP(w, r)
							return
						}
						// TODO: javascript-friendly error message
						httpJsError(w, "react: failed to transform for dom: %s", err.Error())
						return
					}
					distfs = os.DirFS(v.Dir)
				}

				// Read the code
				transformedCode, err := fs.ReadFile(distfs, gopath.Join(v.OutDir, path+".js"))
				if err != nil {
					httpJsError(w, "react: failed to read dom file: %s", err.Error())
					return
				}

				// Cache the render function when we're using the static filesystem
				if v.Cache {
					v.serveCache.Set(path, transformedCode)
				}

				code = transformedCode
			}

			// Serve the code
			w.Header().Set("Content-Type", "application/javascript")
			w.Write(code)
		})
	}
}

type View struct {
	Path  string
	Props any
}

func (v *Viewer) Layout(path string, props any) *Renderer {
	return &Renderer{v, &View{path, props}}
}

type Renderer struct {
	v      *Viewer
	layout *View
}

func (v *Viewer) Render(w io.Writer, path string, props any) error {
	renderer := &Renderer{v, nil}
	return renderer.Render(w, path, props)
}

func (r *Renderer) Render(w io.Writer, path string, props any) error {
	vm := r.v.pool.Acquire()
	defer r.v.pool.Release(vm)

	// Render page
	page, err := r.render(vm, path, props)
	if err != nil {
		return fmt.Errorf("react: unable to render page %q: %w", path, err)
	}

	// If there's a layout, render the layout
	// if r.layout != nil {
	// 	layout, err := r.render(vm, path, props)
	// 	if err != nil {
	// 		return fmt.Errorf("react: unable to render page %q: %w", path, err)
	// 	}
	// }
	// fmt.Println("Rendering", r.layout.Path)

	// Write the result
	_, err = io.WriteString(w, page)
	if err != nil {
		return fmt.Errorf("react: unable to write rendered output: %w", err)
	}

	return nil
}

func (r *Renderer) render(vm *goja.Runtime, path string, props any) (string, error) {
	render, ok := r.v.renderCache.Get(path)
	if !ok {
		distfs := r.v.FS
		// Transform the live path when we're not caching or when there's no
		// embedded filesystem present. Often a static filesystem will be passed in
		// during development but we still want the current version of the code.
		if !r.v.Cache || distfs == nil {
			if err := r.v.TransformSSR(path); err != nil {
				return "", fmt.Errorf("react: failed to transform for ssr: %w", err)
			}
			distfs = os.DirFS(r.v.Dir)
		}

		// Read the code
		code, err := fs.ReadFile(distfs, gopath.Join(r.v.OutDir, path+".js"))
		if err != nil {
			return "", fmt.Errorf("react: failed to read ssr file: %w", err)
		}

		// Evaluate the code
		if _, err = vm.RunString(string(code)); err != nil {
			return "", fmt.Errorf("react: failed to evaluate code for %s: %w", path, err)
		}

		// Get the exported render function
		render, err = getRenderFunction(vm, path)
		if err != nil {
			return "", fmt.Errorf("react: failed to get render function for %s: %w", path, err)
		}

		// Cache the render function when we're using the static filesystem
		if r.v.Cache {
			r.v.renderCache.Set(path, render)
		}
	}

	// Call the render function
	result, err := render(goja.Undefined(), vm.ToValue(path), vm.ToValue(props))
	if err != nil {
		return "", fmt.Errorf("react: unable to render the component: %w", err)
	}

	return result.String(), nil
}

// Returns true if the current executable is being run via `go run` or `go test`
func isGoRun() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if strings.Contains(exe, string(filepath.Separator)+"go-build") {
		return true
	}
	// Otherwise, trap interrupts by default
	return false
}

func (v *Viewer) serveLiveRoute(w http.ResponseWriter, r *http.Request) {
	httpJsError(w, "react: livereload not implemented yet")
}

func httpJsError(w http.ResponseWriter, format string, a ...any) {
	w.Header().Set("Content-Type", "application/javascript")
	msg := fmt.Sprintf(format, a...)
	js := fmt.Sprintf(`console.error(%q);`, msg)
	w.Write([]byte(js))
}
