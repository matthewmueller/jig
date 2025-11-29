package preact

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/matthewmueller/elmo/internal/asset"
	"github.com/matthewmueller/elmo/internal/esbuild"
	"github.com/matthewmueller/elmo/internal/pagedir"
)

type Client struct {
	Dir    string
	Env    map[string]string
	Minify bool
	Bundle bool
}

// Build builds the client-side assets.
func (c *Client) Build(ctx context.Context, pages ...*pagedir.Page) (files []*asset.File, err error) {
	defines := make(map[string]string, len(c.Env)+2)
	defines["process"] = "{}"
	defines["process.env"] = "{}"
	for key, value := range c.Env {
		defines[`process.env.`+key] = strconv.Quote(value)
	}
	jsFiles, err := c.buildJS(pages, defines)
	if err != nil {
		return nil, err
	}
	for _, outfile := range jsFiles {
		files = append(files, &asset.File{
			Path:    outfile.Path,
			Data:    outfile.Contents,
			ModTime: time.Now(),
			Mode:    0644,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func (c *Client) buildJS(pages []*pagedir.Page, defines map[string]string) (outFiles []*esbuild.File, err error) {
	absDir, err := filepath.Abs(c.Dir)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to get absolute path for %s: %w", c.Dir, err)
	}
	entrypoints := make([]esbuild.EntryPoint, len(pages))
	for i, page := range pages {
		entrypoints[i] = page.Entrypoint()
	}
	buildOptions := esbuild.BuildOptions{
		EntryPointsAdvanced: entrypoints,
		AbsWorkingDir:       absDir,
		Outdir:              ".",
		MinifyWhitespace:    c.Minify,
		MinifyIdentifiers:   c.Minify,
		MinifySyntax:        c.Minify,
		JSXImportSource:     "preact",
		JSX:                 esbuild.JSXAutomatic,
		Format:              esbuild.FormatESModule,
		Alias:               map[string]string{},
		Platform:            esbuild.PlatformBrowser,
		ChunkNames:          "[name]-[hash]",
		Splitting:           true,
		Bundle:              true,
		Define:              defines,
		PublicPath:          "/",
		Plugins: []esbuild.Plugin{
			esbuild.Virtual(`\.entry\.js$`, c.loadClientEntry(absDir, func(path string) (*pagedir.Page, error) {
				for _, page := range pages {
					if page.Entrypoint().InputPath == path {
						return page, nil
					}
				}
				return nil, fmt.Errorf("preact: unable to find page for %s", path)
			})),
			esbuild.Virtual(`^client_runtime\.tsx$`, c.loadClientRuntime(absDir)),
			esbuild.JSX("client_runtime.tsx"),
		},
	}
	if !c.Bundle {
		buildOptions.Splitting = false
		buildOptions.Plugins = append(buildOptions.Plugins, esbuild.ExternalNodeModules("/node_modules"))
	}
	files, _, err := esbuild.Build(buildOptions)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to build js: %w", err)
	}
	outFiles = make([]*esbuild.File, len(files))
	for i, file := range files {
		rel, err := filepath.Rel(absDir, file.Path)
		if err != nil {
			return nil, fmt.Errorf("preact: unable to get relative path for %s: %w", file.Path, err)
		}
		file.Path = rel
		outFiles[i] = &file
	}
	return outFiles, nil
}

//go:embed client_entry.gotext
var clientEntry string

var clientGenerator = template.Must(template.New("client_entry.gotext").Parse(clientEntry))

type clientEntryData struct {
	*pagedir.Page
}

func (c *Client) loadClientEntry(absDir string, findPage func(path string) (*pagedir.Page, error)) func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
	return func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
		page, err := findPage(args.Path)
		if err != nil {
			return result, fmt.Errorf("preact: unable to load client entry: %w", err)
		}
		buffer := new(bytes.Buffer)
		data := &clientEntryData{page}
		if err := clientGenerator.Execute(buffer, data); err != nil {
			return result, err
		}
		contents := buffer.String()
		result.Contents = &contents
		result.ResolveDir = absDir
		return result, nil
	}
}

//go:embed client_runtime.tsx
var clientRuntime string

func (c *Client) loadClientRuntime(absDir string) func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
	return func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
		result.Contents = &clientRuntime
		result.ResolveDir = absDir
		result.Loader = esbuild.LoaderTSX
		return result, nil
	}
}

// Serve serves the client-side assets.
func (c *Client) Serve(ctx context.Context, page *pagedir.Page) (*asset.File, error) {
	defines := make(map[string]string, len(c.Env)+2)
	defines["process"] = "{}"
	defines["process.env"] = "{}"
	for key, value := range c.Env {
		defines[`process.env.`+key] = strconv.Quote(value)
	}
	jsFile, err := c.buildJSPage(defines, page)
	if err != nil {
		return nil, err
	}
	return &asset.File{
		Path:    jsFile.Path,
		Data:    jsFile.Contents,
		ModTime: time.Now(),
		Mode:    0644,
	}, nil
}

func (c *Client) buildJSPage(defines map[string]string, page *pagedir.Page) (*esbuild.File, error) {
	absDir, err := filepath.Abs(c.Dir)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to get absolute path for %s: %w", c.Dir, err)
	}
	buildOptions := esbuild.BuildOptions{
		EntryPointsAdvanced: []esbuild.EntryPoint{page.Entrypoint()},
		AbsWorkingDir:       absDir,
		Outdir:              ".",
		MinifyWhitespace:    c.Minify,
		MinifyIdentifiers:   c.Minify,
		MinifySyntax:        c.Minify,
		JSXImportSource:     "preact",
		JSX:                 esbuild.JSXAutomatic,
		Format:              esbuild.FormatESModule,
		Alias:               map[string]string{},
		Platform:            esbuild.PlatformBrowser,
		ChunkNames:          "[name]-[hash]",
		Splitting:           true,
		Bundle:              true,
		Define:              defines,
		PublicPath:          "/",
		Plugins: []esbuild.Plugin{
			esbuild.Virtual(`\.entry\.js$`, c.loadClientEntry(absDir, func(pagePath string) (*pagedir.Page, error) {
				return page, nil
			})),
			esbuild.Virtual(`^client_runtime\.tsx$`, c.loadClientRuntime(absDir)),
			esbuild.JSX("client_runtime.tsx"),
		},
	}
	if !c.Bundle {
		buildOptions.Splitting = false
		buildOptions.Plugins = append(buildOptions.Plugins, esbuild.ExternalNodeModules("/node_modules"))
	}
	files, _, err := esbuild.Build(buildOptions)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to build js: %w", err)
	} else if len(files) != 1 {
		return nil, fmt.Errorf("preact: expected 1 js file, got %d", len(files))
	}
	return &files[0], nil
}
