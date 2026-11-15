package jig

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	gopath "path"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/matthewmueller/cjs"
	"github.com/matthewmueller/jig/internal/esbuild"
	"github.com/matthewmueller/jig/internal/framework"
	"github.com/matthewmueller/jig/internal/framework/react"
	"github.com/matthewmueller/jig/internal/gitignore"
	"github.com/matthewmueller/jig/internal/goja"
	"github.com/matthewmueller/jig/internal/livereload"
	"github.com/matthewmueller/jig/internal/pages"
	"github.com/matthewmueller/jig/internal/watch"
	"github.com/matthewmueller/logs"
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
	Cache  bool
	Log    *slog.Logger
	Minify bool
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

func defaults() *options {
	define := map[string]string{
		"process":     "{}",
		"process.env": "{}",
	}
	opts := &options{
		Dir:    ".",
		OutDir: "dist",
		FS:     nil,
		Define: define,
		Log:    logs.Default(),
	}
	if isProduction() {
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

	// Minified builds are production builds. This takes precedence over the
	// environment, so React's development build isn't shipped when building
	// from a shell with NODE_ENV unset or set to development.
	if opts.Minify {
		opts.Define["process.env.NODE_ENV"] = `"production"`
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
			return nil, fmt.Errorf("jig: live route must start with '/'")
		}
		liveReload = livereload.New(opts.Log, opts.LiveRoute)
	}

	eg := new(errgroup.Group)

	// Start the watcher
	watcher := watch.Noop()
	if opts.Watch {
		watcher, err = watch.New(opts.Log, opts.Dir, ignorer(opts.Dir, opts.OutDir))
		if err != nil {
			return nil, err
		}
	}

	// Frameworks by the extensions of the views they render
	r := react.New(opts.Dir)
	frameworks := map[string]framework.Framework{
		".jsx": r,
		".tsx": r,
	}

	viewer = &Viewer{
		opts,
		frameworks,
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

// ignorer ignores the out directory and gitignored paths within dir
func ignorer(dir, outDir string) func(path string) bool {
	outDir = filepath.Join(dir, outDir)
	gitignored := gitignore.From(dir)
	return func(path string) bool {
		return path == outDir || strings.HasPrefix(path, outDir+string(filepath.Separator)) || gitignored(path)
	}
}

// View is a view file, relative to the viewer's directory, and the props to
// render it with. It's an alias of an unnamed type, so other packages can
// declare the same type and call Render without importing jig.
type View = struct {
	Name  string
	Props any
}

type Viewer struct {
	options     *options
	frameworks  map[string]framework.Framework
	pool        *goja.Pool
	renderCache *renderCache
	serveCache  *serveCache
	liveReload  livereload.LiveReload
	watcher     watch.Watcher
	eg          *errgroup.Group
}

// Views import the types generated for Go files from the Go files themselves,
// like "./props.go". Type-only imports can still leave a side-effect import
// behind, so load Go files as empty modules.
var loaders = map[string]esbuild.Loader{
	".go": esbuild.LoaderEmpty,
}

// inject.js polyfills browser APIs that goja lacks for server rendering
//
//go:embed inject.js
var injectJs string

// navigate.js swaps in pages when links are clicked, instead of loading them
//
//go:embed navigate.js
var navigateJs string

// ssrBundle names the bundle of views for the server
const ssrBundle = "views"

// transformSSR bundles pages for the server, along with the views that pages
// render inside of. The bundle evaluates into __jig__[key]. Outputs are kept
// in memory.
func (v *Viewer) transformSSR(fw framework.Framework, key string, pages []*pages.Page, views ...string) ([]esbuild.OutputFile, error) {
	plugins := append(fw.Server(pages, views...),
		esbuild.Virtual(`^inject:inject.js$`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
			return esbuild.OnLoadResult{
				Contents:   &injectJs,
				ResolveDir: v.options.Dir,
			}, nil
		}),
	)
	// Build the file
	result, err := esbuild.Build(esbuild.BuildOptions{
		EntryPointsAdvanced: []esbuild.EntryPoint{
			{
				InputPath:  "entry:" + ssrBundle,
				OutputPath: ssrBundle,
			},
		},
		AbsWorkingDir:     v.options.Dir,
		Bundle:            true,
		Splitting:         false,
		Outdir:            filepath.Join(v.options.OutDir, "ssr"),
		Format:            esbuild.FormatIIFE,
		Target:            esbuild.ES2015,
		MinifyWhitespace:  v.options.Minify,
		MinifyIdentifiers: v.options.Minify,
		MinifySyntax:      v.options.Minify,
		Platform:          esbuild.PlatformBrowser,
		GlobalName:        fmt.Sprintf("__jig__[%q]", key),
		Define:            v.options.Define,
		JSX:               esbuild.JSXAutomatic,
		Loader:            loaders,
		Inject:            []string{"inject:inject.js"},
		Plugins:           plugins,
	})
	if err != nil {
		return nil, fmt.Errorf("jig: failed to bundle ssr: %w", err)
	} else if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("jig: no output files after ssr bundle")
	}
	for _, warning := range result.Warnings {
		v.options.Log.Warn("esbuild warning", slog.String("message", warning.Text))
	}
	return result.OutputFiles, nil
}

