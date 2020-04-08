package multiwerf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/flant/shluz"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/bintray"
	"github.com/flant/multiwerf/pkg/output"
	"github.com/flant/multiwerf/pkg/util"
)

const SelfUpdateLockName = "self-update"

// update multiwerf binary (self-update)
func PerformSelfUpdate(printer output.Printer, skipSelfUpdate bool) (err error) {
	messages := make(chan ActionMessage, 0)
	selfPath := ""

	go func() {
		if skipSelfUpdate {
			// self-update is disabled. Silently skip it
			messages <- ActionMessage{
				msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
				msgType: OkMsgType,
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
				msgType: WarnMsgType,
			}

			messages <- ActionMessage{
				msg:     fmt.Sprintf("Skip Self-update: Cannot acquire a lock: %v", err),
				msgType: WarnMsgType,
			}

			messages <- ActionMessage{action: "exit"}

			return
		} else {
			if !isAcquired {
				messages <- ActionMessage{
					msg:     fmt.Sprintf("%s %s", app.AppName, app.Version),
					msgType: WarnMsgType,
				}

				messages <- ActionMessage{
					msg:     "Self-update has been skipped because the operation is performing by another process",
					msgType: WarnMsgType,
				}

				messages <- ActionMessage{action: "exit"}

				return
			}
		}

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

	var repoName string
	if app.Experimental {
		repoName = app.SelfExperimentalBintrayRepo
	} else {
		repoName = app.SelfBintrayRepo
	}

	btClient := bintray.NewBintrayClient(app.SelfBintraySubject, repoName, app.SelfBintrayPackage)

	pkgInfo, err := btClient.GetPackageInfo()
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("Self-update: Package %s GET info error: %v", app.SelfBintrayPackage, err),
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     "Self-update: no versions found",
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
	} else {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("Self-update: Discover %d versions: %+v", len(versions), versions),
			debug: true}
	}

	// Calc latest version for channel
	latestVersion, err := HighestSemverVersion(versions)
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("Self-update: Cannot choose the latest version: %v", err),
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
	}
	if latestVersion == "" {
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

	files := ReleaseFiles(app.SelfBintrayPackage, latestVersion, app.OsArch)
	downloadFiles := map[string]string{
		"program": files["program"],
	}
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is %s, downloadFiles: %+v", selfDir, downloadFiles),
		debug: true}

	messages <- ActionMessage{msg: "Self-update: Downloading ...", debug: true}
	err = btClient.DownloadFiles(latestVersion, selfDir, downloadFiles)
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("Self-update: Download release error: %v", err),
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
	}

	// TODO add hash verification!
	sha256sums, err := btClient.GetFileContent(latestVersion, files["hash"])
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("Self-update: Download %s error: %v", files["hash"], err),
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
	}

	// check hash of local binary
	hashes := LoadHashMap(strings.NewReader(sha256sums))
	match, err := VerifyReleaseFileHashFromHashes(messages, selfDir, hashes, files["program"])
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("Self-update: %s hash verification error: %v", files["program"], err),
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
	}
	if !match {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("Self-update: %s hash is not verified", files["program"]),
			msgType: FailMsgType,
			stage:   "self-update"}
		return ""
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
