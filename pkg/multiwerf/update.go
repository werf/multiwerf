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
	"github.com/flant/multiwerf/pkg/util"
)

type BinaryInfo struct {
	BinaryPath        string
	Version           string
	HashVerified      bool
	AvailableVersions []string
}

type BinaryUpdater interface {
	// DownloadLatest updates a binary in local storage from remote source
	//
	// Check remote latest → get local latest → if local need update → download and verify a hash
	// ↑no remote — error/exit
	//                       ↑no local — no error
	//                                                                 ↑ error if download failed or hash not verified
	DownloadLatest(version string, channel string) (binInfo BinaryInfo)

	// GetLatestBinaryInfo returns BinaryInfo instance with path to the program of latest version.
	//
	// If remote is enabled, then method tries to get a latest version from remote source and verify a hash.
	//
	// If remote is not enabled or remote update was not successfull, then method check if local latest version is good.
	GetLatestBinaryInfo(version string, channel string) (binInfo BinaryInfo)

	// SetRemoteEnabled sets remoteEnabled flag
	SetRemoteEnabled(enabled bool)
}

type MainBinaryUpdater struct {
	BintrayClient bintray.BintrayClient
	Messages      chan ActionMessage
	RemoteEnabled bool
}

func NewBinaryUpdater(messages chan ActionMessage) BinaryUpdater {
	result := &MainBinaryUpdater{
		RemoteEnabled: false,
	}
	result.BintrayClient = bintray.NewBintrayClient(app.BintraySubject, app.BintrayRepo, app.BintrayPackage)
	result.Messages = messages
	return result
}

// DownloadLatest check for new latest version in bintray.
// Exit with error on connection problems or if no versions found for version/channel
// 1. Check for new version → print version
// 2. Check hashes  existed binaries → print 'version stays'
// 3. Download and check if no binaries are existed — print 'updated to'
func (u *MainBinaryUpdater) DownloadLatest(version string, channel string) (binInfo BinaryInfo) {
	u.Messages <- ActionMessage{msg: "Start DownloadLatest", debug: true}

	remoteBinInfo, err := RemoteLatestBinaryInfo(version, channel, u.Messages, u.BintrayClient)
	if err != nil {
		u.Messages <- ActionMessage{err: err}
		return
	}

	latestVersion := remoteBinInfo.Version

	u.Messages <- ActionMessage{
		msg:     fmt.Sprintf("detect version '%s' as latest for channel %s/%s", latestVersion, version, channel),
		msgType: "ok"}

	binInfo, _ = GetVerifiedBinaryInfo(latestVersion, u.Messages)
	if binInfo.HashVerified {
		u.Messages <- ActionMessage{
			comment: "no update needed",
			msg:     fmt.Sprintf("%s %s/%s stays at %s", app.BintrayPackage, version, channel, latestVersion),
			msgType: "ok"}
		return
	}

	// Download a release if no local latest binary or hash is not verified
	binInfo, err = DownloadRelease(latestVersion, u.Messages, u.BintrayClient)
	if err != nil {
		u.Messages <- ActionMessage{
			err: fmt.Errorf("%s %s/%s: %v", app.BintrayPackage, version, channel, err)}
		return
	}

	u.Messages <- ActionMessage{
		comment: fmt.Sprintf("# update %s success", app.BintrayPackage),
		msg:     fmt.Sprintf("%s %s/%s updated to %s", app.BintrayPackage, version, channel, latestVersion),
		msgType: "ok"}

	return binInfo
}

