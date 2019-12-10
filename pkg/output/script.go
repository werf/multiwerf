package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/fatih/color"
)

type ScriptPrint struct {
}

func NewScriptPrint() *ScriptPrint {
	return &ScriptPrint{}
}

func (s *ScriptPrint) Cprintf(colorAttribute *color.Attribute, format string, args ...interface{}) (n int, err error) {
	msg := fmt.Sprintf(format, args...)
	if msg == "" {
		return
	}

	if colorAttribute == nil {
		return fmt.Printf("echo %s\n", shellescape.Quote(msg))
	}

	return fmt.Printf("echo %s\n", color.New(*colorAttribute).Sprintf(shellescape.Quote(msg)))
}

func (s *ScriptPrint) CommentPrintf(format string, args ...interface{}) (n int, err error) {
	return fmt.Print("# ", fmt.Sprintf(format, args...), "\n")
}

// Message output comment and message in script form:
// # comment
// echo -e color_code message stop_color_code
func (s *ScriptPrint) Message(msg string, colorAttribute *color.Attribute, comment string) {
	s.CommentPrintf(comment)
	s.Cprintf(colorAttribute, msg)
}

// DebugMessage output a message in gray color
func (s *ScriptPrint) DebugMessage(msg string, comment string) {
	s.CommentPrintf(comment)
	s.Cprintf(nil, msg)
}

func (s *ScriptPrint) Error(err error) {
	if err.Error() != "" {
		s.Cprintf(&RedColor, "%v\n", err)
	}
	fmt.Println("return 1")
	return
}

type Script struct {
	Printer *ScriptPrint
}

func NewScript() *Script {
	return &Script{
		Printer: NewScriptPrint(),
	}
}

// PrintDefaultBinaryAliasFunction prints a shell script with alias function
// TODO Add script block to prevent from loading not in bash/zsh shells (as in rvm script)
func (s *Script) PrintDefaultBinaryAliasFunction(name, path string) error {
	fmt.Printf(`#

# Function with a path to the chosen version of %s binary.
%[1]s() {
  case "$1" in
    --path) echo '%[2]s';;
         *) %[2]s "$@";;
  esac
}

# To start using werf source this output:
# * Bourne shell (sh): tmpfile=$(mktemp) && multiwerf %[3]s > $tmpfile && . $tmpfile
# * Bash (< 4.0):      source /dev/stdin <<<"$(multiwerf %[3]s)"
# * Bash (>=4.0), zsh: source <(multiwerf %[3]s)

# To remove function use the following command: unset -f %[1]s
`, name, filepath.ToSlash(path), strings.Join(os.Args[1:], " "))
	return nil
}

// PrintBinaryAliasFunctionForPowerShell prints a powershell script with alias function
func (s *Script) PrintBinaryAliasFunctionForPowerShell(name, path string) error {
	fmt.Printf(`#

# Function with a path to the chosen version of %s binary.
function %[1]s {
  & %[2]s "$args"
}

# To start using werf source this output:
# * create temporary file: $tmpfile = [IO.Path]::GetTempFileName() | Rename-Item -NewName { $_ -replace 'tmp$', 'ps1' } –PassThru
# * save command output:   multiwerf use 1.0 alpha > $tmpfile
# * source it:             . $tmpfile

# To remove function use the following command: Remove-Item -Path Function:%[1]s
`, name, filepath.ToSlash(path), strings.Join(os.Args[1:], " "))
	return nil
}
