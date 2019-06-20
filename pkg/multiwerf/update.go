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
	BinaryPath        string
	Version           string
	HashVerified      bool
	AvailableVersions []string
}

type BinaryUpdater interface {
	// this method is intended to update a binary from remote source
	// check remote latest → get local latest → if local need update → download and verify a hash
	// ↑no remote — error/exit
	//                       ↑no local — no error
	//                                                                 ↑ error if download failed or hash not verified
	DownloadLatest(version string, channel string) (binInfo BinaryInfo)

	// this method return BinaryInfo instance
	// multiwerf exit with error if no binary found remote or local
	// check remote versions — check local version — if local need update — download and verify a hash
	GetLatestBinaryInfo(version string, channel string) (binInfo BinaryInfo)

	//
	SetRemoteEnabled(enabled bool)
	SetRemoteDelayed(delayed bool)
}

type MainBinaryUpdater struct {
	BintrayClient bintray.BintrayClient
	Messages      chan ActionMessage
	RemoteEnabled bool
	RemoteDelayed bool
}

func NewBinaryUpdater(messages chan ActionMessage) BinaryUpdater {
	result := &MainBinaryUpdater{
		RemoteEnabled: false,
		RemoteDelayed: false,
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

	binInfo, _ = GetBinaryInfo(latestVersion, u.Messages)
	if binInfo.HashVerified {
		u.Messages <- ActionMessage{
			comment: "no update needed",
			msg:     fmt.Sprintf("%s %s/%s stays at %s", app.BintrayPackage, version, channel, latestVersion),
			msgType: "ok"}
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
		msgType: "ok"}

	return binInfo
}

// GetLatestBinaryInfo return BinaryInfo object for latest binary with version/channel.
// Checks for local versions and remote versions. If no remote version is available — use
// local version.
func (u *MainBinaryUpdater) GetLatestBinaryInfo(version string, channel string) (binInfo BinaryInfo) {
	remoteBinInfo := BinaryInfo{}
	remoteLatestVersion := ""
	if u.RemoteEnabled && !u.RemoteDelayed {
		var err error
		remoteBinInfo, err = RemoteLatestBinaryInfo(version, channel, u.Messages, u.BintrayClient)
		if err != nil {
			u.Messages <- ActionMessage{
				msg:     err.Error(),
				msgType: "warn"}
		}
		remoteLatestVersion = remoteBinInfo.Version
	}

	var llbiErr error
	localLatestVersion := ""
	localBinaryInfo, err := LocalLatestBinaryInfo(version, channel, u.Messages)
	if err != nil {
		llbiErr = err
		u.Messages <- ActionMessage{
			msg:     err.Error(),
			msgType: "warn"}
	} else {
		localLatestVersion = localBinaryInfo.Version
	}

	// no remote, no local — exit with error
	if remoteLatestVersion == "" && localLatestVersion == "" {
		u.Messages <- ActionMessage{
			msg:     fmt.Sprintf("Cannot determine latest version for %s/%s neither from bintray package '%s' nor from local storage %s", version, channel, app.BintrayPackage, app.StorageDir),
			msgType: "fail",
		}
		if !u.RemoteEnabled || u.RemoteDelayed {
			u.Messages <- ActionMessage{
				msg:     fmt.Sprintf("Auto update of `%s` is disabled or delayed. Try `multiwerf update %s %s` command.", app.BintrayPackage, version, channel),
				msgType: "warn",
			}
		}
		// Show top 5 versions from remote or from local if remote is disabled
		if len(remoteBinInfo.AvailableVersions) > 0 {
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
			latestAvailable := PickLatestVersions(version, localBinaryInfo.AvailableVersions, 5)
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

	// has local and no remote or local is equal to remote — no update needed, stay at local version
	if localLatestVersion != "" {
		if (remoteLatestVersion == "" || localLatestVersion == remoteLatestVersion) && localBinaryInfo.HashVerified {
			if localBinaryInfo.BinaryPath == "" {
				u.Messages <- ActionMessage{
					err: fmt.Errorf("BUG: empty path. Please, report: rlv=[%v] llv=[%v] v=[%v] av=[%+v] hv=[%v] ver=[%v] ch=[%v] llbierr=[%v]",
						remoteLatestVersion,
						localLatestVersion,
						localBinaryInfo.Version,
						localBinaryInfo.AvailableVersions,
						localBinaryInfo.HashVerified,
						version,
						channel,
						llbiErr)}

			}
			u.Messages <- ActionMessage{
				comment: "no update needed",
				msg:     fmt.Sprintf("%s %s/%s stays at %s", app.BintrayPackage, version, channel, localLatestVersion),
				msgType: "ok"}
			return localBinaryInfo
		}
		if remoteLatestVersion == "" && !localBinaryInfo.HashVerified {
			u.Messages <- ActionMessage{
				err: fmt.Errorf("Cannot determine latest version from bintray package '%s' and local latest binary '%s' is corrupted", app.BintrayPackage, localLatestVersion),
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
			msgType: "ok"}
		return newBinInfo
	}

	// It's a bug if reached!
	u.Messages <- ActionMessage{
		err: fmt.Errorf("BUG: %s %s/%s detect remote version as '%s' and local '%s' but no action is performed.", app.BintrayPackage, version, channel, remoteLatestVersion, localLatestVersion),
	}
	return
}

func (u *MainBinaryUpdater) SetRemoteEnabled(enabled bool) {
	u.RemoteEnabled = enabled
}

func (u *MainBinaryUpdater) SetRemoteDelayed(delayed bool) {
	u.RemoteDelayed = delayed
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
// 2. find latest version for channel and verify a hash for that binary
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

	exactBinInfo, err := GetBinaryInfo(latestVersion, messages)
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

// RemoteLatestBinaryInfo searches for a latest available version in bintray
func RemoteLatestBinaryInfo(version string, channel string, messages chan ActionMessage, btClient bintray.BintrayClient) (binInfo BinaryInfo, err error) {
	binInfo = BinaryInfo{}

	pkgInfo, err := btClient.GetPackage()
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

	pkgInfo, err := btClient.GetPackage()
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

func DownloadVersion(version string, messages chan ActionMessage, btClient bintray.BintrayClient) (err error) {
	// Verify hash for found version
	dstPath := filepath.Join(MultiwerfStorageDir, version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)

	messages <- ActionMessage{msg: "Start downloading", debug: true}

	err = btClient.DownloadFiles(version, dstPath, files)
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
