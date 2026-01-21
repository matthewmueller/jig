package goja

import (
	"log/slog"
	"sync"

	"github.com/dop251/goja"
	"github.com/matthewmueller/views/internal/goja/urls"
)

type (
	Runtime  = goja.Runtime
	Value    = goja.Value
	Callable = goja.Callable
)

var (
	AssertFunction = goja.AssertFunction
	Undefined      = goja.Undefined
)

func New(log *slog.Logger) *Pool {
	return &Pool{
		p: &sync.Pool{
			New: func() any {
				vm := goja.New()
				urls.Enable(vm)
				vm.GlobalObject().Set("console", console(log))
				vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
				return vm
			},
		},
	}
}

type Pool struct {
	p *sync.Pool
}

func (v *Pool) Acquire() *goja.Runtime {
	return v.p.Get().(*goja.Runtime)
}

func (v *Pool) Release(vm *goja.Runtime) {
	v.p.Put(vm)
}
