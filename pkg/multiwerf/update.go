package multiwerf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/bintray"
	"github.com/flant/multiwerf/pkg/util"
)

type BinaryInfo struct {
	BinaryPath   string
	Version      string
	HashVerified bool
}

type BinaryUpdater interface {
	// UpdateChannelVersion downloads the werf binary according to the remote or local channel mapping
	//
	// Get the actual werf version for group/channel from the mapping
	// Check version locally and remotely
	// * if local version exists and it is valid then use it
	// * otherwise, download and verify the actual binary from bintray
	// Override local channel mapping
	UpdateChannelVersion(group, channel string) (binInfo *BinaryInfo)
}

type MainBinaryUpdater struct {
	BintrayClient  bintray.BintrayClient
	Messages       chan ActionMessage
	RemoteEnabled  bool
	ChannelMapping ChannelMapping
}

func NewBinaryUpdater(messages chan ActionMessage, remoteEnabled bool) BinaryUpdater {
	result := &MainBinaryUpdater{}
	result.BintrayClient = bintray.NewBintrayClient(app.BintraySubject, app.BintrayRepo, app.BintrayPackage)
	result.RemoteEnabled = remoteEnabled
	result.Messages = messages
	result.InitChannelMapping()

	return result
}

func (u *MainBinaryUpdater) InitChannelMapping() {
	if u.RemoteEnabled {
		channelMapping, err := newRemoteChannelMapping(app.ChannelMappingUrl)
		if err != nil {
			u.Messages <- ActionMessage{
				msg:     fmt.Sprintf("Getting remote channel mapping from %s failed: %s", app.ChannelMappingUrl, err),
				msgType: "warn",
			}
		}

		if channelMapping != nil {
			u.ChannelMapping = channelMapping
			return
		}
	}

	localChannelMappingPath := filepath.Join(MultiwerfStorageDir, LocalChannelMappingFilename)
	if app.ChannelMappingPath != "" {
		localChannelMappingPath = app.ChannelMappingPath
	}

	if u.RemoteEnabled {
		u.Messages <- ActionMessage{
			msg:     fmt.Sprintf("Trying to get local channel mapping from %s ...", localChannelMappingPath),
			msgType: "warn",
		}
	}

	channelMapping, err := newLocalChannelMapping(localChannelMappingPath)
	if err != nil {
		u.Messages <- ActionMessage{err: fmt.Errorf("getting the local channel mapping failed: %s\nRun command `multiwerf update %s` to download the actual one", err, strings.Join(os.Args[2:], " "))}
		return
	}

	u.ChannelMapping = channelMapping
}

func (u *MainBinaryUpdater) UpdateChannelVersion(group string, channel string) (binInfo *BinaryInfo) {
	u.Messages <- ActionMessage{
		msg:   "Start UpdateChannelVersion",
		debug: true,
	}

	actualChannelVersion, err := u.ChannelMapping.GetChannelVersion(group, channel)
	if err != nil {
		u.Messages <- ActionMessage{err: err}
		return nil
	}

	u.Messages <- ActionMessage{
		msg:     fmt.Sprintf("The version %s is the actual for channel %s/%s", actualChannelVersion, group, channel),
		msgType: "ok",
	}

	strictVerification := u.RemoteEnabled
	verifiedBinaryInfo, err := verifiedLocalBinaryInfo(actualChannelVersion, strictVerification, u.Messages)
	if err != nil {
		u.Messages <- ActionMessage{err: fmt.Errorf("the local version %s verification failed: %s", actualChannelVersion, err.Error())}
		return nil
	} else if verifiedBinaryInfo != nil {
		if !verifiedBinaryInfo.HashVerified {
			u.Messages <- ActionMessage{
				msg:     fmt.Sprintf("The local version %s has invalid or corrupted files and will be overrided", actualChannelVersion),
				msgType: "warn",
			}

			if err := os.RemoveAll(filepath.Dir(verifiedBinaryInfo.BinaryPath)); err != nil {
				u.Messages <- ActionMessage{
					err: fmt.Errorf("remove directory %s failed: %s", filepath.Dir(verifiedBinaryInfo.BinaryPath), err),
				}
				return nil
			}
		} else {
			u.Messages <- ActionMessage{
				msg:     "The actual version is available locally",
				msgType: "ok",
			}

			if !isLocalChannelMappingExist() {
				if err := u.ChannelMapping.Save(); err != nil {
					u.Messages <- ActionMessage{err: fmt.Errorf("save channel mapping failed: %s", err)}
					return nil
				}
			}

			return verifiedBinaryInfo
		}
	}

	if u.RemoteEnabled {
		// Download a actualChannelVersion if local binary does not exist or hash is not verified
		binInfo, err = downloadAndVerifyReleaseFiles(actualChannelVersion, u.Messages, u.BintrayClient)
		if err != nil {
			u.Messages <- ActionMessage{err: fmt.Errorf("%s %s/%s: %v", app.BintrayPackage, group, channel, err)}
			return nil
		}

		if err := u.ChannelMapping.Save(); err != nil {
			u.Messages <- ActionMessage{err: fmt.Errorf("save channel mapping failed: %s", err)}
			return nil
		}

		u.Messages <- ActionMessage{
			comment: fmt.Sprintf("# update %s success", app.BintrayPackage),
			msg:     "The actual version has been successfully downloaded",
			msgType: "ok",
		}

		return binInfo
	}

	u.Messages <- ActionMessage{
		err: fmt.Errorf("The actual version has not been found locally\nThe auto-update is disabled or delayed\nRun command `multiwerf update %s`", strings.Join(os.Args[2:], " ")),
	}

	return nil
}

