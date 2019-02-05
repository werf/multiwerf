package output

import (
	"fmt"
)

type ScriptPrint struct {
}

func NewScriptPrint() *ScriptPrint {
	return &ScriptPrint{}
}

func (s *ScriptPrint) Cprintf(color string, format string, args ...interface{}) (n int, err error) {
	if color == "" || color == "none" {
		return fmt.Printf("echo '%s'\n", fmt.Sprintf(format, args...))
	}

	return fmt.Printf("echo -e %s'%s'%s\n", ColorCodes[color]["quoted"], fmt.Sprintf(format, args...), ColorCodes["stop"]["quoted"])
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
	s.Cprintf("red", "%v\n", err)
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
# To remove function use unset:
# unset -f %[1]s
%[1]s() {
%s "$@"
}

`, name, path)
	return nil
}