// transformDOM builds entries for the browser. Outputs are kept in memory. Code
// is only split into shared chunks when split is set, since chunks can only be
// served from a full build.
func (v *Viewer) transformDOM(fw framework.Framework, split bool, entries ...string) ([]esbuild.OutputFile, error) {
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
		AbsWorkingDir:       v.options.Dir,
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
		Loader:              loaders,
		Inject:              []string{"inject:navigate.js"},
		Plugins: append(fw.Client(),
			esbuild.Virtual(`^inject:navigate.js$`, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
				return esbuild.OnLoadResult{
					Contents:   &navigateJs,
					ResolveDir: v.options.Dir,
				}, nil
			}),
		),
	}
	// Development build. Added after the framework's plugins, since its
	// runtime modules also look like bare imports.
	if !v.options.Minify {
		buildOptions.Plugins = append(
			buildOptions.Plugins,
			esbuild.ExternalNodeModules("/node_modules/"),
		)
	}
	// Build the file
	result, err := esbuild.Build(buildOptions)
	if err != nil {
		return nil, fmt.Errorf("jig: failed to bundle dom: %w", err)
	} else if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("jig: no output files after dom bundle")
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
		return nil, fmt.Errorf("jig: failed to bundle module %q: %w", module, err)
	} else if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("jig: expected exactly one output file after bundling module %q", module)
	}
	for _, warning := range result.Warnings {
		v.options.Log.Warn("esbuild warning", slog.String("module", module), slog.String("message", warning.Text))
	}

	contents := result.OutputFiles[0].Contents
	exports, err := cjs.ParseExports(module, string(contents))
	if err != nil {
		return nil, fmt.Errorf("jig: failed to parse exports of %q: %w", module, err)
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
		return nil, fmt.Errorf("jig: failed to rebundle module %q: %w", module, err)
	} else if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("jig: expected exactly one output file after rebundling %q", module)
	}
	for _, warning := range result.Warnings {
		v.options.Log.Warn("esbuild warning", slog.String("module", module), slog.String("message", warning.Text))
	}

	rewritten, err := cjs.RewriteRequires(module, prefix, string(result.OutputFiles[0].Contents))
	if err != nil {
		return nil, fmt.Errorf("jig: failed to rewrite requires: %w", err)
	}

	return []byte(rewritten), nil
}

// getBundle returns the views and the render function of the SSR bundle that
// evaluated into __jig__[key]
func getBundle(vm *goja.Runtime, key string) (views *goja.Object, render goja.Callable, err error) {
	jigObj := vm.Get("__jig__")
	if jigObj == nil {
		return nil, nil, fmt.Errorf("jig: unable to load __jig__ object")
	}
	bundleObj := jigObj.ToObject(vm).Get(key)
	if bundleObj == nil {
		return nil, nil, fmt.Errorf("jig: unable to load __jig__[%q]", key)
	}
	bundle := bundleObj.ToObject(vm)
	render, ok := goja.AssertFunction(bundle.Get("render"))
	if !ok {
		return nil, nil, fmt.Errorf("jig: unable to load __jig__[%q].render function", key)
	}
	return bundle.Get("views").ToObject(vm), render, nil
}

