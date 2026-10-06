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

// Package resolves "package:<module>" entrypoints like bare imports, so they
// only come from node_modules. esbuild resolves plain entrypoints as files
// relative to the working directory first, which would expose project files.
func Package() esbuild.Plugin {
	return esbuild.Plugin{
		Name: "package",
		Setup: func(epb esbuild.PluginBuild) {
			epb.OnResolve(esbuild.OnResolveOptions{Filter: `^package:`}, func(args esbuild.OnResolveArgs) (result esbuild.OnResolveResult, err error) {
				resolved := epb.Resolve(strings.TrimPrefix(args.Path, "package:"), esbuild.ResolveOptions{
					Kind:       esbuild.ResolveJSImportStatement,
					ResolveDir: args.ResolveDir,
				})
				result.Errors = resolved.Errors
				result.Path = resolved.Path
				result.Namespace = resolved.Namespace
				return result, nil
			})
		},
	}
}
