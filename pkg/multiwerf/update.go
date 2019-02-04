package multiwerf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/bintray"
)

type BinaryInfo struct {
	BinaryPath   string
	Version      string
	HashVerified bool
}

type BinaryUpdater interface {
	// this method is intented to update from remote source
	// check remote — get local — if local need update — download and verify a hash
	// ↑no remote — error/exit
	//                ↑no local — no error
	//                                                   ↑ error if download failed or hash not verified
	DownloadLatest(version string, channel string) (binInfo BinaryInfo)

	// this method return BinaryInfo instance
	// multiwerf exit with error if no binary found remote or local
	// check remote versions — check local version — if local need update — download and verify a hash
	GetLatestBinaryInfo(version string, channel string) (binInfo BinaryInfo)
}

type MainBinaryUpdater struct {
	BintrayClient bintray.BintrayClient
	Messages      chan ActionMessage
}

func NewBinaryUpdater(messages chan ActionMessage) BinaryUpdater {
	result := &MainBinaryUpdater{}
	result.BintrayClient = bintray.NewBintrayClient(app.BintraySubject, app.BintrayRepo, app.BintrayPackage)
	result.Messages = messages
	return result
}

// UpdateBinary check for new latest version in bintray.
// Exit with error on connection problems or if no versions found for version/channel
// 1. Check for new version → print version
// 2. Check hashes  existed binaries → print 'version stays'
// 3. Download and check if no binaries are existed — print 'updated to'
func (u *MainBinaryUpdater) DownloadLatest(version string, channel string) (binInfo BinaryInfo) {
	u.Messages <- ActionMessage{msg: "Start DownloadLatest", debug: true}

	latestVersion, err := RemoteLatestVersion(version, channel, u.Messages, u.BintrayClient)
	if err != nil {
		u.Messages <- ActionMessage{err: err, action: "exit"}
		return
	}

	u.Messages <- ActionMessage{
		msg:     fmt.Sprintf("detect version '%s' as latest for channel %s/%s", latestVersion, version, channel),
		msgType: "ok"}

	binInfo, _ = GetBinaryInfo(latestVersion, u.Messages)
	if binInfo.HashVerified {
		u.Messages <- ActionMessage{
			comment: "no update needed",
			msg:     fmt.Sprintf("%s %s/%s stays at %s", app.BintrayPackage, version, channel, latestVersion),
			msgType: "ok",
			action:  "exit"}
		return
	}

	// If no locale binary or hash not verified: download
	err = DownloadVersion(latestVersion, u.Messages, u.BintrayClient)
	if err != nil {
		u.Messages <- ActionMessage{err: err}
		return
	}

	// Check hash of the binary
	u.Messages <- ActionMessage{msg: "Check hash...", debug: true}

	binInfo, err = GetBinaryInfo(latestVersion, u.Messages)
	if err != nil {
		u.Messages <- ActionMessage{
			err: fmt.Errorf("verifying release error: %v", err)}
		return
	}
	if !binInfo.HashVerified {
		// Not match — ERROR and exit
		u.Messages <- ActionMessage{
			err: fmt.Errorf("hash for version %s is not verified!", latestVersion)}
		return
	}

	u.Messages <- ActionMessage{
		comment: fmt.Sprintf("# update %s success", app.BintrayPackage),
		msg:     fmt.Sprintf("%s %s/%s updated to %s", app.BintrayPackage, version, channel, latestVersion),
		msgType: "ok",
		action:  "exit"}

	return binInfo
}

// GetLatestBinaryInfo return BinaryInfo object for latest binary with version/channel.
// Checks for local versions and remote versions. If no remote version is available — use
// local version.
func (u *MainBinaryUpdater) GetLatestBinaryInfo(version string, channel string) (binInfo BinaryInfo) {
	remoteLatestVersion := ""
	if app.Update == "yes" {
		var err error
		remoteLatestVersion, err = RemoteLatestVersion(version, channel, u.Messages, u.BintrayClient)
		if err != nil {
			u.Messages <- ActionMessage{
				msg:     err.Error(),
				msgType: "warn"}
		}
	}

	localBinaryInfo, err := LocalLatestBinaryInfo(version, channel, u.Messages)
	if err != nil {
		u.Messages <- ActionMessage{
			msg:     err.Error(),
			msgType: "warn"}
	}

	localLatestVersion := localBinaryInfo.Version

	// no remote, no local — exit with error
	if remoteLatestVersion == "" && localLatestVersion == "" {
		u.Messages <- ActionMessage{
			err: fmt.Errorf("Cannot determine latest version neither from bintray package '%s' nor from local storage %s", app.BintrayPackage, app.StorageDir),
		}
		return
	}

	// has local and no remote or local is equal to remote — no update needed, stay at local version
	if localLatestVersion != "" {
		if (remoteLatestVersion == "" || localLatestVersion == remoteLatestVersion) && localBinaryInfo.HashVerified {
			u.Messages <- ActionMessage{
				comment: "no update needed",
				msg:     fmt.Sprintf("%s %s/%s stays at %s", app.BintrayPackage, version, channel, localLatestVersion),
				msgType: "ok",
				action:  "exit"}
			return localBinaryInfo
		}
		if remoteLatestVersion == "" && !localBinaryInfo.HashVerified {
			u.Messages <- ActionMessage{
				err: fmt.Errorf("Cannot determine latest version from bintray package '%s' and local binary '%s' is corrupted", app.BintrayPackage, localLatestVersion),
			}
			return
		}
	}

	// localVersion is "" or localVersion is not equal to remoteVersion or local hash is not verified
	if remoteLatestVersion != "" {
		u.Messages <- ActionMessage{
			msg:     fmt.Sprintf("Detect version '%s' as latest for channel %s/%s", remoteLatestVersion, version, channel),
			msgType: "ok"}

		// Download
		err = DownloadVersion(remoteLatestVersion, u.Messages, u.BintrayClient)
		if err != nil {
			u.Messages <- ActionMessage{err: err}
			return
		}

		// Check hash of the binary
		u.Messages <- ActionMessage{msg: "Check hash...", debug: true}

		newBinInfo, err := GetBinaryInfo(remoteLatestVersion, u.Messages)
		if err != nil {
			u.Messages <- ActionMessage{
				err: fmt.Errorf("verifying release error: %v", err)}
			return
		}
		if !newBinInfo.HashVerified {
			// Not match — ERROR and exit
			u.Messages <- ActionMessage{
				err: fmt.Errorf("hash for version %s is not verified!", remoteLatestVersion)}
			return
		}

		u.Messages <- ActionMessage{
			comment: fmt.Sprintf("# update %s success", app.BintrayPackage),
			msg:     fmt.Sprintf("%s %s/%s updated to %s", app.BintrayPackage, version, channel, remoteLatestVersion),
			msgType: "ok",
			action:  "exit"}
		return newBinInfo
	}

	// It's a bug if reached!
	u.Messages <- ActionMessage{
		err: fmt.Errorf("BUG: %s %s/%s detect remote version as '%s' and local '%s' but no action is performed.", app.BintrayPackage, version, channel, remoteLatestVersion, localLatestVersion),
	}
	return
}

