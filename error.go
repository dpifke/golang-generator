package generator

import (
	"bytes"
	"fmt"
)

// ParseError provides an error implementation which optionally includes
// source filename and line or row number information.  It can encapsulate
// another error, or be built with an error string.
type ParseError struct {
	Filename  string
	Line, Row int
	Err       error
	ErrStr    string
}

func (e *ParseError) Error() string {
	var b bytes.Buffer

	// File location can be either row (for CSV files) or line.  Both are
	// optional.
	if e.Row > 0 {
		b.WriteString(fmt.Sprintf("row %d", e.Row))
	} else if e.Line > 0 {
		b.WriteString(fmt.Sprintf("line %d", e.Line))
	}

	if b.Len() > 0 && e.Filename != "" {
		b.WriteString(" of ")
	}
	b.WriteString(e.Filename)

	// Error can be nested, or a simple string.
	var errStr string
	if e.Err != nil {
		errStr = e.Err.Error()
	} else {
		errStr = e.ErrStr
	}
	if b.Len() > 0 && errStr != "" {
		b.WriteString(": ")
	}
	b.WriteString(errStr)

	return b.String()
}
