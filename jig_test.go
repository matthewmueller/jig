package jig_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/livebud/npm"
	"github.com/matryer/is"
	"github.com/matthewmueller/diff"
	"github.com/matthewmueller/jig"
	"github.com/matthewmueller/logs"
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
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithLog(logs.Default()),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}))
	expect := `<!doctype html><html><head><link rel="modulepreload" href="/index.jsx.js"/><script async="" type="module" src="/.live"></script></head><body><div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div><script type="application/json" id="react-props">{"name":"World"}</script><script type="module">import mount from "/index.jsx.js"; mount()</script></body></html>`
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
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}))
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
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "about.jsx", Props: map[string]any{"name": "World"}}))
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
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}))
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
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "about.jsx", Props: map[string]any{"name": "World"}}))
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

func TestRenderEnv(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index() {
				return <div>{process.env.APP_NAME} on {process.env.PORT}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithEnv(map[string]string{
			"APP_NAME": "Jig",
			"PORT":     "9000",
		}),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx"}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<div>Jig<!-- --> on <!-- -->9000</div>`)

	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	res := rec.Result()
	is.Equal(res.StatusCode, http.StatusOK)
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.NoErr(res.Body.Close())
	contains(t, string(body), `"Jig"`)
	contains(t, string(body), `"9000"`)
}

func TestRenderDefaultEnv(t *testing.T) {
	is := is.New(t)
	t.Setenv("JIG_RENDER_ENV_DEFAULT", "from-os")
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index() {
				return <div>{process.env.JIG_RENDER_ENV_DEFAULT}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx"}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<div>from-os</div>`)

	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	res := rec.Result()
	is.Equal(res.StatusCode, http.StatusOK)
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.NoErr(res.Body.Close())
	contains(t, string(body), `"from-os"`)
}

func TestRenderMissingEnv(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index() {
				return <div>{process.env.API_URL ? "defined" : "undefined"}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithEnv(map[string]string{
			"APP_NAME": "Jig",
		}),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx"}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<div>undefined</div>`)
}

func TestRenderImportTSX(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			import Form from './form'
			export default function Index() {
				return <div><Form /></div>;
			}
		`,
		"form.tsx": `
			import React from 'react';
			type Fields = { [key: string]: string };
			export default class Form<F extends Fields> extends React.Component {
				name = <K extends keyof F>(name: K): string => {
					return String(name);
				}
				render() {
					return <form>Form</form>;
				}
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx"}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<div><form>Form</form></div>`)

	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusOK)
	contains(t, rec.Body.String(), `Form`)
}

func TestRenderImportGoTypes(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		// Keeps the side-effect import of type-only imports
		"tsconfig.json": `{ "compilerOptions": { "verbatimModuleSyntax": true } }`,
		"props.go": `
			package views
			type Props struct {
				Name string ` + "`json:\"name\"`" + `
			}
		`,
		"props.go.ts": `export type Props = { name: string }`,
		"greet.ts": `
			import { type Props } from "./props.go"
			export const greet = (props: Props) => "Hello, " + props.name
		`,
		"index.jsx": `
			import React from 'react';
			import { greet } from './greet.ts';
			export default function Index(props) {
				return <div>{greet(props)}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "Jig"}}))
	contains(t, actual.String(), `<div id="root"><div>Hello, Jig</div></div>`)
	handler := viewer.Middleware(http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil))
	is.Equal(rec.Code, http.StatusOK)
	contains(t, rec.Body.String(), `"Hello, "`)
}

func TestRenderImportGenericFormTSX(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"tsconfig.json": `
			{
				"compilerOptions": {
					"paths": {
						"~/*": ["./*"]
					}
				}
			}
		`,
		"index.jsx": `
			import React from 'react';
			import Form from '~/form';
			export default function Index() {
				return <Form defaults={{ email: '' }} />;
			}
		`,
		"form.tsx": `
			import { Component, type InputEvent } from 'react';

			type Props<Fields> = {
				defaults: { [Key in keyof Fields]: Fields[Key] };
			};

			type State<Fields> = {
				fields: Fields;
				disabled: boolean;
			};

			export type Fields = {
				[key: string]: string;
			};

			export default class Form<F extends Fields> extends Component<Props<F>, State<F>> {
				id = 1;
				state: State<F> = {
					fields: this.props.defaults,
					disabled: false,
				};

				name = <K extends keyof F>(name: K): string => {
					return String(name) + '-' + this.id;
				};

				onInput = <K extends keyof F>(key: K) => {
					return (e: InputEvent<HTMLInputElement>) => {
						const fields = { ...this.state.fields };
						fields[key] = e.currentTarget.value as F[K];
						this.setState({ fields });
					};
				};

				render() {
					return <form>{this.name('email' as keyof F)}</form>;
				}
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx"}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<form>email-1</form>`)

	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusOK)
	contains(t, rec.Body.String(), `String(name)`)
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
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithMinify(true),
		jig.WithCache(true),
		jig.WithEnv(map[string]string{
			"NODE_ENV": "production",
		}),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	expect := `<!doctype html><html><head><link rel="modulepreload" href="/index.jsx.js"/></head><body><div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div><script type="application/json" id="react-props">{"name":"World"}</script><script type="module">import mount from "/index.jsx.js"; mount()</script></body></html>`
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}))
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
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest("GET", "/index.jsx.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	res := rec.Result()

	is.Equal(res.StatusCode, http.StatusOK)
	is.Equal(res.Header.Get("Content-Type"), "application/javascript")
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	contains(t, string(body), `"Hello, ",`)
	contains(t, string(body), `props.name,`)
	// Pages navigate without full loads, and export mount for navigation to
	// call again
	contains(t, string(body), `window.__jig_navigate = true`)
	contains(t, string(body), "mount as default")
}

func TestRenderConcurrent(t *testing.T) {
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
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithCache(true),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	// Warm the cache
	is.NoErr(viewer.Render(ctx, new(bytes.Buffer), jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}))
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			actual := new(bytes.Buffer)
			if err := viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}); err != nil {
				t.Error(err)
			} else if !strings.Contains(actual.String(), `<div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div>`) {
				t.Errorf("unexpected render %q", actual.String())
			}
		})
	}
	wg.Wait()
}

