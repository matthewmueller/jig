package gotext

import (
	"bytes"
	"text/template"
)

func MustCompile(path string, code string) *Template {
	template, err := Compile(path, code)
	if err != nil {
		panic(err)
	}
	return template
}

func Compile(path string, code string) (*Template, error) {
	tpl, err := template.New(path).Parse(code)
	if err != nil {
		return nil, err
	}
	return &Template{tpl}, nil
}

type Template struct {
	tpl *template.Template
}

func (t *Template) Render(data any) (string, error) {
	var out bytes.Buffer
	err := t.tpl.Execute(&out, data)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
