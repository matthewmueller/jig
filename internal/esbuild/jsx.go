package esbuild

import (
	"os"
	"path/filepath"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/matthewmueller/jsx"
	"github.com/matthewmueller/views/internal/head"
	"github.com/matthewmueller/views/internal/styledjsx"
)

func JSX(runtime string) esbuild.Plugin {
	transformFile := func(ext string, loader esbuild.Loader) func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
		return func(args esbuild.OnLoadArgs) (result esbuild.OnLoadResult, err error) {
			// Skip layout files
			if filepath.Base(args.Path) == "layout"+ext {
				return result, nil
			}
			code, err := os.ReadFile(args.Path)
			if err != nil {
				return result, err
			}
			script, err := jsx.Parse(args.Path, string(code))
			if err != nil {
				return result, err
			}
			// TODO: this is very hacky, hoist the conditional up
			if strings.Contains(runtime, "ssr_runtime") {
				if err := styledjsx.Rewrite(args.Path, runtime, script); err != nil {
					return result, err
				}
			} else {
				if err := styledjsx.Remove(args.Path, script); err != nil {
					return result, err
				}
			}
			if err := head.Rewrite(args.Path, runtime, script); err != nil {
				return result, err
			}
			contents := script.String()
			result.Contents = &contents
			result.ResolveDir = filepath.Dir(args.Path)
			result.Loader = loader
			return result, nil
		}
	}
	return esbuild.Plugin{
		Name: "jsx",
		Setup: func(epb esbuild.PluginBuild) {
			epb.OnLoad(esbuild.OnLoadOptions{Filter: ".tsx$"}, transformFile(".tsx", esbuild.LoaderTSX))
			epb.OnLoad(esbuild.OnLoadOptions{Filter: ".jsx$"}, transformFile(".jsx", esbuild.LoaderJSX))
		},
	}
}