// Build the pages in dir for production, replacing previous builds. dir is
// relative to the viewer's directory, which pages are still relative to.
func (v *Viewer) Build(ctx context.Context, dir string) error {
	// Group the entries by the framework that renders them
	entries := map[framework.Framework][]string{}
	ignored := ignorer(v.options.Dir, v.options.OutDir)
	dir = filepath.Join(v.options.Dir, dir)
	err := filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		} else if ignored(path) {
			if de.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		fw, ok := v.frameworks[filepath.Ext(path)]
		if !ok || de.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(v.options.Dir, path)
		if err != nil {
			return err
		}
		entries[fw] = append(entries[fw], filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return fmt.Errorf("jig: failed to find pages: %w", err)
	}

	// Build each framework's pages. Pages of the same framework share chunks.
	var files []esbuild.OutputFile
	for fw, entries := range entries {
		found, err := pages.Find(v.options.Dir, entries...)
		if err != nil {
			return fmt.Errorf("jig: failed to find pages: %w", err)
		} else if len(found) == 0 {
			continue
		}
		// Bundle every layout, since pages can render inside any one of them
		var layouts []string
		for _, entry := range entries {
			if pages.IsLayout(entry) {
				layouts = append(layouts, entry)
			}
		}
		ssrFiles, err := v.transformSSR(fw, ssrBundle, found, layouts...)
		if err != nil {
			return fmt.Errorf("jig: failed to build: %w", err)
		}
		pageEntries := make([]string, len(found))
		for i, page := range found {
			pageEntries[i] = page.Entry
		}
		domFiles, err := v.transformDOM(fw, true, pageEntries...)
		if err != nil {
			return fmt.Errorf("jig: failed to build: %w", err)
		}
		files = slices.Concat(files, ssrFiles, domFiles)
	}
	if len(files) == 0 {
		return fmt.Errorf("jig: no pages in %s", dir)
	}

	// Keep the previous build when canceled
	if err := ctx.Err(); err != nil {
		return err
	}

	// Replace previous builds so removed pages and stale chunks aren't left
	// behind. Only done once the build succeeds.
	for _, dir := range []string{"ssr", "dom"} {
		if err := os.RemoveAll(filepath.Join(v.options.Dir, v.options.OutDir, dir)); err != nil {
			return fmt.Errorf("jig: failed to build: %w", err)
		}
	}
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.Path), 0755); err != nil {
			return fmt.Errorf("jig: failed to build: %w", err)
		}
		if err := os.WriteFile(file.Path, file.Contents, 0644); err != nil {
			return fmt.Errorf("jig: failed to build: %w", err)
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
	return nil, fmt.Errorf("jig: no output file for %q", path)
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
			// Let browsers revalidate packages across restarts without rebuilding
			etag, err := packageETag(v.options.Dir)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "no-cache")
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
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
			consoleError(w, "jig: failed to serve %s: %s", r.URL.Path, err.Error())
			return
		}

		// Cached code only changes on deploy, so let browsers revalidate it
		// instead of downloading it again
		if v.options.Cache {
			sum := sha256.Sum256(code)
			etag := `"` + hex.EncodeToString(sum[:8]) + `"`
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "no-cache")
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Serve the code
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(code)
	}))
}

