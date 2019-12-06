package output

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ScriptPrint struct {
}

func NewScriptPrint() *ScriptPrint {
	return &ScriptPrint{}
}

func (s *ScriptPrint) Cprintf(color string, format string, args ...interface{}) (n int, err error) {
	msg := fmt.Sprintf(format, args...)
	if msg == "" {
		return
	}

	if color == "" || color == "none" {
		return fmt.Printf("echo '%s'\n", EscapeSingleQuotes(msg))
	}

	return fmt.Printf("echo -e %s'%s'%s\n", ColorCodes[color]["quoted"], EscapeSingleQuotes(msg), ColorCodes["stop"]["quoted"])
}

func (s *ScriptPrint) CommentPrintf(format string, args ...interface{}) (n int, err error) {
	return fmt.Print("# ", fmt.Sprintf(format, args...), "\n")
}

// Message output comment and message in script form:
// # comment
// echo -e color_code message stop_color_code
func (s *ScriptPrint) Message(msg string, color string, comment string) {
	s.CommentPrintf(comment)
	s.Cprintf(color, msg)
}

// DebugMessage output a message in gray color
func (s *ScriptPrint) DebugMessage(msg string, comment string) {
	s.CommentPrintf(comment)
	s.Cprintf("none", msg)
}

func (s *ScriptPrint) Error(err error) {
	if err.Error() != "" {
		s.Cprintf("red", "%v\n", err)
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

// PrintBinaryAliasFunction prints a shell script with alias function
// TODO Add script block to prevent from loading not in bash/zsh shells (as in rvm script)
func (s *Script) PrintBinaryAliasFunction(name, path string) error {
	fmt.Printf(`#

# Function with path to chosen version of %s binary.
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

func EscapeSingleQuotes(s string) string {
	re := regexp.MustCompile(`'`)
	return re.ReplaceAllString(s, "'\"'\"'")
}