func TestRenderPropsEscaped(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>{props.content}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"content": "</script><script>alert(1)</script>"}}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	// Only the props and the page's mount script, nothing injected
	is.Equal(doc.Find("script:not([src])").Length(), 2)
	propsText := doc.Find("#react-props").Text()
	is.True(!strings.Contains(propsText, "</"))
	var props map[string]string
	is.NoErr(json.Unmarshal([]byte(propsText), &props))
	is.Equal(props["content"], "</script><script>alert(1)</script>")
}

func TestRenderPropsJSON(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>{Object.keys(props).join(",")}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	type props struct {
		Name  string `json:"name"`
		Empty string `json:"empty,omitempty"`
		Ptr   *int   `json:"ptr,omitzero"`
	}
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: props{Name: "World"}}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<div>name</div>`)
	diff.TestString(t, doc.Find("#react-props").Text(), `{"name":"World"}`)
}

func TestRenderPropsNilSlice(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>{props.items.length}:{Object.keys(props.meta).length}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	type props struct {
		Items []string       `json:"items"`
		Meta  map[string]int `json:"meta"`
	}
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: props{}}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<div>0<!-- -->:<!-- -->0</div>`)
	diff.TestString(t, doc.Find("#react-props").Text(), `{"items":[],"meta":{}}`)
}

func TestServeNodeModulesTraversal(t *testing.T) {
	is := is.New(t)
	root := t.TempDir()
	dir := filepath.Join(root, "app")
	is.NoErr(virt.Write(virt.Map{
		"secret.json":      `{"password":"hunter2"}`,
		"app/server.json":  `{"token":"abc123"}`,
		"app/package.json": packageJson,
		"app/index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
	}, root))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/node_modules/react/../../secret.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusNotFound)
	is.True(!strings.Contains(rec.Body.String(), "hunter2"))
	// Bare paths must resolve as packages, not as files in the project
	req = httptest.NewRequest(http.MethodGet, "/node_modules/server.json", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.True(!strings.Contains(rec.Body.String(), "abc123"))
	// Packages still resolve, including subpath exports
	req = httptest.NewRequest(http.MethodGet, "/node_modules/react-dom/client", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusOK)
	contains(t, rec.Body.String(), "createRoot")
}

