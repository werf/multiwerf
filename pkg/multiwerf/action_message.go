package multiwerf

import (
	"github.com/fatih/color"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/output"
)

type MsgType string

var (
	OkMsgType   MsgType = "ok"
	WarnMsgType MsgType = "warn"
	FailMsgType MsgType = "fail"
)

// ActionMessage is used to send messages from go routines
type ActionMessage struct {
	stage   string  // stage of a program
	action  string  // action to perform (exit with error, exit 0, ...)
	msg     string  // text to print to the screen
	msgType MsgType // message type: ok, warn, fail
	err     error   // error message — display it in case of critical error before graceful exit
	comment string  // minor message that displayed as a comment in a script output (can be grayed)
	debug   bool    // debug msg and comment are displayed only if --debug=yes flag is set
}

// PrintActionMessages handle ActionMessage events and print messages with printer object
func PrintActionMessages(messages chan ActionMessage, printer output.Printer) error {
	for {
		select {
		case msg := <-messages:
			PrintActionMessage(msg, printer)
			if msg.err != nil {
				// TODO add special error to exit with 1 and not print error message with kingpin
				return msg.err
			}

			if msg.action == "exit" {
				return nil
			}
		}
	}
}

func PrintActionMessage(msg ActionMessage, printer output.Printer) {
	if msg.err != nil {
		printer.Error(msg.err)
		return
	}

	// ignore debug messages if no --debug=yes flag
	if msg.debug {
		if app.DebugMessages == "yes" && msg.msg != "" {
			printer.DebugMessage(msg.msg, msg.comment)
		}
		return
	}

	if msg.msg != "" {
		var colorAttribute *color.Attribute
		switch msg.msgType {
		case OkMsgType:
			colorAttribute = &output.GreenColor
		case WarnMsgType:
			colorAttribute = &output.YellowColor
		case FailMsgType:
			colorAttribute = &output.RedColor
		}

		printer.Message(msg.msg, colorAttribute, msg.comment)
	}
}
