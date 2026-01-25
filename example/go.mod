module github.com/matthewmueller/views/example

replace github.com/matthewmueller/views => ../

replace github.com/matthewmueller/livereload => ../../livereload

go 1.25.5

require (
	github.com/livebud/mux v0.4.0
	github.com/matthewmueller/logs v0.0.8
	github.com/matthewmueller/views v0.0.0-00010101000000-000000000000
)

require (
	github.com/bep/debounce v1.2.1 // indirect
	github.com/buildkite/terminal-to-html/v3 v3.16.8 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dop251/goja v0.0.0-20251121114222-56b1242a5f86 // indirect
	github.com/evanw/esbuild v0.27.2 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/livebud/cli v0.0.22 // indirect
	github.com/livebud/color v0.0.2 // indirect
	github.com/livebud/sse v0.0.5 // indirect
	github.com/matthewmueller/cjs v0.0.2 // indirect
	github.com/matthewmueller/css v0.0.4 // indirect
	github.com/matthewmueller/enroute v0.1.0 // indirect
	github.com/matthewmueller/glob v0.1.1 // indirect
	github.com/matthewmueller/jsx v0.0.12 // indirect
	github.com/matthewmueller/styledjsx v0.0.11 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	github.com/tdewolff/parse/v2 v2.8.5 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)

tool github.com/matthewmueller/views/cmd/react
