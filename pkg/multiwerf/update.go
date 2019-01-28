package multiwerf

import (
	"fmt"
	"path/filepath"
	"os"
	"syscall"

	"github.com/flant/multiwerf/pkg/bintray"
	"github.com/flant/multiwerf/pkg/app"
)

type BinaryInfo struct {
	BinaryPath string
	Version string
}

// 1. Check for new version → print version
// 2. Check hashes  existed binaries → print 'version stays'
// 3. Download and check if no binaries are existed — print 'updated to'
func UpdateBinary(version string, channel string, messages chan ActionMessage) (binInfo BinaryInfo) {
	btClient := bintray.NewBintrayClient(app.BintraySubject, app.BintrayRepo, app.BintrayPackage)

	// TODO add search of local version in case of GetPackage errors
	// TODO do not exit if internet is not available — for use command
	pkgInfo, err := btClient.GetPackage()
	if err != nil {
		messages <- ActionMessage{
			err: fmt.Errorf("package %s/%s/%s GET info error: %v", app.BintraySubject, app.BintrayRepo, app.BintrayPackage, err),
			state: "exit"}
		return
	}


	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		messages <- ActionMessage{
			msg: fmt.Sprintf("No versions found for package %s/%s/%s", app.BintraySubject, app.BintrayRepo, app.BintrayPackage),
			state: "exit"}
		return
	} else {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Discover %d versions of package %s", len(versions), app.BintrayPackage),
			debug: true}
	}

	// Calc latest version for channel
	latestVersion, err := ChooseLatestVersion(version, channel, versions, AvailableChannels)
	if err != nil {
		messages <- ActionMessage{err: err, state: "exit"}
		return
	}
	if latestVersion == "" {
		messages <- ActionMessage{
			err: fmt.Errorf("No latest version found for %s version of package %s/%s/%s", version, app.BintraySubject, app.BintrayRepo, app.BintrayPackage),
			state: "exit"}
		return
	}
	messages <- ActionMessage{msg: fmt.Sprintf("Detect version '%s' as latest for channel %s@%s", latestVersion, version, channel)}

	werfStorageDir, err := TildeExpand(app.StorageDir)
	if err != nil {
		messages <- ActionMessage{err: err, state: "exit"}
		return
	}

	dstPath := filepath.Join(werfStorageDir, latestVersion)
	files := ReleaseFiles(app.BintrayPackage, latestVersion, app.OsArch)
	messages <- ActionMessage{
		msg: fmt.Sprintf("dstPath is '%s', files: %+v", dstPath, files),
		debug: true}


	// check hash of local binary
	match, err := VerifyReleaseFileHash(dstPath, files["hash"], files["program"])
	if err != nil {
		messages <- ActionMessage{
			msg: fmt.Sprintf("verifying local file error: %v", err ),
			debug: true}
	}
	if match {
		messages <- ActionMessage{
			msg: fmt.Sprintf("werf %s@%s stays at %s", version, channel, latestVersion),
			state: "success"}
		return BinaryInfo{
			Version: latestVersion,
			BinaryPath: filepath.Join(dstPath, files["program"]),
		}
	}

	// If no binary or hash not verified: download
	messages <- ActionMessage{msg: "Start downloading", debug: true}
	err = btClient.DownloadRelease(latestVersion, dstPath, files)
	if err != nil {
		messages <- ActionMessage{err: err, state: "exit"}
		return
	}

	// chmod +x for files["program"]
	err = os.Chmod(filepath.Join(dstPath, files["program"]), 0755)
	if err != nil {
		messages <- ActionMessage{
			err: fmt.Errorf("chmod 755 failed for %s: %v", files["program"], err),
			state: "exit"}
		return
	}

	// Check hash of the binary
	messages <- ActionMessage{msg: "Check hash...", debug: true}

	match, err = VerifyReleaseFileHash(dstPath, files["hash"], files["program"])
	if err != nil {
		messages <- ActionMessage{
			err: fmt.Errorf("verifying release error: %v", err),
			state: "exit"}
		return
	}
	if !match {
		// Not match — ERROR and exit
		messages <- ActionMessage{err: fmt.Errorf("hash of '%s' is not verified!", files["program"]), state: "exit"}
		return
	}

	messages <- ActionMessage{
		msg: fmt.Sprintf("werf %s@%s updated to %s", version, channel, latestVersion),
		state: "success"}
	return BinaryInfo{
		Version: latestVersion,
		BinaryPath: filepath.Join(dstPath, files["program"]),
	}
}


