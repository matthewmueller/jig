package goja

import (
	"testing"

	js "github.com/dop251/goja"
	"github.com/matryer/is"
)

func TestFormatConsole(t *testing.T) {
	is := is.New(t)
	runtime := js.New()
	actual := formatConsole([]js.Value{
		runtime.ToValue("Received `%s` for a non-boolean attribute `%s`. If you want to write it to the DOM, pass a string instead: %s=\"%s\" or %s={value.toString()}."),
		runtime.ToValue(true),
		runtime.ToValue("global"),
		runtime.ToValue("global"),
		runtime.ToValue(true),
		runtime.ToValue("global"),
	})
	is.Equal(actual, "Received `true` for a non-boolean attribute `global`. If you want to write it to the DOM, pass a string instead: global=\"true\" or global={value.toString()}.")
}
