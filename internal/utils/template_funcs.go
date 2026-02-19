package utils

import (
	"text/template"
)

var Funcs = template.FuncMap{
	"add": add,
}

func add(a, b int) int {
	return a + b
}