func TestServeNodeModulesProduction(t *testing.T) {
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
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithMinify(true),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	// Minified builds bundle node_modules, so the route isn't needed
	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/node_modules/react", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusNotFound)
}

func TestServeStaticJSPassthrough(t *testing.T) {
	is := is.New(t)
	root := t.TempDir()
	dir := filepath.Join(root, "app")
	is.NoErr(virt.Write(virt.Map{
		"secret.jsx":       `export default "hunter2"`,
		"app/package.json": packageJson,
		"app/index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
	}, root))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithCache(true),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	handler := viewer.Middleware(http.NotFoundHandler())
	// Non-page scripts are left to the next handler
	req := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusNotFound)
	// Missing pages too
	req = httptest.NewRequest(http.MethodGet, "/missing.jsx.js", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusNotFound)
	// Pages outside the directory aren't built
	req = httptest.NewRequest(http.MethodGet, "/../secret.jsx.js", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusNotFound)
	is.True(!strings.Contains(rec.Body.String(), "hunter2"))
	// Pages are still served
	req = httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusOK)
	contains(t, rec.Body.String(), `"Hello, "`)
}

func TestServeFS(t *testing.T) {
	is := is.New(t)
	viewer, err := jig.Load(
		jig.WithDir(t.TempDir()),
		jig.WithWatch(false),
		jig.WithCache(true),
		jig.WithFS(fstest.MapFS{
			"dist/dom/index.jsx.js": {Data: []byte(`import "./chunk-ABC.js"`)},
			"dist/dom/chunk-ABC.js": {Data: []byte(`console.log("chunk")`)},
		}),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/chunk-ABC.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusOK)
	is.Equal(rec.Body.String(), `console.log("chunk")`)
	req = httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusNotFound)
}

func TestServeFSETag(t *testing.T) {
	is := is.New(t)
	fsys := fstest.MapFS{
		"dist/dom/index.jsx.js": {Data: []byte(`console.log("v1")`)},
	}
	load := func() http.Handler {
		viewer, err := jig.Load(
			jig.WithDir(t.TempDir()),
			jig.WithWatch(false),
			jig.WithCache(true),
			jig.WithFS(fsys),
		)
		is.NoErr(err)
		t.Cleanup(func() {
			is.NoErr(viewer.Close())
		})
		return viewer.Middleware(http.NotFoundHandler())
	}
	serve := func(handler http.Handler, etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil)
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	handler := load()
	rec := serve(handler, "")
	is.Equal(rec.Code, http.StatusOK)
	is.Equal(rec.Header().Get("Cache-Control"), "no-cache")
	etag := rec.Header().Get("ETag")
	is.True(etag != "")
	// Unchanged code isn't sent again
	rec = serve(handler, etag)
	is.Equal(rec.Code, http.StatusNotModified)
	is.Equal(rec.Body.Len(), 0)
	// A deploy with new code is sent again
	fsys["dist/dom/index.jsx.js"] = &fstest.MapFile{Data: []byte(`console.log("v2")`)}
	rec = serve(load(), etag)
	is.Equal(rec.Code, http.StatusOK)
	is.Equal(rec.Body.String(), `console.log("v2")`)
	is.True(rec.Header().Get("ETag") != etag)
}

func TestRenderErrorKeepsErrorView(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				throw new Error(props.fail);
			}
		`,
		"error.jsx": `
			import React from 'react';
			export default function Error(props) {
				if (props.error.message === "error view fails") {
					throw props.error;
				}
				return <div>Custom: {props.error.message}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithCache(true),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	// The error view fails, so the default error view is used
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"fail": "error view fails"}}))
	contains(t, actual.String(), "An error occurred: <!-- -->error view fails")
	// The next error still uses the custom error view
	actual = new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"fail": "boom"}}))
	contains(t, actual.String(), "Custom: <!-- -->boom")
}

