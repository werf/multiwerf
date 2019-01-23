package multiwerf

import (
	"fmt"
	"path/filepath"

	"github.com/flant/multiwerf/pkg/bintray"
	"github.com/flant/multiwerf/pkg/app"
	"os"
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

	pkgInfo, err := btClient.GetPackage()
	if err != nil {
		messages <- ActionMessage{err: err, state: "exit"}
		return
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		messages <- ActionMessage{
			msg: fmt.Sprintf("No versions found for package %s/%s/%s", app.BintraySubject, app.BintrayRepo, app.BintrayPackage),
			state: "exit"}
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
	if match {
		messages <- ActionMessage{
			msg: fmt.Sprintf("werf %s@%s updated to %s", version, channel, latestVersion),
			state: "success"}
		return BinaryInfo{
			Version: latestVersion,
			BinaryPath: filepath.Join(dstPath, files["program"]),
		}

	}

	// Not match — ERROR and grace exit
	messages <- ActionMessage{err: fmt.Errorf("hash of release is not verified"), state: "exit"}
	return
}


