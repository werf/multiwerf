package output

import (
	"fmt"
)

type SimplePrint struct {
}

func NewSimplePrint() *SimplePrint {
	return &SimplePrint{}
}

func (p *SimplePrint) Cprintf(color string, format string, args ...interface{}) (n int, err error) {
	if color == "" || color == "none" {
		return fmt.Printf(format, args...)
	}

	return fmt.Print(ColorCodes[color]["code"], fmt.Sprintf(format, args...), ColorCodes["stop"]["code"])
}

func (p *SimplePrint) Error(err error) {
	if err.Error() != "" {
		p.Cprintf("red", "%v\n", err)
	}
}

func (p *SimplePrint) DebugMessage(message, comment string) {
	fmt.Printf("%s (%s)\n", message, comment)
}

func (p *SimplePrint) Message(message, color, comment string) {
	if message != "" {
		p.Cprintf(color, "%s\n", message)
	}
}
