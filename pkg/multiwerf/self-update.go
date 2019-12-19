package multiwerf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/bintray"
	"github.com/flant/multiwerf/pkg/util"
)

var multiwerfProlog = fmt.Sprintf("%s %s self-update", app.AppName, app.Version)

// SelfUpdate checks for new version of multiwerf, download it and execute as a new process.
// Note: multiwerf has no option to exit on self-update errors.
func SelfUpdate(messages chan ActionMessage) string {
	// TODO check if executable is writable and stop self update if it is not.
	selfPath, err := GetSelfExecutableInfo()
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("%s: get executable file info error: %v", multiwerfProlog, err),
			stage: "self-update-error"}
		return ""
	}

	err = CheckIsFileWritable(selfPath)
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("%s: check for writable file error: %v", multiwerfProlog, err),
			debug: true}
		messages <- ActionMessage{
			comment: "self update warning",
			msg:     fmt.Sprintf("Skip self-update: executable file is not writable."),
			msgType: "warn",
			stage:   "self-update"}
		return ""
	}

	selfDir := filepath.Dir(selfPath)
	selfName := filepath.Base(selfPath)

	btClient := bintray.NewBintrayClient(app.SelfBintraySubject, app.SelfBintrayRepo, app.SelfBintrayPackage)

	pkgInfo, err := btClient.GetPackageInfo()
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: package %s GET info error: %v", multiwerfProlog, app.SelfBintrayPackage, err),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: no versions found", multiwerfProlog),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	} else {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("%s: discover %d versions: %+v", multiwerfProlog, len(versions), versions),
			debug: true}
	}

	// Calc latest version for channel
	latestVersion, err := HighestSemverVersion(versions)
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: cannot choose latest version: %v", multiwerfProlog, err),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}
	if latestVersion == "" {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: no latest version found", multiwerfProlog),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}

	if latestVersion == app.Version {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("%s: already latest version", multiwerfProlog),
			msgType: "ok",
			stage:   "self-update"}
		return ""
	}

	messages <- ActionMessage{
		msg:     fmt.Sprintf("%s: detect version %s as latest", multiwerfProlog, latestVersion),
		msgType: "ok",
		stage:   "self-update"}

	files := ReleaseFiles(app.SelfBintrayPackage, latestVersion, app.OsArch)
	downloadFiles := map[string]string{
		"program": files["program"],
	}
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is %s, downloadFiles: %+v", selfDir, downloadFiles),
		debug: true}

	messages <- ActionMessage{msg: fmt.Sprintf("%s: start downloading", multiwerfProlog), debug: true}
	err = btClient.DownloadFiles(latestVersion, selfDir, downloadFiles)
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: download release error: %v", multiwerfProlog, err),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}

	// TODO add hash verification!
	sha256sums, err := btClient.GetFileContent(latestVersion, files["hash"])
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: download %s error: %v", multiwerfProlog, files["hash"], err),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}

	// check hash of local binary
	hashes := LoadHashMap(strings.NewReader(sha256sums))
	match, err := VerifyReleaseFileHashFromHashes(selfDir, hashes, files["program"])
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: %s hash verification error: %v", multiwerfProlog, files["program"], err),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}
	if !match {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: %s hash is not verified", multiwerfProlog, files["program"]),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}

	// chmod +x for files["program"]
	err = os.Chmod(filepath.Join(selfDir, downloadFiles["program"]), 0755)
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: chmod 755 failed for %s: %v", multiwerfProlog, files["program"], err),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}

	err = ReplaceBinaryFile(selfDir, selfName, downloadFiles["program"])
	if err != nil {
		messages <- ActionMessage{
			comment: "self update error",
			msg:     fmt.Sprintf("%s: replace executable error: %v", multiwerfProlog, err),
			msgType: "fail",
			stage:   "self-update"}
		return ""
	}

	messages <- ActionMessage{
		msg:     fmt.Sprintf("%s: successfully updated to %s", multiwerfProlog, latestVersion),
		msgType: "ok",
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
	oldPath := filepath.Join(dir, fmt.Sprintf(".%s.old", currentName))
	// delete any existing old exec file - this is necessary on Windows for two reasons:
	// 1. after a successful update, Windows can't remove the .old file because the process is still running
	// 2. windows rename operations fail if the destination file already exists
	_ = os.Remove(oldPath)

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

	// move successful, remove the old binary if needed
	//if removeOld {
	errRemove := os.Remove(oldPath)

	// windows has trouble with removing old binaries, so hide it instead
	if errRemove != nil {
		//	_ = hideFile(oldPath)
		return errRemove
	}
	//}

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
	err := syscall.Exec(path, newArgs, os.Environ())
	if err != nil {
		return err
	}
	// Cannot be reached
	return nil
}
