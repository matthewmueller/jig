package react_test

import (
	"bytes"
	"context"
	"testing"

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
	log := logs.Default()
	react := react.New(log)
	react.Dir = dir
	actual := new(bytes.Buffer)
	err := react.Render(actual, "index.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
	expect := `<div>Hello, <!-- -->World<!-- -->!</div>`
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
	log := logs.Default()
	react := react.New(log)
	react.Dir = dir
	actual := new(bytes.Buffer)
	err := react.Render(actual, "index.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
	expect := `<div>Hello, <!-- -->World<!-- -->!</div>`
	diff.TestString(t, actual.String(), expect)

	actual = new(bytes.Buffer)
	err = react.Render(actual, "about.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
	expect = `<div>I&#x27;m <!-- -->World<!-- -->!</div>`
	diff.TestString(t, actual.String(), expect)

	actual = new(bytes.Buffer)
	err = react.Render(actual, "index.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
	expect = `<div>Hello, <!-- -->World<!-- -->!</div>`
	diff.TestString(t, actual.String(), expect)

	actual = new(bytes.Buffer)
	err = react.Render(actual, "about.jsx", map[string]any{"name": "World"})
	is.NoErr(err)
	expect = `<div>I&#x27;m <!-- -->World<!-- -->!</div>`
	diff.TestString(t, actual.String(), expect)

}
