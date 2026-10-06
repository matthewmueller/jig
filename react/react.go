package react

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	gopath "path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/buildkite/terminal-to-html/v3"
	"github.com/matthewmueller/cjs"
	"github.com/matthewmueller/logs"
	"github.com/matthewmueller/views/internal/esbuild"
	"github.com/matthewmueller/views/internal/gitignore"
	"github.com/matthewmueller/views/internal/goja"
	"github.com/matthewmueller/views/internal/gotext"
	"github.com/matthewmueller/views/internal/livereload"
	"github.com/matthewmueller/views/internal/pages"
	"github.com/matthewmueller/views/internal/watch"
	"golang.org/x/sync/errgroup"
)

type options struct {
	Dir       string
	OutDir    string
	FS        fs.FS
	Watch     bool
	LiveRoute string
	Env       map[string]string
	Define    map[string]string
	// envSet    bool
	Cache     bool
	Log       *slog.Logger
	Minify    bool
	FindPages func(dir string, entries ...string) (Pages, error)
}

type Option func(*options)

func WithDir(baseDir string) Option {
	return func(o *options) {
		o.Dir = baseDir
	}
}

func WithOutDir(outDir string) Option {
	return func(o *options) {
		o.OutDir = outDir
	}
}

func WithFS(buildFS fs.FS) Option {
	return func(o *options) {
		o.FS = buildFS
	}
}

func WithWatch(watch bool) Option {
	return func(o *options) {
		o.Watch = watch
	}
}

func WithLiveRoute(liveRoute string) Option {
	return func(o *options) {
		o.LiveRoute = liveRoute
	}
}

func WithEnv(env map[string]string) Option {
	return func(o *options) {
		if o.Env == nil {
			o.Env = map[string]string{}
		}
		maps.Copy(o.Env, env)
	}
}

func WithCache(cache bool) Option {
	return func(o *options) {
		o.Cache = cache
	}
}

func WithLog(log *slog.Logger) Option {
	return func(o *options) {
		o.Log = log
	}
}

func WithMinify(minify bool) Option {
	return func(o *options) {
		o.Minify = minify
	}
}

func WithFindPages(fn func(dir string, entries ...string) (Pages, error)) Option {
	return func(o *options) {
		o.FindPages = fn
	}
}

func defaultFinder(dir string, entries ...string) (Pages, error) {
	return pages.Find(dir, entries...)
}

func defaults() *options {
	define := map[string]string{
		"process":     "{}",
		"process.env": "{}",
	}
	opts := &options{
		Dir:       ".",
		OutDir:    "dist",
		FS:        nil,
		Define:    define,
		Log:       logs.Default(),
		FindPages: defaultFinder,
	}
	if isProduction() {
		define["process.env.NODE_ENV"] = `"production"`
		opts.Cache = true
		opts.Minify = true
	} else {
		opts.LiveRoute = "/.live"
		opts.Watch = true
		opts.Cache = false
		opts.Minify = false
	}
	return opts
}

func environ() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			env[key] = value
		}
	}
	return env
}

func Load(options ...Option) (viewer *Viewer, err error) {
	// Default options
	opts := defaults()

	// Apply the user configuration
	for _, o := range options {
		o(opts)
	}

	// Define the environment variables. Defaults to the current environment.
	if opts.Env == nil {
		opts.Env = environ()
	}
	for key, value := range opts.Env {
		if _, ok := opts.Define["process.env."+key]; !ok {
			opts.Define["process.env."+key] = strconv.Quote(value)
		}
	}

	// Set the directory to an absolute dir
	opts.Dir, err = filepath.Abs(opts.Dir)
	if err != nil {
		return nil, err
	}

	// Add a root prefix to the live route
	liveReload := livereload.Noop()
	if opts.LiveRoute != "" {
		if !strings.HasPrefix(opts.LiveRoute, "/") {
			return nil, fmt.Errorf("react: live route must start with '/'")
		}
		liveReload = livereload.New(opts.Log, opts.LiveRoute)
	}

	eg := new(errgroup.Group)

	// Start the watcher
	watcher := watch.Noop()
	if opts.Watch {
		outDir := filepath.Join(opts.Dir, opts.OutDir)
		gitignored := gitignore.From(opts.Dir)
		watcher, err = watch.New(opts.Log, opts.Dir, func(path string) bool {
			return path == outDir || strings.HasPrefix(path, outDir+string(filepath.Separator)) || gitignored(path)
		})
		if err != nil {
			return nil, err
		}
	}

	viewer = &Viewer{
		opts,
		goja.New(opts.Log),
		&renderCache{},
		&serveCache{},
		liveReload,
		watcher,
		eg,
	}

	// Start the watcher
	if opts.Watch {
		eg.Go(func() error {
			return watcher.Watch(viewer.onChange)
		})
	}

	return viewer, nil
}

