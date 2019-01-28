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

	messages = make(chan ActionMessage, 0)

	var binaryInfo BinaryInfo
	var selfPath string
	go func() {
		if app.SelfUpdate == "yes" {
			selfPath = SelfUpdate(messages)
			if selfPath != "" {
				err := ExecUpdatedBinary(selfPath)
				if err != nil {
					messages <- ActionMessage{
						msg: fmt.Sprintf("multiwerf %s self-update: exec from updated binary failed: %v", app.Version, err),
						state: "self-update-error"}
				} else {
					// Cannot be reached because of exec syscall.
					return
				}
			}
		}
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

			if msg.state == "self-update-error" && msg.msg != "" {
				if msg.msg != "" {
					fmt.Printf("# self-update-error\necho -e \"\\e[31m\" %s \"\\e[0m\"\n", msg.msg)
				}
				break
			}
			if msg.state == "self-update-warning" && msg.msg != "" {
				if msg.msg != "" {
					fmt.Printf("# self-update-warning\necho -e \"\\e[33m\" %s \"\\e[0m\"\n", msg.msg)
				}
				break
			}
			if msg.state == "self-update-success" {
				if msg.msg != "" {
					fmt.Printf("# self-update-success\necho -e \"\\e[32m\" %s \"\\e[0m\"\n#\n#\n", msg.msg)
				}
				break
			}

			// print "return 1" to fail source command
			if msg.err != nil {
				PrintErrorScript()
				return msg.err
			}

			// break for-loop on successful update of binary
			if msg.state == "success" {
				fmt.Printf("# update %s success\necho -e \"\\e[32m\" %s\"\\e[0m\"\n\n", app.BintrayPackage, msg.msg)
				break READ_MESSAGES
			}

			if msg.msg != "" {
				// print messages as comments for source
				fmt.Printf("# %s\n", msg.msg)
				break
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


// TODO Add script block to prevent from loading not in bash/zsh shells (as in rvm script)
func PrintUseScript(info BinaryInfo) error {
	fmt.Printf(`#
# Function with path to choosen version of %s binary.
# To remove function use unset:
# unset -f %[1]s
%[1]s()
{
%s "$@"
}

`, app.BintrayPackage, info.BinaryPath)
	return nil
}

func PrintErrorScript() error {
	fmt.Println("return 1")
	return nil
}
