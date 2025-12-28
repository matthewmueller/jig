package livereload

import (
	"net/http"
)

func Noop() LiveReload {
	return noop{}
}

var _ LiveReload = (*noop)(nil)

type noop struct{}

func (noop) Middleware(next http.Handler) http.Handler {
	return next
}

func (noop) Publish(event *Event) error {
	return nil
}

func (noop) Rewrite(html string) string {
	return html
}
