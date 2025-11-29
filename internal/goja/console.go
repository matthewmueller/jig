package goja

import (
	"fmt"
	"log/slog"

	"github.com/dop251/goja"
)

func console(log *slog.Logger) map[string]any {
	return map[string]any{
		"log": func(args ...goja.Value) {
			var params []any
			for _, arg := range args {
				params = append(params, arg.String())
			}
			log.Info(fmt.Sprintln(params...))
		},
		"warn": func(args ...goja.Value) {
			var params []any
			for _, arg := range args {
				params = append(params, arg.String())
			}
			log.Warn(fmt.Sprintln(params...))
		},
		"error": func(args ...goja.Value) {
			var params []any
			for _, arg := range args {
				params = append(params, arg.String())
			}
			log.Error(fmt.Sprintln(params...))
		},
	}
}
