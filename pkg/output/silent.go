package output

import (
	"os"

	"github.com/fatih/color"
)

type SilentPrint struct {
	*SimplePrint
}

func NewSilentPrint() *SilentPrint {
	return &SilentPrint{
		SimplePrint: NewSimplePrint(os.Stdout),
	}
}

func (p *SilentPrint) Cprintf(_ *color.Attribute, format string, args ...interface{}) (n int, err error) {
	return 0, nil
}

func (p *SilentPrint) Message(message string, _ *color.Attribute, comment string) {
	return
}
