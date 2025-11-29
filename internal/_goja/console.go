package goja

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/matthewmueller/elmo/internal/log"
)

func newConsole(log log.Interface) map[string]interface{} {
	return map[string]interface{}{
		"log": func(args ...goja.Value) {
			var params []interface{}
			for _, arg := range args {
				params = append(params, arg.String())
			}
			log.Info(fmt.Sprintln(params...))
		},
		"warn": func(args ...goja.Value) {
			var params []interface{}
			for _, arg := range args {
				params = append(params, arg.String())
			}
			log.Warn(fmt.Sprintln(params...))
		},
		"error": func(args ...goja.Value) {
			var params []interface{}
			for _, arg := range args {
				params = append(params, arg.String())
			}
			log.Error(fmt.Sprintln(params...))
		},
	}
}
