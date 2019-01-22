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

	// run Update



	fmt.Printf("use command invoked\nversion=%s channel=%s\n args=%+v\n", version, channel, args)
	return nil
}

func Update(version string, channel string, args []string ) error {
	err := CheckMajorMinor(version)
	if err != nil {
		return err
	}

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

	return err


	//fmt.Printf("update command invoked\nversion=%s channel=%s\n args=%+v\n", version, channel, args)
	//return nil
}
