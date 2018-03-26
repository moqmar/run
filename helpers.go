package main

import (
	"strings"
)

func min(a int, b int) int {
	if a > b {
		return b
	}
	return a
}

func wrap(msg string, indent int, width int) string {
	msgRunes := []rune(msg)
	msgLen := len(msgRunes)
	lineLen := width - indent
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += " "
	}

	if strings.Contains(msg, "\n") {
		x := ""
		for _, y := range strings.Split(msg, "\n") {
			x += indentStr + wrap(y, indent, width) + "\n"
		}
		return strings.Trim(strings.TrimPrefix(x, indentStr), "\n ")
	}

	res := msg
	if lineLen > 0 {
		var pos int
		res = string(msgRunes[pos:min(pos+lineLen, msgLen)])
		for pos += lineLen; pos < msgLen; pos += lineLen {
			res += "\n" + indentStr + string(msgRunes[pos:min(pos+lineLen, msgLen)])
		}
	}

	return res
}
