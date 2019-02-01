package multiwerf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/bintray"
)

type BinaryInfo struct {
	BinaryPath string
	Version    string
}

func LocalLatestVersion(version string, channel string, messages chan ActionMessage) string {
	return "DA"
}

func RemoteLatestVersion(version string, channel string, messages chan ActionMessage) string {
	return "NET"
}

// 1. Check for new version → print version
// 2. Check hashes  existed binaries → print 'version stays'
// 3. Download and check if no binaries are existed — print 'updated to'
func UpdateBinary(version string, channel string, messages chan ActionMessage) (binInfo BinaryInfo) {
	btClient := bintray.NewBintrayClient(app.BintraySubject, app.BintrayRepo, app.BintrayPackage)

	// TODO retrieve local available versions

	// TODO do not exit if internet is not available
	pkgInfo, err := btClient.GetPackage()
	if err != nil {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("package %s/%s/%s GET info error: %v", app.BintraySubject, app.BintrayRepo, app.BintrayPackage, err),
			msgType: "fail",
			action:  "exit"}
		return
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("No versions found for package %s/%s/%s", app.BintraySubject, app.BintrayRepo, app.BintrayPackage),
			msgType: "warn",
			action:  "exit"}
		return
	} else {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("Discover %d versions of package %s", len(versions), app.BintrayPackage),
			debug: true}
	}

	// Calc latest version for channel
	latestVersion, err := ChooseLatestVersion(version, channel, versions, AvailableChannels)
	if err != nil {
		messages <- ActionMessage{err: err, action: "exit"}
		return
	}

	// TODO use local version if no remote latest version!
	if latestVersion == "" {
		messages <- ActionMessage{
			err:    fmt.Errorf("No %s version found for %s version of package %s/%s/%s", channel, version, app.BintraySubject, app.BintrayRepo, app.BintrayPackage),
			action: "exit"}
		return
	}
	messages <- ActionMessage{
		msg:     fmt.Sprintf("Detect version '%s' as latest for channel %s@%s", latestVersion, version, channel),
		msgType: "ok"}

	werfStorageDir, err := TildeExpand(app.StorageDir)
	if err != nil {
		messages <- ActionMessage{err: err, stage: "exit"}
		return
	}

	dstPath := filepath.Join(werfStorageDir, latestVersion)
	files := ReleaseFiles(app.BintrayPackage, latestVersion, app.OsArch)
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is '%s', files: %+v", dstPath, files),
		debug: true}

	// check hash of local binary
	match, err := VerifyReleaseFileHash(dstPath, files["hash"], files["program"])
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("verifying local file error: %v", err),
			debug: true}
	}
	if match {
		messages <- ActionMessage{
			comment: "no update needed",
			msg:     fmt.Sprintf("werf %s@%s stays at %s", version, channel, latestVersion),
			msgType: "ok",
			action:  "exit"}
		return BinaryInfo{
			Version:    latestVersion,
			BinaryPath: filepath.Join(dstPath, files["program"]),
		}
	}

	// If no binary or hash not verified: download
	messages <- ActionMessage{msg: "Start downloading", debug: true}
	err = btClient.DownloadRelease(latestVersion, dstPath, files)
	if err != nil {
		messages <- ActionMessage{err: err, stage: "exit"}
		return
	}

	// chmod +x for files["program"]
	err = os.Chmod(filepath.Join(dstPath, files["program"]), 0755)
	if err != nil {
		messages <- ActionMessage{
			err:   fmt.Errorf("chmod 755 failed for %s: %v", files["program"], err),
			stage: "exit"}
		return
	}

	// Check hash of the binary
	messages <- ActionMessage{msg: "Check hash...", debug: true}

	match, err = VerifyReleaseFileHash(dstPath, files["hash"], files["program"])
	if err != nil {
		messages <- ActionMessage{
			err:    fmt.Errorf("verifying release error: %v", err),
			action: "exit"}
		return
	}
	if !match {
		// Not match — ERROR and exit
		messages <- ActionMessage{
			err:   fmt.Errorf("hash of '%s' is not verified!", files["program"]),
			stage: "exit"}
		return
	}

	messages <- ActionMessage{
		comment: fmt.Sprintf("# update %s success", app.BintrayPackage),
		msg:     fmt.Sprintf("werf %s@%s updated to %s", version, channel, latestVersion),
		msgType: "ok",
		action:  "exit"}
	return BinaryInfo{
		Version:    latestVersion,
		BinaryPath: filepath.Join(dstPath, files["program"]),
	}
}
