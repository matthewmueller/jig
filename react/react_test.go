package react_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/livebud/npm"
	"github.com/matryer/is"
	"github.com/matthewmueller/diff"
	"github.com/matthewmueller/logs"
	"github.com/matthewmueller/views/react"
	"github.com/matthewmueller/virt"
)

const packageJson = `
{
	"name": "test-app",
	"version": "1.0.0",
	"main": "index.jsx",
	"dependencies": {
		"react": "^19.2.0",
		"react-dom": "^19.2.0"
	}
}
`

func TestRenderBasic(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
		react.WithLog(logs.Default()),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", map[string]any{"name": "World"})
	expect := `<!doctype html><html><head><script defer="" type="module" src="/index.jsx.js"></script><script async="" type="module" src="/.live"></script></head><body><div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div><script type="application/json" id="react-props">{"name":"World"}</script></body></html>`
	diff.TestString(t, actual.String(), expect)
}

func TestRenderMultiple(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
		"about.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>I'm {props.name}!</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", map[string]any{"name": "World"})
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHtml, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHtml, `<div>Hello, <!-- -->World<!-- -->!</div>`)
	propsText := doc.Find("#react-props").Text()
	is.NoErr(err)
	expectProps := `{"name":"World"}`
	diff.TestString(t, propsText, expectProps)

	actual = new(bytes.Buffer)
	viewer.Render(actual, "about.jsx", map[string]any{"name": "World"})
	doc, err = goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHtml, err = doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHtml, `<div>I&#39;m <!-- -->World<!-- -->!</div>`)
	propsText = doc.Find("#react-props").Text()
	is.NoErr(err)
	expectProps = `{"name":"World"}`
	diff.TestString(t, propsText, expectProps)

	actual = new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", map[string]any{"name": "World"})
	doc, err = goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHtml, err = doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHtml, `<div>Hello, <!-- -->World<!-- -->!</div>`)
	propsText = doc.Find("#react-props").Text()
	is.NoErr(err)
	expectProps = `{"name":"World"}`
	diff.TestString(t, propsText, expectProps)

	actual = new(bytes.Buffer)
	viewer.Render(actual, "about.jsx", map[string]any{"name": "World"})
	doc, err = goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHtml, err = doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHtml, `<div>I&#39;m <!-- -->World<!-- -->!</div>`)
	propsText = doc.Find("#react-props").Text()
	is.NoErr(err)
	expectProps = `{"name":"World"}`
	diff.TestString(t, propsText, expectProps)
}

func TestRenderProduction(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
		"about.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>I'm {props.name}!</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
		react.WithMinify(true),
		react.WithCache(true),
		react.WithEnv(map[string]string{
			"NODE_ENV": "production",
		}),
		react.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	expect := `<!doctype html><html><head><script defer="" type="module" src="/index.jsx.js"></script></head><body><div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div><script type="application/json" id="react-props">{"name":"World"}</script></body></html>`
	actual := new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", map[string]any{"name": "World"})
	diff.TestString(t, actual.String(), expect)
}

func contains(t testing.TB, actual, expect string) {
	t.Helper()
	if !strings.Contains(actual, expect) {
		if len(actual) > 100 {
			actual = actual[:100] + "..."
		}
		if len(expect) > 100 {
			expect = expect[:100] + "..."
		}
		t.Fatalf("actual %q does not contain %q", actual, expect)
	}
}

func TestServeClient(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
		"about.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>I'm {props.name}!</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	handler := viewer.Middleware()(http.NotFoundHandler())
	req := httptest.NewRequest("GET", "/index.jsx.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	res := rec.Result()

	is.Equal(res.StatusCode, http.StatusOK)
	is.Equal(res.Header.Get("Content-Type"), "application/javascript")
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	contains(t, string(body), `"Hello, ",`)
	contains(t, string(body), `props2.name,`)
}
