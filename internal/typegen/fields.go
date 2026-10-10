package typegen

import (
	"cmp"
	"errors"
	"fmt"
	"go/types"
	"reflect"
	"slices"
	"strings"
)

// field is a JSON object member of a Go struct
type field struct {
	goName   string
	index    []int
	typ      types.Type
	optional bool
	options
}

// options of a struct field's json tag
type options struct {
	name      string
	hasName   bool
	embed     bool
	omitzero  bool
	omitempty bool
	string    bool
	format    string
}

// parseOptions parses the json tag of v like encoding/json/v2. Returns false
// when v is ignored.
func parseOptions(v *types.Var, tag string) (opts options, ok bool) {
	value, _ := reflect.StructTag(tag).Lookup("json")
	if value == "-" || (!v.Exported() && !v.Embedded()) {
		return options{}, false
	}
	opts.name = v.Name()
	name, rest, _ := strings.Cut(value, ",")
	if name != "" {
		opts.name, opts.hasName = unquote(name), true
	}
	for rest != "" {
		// The format is always last, so it can contain commas
		if format, ok := strings.CutPrefix(rest, "format:"); ok {
			opts.format = unquote(format)
			break
		}
		var opt string
		opt, rest, _ = strings.Cut(rest, ",")
		switch opt {
		case "embed":
			opts.embed = true
		case "omitzero":
			opts.omitzero = true
		case "omitempty":
			opts.omitempty = true
		case "string":
			opts.string = true
		}
	}
	return opts, true
}

// unquote a single-quoted tag value
func unquote(value string) string {
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}

// structFields returns the JSON object members of st, in marshal order, and the
// value type of an embedded map that holds any other members. Follows
// makeStructFields in encoding/json/v2.
func structFields(st *types.Struct) (fields []field, fallback types.Type, err error) {
	type entry struct {
		st       *types.Struct
		index    []int
		optional bool // embedded through a pointer, which is omitted when nil
		visit    bool // whether to visit the structs it embeds
	}
	queue := []entry{{st, nil, false, true}}
	seen := map[*types.Struct]bool{st: true}
	var fallbacks []field

	// Search the embedded structs breadth-first, so fields are found from the
	// shallowest to the deepest
	for len(queue) > 0 {
		e := queue[0]
		queue = queue[1:]
		names := map[string]bool{}
		var tagged, members bool
		for i := range e.st.NumFields() {
			v, tag := e.st.Field(i), e.st.Tag(i)
			if _, ok := reflect.StructTag(tag).Lookup("json"); ok {
				tagged = true
			}
			opts, ok := parseOptions(v, tag)
			if !ok {
				continue
			}
			members = true
			f := field{
				goName:   v.Name(),
				index:    append(slices.Clone(e.index), i),
				typ:      v.Type(),
				optional: e.optional || opts.omitzero || opts.omitempty,
				options:  opts,
			}
			embedded := types.Unalias(v.Type())
			ptr, isPtr := embedded.(*types.Pointer)
			if isPtr {
				embedded = types.Unalias(ptr.Elem())
			}
			if v.Embedded() && !opts.hasName {
				if _, ok := embedded.Underlying().(*types.Struct); !ok {
					return nil, nil, fmt.Errorf("embedded field %s needs a JSON name", v.Name())
				}
				f.embed = true
			}

			// Regular fields are members
			if !f.embed {
				if !v.Exported() {
					return nil, nil, fmt.Errorf("field %s isn't exported", v.Name())
				} else if names[f.name] {
					return nil, nil, fmt.Errorf("fields conflict over the JSON name %q", f.name)
				}
				names[f.name] = true
				fields = append(fields, f)
				continue
			}

			// Embedded fields add their members to this struct
			if isType(embedded, "encoding/json/jsontext", "Value") {
				f.typ = embedded
				fallbacks = append(fallbacks, f)
				continue
			} else if _, ok := marshaler(embedded); ok {
				return nil, nil, fmt.Errorf("embedded field %s can't have marshal methods", v.Name())
			}
			switch under := embedded.Underlying().(type) {
			case *types.Struct:
				if e.visit {
					queue = append(queue, entry{under, f.index, f.optional || isPtr, !seen[under]})
				}
				seen[under] = true
			case *types.Map:
				if basic, ok := under.Key().Underlying().(*types.Basic); !ok || basic.Info()&types.IsString == 0 {
					return nil, nil, fmt.Errorf("embedded map field %s needs string keys", v.Name())
				}
				f.typ = under.Elem()
				fallbacks = append(fallbacks, f)
			default:
				return nil, nil, fmt.Errorf("embedded field %s must be a struct, map or jsontext.Value", v.Name())
			}
		}
		// json refuses structs without any fields it can marshal, like errors
		if e.st.NumFields() > 0 && !tagged && !members {
			return nil, nil, errors.New("struct has no exported fields")
		}
	}

	// Keep the dominant field of each name: the shallowest one, or the only
	// one at that depth that's tagged with the name
	slices.SortStableFunc(fields, func(x, y field) int {
		return cmp.Or(
			strings.Compare(x.name, y.name),
			cmp.Compare(len(x.index), len(y.index)),
			compareBools(!x.hasName, !y.hasName),
		)
	})
	dominant := fields[:0]
	for len(fields) > 0 {
		n := 1
		for n < len(fields) && fields[n-1].name == fields[n].name {
			n++
		}
		if n == 1 || len(fields[0].index) != len(fields[1].index) || fields[0].hasName != fields[1].hasName {
			dominant = append(dominant, fields[0])
		}
		fields = fields[n:]
	}
	slices.SortFunc(dominant, func(x, y field) int {
		return slices.Compare(x.index, y.index)
	})

	// The shallowest embedded map or jsontext.Value holds the other members
	if len(fallbacks) > 1 && len(fallbacks[0].index) == len(fallbacks[1].index) {
		return nil, nil, fmt.Errorf("embedded fields %s and %s can't both be a map or jsontext.Value", fallbacks[0].goName, fallbacks[1].goName)
	} else if len(fallbacks) > 0 {
		fallback = fallbacks[0].typ
	}
	return dominant, fallback, nil
}

// compareBools orders false before true
func compareBools(x, y bool) int {
	switch {
	case x == y:
		return 0
	case !x:
		return -1
	}
	return 1
}