// GetLatestBinaryInfo return BinaryInfo object for latest binary with version/channel.
// Checks for local versions and remote versions. If no remote version is available — use
// local version.
func (u *MainBinaryUpdater) GetLatestBinaryInfo(version string, channel string) (binInfo BinaryInfo) {
	localLatestVersion := ""
	localAvailableVersions := []string{}
	localBinaryInfo, err := LocalLatestBinaryInfo(version, channel, u.Messages)
	if err != nil {
		u.Messages <- ActionMessage{
			msg:     err.Error(),
			msgType: "warn"}
	} else {
		localLatestVersion = localBinaryInfo.Version
		localAvailableVersions = localBinaryInfo.AvailableVersions
	}
	// If remote is disabled, return localBinaryInfo immediately.
	if !u.RemoteEnabled && localLatestVersion != "" {
		u.Messages <- ActionMessage{
			comment: "update is disabled",
			msg:     fmt.Sprintf("%s %s/%s: update is disabled, use local latest %s", app.BintrayPackage, version, channel, localLatestVersion),
			msgType: "ok"}
		return localBinaryInfo
	}

	remoteBinInfo := BinaryInfo{}
	remoteLatestVersion := ""
	if u.RemoteEnabled {
		var err error
		remoteBinInfo, err = RemoteLatestBinaryInfo(version, channel, u.Messages, u.BintrayClient)
		if err != nil {
			u.Messages <- ActionMessage{
				msg:     err.Error(),
				msgType: "warn"}
		} else {
			remoteLatestVersion = remoteBinInfo.Version
		}
	}

	// There are 3 ways now:
	// - no remote version, no local — that is an error
	// - no remote version or remote version is equal to local — use local version
	// - remote version is not equal to local — download new release and use it

	// no remote, no local — exit with error
	if remoteLatestVersion == "" && localLatestVersion == "" {
		u.Messages <- ActionMessage{
			msg:     fmt.Sprintf("Cannot determine latest version for %s/%s neither from bintray package '%s' nor from local storage %s", version, channel, app.BintrayPackage, app.StorageDir),
			msgType: "fail",
		}
		if !u.RemoteEnabled {
			u.Messages <- ActionMessage{
				msg:     fmt.Sprintf("Auto update of `%s` is disabled or delayed. Try `multiwerf update %s %s` command.", app.BintrayPackage, version, channel),
				msgType: "warn",
			}
		}
		// Show top 5 versions from remote or from local if remote is disabled
		if u.RemoteEnabled && len(remoteBinInfo.AvailableVersions) > 0 {
			latestAvailable := PickLatestVersions(version, remoteBinInfo.AvailableVersions, 5)
			if len(latestAvailable) > 0 {
				msg := strings.Join(latestAvailable, "\n")
				u.Messages <- ActionMessage{
					msg:     fmt.Sprintf("Top %d latest versions for '%s' from bintray package:", len(latestAvailable), version),
					msgType: "warn",
				}
				u.Messages <- ActionMessage{
					msg:     msg,
					msgType: "warn",
				}
			}
		} else {
			latestAvailable := PickLatestVersions(version, localAvailableVersions, 5)
			if len(latestAvailable) > 0 {
				msg := strings.Join(latestAvailable, "\n")
				u.Messages <- ActionMessage{
					msg:     fmt.Sprintf("Top %d latest versions for '%s' from local storage:", len(latestAvailable), version),
					msgType: "warn",
				}
				u.Messages <- ActionMessage{
					msg:     msg,
					msgType: "warn",
				}
			}
		}
		u.Messages <- ActionMessage{
			err: fmt.Errorf(""),
		}
		return
	}

	// remote is disabled or error or remote version is same as local latest version. No update needed, stay at local version.
	if localLatestVersion != "" {
		if remoteLatestVersion == "" || remoteLatestVersion == localLatestVersion {
			u.Messages <- ActionMessage{
				comment: "no update needed",
				msg:     fmt.Sprintf("%s %s/%s: no update needed, use local latest %s", app.BintrayPackage, version, channel, localLatestVersion),
				msgType: "ok"}
			return localBinaryInfo
		}
	}

	// remote returns valid latest version not equal to local, update is needed.
	if remoteLatestVersion != "" {
		u.Messages <- ActionMessage{
			msg:     fmt.Sprintf("Detect version '%s' as latest for channel %s/%s", remoteLatestVersion, version, channel),
			msgType: "ok"}

		// Download and verify release files
		binInfo, err = DownloadRelease(remoteLatestVersion, u.Messages, u.BintrayClient)
		if err != nil {
			if localLatestVersion == "" {
				u.Messages <- ActionMessage{
					msg:     fmt.Sprintf("%s %s/%s: no local version found and download release %s failed: %v", app.BintrayPackage, version, channel, remoteLatestVersion, err),
					msgType: "fail",
				}
				u.Messages <- ActionMessage{
					err: fmt.Errorf("")}
				return
			} else {
				u.Messages <- ActionMessage{
					comment: fmt.Sprintf("download release %s failed", remoteLatestVersion),
					msg:     fmt.Sprintf("Download %s is failed: %v", remoteLatestVersion, err),
					msgType: "fail"}
				u.Messages <- ActionMessage{
					comment: fmt.Sprintf("use local latest version %s", localLatestVersion),
					msg:     fmt.Sprintf("%s %s/%s: use local latest %s", app.BintrayPackage, version, channel, localLatestVersion),
					msgType: "ok"}
				return localBinaryInfo
			}
		}

		u.Messages <- ActionMessage{
			comment: fmt.Sprintf("# update %s success", app.BintrayPackage),
			msg:     fmt.Sprintf("%s %s/%s: update successful, use %s", app.BintrayPackage, version, channel, remoteLatestVersion),
			msgType: "ok"}
		return binInfo
	}

	// It's a bug if reached!
	u.Messages <- ActionMessage{
		err: fmt.Errorf("BUG: %s %s/%s: detect remote version as '%s' and local '%s' but no action is performed.", app.BintrayPackage, version, channel, remoteLatestVersion, localLatestVersion),
	}
	return
}

