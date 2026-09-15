package goja_test

import (
	"testing"

	"github.com/matryer/is"
	"github.com/matthewmueller/logs"
	"github.com/matthewmueller/views/internal/goja"
)

func TestNumberExponent(t *testing.T) {
	is := is.New(t)
	pool := goja.New(logs.Default())
	vm := pool.Acquire()
	t.Cleanup(func() {
		pool.Release(vm)
	})

	value, err := vm.RunString(`9e3`)
	is.NoErr(err)
	is.Equal(value.ToInteger(), int64(9000))
}
