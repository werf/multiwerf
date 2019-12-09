package multiwerf

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/lock"
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
// TODO make script more responsive: print messages immediately
func Use(version string, channel string, forceRemoteCheck bool, args []string) (err error) {
	messages := make(chan ActionMessage, 0)
	script := output.NewScript()
	printer := script.Printer

	err = SetupVersionAndStorageDir(version, printer)
	if err != nil {
		return nil
	}

	err = lock.Init(MultiwerfStorageDir)
	if err != nil {
		return nil
	}

	err = PerformSelfUpdate(printer)
	if err != nil {
		return nil
	}

	// No lock is needed if update is disabled
	// Do not check for delays if update is disabled
	enableUpdate := false
	if app.Update == "yes" {
		enableUpdate = true
		lockName := fmt.Sprintf("update-ver-%s", version)

		isAcquired, err := lock.TryLock(lockName, lock.TryLockOptions{ReadOnly: false})
		defer func() { _ = lock.Unlock(lockName) }()
		if err != nil {
			PrintActionMessage(ActionMessage{
				msg:     fmt.Sprintf("Cannot acquire a lock for update command"),
				msgType: "fail",
			}, printer)
			return err
		} else {
			if !isAcquired {
				PrintActionMessage(ActionMessage{
					msg:     fmt.Sprintf("Update for version %s is skipped: update is already performed by another process", version),
					msgType: "warn",
				}, printer)
				enableUpdate = false
			}
		}
	}

	binUpdater := NewBinaryUpdater(messages)
	binUpdater.SetRemoteEnabled(enableUpdate)

	// If update is enabled, check and update delay files
	if enableUpdate {
		// Check if delay for channel is passed
		updateDelay := UpdateDelay{
			Filename: filepath.Join(MultiwerfStorageDir, fmt.Sprintf("update-%s-%s-%s.delay", app.BintrayPackage, version, channel)),
		}
		if channel == "alpha" || channel == "beta" {
			updateDelay.WithDelay(app.AlphaBetaUpdateDelay)
		} else {
			updateDelay.WithDelay(app.UpdateDelay)
		}
		remains := updateDelay.TimeRemains()
		if remains != "" {
			// Delay is not passed
			if forceRemoteCheck {
				PrintActionMessage(ActionMessage{
					msg:     fmt.Sprintf("Delayed werf update is forced by flag"),
					msgType: "ok",
				}, printer)
				binUpdater.SetRemoteEnabled(true)
			} else {
				PrintActionMessage(ActionMessage{
					msg:     fmt.Sprintf("werf update is delayed: %s remains", remains),
					msgType: "ok",
				}, printer)
				binUpdater.SetRemoteEnabled(false)
			}
		} else {
			// If delay is passed, update delay for channel and for all less stable channels
			for _, availableChannel := range AvailableChannels {
				updateDelay := UpdateDelay{
					Filename: filepath.Join(MultiwerfStorageDir, fmt.Sprintf("update-%s-%s-%s.delay", app.BintrayPackage, version, availableChannel)),
				}
				if availableChannel == "alpha" || availableChannel == "beta" {
					updateDelay.WithDelay(app.AlphaBetaUpdateDelay)
				} else {
					updateDelay.WithDelay(app.UpdateDelay)
				}
				updateDelay.UpdateTimestamp()
				if availableChannel == channel {
					break
				}
			}
		}
	}

	// Update to latest version if needed. Use local version if remote communication failed.
	// Exit with error if no binaries found.
	var binaryInfo BinaryInfo
	go func() {
		binaryInfo = binUpdater.GetLatestBinaryInfo(version, channel)

		// Stop PrintActionMessages after return from GetLatestBinaryInfo
		messages <- ActionMessage{action: "exit"}
	}()
	err = PrintActionMessages(messages, printer)
	if err != nil {
		return err
	}

	return script.PrintBinaryAliasFunction(app.BintrayPackage, binaryInfo.BinaryPath)
}

