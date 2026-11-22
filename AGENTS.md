# AGENTS.md

jig is a Go library and CLI (`cmd/jig`) that server-renders React views in goja, bundled with esbuild. No Node.js at runtime.

## Commands

- Setup: Go version from `go.mod` (currently 1.27.0). If `go` is older and the auto toolchain download fails, install it from `https://dl.google.com/go/go1.27.0.linux-amd64.tar.gz` and put its `bin` first on `PATH`. Then run `go mod download`.
- Build: `go build ./...`
- Test: `make test` runs `go mod tidy`, `go vet`, `modernize -fix`, `staticcheck` and `go test -race ./...`, and takes about 4 minutes. It rewrites files, so check `git diff` afterwards. For quick runs use `go test ./... -run TestName`.
- Run the CLI: `go run ./cmd/jig -h`
- Run the example app: `cd example && npm ci && go generate ./... && go run .`, then open http://localhost:8000. `example/` is a separate Go module that uses `replace ../`.

## Notes

- Tests need network access: `jig_test.go` installs react from registry.npmjs.org into temp dirs through `github.com/livebud/npm`. They don't need any services or env vars, and the root `npm install` isn't needed for tests.
- Generated code: `example/dashboard/props.go.ts` is generated from `props.go` by `go generate` in `example/`. Commit it, and regenerate it whenever a props struct changes. `example/dist/` is build output, so only commit `example/dist/.gitkeep`.
- Add a line to `Changelog.md` for every user-facing change.

## Never run

- `make release`: it commits, tags, pushes `main` and creates a GitHub release.
- `git push origin main`, or creating tags or releases.
