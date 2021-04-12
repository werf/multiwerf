package multiwerf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/werf/lockgate"

	"github.com/werf/multiwerf/pkg/app"
	"github.com/werf/multiwerf/pkg/locker"
	"github.com/werf/multiwerf/pkg/output"
	"github.com/werf/multiwerf/pkg/repo"
	"github.com/werf/multiwerf/pkg/util"
)

const SelfUpdateLockName = "self-update"

// update multiwerf binary (self-update)
func PerformSelfUpdate(printer output.Printer, skipSelfUpdate bool) (err error) {
	messages := make(chan ActionMessage, 0)
	selfPath := ""

	go func() {
		if skipSelfUpdate {
			messages <- ActionMessage{
				msg:   "self-update is disabled",
				debug: true,
			}

			messages <- ActionMessage{action: "exit"}

			return
		}

		// Acquire a lock
		isAcquired, lockHandle, err := locker.Locker.Acquire(SelfUpdateLockName, lockgate.AcquireOptions{NonBlocking: true})
		if err != nil {
			messages <- ActionMessage{
				msg:     fmt.Sprintf("Self-update: Cannot acquire a lock %s: %v", SelfUpdateLockName, err),
				msgType: WarnMsgType,
			}

			messages <- ActionMessage{action: "exit"}

			return
		} else if !isAcquired {
			messages <- ActionMessage{
				msg:     "Self-update: Skipped due to performing the operation by another process",
				msgType: WarnMsgType,
			}

			messages <- ActionMessage{action: "exit"}

			return
		}

		defer func() { _ = locker.Locker.Release(lockHandle) }()

		if !app.Experimental {
			// Check for delay of self update
			selfUpdateDelay := DelayFile{
				Filename: filepath.Join(StorageDir, "self-update.delay"),
			}
			selfUpdateDelay.WithDelay(app.SelfUpdateDelay)

			// self update is enabled here, so check for delay and disable self update if needed
			remains := selfUpdateDelay.TimeRemains()
			if remains != "" {
				messages <- ActionMessage{
					msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
					msgType: OkMsgType,
				}

				messages <- ActionMessage{
					msg:     fmt.Sprintf("Self-update has been delayed: %s left till next attempt", remains),
					msgType: OkMsgType,
				}

				messages <- ActionMessage{action: "exit"}

				return
			} else {
				// FIXME: self-update can be erroneous: new version exists, but with bad hash. Should we set a lower delay with progressive increase in this case?
				if err := selfUpdateDelay.UpdateTimestamp(); err != nil {
					messages <- ActionMessage{err: err}
					return
				}
			}
		}

		// Do self-update: check the latest version, download, replace a binary
		messages <- ActionMessage{
			msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
			msgType: OkMsgType,
		}

		messages <- ActionMessage{
			msg:     fmt.Sprintf("Starting multiwerf self-update ..."),
			msgType: OkMsgType,
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
					msg:     fmt.Sprintf("Self-update: Exec of updated binary failed: %v", err),
					msgType: FailMsgType,
					stage:   "self-update",
				},
				printer,
			)
		}
	}

	return nil
}