// serveDOM returns the browser code for path. Returns fs.ErrNotExist when path
// isn't a page or a built file.
func (v *Viewer) serveDOM(path string) ([]byte, error) {
	// Serve pages and shared chunks from the prebuilt filesystem
	if v.prebuilt() {
		return fs.ReadFile(v.options.FS, gopath.Join(v.options.OutDir, "dom", path+".js"))
	}

	// Otherwise build the page on demand, caching the build when enabled
	if code, ok := v.serveCache.Get(path); ok {
		return code, nil
	}
	fw, ok := v.frameworks[gopath.Ext(path)]
	if !ok {
		return nil, fs.ErrNotExist
	}
	if _, err := os.Stat(filepath.Join(v.options.Dir, path)); err != nil {
		return nil, err
	}
	files, err := v.transformDOM(fw, false, path)
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

// prebuilt reports whether views are served from a previous Build
func (v *Viewer) prebuilt() bool {
	return v.options.Cache && v.options.FS != nil
}

// Render the views into w as one HTML document. Views are listed outermost
// first: an optional layout, which is named layout.* and only renders on the
// server, then the page. Nothing is written when an error is returned. A
// missing view returns an error that matches fs.ErrNotExist. Errors thrown by
// components render the page's error view.
func (v *Viewer) Render(ctx context.Context, w io.Writer, views ...View) error {
	if len(views) == 0 {
		return errors.New("jig: no views to render")
	}
	page := views[len(views)-1].Name
	if pages.IsLayout(page) {
		return fmt.Errorf("jig: layout %q can't be the page", page)
	}
	for i, view := range views[:len(views)-1] {
		if !pages.IsLayout(view.Name) {
			return fmt.Errorf("jig: %q can't wrap a page yet", view.Name)
		} else if i > 0 {
			return fmt.Errorf("jig: layout %q must come first", view.Name)
		}
	}

	vm := v.pool.Acquire()

	// Stop the render when the context is canceled
	stop := context.AfterFunc(ctx, func() {
		vm.Interrupt(ctx.Err())
	})
	html, err := v.render(vm, views)
	// Interrupts skip JS finally blocks, which can leave the VM in a bad state,
	// so only reuse VMs that weren't interrupted
	if stop() {
		v.pool.Release(vm)
	}
	if err != nil {
		return fmt.Errorf("jig: unable to render page %q: %w", page, err)
	}

	_, err = io.WriteString(w, html)
	return err
}

func (v *Viewer) render(vm *goja.Runtime, views []View) (string, error) {
	names := make([]string, len(views))
	props := make([]string, len(views))
	for i, view := range views {
		names[i] = view.Name
		// Encode props with Go's JSON rules so struct tags are respected, SSR and
		// the browser see the same props, nil slices and maps become [] and {}, and
		// "<" is escaped for the script tag.
		data, err := json.Marshal(view.Props, jsontext.EscapeForHTML(true), jsontext.EscapeForJS(true))
		if err != nil {
			return "", fmt.Errorf("jig: unable to encode props: %w", err)
		}
		props[i] = string(data)
	}

	// The prebuilt bundle has every view. Otherwise each stack of views is
	// bundled when it's rendered, so the stack is the key.
	key := ssrBundle
	if !v.prebuilt() {
		key = strings.Join(names, ",")
	}

	// Pooled VMs don't share globals, so evaluate the bundle in this VM when
	// it's missing. Without caching, always evaluate to pick up the latest code.
	bundled, render, err := getBundle(vm, key)
	if err != nil || !v.options.Cache {
		program, err := v.compile(key, names)
		if err != nil {
			return "", err
		}
		if _, err = vm.RunProgram(program); err != nil {
			return "", fmt.Errorf("jig: failed to evaluate code for %s: %w", key, err)
		}
		bundled, render, err = getBundle(vm, key)
		if err != nil {
			return "", fmt.Errorf("jig: failed to get render function for %s: %w", key, err)
		}
	}

	// The prebuilt bundle only has the views that existed when it was built
	for _, name := range names {
		if bundled.Get(name) == nil {
			return "", fmt.Errorf("jig: view %q: %w", name, fs.ErrNotExist)
		}
	}

	// Call the render function
	result, err := render(goja.Undefined(), vm.ToValue(names), vm.ToValue(props))
	if err != nil {
		return "", fmt.Errorf("jig: unable to render the component: %w", err)
	}

	html := result.String()
	html = v.liveReload.Rewrite(html)
	return html, nil
}

// readSSR reads the prebuilt SSR bundle from the filesystem when caching,
// otherwise it bundles views to get the current version of the code.
func (v *Viewer) readSSR(key string, views []string) ([]byte, error) {
	if v.prebuilt() {
		code, err := fs.ReadFile(v.options.FS, gopath.Join(v.options.OutDir, "ssr", ssrBundle+".js"))
		if err != nil {
			return nil, fmt.Errorf("jig: failed to read ssr file: %w", err)
		}
		return code, nil
	}
	page := views[len(views)-1]
	fw, ok := v.frameworks[gopath.Ext(page)]
	if !ok {
		return nil, fmt.Errorf("jig: no framework renders %q files", gopath.Ext(page))
	}
	// Check for missing views here, because the bundler's errors don't match
	// fs.ErrNotExist
	for _, view := range views {
		if _, err := os.Stat(filepath.Join(v.options.Dir, view)); err != nil {
			return nil, err
		}
	}
	pages, err := pages.Find(v.options.Dir, page)
	if err != nil {
		return nil, fmt.Errorf("jig: failed to find pages: %w", err)
	}
	files, err := v.transformSSR(fw, key, pages, views[:len(views)-1]...)
	if err != nil {
		return nil, fmt.Errorf("jig: failed to transform for ssr: %w", err)
	}
	// One entry without code splitting builds into one file
	return files[0].Contents, nil
}

// compile the SSR bundle for views. Programs aren't tied to a VM, so they can
// be cached and shared across the pool.
func (v *Viewer) compile(key string, views []string) (*goja.Program, error) {
	if program, ok := v.renderCache.Get(key); ok {
		return program, nil
	}
	code, err := v.readSSR(key, views)
	if err != nil {
		return nil, err
	}

	program, err := goja.Compile(key, string(code), false)
	if err != nil {
		return nil, fmt.Errorf("jig: failed to compile code for %s: %w", key, err)
	}

	if v.options.Cache {
		v.renderCache.Set(key, program)
	}
	return program, nil
}

// reloadEvent is a file change sent to the livereload client
type reloadEvent struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

// Files that change when packages are installed
var packageFiles = []string{
	"package.json",
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"bun.lock",
}

// packageETag hashes what node_modules bundles are built from: the package
// files in dir, which change when packages are installed, and the versions of
// the modules that build them
func packageETag(dir string) (string, error) {
	hash := sha256.New()
	hash.Write([]byte(buildVersions()))
	for _, name := range packageFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return "", err
		}
		hash.Write([]byte(name))
		hash.Write(data)
	}
	return `"` + hex.EncodeToString(hash.Sum(nil)[:8]) + `"`, nil
}

