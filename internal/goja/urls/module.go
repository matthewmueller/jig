package urls

import (
	"github.com/dop251/goja"
)

type urlModule struct {
	r *goja.Runtime

	URLSearchParamsPrototype         *goja.Object
	URLSearchParamsIteratorPrototype *goja.Object
}

func Enable(runtime *goja.Runtime) {
	m := &urlModule{r: runtime}
	runtime.Set("URL", m.createURLConstructor())
	runtime.Set("URLSearchParams", m.createURLSearchParamsConstructor())
}
