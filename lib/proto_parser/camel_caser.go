package proto_parser

import (
	"strings"
)

// Given a string like:
// set_foo
//
// Turn it into:
// setFoo
func CamelCaseName(name string) string {
	newName := ""
	nextIsUpper := false
	for _, c := range name {
		s := string(c)

		if s == "_" {
			nextIsUpper = true
			continue
		}

		if nextIsUpper {
			newName += strings.ToUpper(s)
			nextIsUpper = false
		} else {
			newName += s
		}
	}
	return newName
}