func (u *MainBinaryUpdater) SetRemoteEnabled(enabled bool) {
	u.RemoteEnabled = enabled
}

// Get local latest for version/channel
// Get remote latest for version/channel

// no remote no local — error
// if equals — go check local sha256sum
// remote errored or no version at all but local version is present — go to sha256 sum
// if no local but remote — skip local check, go to download

// LocalLatestBinaryInfo returns BinaryInfo for latest locally available version
//
// 1. find version dirs in ~/.multiwerf
// 2. find latest version for channel
// 3. check if binary exists in directory with version
//
// Note that this function doesn't verify a file hash
func LocalLatestBinaryInfo(version string, channel string, messages chan ActionMessage) (binInfo BinaryInfo, err error) {

	// create list of directories that names look like a semver
	// FIXME: handle warn message properly
	subDirs, _, err := FindSemverDirs(MultiwerfStorageDir)
	if err != nil {
		return
	}

	binInfo.AvailableVersions = subDirs

	latestVersion, err := ChooseLatestVersion(version, channel, subDirs, AvailableChannels)
	if err != nil {
		return
	}
	if latestVersion == "" {
		err = fmt.Errorf("No valid versions found for %s/%s in local storage %s", version, channel, app.StorageDir)
		return
	}

	exactBinInfo, err := GetLocalReleaseInfo(latestVersion, messages)
	if err != nil {
		return
	}

	exactBinInfo.AvailableVersions = subDirs

	return exactBinInfo, nil
}