// Get local latest for version/channel
// Get remote latest for version/channel

// no remote no local — error
// if equals — go check local sha256sum
// remote errored or no version at all but local version is present — go to sha256 sum
// if no local but remote — skip local check, go to download

// LocalLatestBinaryInfo return BinaryInfo for latest localy available version
//
// 1. find version dirs in ~/.multiwerf
// 2. find latest version for channel and verify a hash for that binary
func LocalLatestBinaryInfo(version string, channel string, messages chan ActionMessage) (binInfo BinaryInfo, err error) {

	// create list of directories that names look like a semver
	subDirs := []string{}
	errMsgs := []string{}

	filepath.Walk(MultiwerfStorageDir, func(filePath string, fi os.FileInfo, err error) error {
		if err != nil {
			errMsgs = append(errMsgs, err.Error())
			return nil
		}

		if fi.IsDir() {
			// No action for initial directory
			if filePath == MultiwerfStorageDir {
				return nil
			}
			// Skip hidden directories inside initial directory
			if strings.HasPrefix(fi.Name(), ".") {
				return filepath.SkipDir
			}

			v, vErr := semver.NewVersion(fi.Name())
			if vErr != nil {
				errMsgs = append(errMsgs, err.Error())
				return filepath.SkipDir
			}
			subDirs = append(subDirs, v.Original())
			// Do not go deeper
			return filepath.SkipDir
		}

		return nil
	})
	if len(errMsgs) > 0 {
		err = errors.New(strings.Join(errMsgs, "\n"))
	}

	latestVersion, err := ChooseLatestVersion(version, channel, subDirs, AvailableChannels)
	if err != nil {
		return
	}
	if latestVersion == "" {
		err = fmt.Errorf("No valid versions found for %s/%s in local storage %s", version, channel, app.StorageDir)
		return
	}

	return GetBinaryInfo(latestVersion, messages)
}

// GetBinaryInfo return BinaryInfo object for binary with exact version if it is
// stored in MultiwerfStorageDir. Empty object is returned if no binary found.
// Hash of binary is verified with SHA256SUMS files.
func GetBinaryInfo(version string, messages chan ActionMessage) (binInfo BinaryInfo, err error) {
	binInfo = BinaryInfo{}

	// Verify hash for found version
	dstPath := filepath.Join(MultiwerfStorageDir, version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is '%s', files: %+v", dstPath, files),
		debug: true}

	binInfo.Version = version
	binInfo.BinaryPath = filepath.Join(dstPath, files["program"])

	// check hash of local binary
	match, err := VerifyReleaseFileHash(dstPath, files["hash"], files["program"])
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("verifying local file error: %v", err),
			debug: true}
		return
	}

	binInfo.HashVerified = match

	return
}

// RemoteLatestVersion searches for a latest available version in bintray
func RemoteLatestVersion(version string, channel string, messages chan ActionMessage, btClient bintray.BintrayClient) (latestVersion string, err error) {
	pkgInfo, err := btClient.GetPackage()
	if err != nil {
		return "", fmt.Errorf("Get info for package '%s' error: %v", app.BintrayPackage, err)
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		return "", fmt.Errorf("No versions found in bintray for '%s'", app.BintrayPackage)
	} else {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("Discover %d versions for channel %s/%s of package %s", len(versions), version, channel, app.BintrayPackage),
			debug: true}
	}

	// Calc a latest version for version/channel
	latestVersion, err = ChooseLatestVersion(version, channel, versions, AvailableChannels)
	if err != nil {
		return
	}
	if latestVersion == "" {
		return "", fmt.Errorf("No valid versions found for %s/%s channel of package %s/%s/%s", version, channel, app.BintraySubject, app.BintrayRepo, app.BintrayPackage)
	}
	return
}

func DownloadVersion(version string, messages chan ActionMessage, btClient bintray.BintrayClient) (err error) {
	// Verify hash for found version
	dstPath := filepath.Join(MultiwerfStorageDir, version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)

	messages <- ActionMessage{msg: "Start downloading", debug: true}

	err = btClient.DownloadRelease(version, dstPath, files)
	if err != nil {
		return err
	}

	// chmod +x for files["program"]
	err = os.Chmod(filepath.Join(dstPath, files["program"]), 0755)
	if err != nil {
		return fmt.Errorf("chmod 755 failed for %s: %v", files["program"], err)
	}

	return nil
}
