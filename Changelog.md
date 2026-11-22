# 0.2.0 / 2026-11-22

- fix make release
- upgrade goja for lazy string concatenation
- navigate between pages without full page loads
- keep idle goja VMs across garbage collections
- render an explicit stack of views with an optional server-only layout
- fix the modernize path in make test
- add readme and license
- test generating types that reference another module
- pass the views directory to jig so views can live in a subdirectory
- generate typescript types for go files in the views directory
- make jig framework-agnostic with a single jig command
- rename views to jig
- add ETags to production page and chunk responses

# 0.1.0 / 2026-10-07

- switch to using encoding/json/v2
- fix copy-pasted error messages in the dom and node_modules builds
- test rendering head elements
- drop the in-memory node_modules cache in favor of ETags
- cache node_modules bundles and let browsers revalidate them
- only inject the goja shims into server bundles
- send livereload events as JSON
- return render errors and accept a context in Render
- define NODE_ENV=production for minified builds
- clean previous ssr and dom output when building
- keep on-demand builds in memory
- fix data race in the file watcher's debounce
- skip ignored directories when watching and match gitignore relative to dir
- don't mutate the shared page when falling back to default views
- only intercept .js requests for pages and built files
- restrict the /node_modules/ route to packages in dev builds
- encode props with encoding/json before rendering
- fix render cache sharing a goja runtime across pooled VMs
- add a better console api
- better console.log support
- transform tsx into jsx before running through styledjsx
- add test for when environment variables aren't defined
- add support for process.env. bump goja to support better regexp2
- upgrade mux
- remove unused code
- fix styled-jsx on ~/Component.
- support json struct tags when passing in props.
- reduce head DOM modifications on unchanged elements

# 0.0.1 / 2026-01-12

- initial release
