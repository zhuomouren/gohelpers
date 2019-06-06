package goini

import (
	"fmt"
)

// ini 文件语法错误
type syntaxError struct {
	File    string
	Line    int
	Message string
}

func (se *syntaxError) Error() string {
	return fmt.Sprintf("invalid syntax in %s on line %d: %s", se.File, se.Line, se.Message)
}

func newSyntaxError(file string, line int, message string) *syntaxError {
	return &syntaxError{
		File:    file,
		Line:    line,
		Message: message,
	}
}
