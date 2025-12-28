package livereload

import (
	"context"
	_ "embed"
	"log/slog"
	"net/http"
	"strings"

	"github.com/livebud/sse"
)

//go:embed script.js
var liveScript string

type Event = sse.Event

type LiveReload interface {
	Middleware(next http.Handler) http.Handler
	Publish(event *Event) error
	Rewrite(html string) string
}

func New(log *slog.Logger, liveRoute string) LiveReload {
	liveScript := strings.ReplaceAll(liveScript, "__LIVE_ROUTE__", liveRoute)
	return &Reloader{liveRoute, liveScript, sse.New(log)}
}

type Reloader struct {
	liveRoute  string
	liveScript string
	sse        *sse.Handler
}

var backgroundCtx = context.Background()

func (lr *Reloader) Publish(event *Event) error {
	return lr.sse.Publish(backgroundCtx, event)
}

func (lr *Reloader) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Passthrough non-livereload requests
		if r.Method != http.MethodGet || r.URL.Path != lr.liveRoute {
			next.ServeHTTP(w, r)
			return
		}

		// Handle server-sent events
		if r.Header.Get("Accept") == "text/event-stream" {
			lr.sse.ServeHTTP(w, r)
			return
		}

		// Serve the livereload client
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(lr.liveScript))
	})
}

func (lr *Reloader) Rewrite(html string) string {
	index := strings.Index(html, "</head>")
	if index < 0 {
		return html
	}
	script := `<script async="" type="module" src="` + lr.liveRoute + `"></script>`
	return html[:index] + script + html[index:]
}
