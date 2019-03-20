package multiwerf

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

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
var AvailableChannelsStableFirst = []string{
	"stable",
	"ea",
	"rc",
	"beta",
	"alpha",
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
func Use(version string, channel string, forceRemoteCheck bool, args []string) (err error) {
	messages := make(chan ActionMessage, 0)
	script := output.NewScript()

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

	// Check for delay of self update
	selfUpdateDelay := UpdateDelay{
		Filename: filepath.Join(MultiwerfStorageDir, "update-multiwerf.delay"),
	}
	selfUpdateDelay.SetDelay(app.SelfUpdateDelay)
	// if self update is enabled, check for delay and disable self update if needed
	if app.SelfUpdate == "yes" {
		if selfUpdateDelay.IsDelayPassed() {
			selfUpdateDelay.UpdateTimestamp()
		} else {
			app.SelfUpdate = "no"
		}
	}

	// update multiwerf binary (self update)
	go func() {
		SelfUpdate(messages)
	}()
	err = PrintMessages(messages, script.Printer)
	if err != nil {
		return err
	}

	binUpdater := NewBinaryUpdater(messages)

	// Check for delay of werf update
	updateDelay := UpdateDelay{
		Filename: filepath.Join(MultiwerfStorageDir, fmt.Sprintf("update-%s-%s-%s.delay", app.BintrayPackage, version, channel)),
	}
	if channel == "alpha" || channel == "beta" {
		updateDelay.SetDelay(app.AlphaBetaUpdateDelay)
	} else {
		updateDelay.SetDelay(app.UpdateDelay)
	}

	if app.Update == "yes" {
		binUpdater.SetRemoteEnabled(true)
		if updateDelay.IsDelayPassed() {
			updateDelay.UpdateTimestamp()
			binUpdater.SetRemoteDelayed(false)
		} else {
			binUpdater.SetRemoteDelayed(!forceRemoteCheck)
		}
	}

	// Update to latest version if needed. Use local version if remote communication failed.
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

	// Check for delay of self update
	selfUpdateDelay := UpdateDelay{
		Filename: filepath.Join(MultiwerfStorageDir, "update-multiwerf.delay"),
	}
	selfUpdateDelay.SetDelay(app.SelfUpdateDelay)
	// if self update is enabled, check for delay and disable self update if needed
	if app.SelfUpdate == "yes" {
		if selfUpdateDelay.IsDelayPassed() {
			selfUpdateDelay.UpdateTimestamp()
		} else {
			app.SelfUpdate = "no"
		}
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
	binUpdater := NewBinaryUpdater(messages)
	binUpdater.SetRemoteEnabled(true)
	binUpdater.SetRemoteDelayed(false)
	go binUpdater.DownloadLatest(version, channel)
	err = PrintMessages(messages, printer)
	if err != nil {
		return err
	}

	// Update timestamp of delay of werf update. Also update timestamp for less stable channels.
	for _, availableChannel := range AvailableChannels {
		updateDelay := UpdateDelay{
			Filename: filepath.Join(MultiwerfStorageDir, fmt.Sprintf("update-%s-%s-%s.delay", app.BintrayPackage, version, channel)),
		}
		updateDelay.UpdateTimestamp()
		if channel == availableChannel {
			break
		}
	}

	return nil
}

func AvailableReleases(version string, channel string, outputFormat string) (err error) {
	messages := make(chan ActionMessage, 0)
	printer := output.NewPlainPrint()

	if version != "" {
		// Check version argument
		go func() {
			err := CheckMajorMinor(version)
			if err != nil {
				messages <- ActionMessage{err: err}
			}
			messages <- ActionMessage{action: "exit"}
		}()
		err = PrintMessages(messages, printer)
		if err != nil {
			return err
		}
	}

	informer := NewAvailableReleasesInformer(messages)
	go func() {
		if version == "" && channel == "" {
			messages <- ActionMessage{msg: "Start GetRelease", debug: true}
			_, err := informer.GetMajorMinorReleases()
			if err != nil {
				messages <- ActionMessage{err: err, action: "exit"}
				return
			}
			messages <- ActionMessage{msg: "major.minor releases", msgType: "ok"}
		} else {
			if channel == "" {
				messages <- ActionMessage{msg: "Start GetAllChannelsReleases", debug: true}
				releases, err := informer.GetAllChannelsReleases(version)
				if err != nil {
					messages <- ActionMessage{err: err, action: "exit"}
					return
				}
				msg := ""
				if outputFormat == "text" {
					outMessages := []string{}
					for _, release := range releases.OrderedReleases {
						outMessages = append(outMessages, fmt.Sprintf("%s %v", release, releases.Releases[release]))
					}
					msg = strings.Join(outMessages, "\n")
				}
				if outputFormat == "json" {
					b, err := json.Marshal(releases)
					if err != nil {
						messages <- ActionMessage{err: err, action: "exit"}
						return
					}
					msg = string(b)
				}
				messages <- ActionMessage{msg: msg, msgType: "ok"}
			} else {
				messages <- ActionMessage{msg: "Start GetRelease", debug: true}
				release, err := informer.GetRelease(version, channel)
				if err != nil {
					messages <- ActionMessage{err: err, action: "exit"}
					return
				}
				messages <- ActionMessage{msg: release, msgType: "ok"}
			}
		}

		// Stop printing
		messages <- ActionMessage{action: "exit"}
	}()
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
				// TODO add special error to exit with 1 and not print error message with kingpin
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
