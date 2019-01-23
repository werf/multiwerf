package multiwerf

import (
	"fmt"

	"github.com/flant/multiwerf/pkg/app"
)

var AvailableChannels = []string{
	"alpha",
	"beta",
	"rc",
	"stable",
}

// use and update actions send messages
type ActionMessage struct {
	msg string
	err error
	state string
	// more options
	debug bool
}
var messages chan ActionMessage


func Use(version string, channel string, args []string ) error {
	err := CheckMajorMinor(version)
	if err != nil {
		return err
	}

	// TODO add self-update

	messages = make(chan ActionMessage, 0)
	var binaryInfo BinaryInfo
	go func() {
		binaryInfo = UpdateBinary(version, channel, messages)
	}()

READ_MESSAGES:
	for {
		select {
		case msg := <-messages:
			// ignore debug messages if no --debug=yes flag
			if msg.debug && app.DebugMessages != "yes" {
				break
			}

			// print "return 1" to fail source command
			if msg.err != nil {
				PrintErrorScript()
				return msg.err
			}

			// print messages as comments for source
			if msg.msg != "" {
				fmt.Printf("# %s\n", msg.msg)
			}

			// print use script on successful update
			if msg.state == "success" {
				break READ_MESSAGES
			}

			// No script actions on exit
			if msg.state == "exit" {
				return nil
			}
		}
	}

	return PrintUseScript(binaryInfo)
}

func Update(version string, channel string, args []string ) error {
	err := CheckMajorMinor(version)
	if err != nil {
		return err
	}

	// TODO add self-update

	messages = make(chan ActionMessage, 0)
	go UpdateBinary(version, channel, messages)

	for {
		select {
		case msg := <-messages:
			if msg.debug && app.DebugMessages != "yes" {
				break
			}
			if msg.err != nil {
				return msg.err
			}

			if msg.msg != "" {
				fmt.Printf("%s\n", msg.msg)
			}

			if msg.state == "success" {
				return nil
			}
			if msg.state == "exit" {
				return nil
			}
		}
	}

	return nil
}


func PrintUseScript(info BinaryInfo) error {
	fmt.Printf(`# set werf path
# TODO Add block for prevent from loading in sh shells (as in rvm script)
%s()
{
%s "$@"
}

# to remove function use unset:
# unset -f %[1]s
`, app.BintrayPackage, info.BinaryPath)
	return nil
}

func PrintErrorScript() error {
	fmt.Println("return 1")
	return nil
}
