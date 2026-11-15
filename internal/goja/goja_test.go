package goja_test

import (
	"runtime"
	"testing"

	"github.com/matryer/is"
	"github.com/matthewmueller/jig/internal/goja"
	"github.com/matthewmueller/logs"
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

// Idle VMs keep the bundles evaluated in them across garbage collections, so
// renders don't evaluate the bundle again
func TestPoolKeepsVMsAcrossGC(t *testing.T) {
	is := is.New(t)
	pool := goja.New(logs.Default())
	vm := pool.Acquire()
	_, err := vm.RunString(`var bundle = 1`)
	is.NoErr(err)
	pool.Release(vm)
	// sync.Pool drops idle values after two collections
	runtime.GC()
	runtime.GC()
	is.Equal(pool.Acquire(), vm)
}