func SelfUpdate(messages chan ActionMessage) string {
	// TODO check executable is writable and stop self update if it is not.
	selfPath, err := GetSelfExecutableInfo()
	if err != nil {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self update: get executable file info error: %v", err),
			state: "self-update-error"}
		return ""
	}

	err = CheckIsFileWritable(selfPath)
	if err != nil {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self-update: check is writable error: %v", err),
			debug: true}
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self-update is disabled. Executable file is not writable."),
			state: "self-update-warning"}
		return ""
	}

	selfDir := filepath.Dir(selfPath)
	selfName := filepath.Base(selfPath)

	btClient := bintray.NewBintrayClient(app.SelfBintraySubject, app.SelfBintrayRepo, app.SelfBintrayPackage)

	pkgInfo, err := btClient.GetPackage()
	if err != nil {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self update: package %s/%s/%s GET info error: %v", app.SelfBintraySubject, app.SelfBintrayRepo, app.SelfBintrayPackage, err),
			state: "self-update-error"}
		return ""
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self update: no versions found"),
			state: "self-update-error"}
		return ""
	} else {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self update: discover %d versions: %+v", len(versions), versions),
			debug: true}
	}

	// Calc latest version for channel
	latestVersion, err := ChooseLatestVersionSimple(versions)
	if err != nil {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self update: cannot choose latest version: %v", err),
			state: "self-update-error"}
		return ""
	}
	if latestVersion == "" {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self update: no latest version found"),
			state: "self-update-error"}
		return ""
	}
	if latestVersion == app.Version {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self update: already latest version %s", app.Version),
			state: "self-update-success"}
		return ""
	}

	messages <- ActionMessage{
		msg: fmt.Sprintf("Multiwerf self update: detect version '%s' as latest", latestVersion)}

	files := ReleaseFiles(app.SelfBintrayPackage, latestVersion, app.OsArch)
	downloadFiles := map[string]string {
		"program": files["program"],
	}
	messages <- ActionMessage{
		msg: fmt.Sprintf("dstPath is '%s', downloadFiles: %+v", selfDir, downloadFiles),
		debug: true}

	messages <- ActionMessage{msg: "Multiwerf self-update: start downloading", debug: true}
	err = btClient.DownloadRelease(latestVersion, selfDir, downloadFiles)
	if err != nil {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self-update: download release error: %v", err),
			state: "self-update-error"}
		return ""
	}

	// TODO add hash verification

	// chmod +x for files["program"]
	err = os.Chmod(filepath.Join(selfDir, downloadFiles["program"]), 0755)
	if err != nil {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self-update: chmod 755 failed for %s: %v", files["program"], err),
			state: "self-update-error"}
		return ""
	}

	err = ReplaceBinaryFile(selfDir, selfName, downloadFiles["program"])
	if err != nil {
		messages <- ActionMessage{
			msg: fmt.Sprintf("Multiwerf self-update: replace executable error: %v", err),
			state: "self-update-error"}
		return ""
	}

	messages <- ActionMessage{
		msg: fmt.Sprintf("Multiwerf self-update: successfully updated to %s", latestVersion),
		state: "self-update-success"}
	return selfPath
}

// GetSelfExecutableInfo return path of an executable file of current process.
// If file is not owned by user of the process and has no 0x400 bit — return error
func GetSelfExecutableInfo() (path string, err error) {
	selfPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot get executabe info: %v", err)
	}

	return selfPath, nil
}

func CheckIsFileWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot stat file '%s': %v", path, err)
	}

	// Check if the user write bit is enabled in file permission
	if info.Mode().Perm() & (1 << (uint(7))) == 0 {
		return fmt.Errorf("write permission bit is not set on '%s'", path)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("no stat_t for file '%s'", path)
	}

	if uint32(os.Geteuid()) != stat.Uid {
		return fmt.Errorf("user %d doesn't have permission to write '%s'", os.Geteuid(), path)
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

	// move the new exectuable in to become the new program
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