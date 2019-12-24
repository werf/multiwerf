package multiwerf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"

	"github.com/flant/shluz"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/output"
)

const UpdateLockName = "update"
const SelfUpdateLockName = "self-update"

var AvailableChannels = []string{
	"alpha",
	"beta",
	"ea",
	"stable",
	"rock-solid",
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

// Use prints a shell script with alias to the actual binary version for the group/channel
// TODO make script more responsive: print messages immediately
func Use(group, channel string, forceRemoteCheck, asFile bool, shell string) (err error) {
	printer := output.NewSilentPrint()
	if err := ValidateGroup(group, printer); err != nil {
		return err
	}

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	useWerfPathLogPath := filepath.Join(MultiwerfStorageDir, "multiwerf_use_first_werf_path.log")
	backgroundUpdateLogPath := filepath.Join(MultiwerfStorageDir, "background_update.log")

	var backgroundUpdateArgs []string
	if !forceRemoteCheck {
		backgroundUpdateArgs = append(backgroundUpdateArgs, "--with-cache")
	}

	backgroundUpdateArgs = append(backgroundUpdateArgs, group, channel)

	var filename = "werf_source"
	var filenameExt string
	var fileContent string

	switch shell {
	case "cmdexe":
		filenameExt = "bat"
		fileContent = fmt.Sprintf(`
FOR /F "tokens=*" %%%%g IN ('multiwerf werf-path %[1]s %[2]s') do (SET WERF_PATH=%%%%g)

IF %%ERRORLEVEL%% NEQ 0 (
    multiwerf update %[1]s %[2]s 
    FOR /F "tokens=*" %%%%g IN ('multiwerf werf-path %[1]s %[2]s') do (SET WERF_PATH=%%%%g)
) ELSE (
    START /B multiwerf update %[3]s >%[5]s 2>&1
)

DOSKEY werf=%%WERF_PATH%% $*
`, group, channel, strings.Join(backgroundUpdateArgs, " "), useWerfPathLogPath, backgroundUpdateLogPath)
	case "powershell":
		filenameExt = "ps1"
		fileContent = fmt.Sprintf(`
if (Invoke-Expression -Command "multiwerf werf-path %[1]s %[2]s >%[4]s 2>&1" | Out-String -OutVariable WERF_PATH) {
    Start-Job { multiwerf update %[3]s >%[5]s 2>&1 }
} else {
    multiwerf update %[1]s %[2]s
    Invoke-Expression -Command "multiwerf werf-path %[1]s %[2]s" | Out-String -OutVariable WERF_PATH
}

function werf { & $WERF_PATH.Trim() $args }
`, group, channel, strings.Join(backgroundUpdateArgs, " "), useWerfPathLogPath, backgroundUpdateLogPath)
	default:
		if runtime.GOOS == "windows" {
			fileContent = fmt.Sprintf(`
if multiwerf werf-path %[1]s %[2]s >%[4]s 2>&1; then
    (multiwerf update %[3]s >%[5]s 2>&1 </dev/null &)
else
    multiwerf update %[1]s %[2]s
fi

WERF_PATH=$(multiwerf werf-path %[1]s %[2]s | sed 's/\\/\//g')
WERF_FUNC=$(cat <<EOF
werf() 
{
    $WERF_PATH "\$@"
}
EOF
)

eval "$WERF_FUNC"
`, group, channel, strings.Join(backgroundUpdateArgs, " "), useWerfPathLogPath, backgroundUpdateLogPath)
		} else {
			fileContent = fmt.Sprintf(`
if multiwerf werf-path %[1]s %[2]s >%[4]s 2>&1; then
    (setsid multiwerf update %[3]s >%[5]s 2>&1 </dev/null &)
else
    multiwerf update %[1]s %[2]s
fi

WERF_PATH=$(multiwerf werf-path %[1]s %[2]s)
WERF_FUNC=$(cat <<EOF
werf() 
{
    $WERF_PATH "\$@"
}
EOF
)

eval "$WERF_FUNC"
`, group, channel, strings.Join(backgroundUpdateArgs, " "), useWerfPathLogPath, backgroundUpdateLogPath)
		}
	}

	fileContent = fmt.Sprintln(strings.TrimSpace(fileContent))

	if !asFile {
		fmt.Printf(fileContent)
	} else {
		if forceRemoteCheck {
			filename = strings.Join([]string{filename, "force_remote_check"}, "_with_")
		}

		if filenameExt != "" {
			filename = strings.Join([]string{filename, filenameExt}, ".")
		}

		fileContentBytes := []byte(fileContent)
		dstPath := filepath.Join(MultiwerfStorageDir, "scripts", strings.Join([]string{group, channel}, "-"), filename)
		tmpDstPath := dstPath + ".tmp"

		if exist, err := FileExists(filepath.Dir(dstPath), filename); err != nil {
			printer.Error(err)
			return err
		} else if exist {
			currentFileContentBytes, err := ioutil.ReadFile(dstPath)
			if err != nil {
				printer.Error(err)
				return err
			}

			if bytes.Equal(currentFileContentBytes, fileContentBytes) {
				fmt.Println(dstPath)
				return nil
			}
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			printer.Error(err)
			return err
		}

		if err := ioutil.WriteFile(tmpDstPath, fileContentBytes, os.ModePerm); err != nil {
			printer.Error(err)
			return err
		}

		err = os.Rename(tmpDstPath, dstPath)
		if err != nil {
			err := fmt.Errorf("rename tmp file failed: %v", err)
			printer.Error(err)
			return err
		}

		fmt.Println(dstPath)
	}

	return nil
}

// Print path to the actual version available for the group/channel locally
func WerfPath(group string, channel string) (err error) {
	messages := make(chan ActionMessage, 0)
	printer := output.NewSilentPrint()

	if err := ValidateGroup(group, printer); err != nil {
		return err
	}

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	var binaryInfo *BinaryInfo
	go func() {
		binUpdater := NewBinaryUpdater(messages, false)
		binaryInfo = binUpdater.UpdateChannelVersion(group, channel)

		messages <- ActionMessage{action: "exit"}
	}()
	if err = PrintActionMessages(messages, printer); err != nil {
		return err
	}

	fmt.Println(binaryInfo.BinaryPath)

	return nil
}

// Exec the latest binary version available for the channel locally with passed args
func WerfExec(group, channel string, args []string) (err error) {
	messages := make(chan ActionMessage, 0)
	printer := output.NewSilentPrint()

	if err := ValidateGroup(group, printer); err != nil {
		return err
	}

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	var binaryInfo *BinaryInfo
	go func() {
		binUpdater := NewBinaryUpdater(messages, false)
		binaryInfo = binUpdater.UpdateChannelVersion(group, channel)

		messages <- ActionMessage{action: "exit"}
	}()
	if err = PrintActionMessages(messages, printer); err != nil {
		return err
	}

	cmd := exec.Command(binaryInfo.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// Update checks for the actual version for group/channel and downloads it to StorageDir
//
// Arguments:
//
// - group - a major.minor version to update
// - channel - a string with channel name
//
// This command is fully locked:
// - if the lock is present then command exits with special message
// - if the lock is acquired then self-update and update perform as usual
func Update(group, channel string, withCache bool) (err error) {
	messages := make(chan ActionMessage, 0)
	printer := output.NewSimplePrint()

	if err := ValidateGroup(group, printer); err != nil {
		return err
	}

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	if err := shluz.Init(filepath.Join(MultiwerfStorageDir, "locks")); err != nil {
		printer.Error(err)
		return err
	}

	if err := PerformSelfUpdate(printer); err != nil {
		return err
	}

	isAcquired, err := shluz.TryLock(UpdateLockName, shluz.TryLockOptions{ReadOnly: false})
	defer func() { _ = shluz.Unlock(UpdateLockName) }()
	if err != nil {
		PrintActionMessage(ActionMessage{
			msg:     fmt.Sprintf("Cannot acquire the lock for the update command"),
			msgType: "fail",
		}, printer)
		return err
	} else {
		if !isAcquired {
			PrintActionMessage(
				ActionMessage{
					msg:     "The update was skipped because it was performing by another process",
					msgType: "warn",
				},
				printer,
			)

			return nil
		}
	}

	remoteEnabled := true
	if withCache {
		// Check if delay for channel is passed
		updateDelay := UpdateDelay{
			Filename: filepath.Join(MultiwerfStorageDir, fmt.Sprintf("update-%s-%s-%s.delay", app.BintrayPackage, group, channel)),
		}

		if channel == "alpha" || channel == "beta" {
			updateDelay.WithDelay(app.AlphaBetaUpdateDelay)
		} else {
			updateDelay.WithDelay(app.UpdateDelay)
		}

		remains := updateDelay.TimeRemains()
		if remains != "" {
			PrintActionMessage(
				ActionMessage{
					msg:     fmt.Sprintf("werf update has been delayed: %s left till next attempt", remains),
					msgType: "ok",
				},
				printer,
			)

			remoteEnabled = false
		} else {
			// If delay is passed, update delay for channel and for all less stable channels
			for _, availableChannel := range AvailableChannels {
				updateDelay := UpdateDelay{
					Filename: filepath.Join(MultiwerfStorageDir, fmt.Sprintf("update-%s-%s-%s.delay", app.BintrayPackage, group, availableChannel)),
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

	go func() {
		binUpdater := NewBinaryUpdater(messages, remoteEnabled)
		binUpdater.UpdateChannelVersion(group, channel)

		messages <- ActionMessage{action: "exit"}
	}()

	return PrintActionMessages(messages, printer)
}

func ValidateGroup(group string, printer output.Printer) error {
	messages := make(chan ActionMessage, 0)

	go func() {
		err := CheckMajorMinor(group)
		if err != nil {
			messages <- ActionMessage{err: err}
		}

		messages <- ActionMessage{
			msg:   fmt.Sprintf("The group %s is the valid major.minor version", group),
			debug: true,
		}

		messages <- ActionMessage{action: "exit"}
	}()

	return PrintActionMessages(messages, printer)
}

func SetupStorageDir(printer output.Printer) error {
	messages := make(chan ActionMessage, 0)

	go func() {
		var err error
		MultiwerfStorageDir, err = ExpandPath(app.StorageDir)
		if err != nil {
			messages <- ActionMessage{
				err: fmt.Errorf("invalid storage dir %s: %s", MultiwerfStorageDir, err),
			}
		}

		messages <- ActionMessage{
			msg:   fmt.Sprintf("storage dir is %s", MultiwerfStorageDir),
			debug: true,
		}

		messages <- ActionMessage{action: "exit"}
	}()

	return PrintActionMessages(messages, printer)
}

// update multiwerf binary (self-update)
func PerformSelfUpdate(printer output.Printer) (err error) {
	messages := make(chan ActionMessage, 0)
	selfPath := ""

	go func() {
		if app.SelfUpdate == "no" {
			// self-update is disabled. Silently skip it
			messages <- ActionMessage{
				msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
				msgType: "ok",
			}

			messages <- ActionMessage{
				msg:   "self-update is disabled",
				debug: true,
			}

			messages <- ActionMessage{action: "exit"}

			return
		}

		// Acquire a shluz
		isAcquired, err := shluz.TryLock(SelfUpdateLockName, shluz.TryLockOptions{ReadOnly: false})
		defer func() { _ = shluz.Unlock(SelfUpdateLockName) }()
		if err != nil {
			messages <- ActionMessage{
				msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
				msgType: "warn",
			}

			messages <- ActionMessage{
				msg:     fmt.Sprintf("Skip self-update: cannot acquire a lock: %v", err),
				msgType: "warn",
			}

			messages <- ActionMessage{action: "exit"}

			return
		} else {
			if !isAcquired {
				messages <- ActionMessage{
					msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
					msgType: "ok",
				}

				messages <- ActionMessage{
					msg:     "Self-update has been skipped because the operation is performing by another process",
					msgType: "ok",
				}

				messages <- ActionMessage{action: "exit"}

				return
			}
		}

		if !app.Experimental {
			// Check for delay of self update
			selfUpdateDelay := UpdateDelay{
				Filename: filepath.Join(MultiwerfStorageDir, "update-multiwerf.delay"),
			}
			selfUpdateDelay.WithDelay(app.SelfUpdateDelay)

			// self update is enabled here, so check for delay and disable self update if needed
			remains := selfUpdateDelay.TimeRemains()
			if remains != "" {
				messages <- ActionMessage{
					msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
					msgType: "ok",
				}

				messages <- ActionMessage{
					msg:     fmt.Sprintf("Self-update has been delayed: %s left till next attempt", remains),
					msgType: "ok",
				}

				messages <- ActionMessage{action: "exit"}

				return
			} else {
				// FIXME: self update can be erroneous: new version exists, but with bad hash. Should we set a lower delay with progressive increase in this case?
				selfUpdateDelay.UpdateTimestamp()
			}
		}

		// Do self-update: check the latest version, download, replace a binary
		messages <- ActionMessage{
			msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
			msgType: "ok",
		}

		messages <- ActionMessage{
			msg:     fmt.Sprintf("Start multiwerf self-update ..."),
			msgType: "ok",
		}

		selfPath = SelfUpdate(messages)

		// Stop PrintActionMessages after return from SelfUpdate
		messages <- ActionMessage{action: "exit"}
	}()
	if err := PrintActionMessages(messages, printer); err != nil {
		return err
	}

	// restart myself if new binary was downloaded
	if selfPath != "" {
		err := ExecUpdatedBinary(selfPath)
		if err != nil {
			PrintActionMessage(
				ActionMessage{
					comment: "self-update error",
					msg:     fmt.Sprintf("%s: exec of updated binary failed: %v", multiwerfProlog, err),
					msgType: "fail",
					stage:   "self-update",
				},
				printer,
			)
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
		var colorAttribute *color.Attribute
		switch msg.msgType {
		case "ok":
			colorAttribute = &output.GreenColor
		case "warn":
			colorAttribute = &output.YellowColor
		case "fail":
			colorAttribute = &output.RedColor
		}

		printer.Message(msg.msg, colorAttribute, msg.comment)
	}
}
