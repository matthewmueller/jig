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

// func (v *VM) Render(w io.Writer, path, code string, props any) error {
// 	vm, ok := v.p.Get().(*goja.Runtime)
// 	if !ok {
// 		return fmt.Errorf("goja: unable to acquire vm")
// 	}
// 	res, err := render(vm, path, code)
// 	v.p.Put(vm)
// 	if err != nil {
// 		return err
// 	}
// 	_, err = io.WriteString(w, res)
// 	return err
// }

// func render(vm *goja.Runtime, path, code string) (string, error) {
// 	value, err := vm.RunScript(path, code)
// 	if err != nil {
// 		return "", err
// 	}
// 	// TODO: remove
// 	if promise, ok := value.Export().(*goja.Promise); ok {
// 		switch promise.State() {
// 		case goja.PromiseStateFulfilled:
// 			return promise.Result().String(), nil
// 		case goja.PromiseStateRejected:
// 			return "", fmt.Errorf("goja: promise was rejected with %q", promise.Result().String())
// 		case goja.PromiseStatePending:
// 			// Note: eventloop should have handled this state
// 			return "", fmt.Errorf("goja: promise is in an unexpected pending state")
// 		}
// 	}
// 	return value.String(), nil
// }
