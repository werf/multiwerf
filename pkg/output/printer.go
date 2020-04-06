package output

import (
	"os"
	"runtime"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

type Printer interface {
	Cprintf(colorAttribute *color.Attribute, format string, args ...interface{}) (n int, err error)
	Error(err error)
	DebugMessage(message, comment string)
	Message(message string, colorAttribute *color.Attribute, comment string)
}

var (
	RedColor    = color.FgRed
	GreenColor  = color.FgGreen
	YellowColor = color.FgYellow
)

func init() {
	color.NoColor = runtime.GOOS == "windows" && !isatty.IsCygwinTerminal(os.Stdout.Fd())
}
