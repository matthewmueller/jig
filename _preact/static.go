package preact

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/matthewmueller/elmo/internal/asset"
	"github.com/matthewmueller/elmo/internal/esbuild"
	"github.com/matthewmueller/elmo/internal/goja"
	"github.com/matthewmueller/elmo/internal/log"
	"github.com/matthewmueller/elmo/internal/pagedir"
)

type Static struct {
	Log    log.Interface
	Dir    string
	Env    map[string]string
	JS     goja.Renderer
	Minify bool
}

// Build builds the static assets.
func (s *Static) Build(ctx context.Context, pages ...*pagedir.Page) (files []*asset.File, err error) {
	defines := make(map[string]string, len(s.Env)+2)
	defines["process"] = "{}"
	defines["process.env"] = "{}"
	for key, value := range s.Env {
		defines[`process.env.`+key] = strconv.Quote(value)
	}
	htmlFiles, err := s.buildHTML(ctx, pages, defines)
	if err != nil {
		return nil, err
	}
	for _, outfile := range htmlFiles {
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

func (s *Static) buildHTML(ctx context.Context, pages []*pagedir.Page, defines map[string]string) ([]*esbuild.File, error) {
	absDir, err := filepath.Abs(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to get absolute path for %s: %w", s.Dir, err)
	}
	entrypoints := make([]esbuild.EntryPoint, len(pages))
	for i, page := range pages {
		entrypoints[i] = page.Entrypoint()
	}
	files, _, err := esbuild.Build(esbuild.BuildOptions{
		EntryPointsAdvanced: entrypoints,
		AbsWorkingDir:       absDir,
		Outdir:              ".",
		MinifyWhitespace:    s.Minify,
		MinifyIdentifiers:   s.Minify,
		MinifySyntax:        s.Minify,
		JSXImportSource:     "preact",
		JSX:                 esbuild.JSXAutomatic,
		Format:              esbuild.FormatIIFE,
		GlobalName:          "elmo",
		Platform:            esbuild.PlatformNeutral,
		MainFields:          []string{"module", "main"},
		Metafile:            true,
		Bundle:              true,
		Define:              defines,
		PublicPath:          "/",
		Plugins: []esbuild.Plugin{
			esbuild.Virtual(`\.entry\.js$`, s.loadStaticEntry(absDir, func(path string) (*pagedir.Page, error) {
				for _, page := range pages {
					if page.Entrypoint().InputPath == path {
						return page, nil
					}
				}
				return nil, fmt.Errorf("preact: unable to find page for %s", path)
			})),
			esbuild.Virtual(`^static_runtime\.tsx$`, s.loadStaticRuntime(absDir)),
			esbuild.JSX("static_runtime.tsx"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("preact: unable to build html: %w", err)
	}
	outFiles := make([]*esbuild.File, len(files))
	for i, file := range files {
		rel, err := filepath.Rel(absDir, file.Path)
		if err != nil {
			return nil, fmt.Errorf("preact: unable to get relative path for %s: %w", file.Path, err)
		}
		file.Path = replaceExt(rel, ".html")
		contents := string(file.Contents)
		html, err := s.JS.Render(ctx, file.Path, contents+"; elmo.render({})")
		if err != nil {
			return nil, fmt.Errorf("preact: error rendering html for %s: %w", file.Path, err)
		}
		file.Contents = []byte(html)
		outFiles[i] = &file
	}
	return outFiles, nil
}

//go:embed static_entry.gotext
var staticEntry string

var staticGenerator = template.Must(template.New("static_entry.gotext").Parse(staticEntry))

type staticEntryData struct {
	*pagedir.Page
}

func (s *Static) loadStaticEntry(absDir string, findPage func(path string) (*pagedir.Page, error)) func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
	return func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
		page, err := findPage(args.Path)
		if err != nil {
			return result, fmt.Errorf("preact: unable to load static entry: %w", err)
		}
		buffer := new(bytes.Buffer)
		data := &staticEntryData{page}
		if err := staticGenerator.Execute(buffer, data); err != nil {
			return result, err
		}
		contents := buffer.String()
		result.Contents = &contents
		result.ResolveDir = absDir
		return result, nil
	}
}

//go:embed static_runtime.tsx
var staticRuntime string

func (s *Static) loadStaticRuntime(absDir string) func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
	return func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
		result.Contents = &staticRuntime
		result.ResolveDir = absDir
		result.Loader = esbuild.LoaderTSX
		return result, nil
	}
}

// Render serves the static assets.
func (s *Static) Render(ctx context.Context, page *pagedir.Page, props map[string]interface{}) (*asset.File, error) {
	defines := make(map[string]string, len(s.Env)+2)
	defines["process"] = "{}"
	defines["process.env"] = "{}"
	for key, value := range s.Env {
		defines[`process.env.`+key] = strconv.Quote(value)
	}
	jsFile, err := s.buildHTMLPage(defines, page)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to build ssr script for %q: %w", page.Path, err)
	}
	contents := string(jsFile.Contents)
	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to marshal props: %w", err)
	}
	htmlCode, err := s.JS.Render(ctx, jsFile.Path, contents+"; elmo["+strconv.Quote(page.Key)+"].render("+string(propBytes)+")")
	if err != nil {
		return nil, fmt.Errorf("preact: error rendering html for %s: %w", jsFile.Path, err)
	}
	return &asset.File{
		Path:    replaceExt(jsFile.Path, ".html"),
		Data:    []byte(htmlCode),
		ModTime: time.Now(),
		Mode:    0644,
	}, nil
}

func (s *Static) buildHTMLPage(defines map[string]string, page *pagedir.Page) (*esbuild.File, error) {
	absDir, err := filepath.Abs(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to get absolute path for %s: %w", s.Dir, err)
	}
	files, _, err := esbuild.Build(esbuild.BuildOptions{
		EntryPointsAdvanced: []esbuild.EntryPoint{page.Entrypoint()},
		AbsWorkingDir:       absDir,
		Outdir:              ".",
		MinifyWhitespace:    s.Minify,
		MinifyIdentifiers:   s.Minify,
		MinifySyntax:        s.Minify,
		JSXImportSource:     "preact",
		JSX:                 esbuild.JSXAutomatic,
		Format:              esbuild.FormatIIFE,
		GlobalName:          "elmo[" + strconv.Quote(page.Key) + "]",
		Platform:            esbuild.PlatformNeutral,
		Target:              esbuild.ES2017,
		MainFields:          []string{"module", "main"},
		Metafile:            true,
		Bundle:              true,
		Define:              defines,
		PublicPath:          "/",
		Plugins: []esbuild.Plugin{
			esbuild.Virtual(`\.entry\.js$`, s.loadStaticEntry(absDir, func(path string) (*pagedir.Page, error) {
				return page, nil
			})),
			esbuild.Virtual(`^static_runtime\.tsx$`, s.loadStaticRuntime(absDir)),
			esbuild.JSX("static_runtime.tsx"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("preact: unable to build html: %w", err)
	} else if len(files) != 1 {
		return nil, fmt.Errorf("preact: expected 1 html file, got %d", len(files))
	}
	file := files[0]
	relPath, err := filepath.Rel(absDir, file.Path)
	if err != nil {
		return nil, fmt.Errorf("preact: unable to get relative path for %s: %w", file.Path, err)
	}
	file.Path = relPath
	return &file, nil
}

func replaceExt(path, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}
