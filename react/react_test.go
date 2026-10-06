package react_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
		react.WithEnv(map[string]string{
			"APP_NAME": "Views",
			"PORT":     "9000",
		}),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", nil)
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<div>Views<!-- --> on <!-- -->9000</div>`)

	handler := viewer.Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/index.jsx.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	res := rec.Result()
	is.Equal(res.StatusCode, http.StatusOK)
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.NoErr(res.Body.Close())
	contains(t, string(body), `"Views"`)
	contains(t, string(body), `"9000"`)
}

func TestRenderDefaultEnv(t *testing.T) {
	is := is.New(t)
	t.Setenv("VIEWS_RENDER_ENV_DEFAULT", "from-os")
	dir := t.TempDir()
	is.NoErr(virt.Write(virt.Map{
		"package.json": packageJson,
		"index.jsx": `
			import React from 'react';
			export default function Index() {
				return <div>{process.env.VIEWS_RENDER_ENV_DEFAULT}</div>;
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
	viewer.Render(actual, "index.jsx", nil)
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
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
		react.WithEnv(map[string]string{
			"APP_NAME": "Views",
		}),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", nil)
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
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", nil)
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
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})

	actual := new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", nil)
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
	contains(t, string(body), `props2.name,`)
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
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
		react.WithCache(true),
		react.WithLiveRoute(""),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	// Warm the cache
	viewer.Render(new(bytes.Buffer), "index.jsx", map[string]any{"name": "World"})
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			actual := new(bytes.Buffer)
			viewer.Render(actual, "index.jsx", map[string]any{"name": "World"})
			contains(t, actual.String(), `<div id="root"><div>Hello, <!-- -->World<!-- -->!</div></div>`)
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
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
	)
	is.NoErr(err)
	t.Cleanup(func() {
		is.NoErr(viewer.Close())
	})
	actual := new(bytes.Buffer)
	viewer.Render(actual, "index.jsx", map[string]any{"content": "</script><script>alert(1)</script>"})
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	is.Equal(doc.Find("script:not([src])").Length(), 1)
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
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithWatch(false),
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
	viewer.Render(actual, "index.jsx", props{Name: "World"})
	doc, err := goquery.NewDocumentFromReader(actual)
	is.NoErr(err)
	rootHTML, err := doc.Find("#root").Html()
	is.NoErr(err)
	diff.TestString(t, rootHTML, `<div>name</div>`)
	diff.TestString(t, doc.Find("#react-props").Text(), `{"name":"World"}`)
}
