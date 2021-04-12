package multiwerf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/werf/lockgate"

	"github.com/werf/multiwerf/pkg/app"
	"github.com/werf/multiwerf/pkg/locker"
	"github.com/werf/multiwerf/pkg/repo"
)

func UpdateChannelVersionBinary(messages chan ActionMessage, group string, channel string, tryRemoteChannelMapping bool) (binInfo *BinaryInfo) {
	messages <- ActionMessage{
		msg:   "Start UpdateChannelVersionBinary",
		debug: true,
	}

	channelMapping, err := GetChannelMapping(messages, tryRemoteChannelMapping)
	if err != nil {
		messages <- ActionMessage{err: err}
		return nil
	}

	actualChannelVersion, err := channelMapping.ChannelVersion(group, channel)
	if err != nil {
		messages <- ActionMessage{err: err}
		return nil
	}

	messages <- ActionMessage{
		msg:     fmt.Sprintf("The version %s is the actual for channel %s/%s", actualChannelVersion, group, channel),
		msgType: OkMsgType,
	}

	_ = lockgate.WithAcquire(locker.Locker, actualChannelVersion, lockgate.AcquireOptions{}, func(_ bool) error {
		localBinaryInfo, err := verifiedLocalBinaryInfo(messages, actualChannelVersion)
		if err != nil {
			messages <- ActionMessage{
				err: fmt.Errorf("the local version %s verification failed: %s", actualChannelVersion, err.Error()),
			}

			return nil
		} else if localBinaryInfo != nil {
			if !localBinaryInfo.HashVerified {
				messages <- ActionMessage{
					msg:     fmt.Sprintf("The local version %s has invalid or corrupted files and will be overrided", actualChannelVersion),
					msgType: WarnMsgType,
				}

				if err := os.RemoveAll(filepath.Dir(localBinaryInfo.BinaryPath)); err != nil {
					messages <- ActionMessage{
						err: fmt.Errorf("remove directory %s failed: %s", filepath.Dir(localBinaryInfo.BinaryPath), err),
					}

					return nil
				}
			} else {
				messages <- ActionMessage{
					msg:     "The actual version is available locally",
					msgType: OkMsgType,
				}

				if err := channelMapping.Save(); err != nil {
					messages <- ActionMessage{err: fmt.Errorf("save channel mapping failed: %s", err)}
					return nil
				}

				binInfo = localBinaryInfo
				return nil
			}
		}

		downloadedBinaryInfo, err := downloadAndVerifyReleaseFiles(messages, actualChannelVersion)
		if err != nil {
			messages <- ActionMessage{err: fmt.Errorf("%s %s/%s: %v", app.AppPackageName, group, channel, err)}
			return nil
		}

		if err := channelMapping.Save(); err != nil {
			messages <- ActionMessage{err: fmt.Errorf("save channel mapping failed: %s", err)}
			return nil
		}

		messages <- ActionMessage{
			msg:     "The actual version has been successfully downloaded",
			msgType: OkMsgType,
		}

		binInfo = downloadedBinaryInfo

		return nil
	})

	return binInfo
}

func UseChannelVersionBinary(messages chan ActionMessage, group string, channel string) (binInfo *BinaryInfo) {
	messages <- ActionMessage{
		msg:   "Starting UseChannelVersionBinary",
		debug: true,
	}

	for _, envName := range []string{
		fmt.Sprintf("MULTIWERF_WERF_PATH_%s_%s_FORCE", strings.ReplaceAll(group, ".", "_"), strings.ToUpper(channel)),
		"MULTIWERF_WERF_PATH_FORCE",
	} {
		binaryPath := os.Getenv(envName)
		if binaryPath != "" {
			messages <- ActionMessage{
				msg:   fmt.Sprintf("Force binary path %s is used", binaryPath),
				debug: true,
			}

			return &BinaryInfo{
				BinaryPath: binaryPath,
			}
		}
	}

	channelMapping, err := GetChannelMapping(messages, false)
	if err != nil {
		messages <- ActionMessage{err: err}
		return nil
	}

	actualChannelVersion, err := channelMapping.ChannelVersion(group, channel)
	if err != nil {
		messages <- ActionMessage{err: err}
		return nil
	}

	messages <- ActionMessage{
		msg:     fmt.Sprintf("The version %s is the actual for channel %s/%s", actualChannelVersion, group, channel),
		msgType: OkMsgType,
	}

	localBinaryInfo, err := localBinaryInfo(messages, actualChannelVersion)
	if err != nil {
		messages <- ActionMessage{err: fmt.Errorf("the local version %s getting failed: %s", actualChannelVersion, err.Error())}
		return nil
	} else if localBinaryInfo != nil {
		messages <- ActionMessage{
			debug: true,
			msg:   "The actual version is available locally",
		}

		return localBinaryInfo
	}

	messages <- ActionMessage{
		err: fmt.Errorf("the actual channel version has not been found locally\nRun command `multiwerf update %s %s`", group, channel),
	}

	return nil
}

// downloadAndVerifyReleaseFiles downloads release files and verifies them.
// If files are good then creates version directory and moves files there
func downloadAndVerifyReleaseFiles(messages chan ActionMessage, version string) (binInfo *BinaryInfo, err error) {
	tmpDir, err := ioutil.TempDir(TmpDir, version+"-")
	if err != nil {
		return nil, fmt.Errorf("create tmp dir failed: %s", err)
	}

	dstPath := localVersionDirPath(version)
	files := ReleaseFiles(app.AppPackageName, version, app.OsArch)

	messages <- ActionMessage{
		msg:     fmt.Sprintf("Downloading the version %s ...", version),
		msgType: OkMsgType,
	}

	shouldBeRemoved := true
	defer func() {
		if shouldBeRemoved {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	repoClients := []repo.Repo{
		NewAppS3Client(),
		NewAppBtClient(),
	}

	for ind, repoClient := range repoClients {
		shouldSkipError := len(repoClients) > ind+1

		err = repoClient.DownloadFiles(version, tmpDir, files)
		if err != nil {
			if shouldSkipError {
				messages <- ActionMessage{
					msg:     fmt.Sprintf("[%s] Downloading the version %s failed", repoClient.String(), version),
					msgType: WarnMsgType,
					stage:   "update",
				}

				continue
			}

			return nil, err
		}

		break
	}

	if err = os.Chmod(filepath.Join(tmpDir, files["program"]), 0755); err != nil {
		return nil, fmt.Errorf("chmod 755 failed for %s: %v", files["program"], err)
	}

	// check hash of local binary
	match, err := VerifyReleaseFileHash(messages, tmpDir, files["hash"], files["program"])
	if err != nil {
		messages <- ActionMessage{
			msg:   fmt.Sprintf("verifying release %s error: %v", version, err),
			debug: true,
		}

		return nil, err
	}

	if match {
		if err = os.Rename(tmpDir, dstPath); err != nil {
			return nil, err
		}

		shouldBeRemoved = false

		binInfo = &BinaryInfo{}
		binInfo.BinaryPath = filepath.Join(dstPath, files["program"])
		binInfo.Version = version
		binInfo.HashVerified = true

		return binInfo, nil
	}

	return
}