func TestRenderConcurrentDev(t *testing.T) {
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
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithCache(false),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	handler := viewer.Middleware(http.NotFoundHandler())
	var wg sync.WaitGroup
	for range 64 {
		wg.Go(func() {
			actual := new(bytes.Buffer)
			if err := viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}); err != nil {
				t.Error(err)
			} else if !strings.Contains(actual.String(), `<div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div>`) {
				t.Errorf("unexpected render %q", actual.String())
			}
		})
		wg.Go(func() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil))
			if !strings.Contains(rec.Body.String(), `"Hello, "`) {
				t.Errorf("unexpected client code %.100q", rec.Body.String())
			}
		})
	}
	wg.Wait()
	// Builds on demand stay in memory
	_, err = os.Stat(filepath.Join(dir, "dist"))
	is.True(errors.Is(err, fs.ErrNotExist))
}

func TestBuild(t *testing.T) {
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
			export default function About(props) {
				return <div>I'm {props.name}!</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	builder, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithMinify(true),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	is.NoErr(builder.Build(ctx, "."))
	is.NoErr(builder.Close())
	// The goja shims are only for the server
	ssrCode, err := os.ReadFile(filepath.Join(dir, "dist", "ssr", "views.js"))
	is.NoErr(err)
	contains(t, string(ssrCode), "_otherPort")
	domFiles, err := filepath.Glob(filepath.Join(dir, "dist", "dom", "*.js"))
	is.NoErr(err)
	is.True(len(domFiles) > 0)
	for _, path := range domFiles {
		code, err := os.ReadFile(path)
		is.NoErr(err)
		if strings.Contains(string(code), "_otherPort") {
			t.Fatalf("%s contains the goja shims", filepath.Base(path))
		}
	}

	// Serve from the built files
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithCache(true),
		jig.WithMinify(true),
		jig.WithLiveRoute(""),
		jig.WithFS(os.DirFS(dir)),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "about.jsx", Props: map[string]any{"name": "World"}}))
	contains(t, actual.String(), `<div id="root"><div>I&#x27;m <!-- -->World<!-- -->!</div></div>`)
	handler := viewer.Middleware(http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil))
	is.Equal(rec.Code, http.StatusOK)
	contains(t, rec.Body.String(), "Hello, ")
}

// files lists the files in dir, relative to dir
func files(t testing.TB, dir string) (paths []string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		paths = append(paths, filepath.ToSlash(rel))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func TestBuildWalk(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		".gitignore":   "ignored/\n",
		"greet.ts":     `export const greet = (name: string) => "Hello, " + name`,
		"index.jsx": `
			import React from 'react';
			import { greet } from './greet.ts';
			export default function Index(props) {
				return <div>{greet(props.name)}</div>;
			}
		`,
		"users/show.tsx": `
			import React from 'react';
			import { greet } from '../greet.ts';
			export default function Show(props: { name: string }) {
				return <div>{greet(props.name)}</div>;
			}
		`,
		"users/layout.tsx": `
			import React from 'react';
			export default function Layout(props: { head: React.ReactNode, children: React.ReactNode }) {
				return <html><head>{props.head}</head><body className="users">{props.children}</body></html>;
			}
		`,
		"users/error.tsx": `
			import React from 'react';
			export default function Error(props: { error: Error }) {
				return <div>{props.error.message}</div>;
			}
		`,
		"ignored/page.jsx":      `export default () => null`,
		"dist/dom/stale.jsx.js": `old`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	is.NoErr(virt.Write(virt.Map{
		"node_modules/fake/page.jsx": `export default () => null`,
	}, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithMinify(true),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	is.NoErr(viewer.Build(ctx, "."))
	// Every view is in one bundle for the server
	is.Equal(files(t, filepath.Join(dir, "dist", "ssr")), []string{"views.js"})
	// .jsx and .tsx pages share chunks
	domFiles := files(t, filepath.Join(dir, "dist", "dom"))
	is.Equal(len(domFiles), 3)
	is.True(strings.HasPrefix(domFiles[0], "chunk-"))
	is.Equal(domFiles[1:], []string{"index.jsx.js", "users/show.tsx.js"})
	// Pages render inside the layouts they're given
	viewer, err = jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithCache(true),
		jig.WithMinify(true),
		jig.WithLiveRoute(""),
		jig.WithFS(os.DirFS(dir)),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual,
		jig.View{Name: "users/layout.tsx"},
		jig.View{Name: "users/show.tsx", Props: map[string]any{"name": "Jig"}},
	))
	contains(t, actual.String(), `<body class="users"><div id="root"><div>Hello, Jig</div></div>`)
	// The bundle only has the views that were built
	actual = new(bytes.Buffer)
	err = viewer.Render(ctx, actual, jig.View{Name: "users/missing.tsx"})
	is.True(errors.Is(err, fs.ErrNotExist))
	is.Equal(actual.Len(), 0)
}

func TestBuildNoPages(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"layout.jsx": `export default (props) => props.children`,
	}, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	err = viewer.Build(context.Background(), ".")
	is.True(err != nil)
	contains(t, err.Error(), "jig: no pages in ")
}

