package utils

import (
	"io"
	"text/template"
)

type Template struct {
	Template *template.Template
	FuncMap  template.FuncMap
}

func NewTemplate(name string) *Template {
	t := &Template{
		Template: template.New(name),
	}
	return t.Funcs(Funcs)
}

func (t *Template) Funcs(funcMap template.FuncMap) *Template {
	t.Template.Funcs(funcMap)
	if t.FuncMap == nil {
		t.FuncMap = template.FuncMap{}
	}
	for name, f := range funcMap {
		if _, ok := t.FuncMap[name]; ok {
			t.FuncMap[name] = f
		}
	}
	return t
}

func (t *Template) Parse(text string) (*Template, error) {
	if _, err := t.Template.Parse(text); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Template) Execute(wr io.Writer, data any) error {
	return t.Template.Execute(wr, data)
}

func TemplateMust(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}
	return t
}
