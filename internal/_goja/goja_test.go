package goja_test

import (
	"context"
	"testing"

	"github.com/matryer/is"
	"github.com/matthewmueller/elmo/internal/goja"
	"github.com/matthewmueller/elmo/internal/log"
)

func TestPromiseResolve(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	log := log.Default()
	vm := goja.New(log)
	result, err := vm.Render(ctx, "test.js", `
		new Promise((resolve, reject) => {
			resolve("hello")
		})
	`)
	is.NoErr(err)
	is.Equal(result, "hello")
}

func TestPromiseReject(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	log := log.Default()
	vm := goja.New(log)
	result, err := vm.Render(ctx, "test.js", `
		new Promise((resolve, reject) => {
			reject("hello")
		})
	`)
	is.True(err != nil)
	is.Equal(err.Error(), `goja: promise was rejected with "hello"`)
	is.Equal(result, "")
}

func TestURLPathname(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	log := log.Default()
	vm := goja.New(log)
	result, err := vm.Render(ctx, "test.js", `
		const url = new URL("https://sub.example.com:8080/p/a/t/h?query=string#hash")
		url.pathname
	`)
	is.NoErr(err)
	is.Equal(result, "/p/a/t/h")
}
