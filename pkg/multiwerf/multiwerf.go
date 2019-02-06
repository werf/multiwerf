package multiwerf

import (
	"fmt"
	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/output"
)

var AvailableChannels = []string{
	"alpha",
	"beta",
	"rc",
	"ea",
	"stable",
}

// MultiwerfStorageDir is an effective path to a storage
var MultiwerfStorageDir string

// ActionMessage is used to send messages from go routines started in use and update commands
type ActionMessage struct {
	stage   string // stage of a program
	action  string // action to perform (exit with error, exit 0, ...)
	msg     string // Text to print to the screen
	msgType string // message type: ok, warn, fail
	err     error  // Error message — display it in case of critical error before graceful exit
	comment string // minor message that displayed as a comment in a script output (can be grayed)
	debug   bool   // debug msg and comment are displayed only if --debug=yes flag is set
}

// Use prints a shell script with alias to the latest binary version available for the channel
func Use(version string, channel string, args []string) (err error) {
	messages := make(chan ActionMessage, 0)
	script := output.NewScript()
	binUpdater := NewBinaryUpdater(messages)

	// Check version argument and storage path
	go func() {
		err := CheckMajorMinor(version)
		if err != nil {
			messages <- ActionMessage{err: err}
		}

		MultiwerfStorageDir, err = ExpandPath(app.StorageDir)
		if err != nil {
			messages <- ActionMessage{err: err}
		}

		messages <- ActionMessage{action: "exit"}
	}()
	err = PrintMessages(messages, script.Printer)
	if err != nil {
		return err
	}

	// update multiwerf binary (self update)
	go func() {
		SelfUpdate(messages)
	}()
	err = PrintMessages(messages, script.Printer)
	if err != nil {
		return err
	}

	// Update to latest version if neede. Use local version if remote communication failed.
	// Exit with error if no binaries found.
	var binaryInfo BinaryInfo
	go func() {
		binaryInfo = binUpdater.GetLatestBinaryInfo(version, channel)
	}()
	err = PrintMessages(messages, script.Printer)
	if err != nil {
		return err
	}

	return script.PrintBinaryAliasFunction(app.BintrayPackage, binaryInfo.BinaryPath)
}

// Update checks for the latest available version and download it to StorageDir
func Update(version string, channel string, args []string) (err error) {
	messages := make(chan ActionMessage, 0)
	printer := output.NewSimplePrint()
	binUpdater := NewBinaryUpdater(messages)

	// Check version argument and storage path
	go func() {
		err := CheckMajorMinor(version)
		if err != nil {
			messages <- ActionMessage{err: err}
		}

		MultiwerfStorageDir, err = ExpandPath(app.StorageDir)
		if err != nil {
			messages <- ActionMessage{err: err}
		}

		messages <- ActionMessage{msg: fmt.Sprintf("Major minor Checks done, storage dir is %s", MultiwerfStorageDir), debug: true}
		messages <- ActionMessage{action: "exit"}
	}()
	err = PrintMessages(messages, printer)
	if err != nil {
		return err
	}

	// update multiwerf binary (self update)
	go func() {
		messages <- ActionMessage{msg: "Start SelfUpdate", debug: true}
		SelfUpdate(messages)
	}()
	err = PrintMessages(messages, printer)
	if err != nil {
		return err
	}
	// Update binary to latest version. Exit with error if remote communication failed.
	go binUpdater.DownloadLatest(version, channel)
	err = PrintMessages(messages, printer)
	if err != nil {
		return err
	}
	return nil
}

// PrintMessages handle ActionMessage events and print messages to the screen
func PrintMessages(messages chan ActionMessage, printer output.Printer) error {
	for {
		select {
		case msg := <-messages:
			if msg.err != nil {
				printer.Error(msg.err)
				// TODO prevent this error print with kingpin default handler
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