func FindSemverDirs(path string) ([]string, string, error) {
	// create list of directories that names look like a semver
	subDirs := []string{}
	errMsgs := []string{}
	warn := ""
	warnMsgs := []string{}

	err := filepath.Walk(MultiwerfStorageDir, func(filePath string, fi os.FileInfo, err error) error {
		if err != nil {
			errMsgs = append(errMsgs, err.Error())
			return nil
		}

		if fi.IsDir() {
			// No action for initial directory
			if filePath == path {
				return nil
			}
			// Skip hidden directories inside initial directory
			if strings.HasPrefix(fi.Name(), ".") {
				return filepath.SkipDir
			}

			_, vErr := semver.NewVersion(fi.Name())
			if vErr != nil {
				warnMsgs = append(warnMsgs, fmt.Sprintf("%s: %v", fi.Name(), vErr.Error()))
				return filepath.SkipDir
			}
			subDirs = append(subDirs, fi.Name())
			// Do not go deeper
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return []string{}, "", err
	}
	if len(errMsgs) > 0 {
		err = errors.New(strings.Join(errMsgs, "\n"))
		return []string{}, "", err
	}
	if len(warnMsgs) > 0 {
		warn = fmt.Sprintf("warnMsgs: %+v\n", warnMsgs)
	}
	return subDirs, warn, nil
}

// GetVerifiedBinaryInfo return BinaryInfo object for binary with exact version if it is
// stored in MultiwerfStorageDir. Empty object is returned if no binary found.
// Hash of binary is verified with SHA256SUMS files.
func GetVerifiedBinaryInfo(version string, messages chan ActionMessage) (binInfo BinaryInfo, err error) {
	binInfo = BinaryInfo{}

	// Verify hash for found version
	dstPath := filepath.Join(MultiwerfStorageDir, version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is '%s', files: %+v", dstPath, files),
		debug: true}

	binInfo.Version = version
	binInfo.BinaryPath = filepath.Join(dstPath, files["program"])

	exist, err := IsReleaseFilesExist(dstPath, files)
	if err != nil {
		return
	}
	if !exist {
		return binInfo, fmt.Errorf("Release directory for %s is corrupted", version)
	}

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

// GetLocalReleaseInfo return BinaryInfo object for binary with exact version if it is
// stored in MultiwerfStorageDir. Empty object is returned if no binary found.
// Hash of binary is NOT verified with SHA256SUMS files.
func GetLocalReleaseInfo(version string, messages chan ActionMessage) (binInfo BinaryInfo, err error) {
	binInfo = BinaryInfo{}

	dstPath := filepath.Join(MultiwerfStorageDir, version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is '%s', files: %+v", dstPath, files),
		debug: true}

	exist, err := IsReleaseFilesExist(dstPath, files)

	if err != nil {
		return
	}

	if !exist {
		return binInfo, fmt.Errorf("Release directory for %s is corrupted", version)
	}

	binInfo.Version = version
	binInfo.BinaryPath = filepath.Join(dstPath, files["program"])
	// binInfo.HashVerified = true

	return
}

// RemoteLatestBinaryInfo searches for a latest available version in bintray
func RemoteLatestBinaryInfo(version string, channel string, messages chan ActionMessage, btClient bintray.BintrayClient) (binInfo BinaryInfo, err error) {
	binInfo = BinaryInfo{}

	pkgInfo, err := btClient.GetPackageInfo()
	if err != nil {
		err = fmt.Errorf("Get info for package '%s' error: %v", app.BintrayPackage, err)
		return
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		err = fmt.Errorf("No versions found in bintray for '%s'", app.BintrayPackage)
		return
	} else {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("Discover %d versions for channel %s/%s of package %s", len(versions), version, channel, app.BintrayPackage),
			debug: true}
	}

	binInfo.AvailableVersions = versions

	// Calc a latest version for version/channel
	latestVersion, err := ChooseLatestVersion(version, channel, versions, AvailableChannels)
	if err != nil {
		return
	}
	if latestVersion == "" {
		err = fmt.Errorf("No valid versions found for %s/%s channel of package %s/%s/%s", version, channel, app.BintraySubject, app.BintrayRepo, app.BintrayPackage)
		return
	}
	binInfo.Version = latestVersion
	return
}

// RemoteLatestChannelsReleases searches for a latest available version in bintray for each channel
func RemoteLatestChannelsReleases(version string, messages chan ActionMessage, btClient bintray.BintrayClient) (orderedReleases []string, releases map[string][]string, err error) {
	orderedReleases = make([]string, 0)
	releases = make(map[string][]string, 0)

	pkgInfo, err := btClient.GetPackageInfo()
	if err != nil {
		err = fmt.Errorf("Get info for package '%s' error: %v", app.BintrayPackage, err)
		return
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		err = fmt.Errorf("No versions found in bintray for '%s'", app.BintrayPackage)
		return
	} else {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("Discover %d versions for %s of package %s", len(versions), version, app.BintrayPackage),
			debug: true}
	}

	releaseForChannel := make(map[string]string)

	for _, channel := range AvailableChannels {
		// Calc a latest version for version/channel
		latestVersion, err := ChooseLatestVersion(version, channel, versions, AvailableChannels)
		if err != nil {
			messages <- ActionMessage{err: err}
		}
		releaseForChannel[channel] = latestVersion
	}

	for _, channel := range AvailableChannelsStableFirst {
		release := releaseForChannel[channel]
		if release == "" {
			continue
		}
		channelTag := fmt.Sprintf("%s %s", version, channel)
		if _, ok := releases[release]; !ok {
			releases[release] = []string{channelTag}
			orderedReleases = append(orderedReleases, release)
		} else {
			releases[release] = append(releases[release], channelTag)
		}
	}

	return
}

// DownloadRelease download all files for release and verify them.
// If files are good, then create version directory and move files there
func DownloadRelease(version string, messages chan ActionMessage, btClient bintray.BintrayClient) (binInfo BinaryInfo, err error) {
	rndStr := util.RndDigitsStr(5)
	tmpDir := filepath.Join(MultiwerfStorageDir, fmt.Sprintf("download-%s-%s", version, rndStr))
	dstPath := filepath.Join(MultiwerfStorageDir, version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)

	messages <- ActionMessage{msg: fmt.Sprintf("Start downloading release %s", version), debug: true}

	removeAll := func(prevErr error) (err error) {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			if prevErr != nil {
				return fmt.Errorf("%v, remove %s dir failed: %v", prevErr, tmpDir, err)
			} else {
				return fmt.Errorf("remove %s dir failed: %v", tmpDir, err)
			}
		}
		return prevErr
	}

	err = btClient.DownloadFiles(version, tmpDir, files)
	if err != nil {
		return binInfo, removeAll(err)
	}

	// chmod +x for files["program"]
	err = os.Chmod(filepath.Join(tmpDir, files["program"]), 0755)
	if err != nil {
		return binInfo, removeAll(fmt.Errorf("chmod 755 failed for %s: %v", files["program"], err))
	}

	// check hash of local binary
	match, err := VerifyReleaseFileHash(tmpDir, files["hash"], files["program"])
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("verifying release %s error: %v", version, err),
			debug: true}
		return binInfo, removeAll(err)
	}

	if match {
		err = os.Rename(tmpDir, dstPath)
		if err != nil {
			return binInfo, removeAll(fmt.Errorf("rename tmp dir failed: %v", err))
		} else {
			binInfo.BinaryPath = filepath.Join(dstPath, files["program"])
			binInfo.Version = version
			binInfo.HashVerified = true
			return binInfo, nil
		}
	}

	return
}
