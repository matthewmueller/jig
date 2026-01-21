package esbuild

import (
	"slices"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

// ExternalNodeModules externalizes node_modules
func ExternalNodeModules(prefix string, ignores ...string) esbuild.Plugin {
	return esbuild.Plugin{
		Name: "external_node_modules",
		Setup: func(epb esbuild.PluginBuild) {
			// Regexp doesn't start with ".", "/", "\", "~"
			epb.OnResolve(esbuild.OnResolveOptions{Filter: `^[a-z0-9@]`}, func(args esbuild.OnResolveArgs) (result esbuild.OnResolveResult, err error) {
				if args.Importer == "" {
					// Ignore entrypoints
					return result, nil
				}
				// Ignore tsconfig common @/*
				// TODO: we should use actually resolve instead, but this is good enough for now
				if strings.HasPrefix(args.Path, "@/") {
					return result, nil
				}
				if slices.Contains(ignores, args.Path) {
					return result, nil
				}
				result.Path = prefix + args.Path
				result.External = true
				return result, nil
			})
		},
	}
}
