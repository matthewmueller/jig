module github.com/matthewmueller/jig/example

replace github.com/matthewmueller/jig => ../

replace github.com/matthewmueller/livereload => ../../livereload

go 1.27.0

require (
	github.com/livebud/mux v0.4.0
	github.com/matthewmueller/jig v0.0.0-00010101000000-000000000000
	github.com/matthewmueller/logs v0.0.8
)

require (
	github.com/dlclark/regexp2/v2 v2.8.0 // indirect
	github.com/dop251/goja v0.0.0-20260911104922-fabc3b8078ad // indirect
	github.com/evanw/esbuild v0.27.2 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/google/pprof v0.0.0-20260906184651-6331bc6350fe // indirect
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
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

tool github.com/matthewmueller/jig/cmd/react
