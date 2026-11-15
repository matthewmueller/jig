package goja

import (
	"log/slog"
	"runtime"

	"github.com/dop251/goja"
	"github.com/matthewmueller/jig/internal/goja/urls"
)

type (
	Runtime  = goja.Runtime
	Value    = goja.Value
	Object   = goja.Object
	Callable = goja.Callable
	Program  = goja.Program
)

var (
	AssertFunction = goja.AssertFunction
	Compile        = goja.Compile
	Undefined      = goja.Undefined
)

func New(log *slog.Logger) *Pool {
	return &Pool{log, make(chan *goja.Runtime, runtime.GOMAXPROCS(0))}
}

// Pool keeps idle VMs, along with the bundles evaluated in them. Unlike
// sync.Pool, idle VMs aren't dropped during garbage collection, which
// rendering triggers often.
type Pool struct {
	log  *slog.Logger
	idle chan *goja.Runtime
}

// Acquire an idle VM or create one
func (v *Pool) Acquire() *goja.Runtime {
	select {
	case vm := <-v.idle:
		return vm
	default:
	}
	vm := goja.New()
	urls.Enable(vm)
	vm.GlobalObject().Set("console", console(v.log))
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	return vm
}

// Release the VM to be reused, dropping it when the pool is full
func (v *Pool) Release(vm *goja.Runtime) {
	select {
	case v.idle <- vm:
	default:
	}
}