func (v *Viewer) Close() (err error) {
	err = errors.Join(err, v.watcher.Close())
	err = errors.Join(err, v.eg.Wait())
	return err
}

type Page = pages.Page

type Pages interface {
	Get(entry string) (*Page, bool)
	List() []*Page
	Entries() []string
}

type Viewer struct {
	options     *options
	pool        *goja.Pool
	renderCache *renderCache
	serveCache  *serveCache
	liveReload  livereload.LiveReload
	watcher     watch.Watcher
	eg          *errgroup.Group
}

//go:embed inject.js
var injectJs string

//go:embed ssr_entry.tsx.gotext
var ssrEntryCode string

var ssrEntryTemplate = gotext.MustCompile("ssr_entry.tsx.gotext", ssrEntryCode)

//go:embed ssr_runtime.tsx
var ssrRuntimeCode string

type ssrEntryData struct {
	Page *Page
}

// transformSSR builds pages for the server. Outputs are kept in memory.
func (v *Viewer) transformSSR(absDir string, pages Pages) ([]esbuild.OutputFile, error) {
	plugins := []esbuild.Plugin{
		esbuild.Virtual(`^entry\:`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			entryPath := strings.TrimPrefix(args.Path, "entry:")
			page, ok := pages.Get(entryPath)
			if !ok {
				return esbuild.OnLoadResult{}, fmt.Errorf("react: no page found for entry %q", entryPath)
			}
			contents, err := ssrEntryTemplate.Render(ssrEntryData{
				Page: page,
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
		esbuild.JSX("runtime:ssr_runtime.tsx"),
	}
	list := pages.List()
	outputs := make([][]esbuild.OutputFile, len(list))
	eg := new(errgroup.Group)
	for i, page := range list {
		eg.Go(func() (err error) {
			outputs[i], err = v.transformPageSSR(plugins, absDir, page.Entry)
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return slices.Concat(outputs...), nil
}

func (v *Viewer) transformPageSSR(plugins []esbuild.Plugin, absDir string, entry string) ([]esbuild.OutputFile, error) {
	// Build the file
	result, err := esbuild.Build(esbuild.BuildOptions{
		EntryPointsAdvanced: []esbuild.EntryPoint{
			{
				InputPath:  "entry:" + entry,
				OutputPath: entry,
			},
		},
		AbsWorkingDir:     absDir,
		Bundle:            true,
		Splitting:         false,
		Outdir:            filepath.Join(v.options.OutDir, "ssr"),
		Format:            esbuild.FormatIIFE,
		Target:            esbuild.ES2015,
		MinifyWhitespace:  v.options.Minify,
		MinifyIdentifiers: v.options.Minify,
		MinifySyntax:      v.options.Minify,
		Platform:          esbuild.PlatformBrowser,
		GlobalName:        fmt.Sprintf("__go_react__[%q]", entry),
		Define:            v.options.Define,
		JSX:               esbuild.JSXAutomatic,
		Inject:            []string{"inject:inject.js"},
		Plugins:           plugins,
	})
	if err != nil {
		return nil, fmt.Errorf("react: failed to bundle ssr: %w", err)
	} else if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("react: no output files after ssr bundle")
	}
	for _, warning := range result.Warnings {
		v.options.Log.Warn("esbuild warning", slog.String("message", warning.Text))
	}
	return result.OutputFiles, nil
}

//go:embed dom_entry.jsx.gotext
var domEntryCode string

var domEntryTemplate = gotext.MustCompile("dom_entry.jsx.gotext", domEntryCode)

//go:embed dom_runtime.tsx
var domRuntimeCode string

type domEntryData struct {
	Entry string
}

// transformDOM builds entries for the browser. Outputs are kept in memory. Code
// is only split into shared chunks when split is set, since chunks can only be
// served from a full build.
func (v *Viewer) transformDOM(absDir string, split bool, entries ...string) ([]esbuild.OutputFile, error) {
	plugins := []esbuild.Plugin{
		esbuild.Virtual(`^entry\:`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			path := strings.TrimPrefix(args.Path, "entry:")
			contents, err := domEntryTemplate.Render(domEntryData{
				Entry: path,
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
		esbuild.Virtual(`^runtime:dom_runtime.tsx$`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			return esbuild.OnLoadResult{
				Loader:     esbuild.LoaderTSX,
				Contents:   &domRuntimeCode,
				ResolveDir: absDir,
			}, nil
		}),
		esbuild.JSX("runtime:dom_runtime.tsx"),
	}

	// Prepare the entrypoints
	var entrypoints []esbuild.EntryPoint
	for _, entry := range entries {
		entrypoints = append(entrypoints, esbuild.EntryPoint{
			InputPath:  "entry:" + entry,
			OutputPath: entry,
		})
	}
	buildOptions := esbuild.BuildOptions{
		EntryPointsAdvanced: entrypoints,
		AbsWorkingDir:       absDir,
		Bundle:              true,
		Splitting:           split && v.options.Minify,
		Outdir:              filepath.Join(v.options.OutDir, "dom"),
		Format:              esbuild.FormatESModule,
		Target:              esbuild.ES2023,
		MinifyWhitespace:    v.options.Minify,
		MinifyIdentifiers:   v.options.Minify,
		MinifySyntax:        v.options.Minify,
		Platform:            esbuild.PlatformBrowser,
		Define:              v.options.Define,
		JSX:                 esbuild.JSXAutomatic,
		Inject:              []string{"inject:inject.js"},
		Plugins:             plugins,
	}
	// Development build
	if !v.options.Minify {
		buildOptions.Plugins = append(
			buildOptions.Plugins,
			esbuild.ExternalNodeModules("/node_modules/"),
		)
	}
	// Build the file
	result, err := esbuild.Build(buildOptions)
	if err != nil {
		return nil, fmt.Errorf("react: failed to bundle ssr: %w", err)
	} else if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("react: no output files after ssr bundle")
	}
	for _, warning := range result.Warnings {
		v.options.Log.Warn("esbuild warning", slog.String("message", warning.Text))
	}
	return result.OutputFiles, nil
}

func (v *Viewer) transformNodeModule(absDir, module string) ([]byte, error) {
	const prefix = "/node_modules/"

	// Build the file
	result, err := esbuild.Build(esbuild.BuildOptions{
		EntryPoints:   []string{"package:" + module},
		AbsWorkingDir: absDir,
		Bundle:        true,
		Splitting:     false,
		Outfile:       ".",
		Format:        esbuild.FormatESModule,
		Target:        esbuild.ES2023,
		Platform:      esbuild.PlatformBrowser,
		JSX:           esbuild.JSXAutomatic,
		Plugins: []esbuild.Plugin{
			esbuild.Package(),
			esbuild.ExternalNodeModules(prefix, module),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cjs: failed to bundle module %q: %w", module, err)
	} else if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("cjs: expected exactly one output file after bundling module %q", module)
	}
	for _, warning := range result.Warnings {
		v.options.Log.Warn("cjs: bundle warning", slog.String("message", warning.Text))
	}

	contents := result.OutputFiles[0].Contents
	exports, err := cjs.ParseExports(module, string(contents))
	if err != nil {
		return nil, fmt.Errorf("cjs: failed to parse exports of %q: %w", module, err)
	}

	// No exports, return as-is since it's likely an ESM module already
	if len(exports) == 0 {
		return contents, nil
	}

	// Re-bundle with an ESM export wrapper as the entrypoint
	exporter := new(bytes.Buffer)
	if len(exports) > 0 {
		exporter.WriteString("export {\n")
		for _, export := range exports {
			exporter.WriteString(export)
			exporter.WriteString(",\n")
		}
		exporter.WriteString("} from \"")
		exporter.WriteString(module)
		exporter.WriteString("\";\n")
	}
	result, err = esbuild.Build(esbuild.BuildOptions{
		Stdin: &esbuild.StdinOptions{
			Contents:   exporter.String(),
			Loader:     esbuild.LoaderJS,
			Sourcefile: module,
			ResolveDir: ".",
		},
		AbsWorkingDir: absDir,
		Bundle:        true,
		Splitting:     false,
		Outfile:       ".",
		Format:        esbuild.FormatESModule,
		Target:        esbuild.ES2023,
		Platform:      esbuild.PlatformBrowser,
		JSX:           esbuild.JSXAutomatic,
		Plugins: []esbuild.Plugin{
			esbuild.ExternalNodeModules(prefix, module),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cjs: failed to rebundle module %q: %w", module, err)
	} else if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("cjs: expected exactly one output file after rebundling %q", module)
	}
	for _, warning := range result.Warnings {
		v.options.Log.Warn("cjs: rebundle warning", slog.String("message", warning.Text))
	}

	rewritten, err := cjs.RewriteRequires(module, prefix, string(result.OutputFiles[0].Contents))
	if err != nil {
		return nil, fmt.Errorf("cjs: failed to rewrite requires: %w", err)
	}

	return []byte(rewritten), nil
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
	absDir, err := filepath.Abs(v.options.Dir)
	if err != nil {
		return err
	}
	pages, err := v.options.FindPages(absDir, paths...)
	if err != nil {
		return err
	}
	ssrFiles, err := v.transformSSR(absDir, pages)
	if err != nil {
		return fmt.Errorf("react: failed to build: %w", err)
	}
	domFiles, err := v.transformDOM(absDir, true, pages.Entries()...)
	if err != nil {
		return fmt.Errorf("react: failed to build: %w", err)
	}
	for _, file := range slices.Concat(ssrFiles, domFiles) {
		if err := os.MkdirAll(filepath.Dir(file.Path), 0755); err != nil {
			return fmt.Errorf("react: failed to build: %w", err)
		}
		if err := os.WriteFile(file.Path, file.Contents, 0644); err != nil {
			return fmt.Errorf("react: failed to build: %w", err)
		}
	}
	return nil
}

// outputFile finds the contents of the output file ending in path. esbuild
// returns absolute paths with symlinks resolved, so compare the suffix.
func outputFile(files []esbuild.OutputFile, path string) ([]byte, error) {
	suffix := string(filepath.Separator) + filepath.FromSlash(path)
	for _, file := range files {
		if strings.HasSuffix(file.Path, suffix) {
			return file.Contents, nil
		}
	}
	return nil, fmt.Errorf("react: no output file for %q", path)
}

func (v *Viewer) Middleware(next http.Handler) http.Handler {
	return v.liveReload.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the live route
		if v.options.LiveRoute != "" && strings.HasPrefix(r.URL.Path, v.options.LiveRoute) {
			// TODO: Implement live route handling
			http.Error(w, "live route not implemented", http.StatusNotImplemented)
			return
		}

		// Unminified DOM builds import node_modules from this route
		if module, ok := strings.CutPrefix(r.URL.Path, "/node_modules/"); ok && !v.options.Minify {
			if !fs.ValidPath(module) {
				next.ServeHTTP(w, r)
				return
			}
			code, err := v.transformNodeModule(v.options.Dir, module)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Serve the code
			w.Header().Set("Content-Type", "application/javascript")
			w.Write(code)
			return
		}

		// Forward along non-JS requests and invalid paths
		path, ok := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, "/"), ".js")
		if !ok || !fs.ValidPath(path) {
			next.ServeHTTP(w, r)
			return
		}

		code, err := v.serveDOM(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				next.ServeHTTP(w, r)
				return
			}
			consoleError(w, "react: failed to serve %s: %s", r.URL.Path, err.Error())
			return
		}

		// Serve the code
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(code)
	}))
}

// serveDOM returns the browser code for path. Returns fs.ErrNotExist when path
// isn't a page or a built file.
func (v *Viewer) serveDOM(path string) ([]byte, error) {
	if code, ok := v.serveCache.Get(path); ok {
		return code, nil
	}

	// Serve pages and shared chunks from the prebuilt filesystem
	if v.options.Cache && v.options.FS != nil {
		code, err := fs.ReadFile(v.options.FS, gopath.Join(v.options.OutDir, "dom", path+".js"))
		if err != nil {
			return nil, err
		}
		v.serveCache.Set(path, code)
		return code, nil
	}

	// Otherwise build the page on demand
	switch gopath.Ext(path) {
	case ".jsx", ".tsx":
	default:
		return nil, fs.ErrNotExist
	}
	if _, err := os.Stat(filepath.Join(v.options.Dir, path)); err != nil {
		return nil, err
	}
	files, err := v.transformDOM(v.options.Dir, false, path)
	if err != nil {
		return nil, err
	}
	code, err := outputFile(files, gopath.Join(v.options.OutDir, "dom", path+".js"))
	if err != nil {
		return nil, err
	}
	if v.options.Cache {
		v.serveCache.Set(path, code)
	}
	return code, nil
}

func (v *Viewer) Render(w io.Writer, path string, props any) {
	vm := v.pool.Acquire()
	defer v.pool.Release(vm)

	// Render page
	html, err := v.render(vm, path, props)
	if err != nil {
		v.renderError(w, path, fmt.Errorf("react: unable to render page %q: %w", path, err))
		return
	}

	// Write the result
	if _, err = io.WriteString(w, html); err != nil {
		v.options.Log.Error("react: unable to write rendered output", "err", err)
		return
	}
}

func (v *Viewer) render(vm *goja.Runtime, path string, props any) (string, error) {
	// Pooled VMs don't share globals, so evaluate the page in this VM when it's
	// missing. Without caching, always evaluate to pick up the latest code.
	render, err := getRenderFunction(vm, path)
	if err != nil || !v.options.Cache {
		program, err := v.compile(path)
		if err != nil {
			return "", err
		}
		if _, err = vm.RunProgram(program); err != nil {
			return "", fmt.Errorf("react: failed to evaluate code for %s: %w", path, err)
		}
		render, err = getRenderFunction(vm, path)
		if err != nil {
			return "", fmt.Errorf("react: failed to get render function for %s: %w", path, err)
		}
	}

	// Encode props with Go's JSON rules so struct tags are respected, SSR and
	// the browser see the same props, and "<" is escaped for the script tag.
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("react: unable to encode props: %w", err)
	}

	// Call the render function
	result, err := render(goja.Undefined(), vm.ToValue(string(propsJSON)))
	if err != nil {
		return "", fmt.Errorf("react: unable to render the component: %w", err)
	}

	html := result.String()
	html = v.liveReload.Rewrite(html)
	return html, nil
}

// readSSR reads the prebuilt SSR code for path from the filesystem when
// caching, otherwise it builds the page to get the current version of the code.
func (v *Viewer) readSSR(path string) ([]byte, error) {
	if v.options.Cache && v.options.FS != nil {
		code, err := fs.ReadFile(v.options.FS, gopath.Join(v.options.OutDir, "ssr", path+".js"))
		if err != nil {
			return nil, fmt.Errorf("react: failed to read ssr file: %w", err)
		}
		return code, nil
	}
	pages, err := v.options.FindPages(v.options.Dir, path)
	if err != nil {
		return nil, fmt.Errorf("react: failed to find pages: %w", err)
	}
	files, err := v.transformSSR(v.options.Dir, pages)
	if err != nil {
		return nil, fmt.Errorf("react: failed to transform for ssr: %w", err)
	}
	return outputFile(files, gopath.Join(v.options.OutDir, "ssr", path+".js"))
}

// compile the SSR code for path. Programs aren't tied to a VM, so they can be
// cached and shared across the pool.
func (v *Viewer) compile(path string) (*goja.Program, error) {
	if program, ok := v.renderCache.Get(path); ok {
		return program, nil
	}
	code, err := v.readSSR(path)
	if err != nil {
		return nil, err
	}

	program, err := goja.Compile(path, string(code), false)
	if err != nil {
		return nil, fmt.Errorf("react: failed to compile code for %s: %w", path, err)
	}

	if v.options.Cache {
		v.renderCache.Set(path, program)
	}
	return program, nil
}

var errorTemplate = template.Must(template.New("error.gohtml").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>{{ $.Title }}</title>
	<style>
		body {
			font-family: ui-monospace, Menlo, monospace;
		}
		pre {
			white-space: pre-wrap;
		}
	</style>
</head>
<body>
	<h1>{{ .Title }}</h1>
	<pre id="error-message">{{ $.Error }}</pre>
</body>
</html>
`))

type errorData struct {
	Title string
	Error template.HTML
}

func (v *Viewer) renderError(w io.Writer, path string, err error) {
	errorHtml := terminal.Render([]byte(err.Error()))
	htmlBuffer := new(bytes.Buffer)
	data := errorData{
		Title: fmt.Sprintf("Unable to render %s", path),
		Error: template.HTML(errorHtml),
	}
	if err := errorTemplate.Execute(htmlBuffer, data); err != nil {
		v.options.Log.Error("react: unable to write render error page", "err", err)
		return
	}
	html := v.liveReload.Rewrite(htmlBuffer.String())
	if _, err = io.WriteString(w, html); err != nil {
		v.options.Log.Error("react: unable to write rendered error page", "err", err)
		return
	}
}

func filterReloadEvents(events []watch.Event) (filtered []watch.Event) {
	filtered = make([]watch.Event, 0, len(events))
	for _, event := range events {
		if event.Op&watch.Remove != 0 {
			continue
		}
		filtered = append(filtered, event)
	}
	return filtered
}

func formatReloadEvents(events []watch.Event) []byte {
	var data bytes.Buffer
	for i, event := range events {
		if i > 0 {
			data.WriteString(";")
		}
		data.WriteString(event.String())
	}
	return data.Bytes()
}

func (v *Viewer) onChange(events []watch.Event) error {
	events = filterReloadEvents(events)
	if len(events) == 0 {
		return nil
	}

	v.options.Log.Debug("livereload: triggering live reload", "events", events)

	// Publish a livereload event
	eventData := formatReloadEvents(events)
	if err := v.liveReload.Publish(&livereload.Event{
		Type: "reload",
		Data: eventData,
	}); err != nil {
		v.options.Log.Warn("livereload: failed to reload", "err", err, "events", string(eventData))
	}

	return nil
}

func isProduction() bool {
	env, ok := os.LookupEnv("NODE_ENV")
	if ok {
		return strings.ToLower(env) == "production"
	}
	return !isGoRun()
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

func consoleError(w http.ResponseWriter, format string, a ...any) {
	w.Header().Set("Content-Type", "application/javascript")
	msg := fmt.Sprintf(format, a...)
	js := fmt.Sprintf(`console.error(%q);`, msg)
	w.Write([]byte(js))
}
