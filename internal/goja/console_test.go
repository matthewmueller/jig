package goja_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/matryer/is"
	"github.com/matthewmueller/jig/internal/goja"
)

func TestConsoleLogObject(t *testing.T) {
	is := is.New(t)
	output := new(bytes.Buffer)
	pool := goja.New(slog.New(slog.NewTextHandler(output, nil)))
	vm := pool.Acquire()
	t.Cleanup(func() {
		pool.Release(vm)
	})

	_, err := vm.RunString(`console.log({ name: "Jig", port: 9000 })`)
	is.NoErr(err)
	is.True(strings.Contains(output.String(), `msg="{ name: 'Jig', port: 9000 }"`))
}

func TestConsoleLogObjectValues(t *testing.T) {
	is := is.New(t)
	output := new(bytes.Buffer)
	pool := goja.New(slog.New(slog.NewTextHandler(output, nil)))
	vm := pool.Acquire()
	t.Cleanup(func() {
		pool.Release(vm)
	})

	_, err := vm.RunString(`console.log("config:", { enabled: true, tags: ["ssr", "jsx"] })`)
	is.NoErr(err)
	is.True(strings.Contains(output.String(), `msg="config: { enabled: true, tags: [ 'ssr', 'jsx' ] }"`))

	_, err = vm.RunString(`console.log({ enabled: true }, "tail")`)
	is.NoErr(err)
	is.True(strings.Contains(output.String(), `msg="{ enabled: true } tail"`))

	_, err = vm.RunString(`console.log({}, [])`)
	is.NoErr(err)
	is.True(strings.Contains(output.String(), `msg="{} []"`))
}
