package output

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

type SilentPrint struct {
}

func NewSilentPrint() *SilentPrint {
	return &SilentPrint{}
}

func (p *SilentPrint) Cprintf(_ *color.Attribute, format string, args ...interface{}) (n int, err error) {
	return 0, nil
}

func (p *SilentPrint) Error(err error) {
	if err.Error() != "" {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

func (p *SilentPrint) DebugMessage(message, comment string) {
	return
}

func (p *SilentPrint) Message(message string, _ *color.Attribute, comment string) {
	return
}
