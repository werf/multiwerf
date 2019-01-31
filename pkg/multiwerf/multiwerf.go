package multiwerf

import (
	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/output"
)

var AvailableChannels = []string{
	"alpha",
	"beta",
	"rc",
	"stable",
}

// use and update actions send messages
type ActionMessage struct {
	stage   string // stage of a program
	action  string // action to perform (exit with error, exit 0, ...)
	msg     string // Text to print to the screen
	msgType string // message type: ok, warn, fail
	err     error  // Error message — display it in case of critical error before graceful exit
	comment string // minor message that displayed as a comment in a script output (can be grayed)
	debug bool // debug msg and comment are displayed only if --debug=yes flag is set
}

func Use(version string, channel string, args []string ) error {
	err := CheckMajorMinor(version)
	if err != nil {
		return err
	}

	messages := make(chan ActionMessage, 0)
	script := output.NewScript()

	SelfUpdate(messages, script.Printer)


	var binaryInfo BinaryInfo
	go func() {
		binaryInfo = UpdateBinary(version, channel, messages)
	}()

	PrintMessages(messages, script.Printer)

	return script.PrintBinaryAliasFunction(app.BintrayPackage, binaryInfo.BinaryPath)
}

func Update(version string, channel string, args []string ) error {
	err := CheckMajorMinor(version)
	if err != nil {
		return err
	}

	messages := make(chan ActionMessage, 0)
	printer := output.NewSimplePrint()

	SelfUpdate(messages, printer)

	go UpdateBinary(version, channel, messages)
	PrintMessages(messages, printer)

	return nil
}

// PrintMessages handle ActionMessage events and print messages to the screen
func PrintMessages(messages chan ActionMessage, printer output.Printer) error {
	for {
		select {
		case msg := <-messages:
			if msg.err != nil {
				printer.Error(msg.err)
				return msg.err
			}

			// ignore debug messages if no --debug=yes flag
			if msg.debug {
				if app.DebugMessages == "yes" && msg.msg != "" {
					printer.DebugMessage(msg.msg, msg.comment)
				}
				break
			}

			if msg.msg != "" {
				color := ""
				switch msg.msgType {
				case "ok":
					color = "green"
				case "warn":
					color = "yellow"
				case "fail":
					color = "red"
				}
				printer.Message(msg.msg, color, msg.comment)
			}

			if msg.action == "exit" {
				return nil
			}
		}
	}
}