// Update checks for the latest available version and download it to StorageDir
//
// Arguments:
//
// - version - a major.minor portion of version to update
// - channel - a string with channel name
// - args - excess arguments from command line (not used yet by "update" command)
//
// This command is fully locked on major.minor basis:
// - if "update-1.0" lock is present, then command is exited with message
// - if "lock is acquired, then self-update and update is performed as usual.
func Update(version string, channel string, args []string) (err error) {
	messages := make(chan ActionMessage, 0)
	printer := output.NewSimplePrint()

	err = SetupVersionAndStorageDir(version, printer)
	if err != nil {
		return nil
	}

	err = lock.Init(MultiwerfStorageDir)
	if err != nil {
		return nil
	}

	err = PerformSelfUpdate(printer)
	if err != nil {
		return nil
	}

	lockName := fmt.Sprintf("update-ver-%s", version)

	isAcquired, err := lock.TryLock(lockName, lock.TryLockOptions{ReadOnly: false})
	defer func() { _ = lock.Unlock(lockName) }()
	if err != nil {
		PrintActionMessage(ActionMessage{
			msg:     fmt.Sprintf("Cannot acquire a lock for update command"),
			msgType: "fail",
		}, printer)
		return err
	} else {
		if !isAcquired {
			PrintActionMessage(ActionMessage{
				msg:     fmt.Sprintf("Update for version %s is already performed by another process. Exiting.", version),
				msgType: "warn",
			}, printer)
			return nil
		}
	}

	// Update binary to latest version. Exit with error if remote communication failed.
	binUpdater := NewBinaryUpdater(messages)
	binUpdater.SetRemoteEnabled(true)

	go func() {
		binUpdater.DownloadLatest(version, channel)
		// Update timestamp of delay for use command. Also update timestamp for less stable channels.
		for _, availableChannel := range AvailableChannels {
			updateDelay := UpdateDelay{
				Filename: filepath.Join(MultiwerfStorageDir, fmt.Sprintf("update-%s-%s-%s.delay", app.BintrayPackage, version, availableChannel)),
			}
			if availableChannel == "alpha" || availableChannel == "beta" {
				updateDelay.WithDelay(app.AlphaBetaUpdateDelay)
			} else {
				updateDelay.WithDelay(app.UpdateDelay)
			}
			updateDelay.UpdateTimestamp()
			if channel == availableChannel {
				break
			}
		}
		// Stop PrintActionMessages after return from DownloadLatest
		messages <- ActionMessage{action: "exit"}
	}()
	err = PrintActionMessages(messages, printer)
	if err != nil {
		return err
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
		err = PrintActionMessages(messages, printer)
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
	err = PrintActionMessages(messages, printer)
	if err != nil {
		return err
	}

	return nil
}

func SetupVersionAndStorageDir(version string, printer output.Printer) (err error) {
	messages := make(chan ActionMessage, 0)
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

		messages <- ActionMessage{msg: fmt.Sprintf("Debug output is enabled. %s is a good major.minor. Storage dir is %s", version, MultiwerfStorageDir), debug: true}
		messages <- ActionMessage{action: "exit"}
	}()
	err = PrintActionMessages(messages, printer)
	if err != nil {
		return err
	}
	return nil
}

// update multiwerf binary (self update)
func PerformSelfUpdate(printer output.Printer) (err error) {
	messages := make(chan ActionMessage, 0)
	selfPath := ""
	lockName := "self-update"

	go func() {
		if app.SelfUpdate == "no" {
			// Self-update is disabled. Silently skip it
			messages <- ActionMessage{
				msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
				msgType: "ok",
			}
			messages <- ActionMessage{msg: "Self-update is disabled", debug: true}
			messages <- ActionMessage{action: "exit"}
			return
		}

		// Acquire a lock
		isAcquired, err := lock.TryLock(lockName, lock.TryLockOptions{ReadOnly: false})
		defer func() { _ = lock.Unlock(lockName) }()
		if err != nil {
			messages <- ActionMessage{
				msg:     fmt.Sprintf("%s %s. Skip self-update: cannot acquire a lock: %v", app.AppName, app.Version, err),
				msgType: "warn",
			}
			messages <- ActionMessage{action: "exit"}
			return
		} else {
			if !isAcquired {
				messages <- ActionMessage{
					msg:     fmt.Sprintf("%s %s. Skip self-update: operation is performed by another process.", app.AppName, app.Version),
					msgType: "ok",
				}
				messages <- ActionMessage{action: "exit"}
				return
			}
		}

		// Check for delay of self update
		selfUpdateDelay := UpdateDelay{
			Filename: filepath.Join(MultiwerfStorageDir, "update-multiwerf.delay"),
		}
		selfUpdateDelay.WithDelay(app.SelfUpdateDelay)
		// self update is enabled here, so check for delay and disable self update if needed
		remains := selfUpdateDelay.TimeRemains()
		if remains != "" {
			messages <- ActionMessage{
				msg:     fmt.Sprintf("%s %s. Self-update is delayed: %s remains till next self-update", app.AppName, app.Version, remains),
				msgType: "ok",
			}
			messages <- ActionMessage{action: "exit"}
			return
		} else {
			// FIXME: self update can be erroneous: new version exists, but with bad hash. Should we set a lower delay with progressive increase in this case?
			selfUpdateDelay.UpdateTimestamp()
		}

		// Do self-update: check latest version, download, replace a binary
		messages <- ActionMessage{msg: fmt.Sprintf("%s %s. Start self update...", app.AppName, app.Version), msgType: "ok"}
		selfPath = SelfUpdate(messages)

		// Stop PrintActionMessages after return from SelfUpdate
		messages <- ActionMessage{action: "exit"}
	}()
	err = PrintActionMessages(messages, printer)
	if err != nil {
		return err
	}

	// restart myself if new binary was downloaded
	if selfPath != "" {
		err := ExecUpdatedBinary(selfPath)
		if err != nil {
			PrintActionMessage(ActionMessage{
				comment: "self update error",
				msg:     fmt.Sprintf("%s: exec of updated binary failed: %v", multiwerfProlog, err),
				msgType: "fail",
				stage:   "self-update"}, printer)
		}
	}

	return nil
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
}