func TestBuildViewsDir(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"view/index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
		"scripts/page.jsx": `export default () => null`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithMinify(true),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	// Only the views directory is built, but pages are still relative to dir
	is.NoErr(viewer.Build(ctx, "view"))
	is.Equal(files(t, filepath.Join(dir, "dist", "ssr")), []string{"views.js"})
	is.Equal(files(t, filepath.Join(dir, "dist", "dom")), []string{"view/index.jsx.js"})
}

func TestBuildCleansOutDir(t *testing.T) {
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
		"dist/ssr/removed.jsx.js": `old`,
		"dist/dom/chunk-OLD.js":   `old`,
		"dist/other.txt":          `keep`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithMinify(true),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	is.NoErr(viewer.Build(ctx, "."))
	_, err = os.Stat(filepath.Join(dir, "dist", "ssr", "removed.jsx.js"))
	is.True(errors.Is(err, fs.ErrNotExist))
	_, err = os.Stat(filepath.Join(dir, "dist", "dom", "chunk-OLD.js"))
	is.True(errors.Is(err, fs.ErrNotExist))
	_, err = os.Stat(filepath.Join(dir, "dist", "ssr", "views.js"))
	is.NoErr(err)
	_, err = os.Stat(filepath.Join(dir, "dist", "dom", "index.jsx.js"))
	is.NoErr(err)
	_, err = os.Stat(filepath.Join(dir, "dist", "other.txt"))
	is.NoErr(err)
}

func TestBuildMinifyProductionReact(t *testing.T) {
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
	// Like cmd/react under go run: not production, but minified
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithMinify(true),
		jig.WithLiveRoute(""),
		jig.WithEnv(map[string]string{}),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	is.NoErr(viewer.Build(ctx, "."))
	// Development builds of React include full warning messages
	for _, path := range []string{"dist/ssr/views.js", "dist/dom/index.jsx.js"} {
		code, err := os.ReadFile(filepath.Join(dir, path))
		is.NoErr(err)
		is.True(!strings.Contains(string(code), "Invalid hook call"))
	}
}

func TestRenderMissingPage(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	err = viewer.Render(ctx, actual, jig.View{Name: "missing.jsx"})
	is.True(errors.Is(err, fs.ErrNotExist))
	is.Equal(actual.Len(), 0)
}

func TestRenderCanceled(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"loop.jsx": `
			import React from 'react';
			export default function Loop() {
				while (true) {}
			}
		`,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithCache(true),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	actual := new(bytes.Buffer)
	err = viewer.Render(timeoutCtx, actual, jig.View{Name: "loop.jsx"})
	is.True(errors.Is(err, context.DeadlineExceeded))
	is.Equal(actual.Len(), 0)
	// Other renders still work
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}))
	contains(t, actual.String(), `<div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div>`)
}

