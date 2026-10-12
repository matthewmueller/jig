# jig

Server-render React views from Go, with typed props generated from your Go structs.

jig bundles views with esbuild and renders them in [goja](https://github.com/dop251/goja), so there's no Node.js process at runtime. In development, views build on demand with live reload. In production, they're served from a prebuilt `dist/` embedded in your binary.

## Install

```sh
go get github.com/matthewmueller/jig
go get -tool github.com/matthewmueller/jig/cmd/jig
npm install react react-dom
```

## Example

Define the props in Go next to the view:

```go
// dashboard/props.go
package dashboard

type Props struct {
	Name string `json:"name"`
}
```

Import the generated types from the Go file:

```tsx
// dashboard/dashboard.tsx
import type { Props } from "./props.go"

export default function Dashboard(props: Props) {
  return <h1>Welcome, {props.name}!</h1>
}
```

Render it from a handler:

```go
package main

import (
	"embed"
	"net/http"

	"github.com/matthewmueller/jig"
	"github.com/matthewmueller/myapp/dashboard"
)

//go:generate go tool jig

//go:embed all:dist
var distfs embed.FS

func main() {
	viewer, err := jig.Load(jig.WithFS(distfs))
	if err != nil {
		panic(err)
	}
	defer viewer.Close()
	http.Handle("/", viewer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		props := dashboard.Props{Name: "Matt"}
		if err := viewer.Render(r.Context(), w, "dashboard/dashboard.tsx", props); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})))
	http.ListenAndServe(":8000", nil)
}
```

- `go run .` builds views on demand and live reloads them.
- `go generate` builds `dist/` and writes `dashboard/props.go.ts`.
- `NODE_ENV=production ./myapp` serves the embedded `dist/`.

Commit an empty `dist/.gitkeep` so `//go:embed all:dist` compiles before the first build.

Pages can sit next to a `layout.tsx` and an `error.tsx`, which apply to the pages in that directory and below.

See [example/](example) for a working app.

## CLI Reference

```sh
jig [-C dir] [--minify=true] [views]
```

`jig` walks the views directory, which defaults to `.`.

- **`.jsx` and `.tsx` files** are built as pages into `dist/` in the project directory (`-C`). Page paths stay relative to the project directory.
- **`.go` files** get a sibling `.go.ts` with TypeScript types matching how `encoding/json/v2` marshals them.

If your views live in a subdirectory, pass it, e.g. `go tool jig view`. Then render with paths like `view/index.tsx`.

## Development

```sh
make test
```