// verifiedLocalBinaryInfo returns BinaryInfo object for the version if it is
// stored in MultiwerfStorageDir. Empty object is returned if no binary found.
// Hash of binary is verified with SHA256SUMS files.
func verifiedLocalBinaryInfo(version string, strict bool, messages chan ActionMessage) (*BinaryInfo, error) {
	var binInfo *BinaryInfo

	// Verify hash for version
	dstPath := filepath.Join(MultiwerfStorageDir, version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is %s, files: %+v", dstPath, files),
		debug: true,
	}

	if exist, err := DirExists(dstPath); err != nil {
		return binInfo, err
	} else if !exist {
		return nil, nil
	}

	binInfo = &BinaryInfo{}
	binInfo.Version = version
	binInfo.BinaryPath = filepath.Join(dstPath, files["program"])
	binInfo.HashVerified = false

	exist, err := IsReleaseFilesExist(dstPath, files)
	if err != nil {
		return nil, err
	} else if !exist {
		return binInfo, nil
	}

	var match bool
	if strict {
		// check hash of local binary
		match, err = VerifyReleaseFileHash(dstPath, files["hash"], files["program"])
		if err != nil {
			return nil, err
		}
	} else {
		match, err = FileExists(dstPath, files["program"])
		if err != nil {
			return nil, err
		}
	}

	binInfo.HashVerified = match

	return binInfo, nil
}

// downloadAndVerifyReleaseFiles downloads all files for the version release and verify them.
// If files are good then creates version directory and moves files there
func downloadAndVerifyReleaseFiles(version string, messages chan ActionMessage, btClient bintray.BintrayClient) (binInfo *BinaryInfo, err error) {
	rndStr := util.RndDigitsStr(5)
	tmpDir := filepath.Join(MultiwerfStorageDir, fmt.Sprintf("download-%s-%s", version, rndStr))
	dstPath := filepath.Join(MultiwerfStorageDir, version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)

	messages <- ActionMessage{
		msg:     fmt.Sprintf("Start downloading version %s ...", version),
		msgType: "ok",
	}

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
		return nil, removeAll(err)
	}

	// chmod +x for files["program"]
	err = os.Chmod(filepath.Join(tmpDir, files["program"]), 0755)
	if err != nil {
		return nil, removeAll(fmt.Errorf("chmod 755 failed for %s: %v", files["program"], err))
	}

	// check hash of local binary
	match, err := VerifyReleaseFileHash(tmpDir, files["hash"], files["program"])
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("verifying release %s error: %v", version, err),
			debug: true,
		}

		return nil, removeAll(err)
	}

	if match {
		err = os.Rename(tmpDir, dstPath)
		if err != nil {
			return binInfo, removeAll(fmt.Errorf("rename tmp dir failed: %v", err))
		}

		binInfo = &BinaryInfo{}
		binInfo.BinaryPath = filepath.Join(dstPath, files["program"])
		binInfo.Version = version
		binInfo.HashVerified = true

		return binInfo, nil
	}

	return
}
