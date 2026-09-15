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

	if _, ok := args[0].(goja.String); !ok {
		return formatConsoleValues(args)
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
		case 's', 'd', 'i', 'f':
			if argIndex >= len(args) {
				message.WriteString(format[i : i+2])
				i++
				continue
			}
			message.WriteString(args[argIndex].String())
			argIndex++
			i++
		case 'o', 'O':
			if argIndex >= len(args) {
				message.WriteString(format[i : i+2])
				i++
				continue
			}
			message.WriteString(formatConsoleValue(args[argIndex], true, make(map[*goja.Object]bool)))
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
		message.WriteString(formatConsoleValue(args[argIndex], false, make(map[*goja.Object]bool)))
	}
	return message.String()
}

func formatConsoleValues(args []goja.Value) string {
	var message strings.Builder
	for i, arg := range args {
		if i > 0 {
			message.WriteByte(' ')
		}
		message.WriteString(formatConsoleValue(arg, false, make(map[*goja.Object]bool)))
	}
	return message.String()
}

func formatConsoleValue(value goja.Value, inspectString bool, seen map[*goja.Object]bool) string {
	if goja.IsUndefined(value) {
		return "undefined"
	}
	if goja.IsNull(value) {
		return "null"
	}
	if _, ok := value.(goja.String); ok {
		if !inspectString {
			return value.String()
		}
		return quoteConsoleString(value.String())
	}

	object, ok := value.(*goja.Object)
	if !ok {
		return value.String()
	}
	if seen[object] {
		return "[Circular]"
	}
	seen[object] = true
	defer delete(seen, object)

	className := object.ClassName()
	if className != "Object" && className != "Array" {
		return object.String()
	}
	keys := object.Keys()
	if className == "Array" {
		if len(keys) == 0 {
			return "[]"
		}
		values := make([]string, 0, len(keys))
		for _, key := range keys {
			values = append(values, formatConsoleValue(object.Get(key), true, seen))
		}
		return "[ " + strings.Join(values, ", ") + " ]"
	}
	if len(keys) == 0 {
		return "{}"
	}

	properties := make([]string, 0, len(keys))
	for _, key := range keys {
		properties = append(properties, formatConsoleKey(key)+": "+formatConsoleValue(object.Get(key), true, seen))
	}
	return "{ " + strings.Join(properties, ", ") + " }"
}

func quoteConsoleString(value string) string {
	return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n", "\r", "\\r").Replace(value) + "'"
}

func formatConsoleKey(key string) string {
	if key == "" {
		return quoteConsoleString(key)
	}
	for i, char := range key {
		if !(char == '_' || char == '$' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || (i > 0 && char >= '0' && char <= '9')) {
			return quoteConsoleString(key)
		}
	}
	return key
}
