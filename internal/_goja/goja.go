package goja

import (
	"context"
	"fmt"
	"runtime"

	"github.com/dop251/goja"
	"github.com/jackc/puddle/v2"
	"github.com/matthewmueller/elmo/internal/goja/url"
	"github.com/matthewmueller/elmo/internal/log"
)

type Renderer interface {
	Render(ctx context.Context, path, code string) (string, error)
}

func New(log log.Interface) *Pool {
	pool, _ := puddle.NewPool(&puddle.Config[*goja.Runtime]{
		Constructor: func(ctx context.Context) (*goja.Runtime, error) {
			vm := goja.New()
			vm.GlobalObject().Set("console", newConsole(log))
			url.Enable(vm)
			return vm, nil
		},
		MaxSize: int32(runtime.NumCPU()),
	})
	return &Pool{pool}
}

type Pool struct {
	pool *puddle.Pool[*goja.Runtime]
}

var _ Renderer = (*Pool)(nil)

func (p *Pool) Render(ctx context.Context, path, code string) (string, error) {
	res, err := p.pool.Acquire(ctx)
	if err != nil {
		return "", fmt.Errorf("goja: unable to acquire vm: %w", err)
	}
	defer res.Release()
	return render(res.Value(), path, code)
}

func render(vm *goja.Runtime, path, code string) (string, error) {
	value, err := vm.RunScript(path, code)
	if err != nil {
		return "", err
	}
	// TODO: remove
	if promise, ok := value.Export().(*goja.Promise); ok {
		switch promise.State() {
		case goja.PromiseStateFulfilled:
			return promise.Result().String(), nil
		case goja.PromiseStateRejected:
			return "", fmt.Errorf("goja: promise was rejected with %q", promise.Result().String())
		case goja.PromiseStatePending:
			// Note: eventloop should have handled this state
			return "", fmt.Errorf("goja: promise is in an unexpected pending state")
		}
	}
	return value.String(), nil
}
