package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

type SimplePrint struct {
	writer io.Writer
}

func NewSimplePrint(w io.Writer) *SimplePrint {
	return &SimplePrint{writer: w}
}

func (p *SimplePrint) Cprintf(colorAttribute *color.Attribute, format string, args ...interface{}) (n int, err error) {
	if colorAttribute == nil {
		return fmt.Fprintf(p.writer, format, args...)
	}

	return color.New(*colorAttribute).Fprintf(p.writer, format, args...)
}

func (p *SimplePrint) Error(err error) {
	if err.Error() != "" {
		_, _ = p.Cprintf(&RedColor, "Error: %v\n", err)
	}
}

func (p *SimplePrint) DebugMessage(message, comment string) {
	_, _ = fmt.Fprintf(p.writer, "%s (%s)\n", message, comment)
}

func (p *SimplePrint) Message(message string, colorAttribute *color.Attribute, comment string) {
	if message != "" {
		_, _ = p.Cprintf(colorAttribute, "%s\n", message)
	}
}
