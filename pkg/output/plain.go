package output

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

type PlainPrint struct {
}

func NewPlainPrint() *PlainPrint {
	return &PlainPrint{}
}

func (p *PlainPrint) Cprintf(_ *color.Attribute, format string, args ...interface{}) (n int, err error) {
	return fmt.Printf(format, args...)
}

func (p *PlainPrint) Error(err error) {
	if err.Error() != "" {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

func (p *PlainPrint) DebugMessage(message, comment string) {
	fmt.Printf("%s (%s)\n", message, comment)
}

func (p *PlainPrint) Message(message string, _ *color.Attribute, comment string) {
	if message != "" {
		fmt.Printf("%s\n", message)
	}
}
