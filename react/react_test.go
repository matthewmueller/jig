package react_test

import (
	"bytes"
	"context"
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
	err = viewer.Render(actual, "index.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
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
	err = viewer.Render(actual, "index.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
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
	err = viewer.Render(actual, "about.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
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
	err = viewer.Render(actual, "index.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
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
	err = viewer.Render(actual, "about.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
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
