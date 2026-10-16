package main

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidFormat is returned when -format is not md, html, or both.
var ErrInvalidFormat = errors.New("invalid -format value: must be md, html, or both")

// outputFormat selects which files are written for each conversation.
type outputFormat string

const (
	formatMarkdown outputFormat = "md"
	formatHTML     outputFormat = "html"
	formatBoth     outputFormat = "both"
)

// parseFormat converts the -format flag value into an outputFormat.
func parseFormat(value string) (outputFormat, error) {
	switch f := outputFormat(strings.ToLower(strings.TrimSpace(value))); f {
	case formatMarkdown, formatHTML, formatBoth:
		return f, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidFormat, value)
	}
}

// writesMarkdown reports whether Markdown files are written.
func (f outputFormat) writesMarkdown() bool {
	return f == formatMarkdown || f == formatBoth
}

// writesHTML reports whether HTML files and the index page are written.
func (f outputFormat) writesHTML() bool {
	return f == formatHTML || f == formatBoth
}