// SelfUpdate checks for new version of multiwerf, download it and execute as a new process.
// Note: multiwerf has no option to exit on self-update errors.
func SelfUpdate(messages chan ActionMessage) string {
	// TODO check if executable is writable and stop self update if it is not.
	selfPath, err := GetSelfExecutableInfo()
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("Self-update: Get executable file info error: %v", err),
			stage: "self-update-error"}
		return ""
	}

	err = CheckIsFileWritable(selfPath)
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("Self-update: Check for writable file error: %v", err),
			debug: true}
		messages <- ActionMessage{
			comment: "self update warning",
			msg:     fmt.Sprintf("Skip Self-update: Executable file is not writable."),
			msgType: WarnMsgType,
			stage:   "self-update"}
		return ""
	}

	selfDir := filepath.Dir(selfPath)
	selfName := filepath.Base(selfPath)

	repoClients := []repo.Repo{
		NewSelfS3Client(),
		NewSelfBtClient(),
	}

	var files, downloadFiles map[string]string
	var latestVersion string
	for ind, repoClient := range repoClients {
		shouldIgnoreError := len(repoClients) > ind+1

		sendMessageFunc := func(msg string) {
			msg = fmt.Sprintf("[%s] %s", repoClient.String(), msg)
			var msgType MsgType
			if shouldIgnoreError {
				msgType = WarnMsgType
			} else {
				msgType = FailMsgType
			}

			messages <- ActionMessage{
				msg:     msg,
				msgType: msgType,
				stage:   "self-update",
			}
		}

		versions, err := repoClient.GetPackageVersions()
		if err != nil {
			msg := fmt.Sprintf("Self-update: Package %s GET info error: %v", app.SelfPackageName, err)
			sendMessageFunc(msg)
			if shouldIgnoreError {
				continue
			}

			return ""
		}

		if len(versions) == 0 {
			msg := "Self-update: no versions found"
			sendMessageFunc(msg)
			if shouldIgnoreError {
				continue
			}

			return ""
		} else {
			messages <- ActionMessage{
				msg:   fmt.Sprintf("Self-update: Discover %d versions: %+v", len(versions), versions),
				debug: true}
		}

		// Calc latest version for channel
		latestVersion, err = HighestSemverVersion(versions)
		if err != nil {
			msg := fmt.Sprintf("Self-update: Cannot choose the latest version: %v", err)
			sendMessageFunc(msg)
			if shouldIgnoreError {
				continue
			}

			return ""
		}

		if latestVersion == "" {
			msg := "Self-update: The latest version not found"
			sendMessageFunc(msg)
			if shouldIgnoreError {
				continue
			}

			messages <- ActionMessage{
				comment: "self update error",
				msg:     "Self-update: The latest version not found",
				msgType: FailMsgType,
				stage:   "self-update"}
			return ""
		}

		if latestVersion == app.Version {
			messages <- ActionMessage{
				msg:     "Self-update: Already the latest version",
				msgType: OkMsgType,
				stage:   "self-update"}
			return ""
		}

		messages <- ActionMessage{
			msg:     fmt.Sprintf("Self-update: Detect version %s as the latest", latestVersion),
			msgType: OkMsgType,
			stage:   "self-update"}

		files = ReleaseFiles(app.SelfPackageName, latestVersion, app.OsArch)
		downloadFiles = map[string]string{
			"program": files["program"],
		}
		messages <- ActionMessage{
			msg:   fmt.Sprintf("dstPath is %q, downloadFiles: %+v", selfDir, downloadFiles),
			debug: true}

		messages <- ActionMessage{msg: "Self-update: Downloading ...", debug: true}

		err = repoClient.DownloadFiles(latestVersion, selfDir, downloadFiles)
		if err != nil {
			msg := fmt.Sprintf("Self-update: Download release error: %v", err)
			sendMessageFunc(msg)
			if shouldIgnoreError {
				continue
			}

			return ""
		}

		// TODO add hash verification!
		sha256sums, err := repoClient.GetFileContent(latestVersion, files["hash"])
		if err != nil {
			msg := fmt.Sprintf("Self-update: Download %s error: %v", files["hash"], err)
			sendMessageFunc(msg)
			if shouldIgnoreError {
				continue
			}

			return ""
		}

		// check hash of local binary
		hashes := LoadHashMap(strings.NewReader(sha256sums))
		match, err := VerifyReleaseFileHashFromHashes(messages, selfDir, hashes, files["program"])
		if err != nil {
			msg := fmt.Sprintf("Self-update: %s hash verification error: %v", files["program"], err)
			sendMessageFunc(msg)
			if shouldIgnoreError {
				continue
			}

			return ""
		}
		if !match {
			msg := fmt.Sprintf("Self-update: %s hash is not verified", files["program"])
			sendMessageFunc(msg)
			if shouldIgnoreError {
				continue
			}

			return ""
		}

		break
	}

	// chmod +x for files["program"]
	err = os.Chmod(filepath.Join(selfDir, downloadFiles["program"]), 0755)
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("Self-update: Chmod 755 failed for %s: %v", files["program"], err),
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
	}

	err = ReplaceBinaryFile(selfDir, selfName, downloadFiles["program"])
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("Self-update: Replace executable error: %v", err),
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
	}

	messages <- ActionMessage{
		msg:     fmt.Sprintf("Self-update: Successfully updated to %s", latestVersion),
		msgType: OkMsgType,
		stage:   "self-update"}

	return selfPath
}

// GetSelfExecutableInfo return path of an executable file of current process.
// If file is not owned by user of the process and has no 0x400 bit — return error
func GetSelfExecutableInfo() (path string, err error) {
	selfPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot get executable info: %v", err)
	}

	return selfPath, nil
}

func CheckIsFileWritable(path string) error {
	if err := util.PathShouldBeWritable(path); err != nil {
		return err
	}

	return nil
}

func ReplaceBinaryFile(dir string, currentName string, newName string) (err error) {
	currentPath := filepath.Join(dir, currentName)
	newPath := filepath.Join(dir, newName)
	// this is where we'll move the executable to so that we can swap in the updated replacement
	oldPath := filepath.Join(TmpDir, fmt.Sprintf(".%s.old", currentName))
	// delete any existing old exec file - this is necessary on Windows for two reasons:
	// 1. after a successful update, Windows can't remove the .old file because the process is still running
	// 2. windows rename operations fail if the destination file already exists

	if exist, err := FileExists(oldPath); err != nil {
		return fmt.Errorf("file exists failed (%s): %s", oldPath, err)
	} else if exist {
		if err = os.Remove(oldPath); err != nil {
			return fmt.Errorf("remove file %s failed: %s", oldPath, err)
		}
	}

	// move the existing executable to a new file in the same directory
	err = os.Rename(currentPath, oldPath)
	if err != nil {
		return err
	}

	// move the new executable in to become the new program
	err = os.Rename(newPath, currentPath)
	if err != nil {
		// move unsuccessful
		//
		// The filesystem is now in a bad state. We have successfully
		// moved the existing binary to a new location, but we couldn't move the new
		// binary to take its place. That means there is no file where the current executable binary
		// used to be!
		// Try to rollback by restoring the old binary to its original path.
		rerr := os.Rename(oldPath, currentPath)
		if rerr != nil {
			return &rollbackErr{err, rerr}
		}

		return err
	}

	// windows has trouble with removing old binaries, so hide it instead
	if runtime.GOOS != "windows" {
		// remove the old binary
		errRemove := os.Remove(oldPath)
		if errRemove != nil {
			return errRemove
		}
	}

	return nil
}

type rollbackErr struct {
	error             // original error
	rollbackErr error // error encountered while rolling back
}

// ExecUpdatedBinary replaces current process with process from path binary
// --self-update=no flag is added to arguments to prevent an infinity loop.
func ExecUpdatedBinary(path string) error {
	newArgs := os.Args[0:]
	newArgs = append(newArgs, "--self-update=no")

	if runtime.GOOS == "windows" {
		cmd := exec.Command(path, newArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			return err
		}

		os.Exit(0)
	} else {
		err := syscall.Exec(path, newArgs, os.Environ())
		if err != nil {
			return err
		}
	}

	// Cannot be reached
	return nil
}
