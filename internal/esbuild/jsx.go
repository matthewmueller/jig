package esbuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/matthewmueller/jig/internal/head"
	"github.com/matthewmueller/jig/internal/styledjsx"
	"github.com/matthewmueller/jsx"
)

func transpileTSX(path, source string) (string, error) {
	result := esbuild.Transform(source, esbuild.TransformOptions{
		Loader:     esbuild.LoaderTSX,
		JSX:        esbuild.JSXPreserve,
		Sourcefile: path,
	})
	if len(result.Errors) > 0 {
		messages := esbuild.FormatMessages(result.Errors, esbuild.FormatMessagesOptions{
			Kind: esbuild.ErrorMessage,
		})
		return "", fmt.Errorf("jsx: failed to transpile typescript: %s", strings.Join(messages, "\n"))
	}
	return string(result.Code), nil
}

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
			contents := string(code)
			outputLoader := loader
			if ext == ".tsx" {
				contents, err = transpileTSX(args.Path, contents)
				if err != nil {
					return result, err
				}
				outputLoader = esbuild.LoaderJSX
			}
			script, err := jsx.Parse(args.Path, contents)
			if err != nil {
				return result, err
			}
			// TODO: this is very hacky, hoist the conditional up
			if strings.Contains(runtime, "ssr_runtime") {
				if err := styledjsx.RewriteSSR(args.Path, runtime, script); err != nil {
					return result, err
				}
			} else {
				if err := styledjsx.RewriteDOM(args.Path, script); err != nil {
					return result, err
				}
			}
			if err := head.Rewrite(args.Path, runtime, script); err != nil {
				return result, err
			}
			contents = script.String()
			result.Contents = &contents
			result.ResolveDir = filepath.Dir(args.Path)
			result.Loader = outputLoader
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