// Modules whose upgrades can change node_modules bundles. The root package's
// path is jig's module path.
var buildModules = []string{
	reflect.TypeFor[Viewer]().PkgPath(),
	"github.com/evanw/esbuild",
}

// buildVersions returns the versions of buildModules in this binary. Local
// replace directives have no version, so changes to those aren't detected.
func buildVersions() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var versions strings.Builder
	for _, module := range append([]*debug.Module{&info.Main}, info.Deps...) {
		if !slices.Contains(buildModules, module.Path) {
			continue
		}
		if module.Replace != nil {
			module = module.Replace
		}
		versions.WriteString(module.Path)
		versions.WriteString("@")
		versions.WriteString(module.Version)
		versions.WriteString("\n")
	}
	return versions.String()
}

func (v *Viewer) onChange(events []watch.Event) error {
	// Removed files don't need a reload
	var reloads []reloadEvent
	for _, event := range events {
		if event.Op&watch.Remove != 0 {
			continue
		}
		path, err := filepath.Rel(v.options.Dir, event.Name)
		if err != nil {
			return err
		}
		reloads = append(reloads, reloadEvent{event.Op.String(), filepath.ToSlash(path)})
	}
	if len(reloads) == 0 {
		return nil
	}

	v.options.Log.Debug("livereload: triggering live reload", "events", reloads)

	// Publish a livereload event
	data, err := json.Marshal(reloads)
	if err != nil {
		return err
	}
	if err := v.liveReload.Publish(&livereload.Event{
		Type: "reload",
		Data: data,
	}); err != nil {
		v.options.Log.Warn("livereload: failed to reload", "err", err, "events", string(data))
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
	return strings.Contains(exe, string(filepath.Separator)+"go-build")
}

func consoleError(w http.ResponseWriter, format string, a ...any) {
	w.Header().Set("Content-Type", "application/javascript")
	msg := fmt.Sprintf(format, a...)
	js := fmt.Sprintf(`console.error(%q);`, msg)
	w.Write([]byte(js))
}
