package goja

import (
	"log/slog"
	"strings"

	"github.com/dop251/goja"
)

func console(log *slog.Logger) map[string]any {
	return map[string]any{
		"log": func(args ...goja.Value) {
			log.Info(formatConsole(args))
		},
		"warn": func(args ...goja.Value) {
			log.Warn(formatConsole(args))
		},
		"error": func(args ...goja.Value) {
			log.Error(formatConsole(args))
		},
	}
}

func formatConsole(args []goja.Value) string {
	if len(args) == 0 {
		return ""
	}

	format := args[0].String()
	var message strings.Builder
	argIndex := 1
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 == len(format) {
			message.WriteByte(format[i])
			continue
		}

		switch format[i+1] {
		case '%':
			message.WriteByte('%')
			i++
		case 'c':
			if argIndex < len(args) {
				argIndex++
			}
			i++
		case 's', 'd', 'i', 'f', 'o', 'O':
			if argIndex >= len(args) {
				message.WriteString(format[i : i+2])
				i++
				continue
			}
			message.WriteString(args[argIndex].String())
			argIndex++
			i++
		default:
			message.WriteByte(format[i])
		}
	}
	for ; argIndex < len(args); argIndex++ {
		if message.Len() > 0 {
			message.WriteByte(' ')
		}
		message.WriteString(args[argIndex].String())
	}
	return message.String()
}
