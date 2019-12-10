package output

import (
	"fmt"

	"github.com/fatih/color"
)

type SimplePrint struct {
}

func NewSimplePrint() *SimplePrint {
	return &SimplePrint{}
}

func (p *SimplePrint) Cprintf(colorAttribute *color.Attribute, format string, args ...interface{}) (n int, err error) {
	if colorAttribute == nil {
		return fmt.Printf(format, args...)
	}

	return color.New(*colorAttribute).Printf(format, args...)
}

func (p *SimplePrint) Error(err error) {
	if err.Error() != "" {
		p.Cprintf(&RedColor, "%v\n", err)
	}
}

func (p *SimplePrint) DebugMessage(message, comment string) {
	fmt.Printf("%s (%s)\n", message, comment)
}

func (p *SimplePrint) Message(message string, colorAttribute *color.Attribute, comment string) {
	if message != "" {
		p.Cprintf(colorAttribute, "%s\n", message)
	}
}