func TestLiveReload(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"index.jsx": `export default function Index() { return null }`,
	}, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(true),
		jig.WithLiveRoute("/.live"),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	server := httptest.NewServer(viewer.Middleware(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/.live", nil)
	is.NoErr(err)
	req.Header.Set("Accept", "text/event-stream")
	res, err := server.Client().Do(req)
	is.NoErr(err)
	defer res.Body.Close()
	// Keep editing until the event arrives, since the subscription is
	// registered asynchronously
	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancel()
	wg.Go(func() {
		for i := 0; ctx.Err() == nil; i++ {
			os.WriteFile(filepath.Join(dir, "index.jsx"), []byte(`export default function Index() { return `+strconv.Itoa(i)+` }`), 0644)
			time.Sleep(50 * time.Millisecond)
		}
	})
	scanner := bufio.NewScanner(res.Body)
	var data string
	for scanner.Scan() {
		if after, ok := strings.CutPrefix(scanner.Text(), "data: "); ok {
			data = after
			break
		}
	}
	is.NoErr(scanner.Err())
	var events []struct {
		Op   string `json:"op"`
		Path string `json:"path"`
	}
	is.NoErr(json.Unmarshal([]byte(data), &events))
	is.Equal(len(events), 1)
	is.True(events[0].Op != "")
	is.Equal(events[0].Path, "index.jsx")
}

func TestServeNodeModulesETag(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json":                    `{"name":"app","dependencies":{"greet":"*"}}`,
		"node_modules/greet/package.json": `{"name":"greet","main":"index.js"}`,
		"node_modules/greet/index.js":     `export const greeting = "v1"`,
	}, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	handler := viewer.Middleware(http.NotFoundHandler())
	serve := func(etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/node_modules/greet", nil)
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	rec := serve("")
	is.Equal(rec.Code, http.StatusOK)
	is.Equal(rec.Header().Get("Cache-Control"), "no-cache")
	etag := rec.Header().Get("ETag")
	is.True(etag != "")
	// Unchanged packages aren't rebuilt or sent again, even after a restart.
	// Break the source without changing the package files to check esbuild
	// doesn't run.
	is.NoErr(os.WriteFile(filepath.Join(dir, "node_modules", "greet", "index.js"), []byte(`export const = `), 0644))
	rec = serve(etag)
	is.Equal(rec.Code, http.StatusNotModified)
	is.Equal(rec.Body.Len(), 0)
	// Without the ETag it's rebuilt
	rec = serve("")
	is.Equal(rec.Code, http.StatusInternalServerError)
	is.NoErr(os.WriteFile(filepath.Join(dir, "node_modules", "greet", "index.js"), []byte(`export const greeting = "v2"`), 0644))
	// Installing packages changes package.json or the lockfile
	is.NoErr(os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{}`), 0644))
	rec = serve(etag)
	is.Equal(rec.Code, http.StatusOK)
	is.True(rec.Header().Get("ETag") != etag)
}

func TestRenderHead(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return (
					<div>
						<head>
							<title>Hello</title>
							<meta charSet="utf-8" />
							<meta name="description" content="A page" />
						</head>
						<h1>Hello, {props.name}!</h1>
					</div>
				);
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	logs := new(bytes.Buffer)
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithLiveRoute(""),
		jig.WithLog(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual, jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}))
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	is.Equal(doc.Find("head title").Text(), "Hello")
	is.Equal(doc.Find("head meta[charset]").Length(), 1)
	is.Equal(doc.Find(`head meta[name="description"]`).AttrOr("content", ""), "A page")
	is.Equal(logs.String(), "")
}

const layoutJsx = `
	import React from 'react';
	export default function Layout(props) {
		return <html><head>{props.head}</head><body><nav>{props.user}</nav>{props.children}</body></html>;
	}
`

func TestRenderLayout(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"layout.jsx":   layoutJsx,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual,
		jig.View{Name: "layout.jsx", Props: map[string]any{"user": "Ann", "head": "replaced"}},
		jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}},
	))
	// The layout's props render on the server, but aren't sent to the browser.
	// The head replaces a prop of the same name.
	expect := `<!doctype html><html><head><link rel="modulepreload" href="/index.jsx.js"/></head><body><nav>Ann</nav><div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div><script type="application/json" id="react-props">{"name":"World"}</script><script type="module">import mount from "/index.jsx.js"; mount()</script></body></html>`
	diff.TestString(t, actual.String(), expect)
}

func TestRenderStackInvalid(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"layout.jsx": layoutJsx,
		"frame.jsx":  `export default (props) => props.children`,
		"index.jsx":  `export default () => null`,
	}, dir))
	ctx := context.Background()
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)

	err = viewer.Render(ctx, actual)
	is.True(err != nil)
	is.Equal(err.Error(), `jig: no views to render`)

	// A layout can't be the page
	err = viewer.Render(ctx, actual, jig.View{Name: "layout.jsx"})
	is.True(err != nil)
	is.Equal(err.Error(), `jig: layout "layout.jsx" can't be the page`)
	err = viewer.Render(ctx, actual, jig.View{Name: "index.jsx"}, jig.View{Name: "layout.jsx"})
	is.True(err != nil)
	is.Equal(err.Error(), `jig: layout "layout.jsx" can't be the page`)

	// Only a layout wraps a page, and it comes first
	err = viewer.Render(ctx, actual, jig.View{Name: "frame.jsx"}, jig.View{Name: "index.jsx"})
	is.True(err != nil)
	is.Equal(err.Error(), `jig: "frame.jsx" can't wrap a page yet`)
	err = viewer.Render(ctx, actual, jig.View{Name: "layout.jsx"}, jig.View{Name: "admin/layout.jsx"}, jig.View{Name: "index.jsx"})
	is.True(err != nil)
	is.Equal(err.Error(), `jig: layout "admin/layout.jsx" must come first`)

	err = viewer.Render(ctx, actual, jig.View{Name: "admin/layout.jsx"}, jig.View{Name: "index.jsx"})
	is.True(errors.Is(err, fs.ErrNotExist))

	is.Equal(actual.Len(), 0)
}

func TestRenderLayoutErrorView(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"layout.jsx":   layoutJsx,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				throw new Error("boom");
			}
		`,
		"error.jsx": `
			import React from 'react';
			export default function Error(props) {
				return <div>Custom: {props.error.message}</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	is.NoErr(viewer.Render(ctx, actual,
		jig.View{Name: "layout.jsx", Props: map[string]any{"user": "Ann"}},
		jig.View{Name: "index.jsx"},
	))
	// The error view renders inside the layout, and doesn't hydrate
	expect := `<!doctype html><html><head></head><body><nav>Ann</nav><div id="root"><div>Custom: <!-- -->boom</div></div></body></html>`
	diff.TestString(t, actual.String(), expect)
}

func TestRenderConcurrentStacks(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"layout.jsx":   layoutJsx,
		"index.jsx": `
			import React from 'react';
			export default function Index(props) {
				return <div>Hello, {props.name}!</div>;
			}
		`,
		"about.jsx": `
			import React from 'react';
			export default function About(props) {
				return <div>I'm {props.name}!</div>;
			}
		`,
	}, dir))
	ctx := context.Background()
	is.NoErr(npm.Install(ctx, dir))
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithWatch(false),
		jig.WithCache(true),
		jig.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	layout := jig.View{Name: "layout.jsx", Props: map[string]any{"user": "Ann"}}
	index := jig.View{Name: "index.jsx", Props: map[string]any{"name": "World"}}
	about := jig.View{Name: "about.jsx", Props: map[string]any{"name": "World"}}
	render := func(expect string, views ...jig.View) {
		actual := new(bytes.Buffer)
		if err := viewer.Render(ctx, actual, views...); err != nil {
			t.Error(err)
		} else if !strings.Contains(actual.String(), expect) {
			t.Errorf("render %q does not contain %q", actual.String(), expect)
		}
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			render(`<body><div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div>`, index)
		})
		wg.Go(func() {
			render(`<body><nav>Ann</nav><div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div>`, layout, index)
		})
		wg.Go(func() {
			render(`<body><nav>Ann</nav><div id="root"><div>I&#x27;m <!-- -->World<!-- -->!</div></div>`, layout, about)
		})
	}
	wg.Wait()
}
