package multiwerf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/flant/multiwerf/pkg/app"
)

const DefaultLocalChannelMappingFilename = "multiwerf.json"

type LocalChannelMappingNotFoundError struct {
	error
}

type ChannelMapping interface {
	ChannelVersion(group, channel string) (string, error)
	Save() error
}

type ChannelMappingBase struct {
	Multiwerf []struct {
		Group    string `json:"group"`
		Channels []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"channels"`
	} `json:"multiwerf"`
}

func (c *ChannelMappingBase) ChannelVersion(group, channel string) (string, error) {
	for _, g := range c.Multiwerf {
		if g.Group == group {
			for _, c := range g.Channels {
				if c.Name == channel {
					return c.Version, nil
				}
			}
		}
	}

	return "", fmt.Errorf("the version for %s/%s is not found", group, channel)
}

func (c *ChannelMappingBase) AllVersions() []string {
	var versions []string

	for _, g := range c.Multiwerf {
		for _, c := range g.Channels {
			versions = append(versions, c.Version)
		}
	}

	return versions
}

func (c *ChannelMappingBase) Save() error {
	return nil
}

func (c *ChannelMappingBase) Marshal() ([]byte, error) {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return nil, err
	}

	data = append(data, []byte("\n")...)

	return data, nil
}

type ChannelMappingLocal struct {
	ChannelMappingBase
}

type ChannelMappingRemote struct {
	ChannelMappingBase
}

func (c *ChannelMappingRemote) Save() error {
	newData, err := c.Marshal()

	// compare new channel mapping and current
	currentExist, err := FileExists(localChannelMappingPath())
	if err != nil {
		return fmt.Errorf("file exists failed %s: %s", localChannelMappingPath(), err)
	}

	if currentExist {
		currentData, err := ioutil.ReadFile(localChannelMappingPath())
		if err != nil {
			return fmt.Errorf("read file failed %s: %s", localChannelMappingPath(), err)
		}

		if bytes.Equal(currentData, newData) {
			return nil
		}

		// compare new channel mapping and old
		var oldData []byte
		oldExist, err := FileExists(localOldChannelMappingPath())
		if oldExist {
			oldData, err = ioutil.ReadFile(localOldChannelMappingPath())
			if err != nil {
				return fmt.Errorf("read file failed %s: %s", localOldChannelMappingPath(), err)
			}
		}

		if !oldExist || !bytes.Equal(oldData, currentData) {
			if err := ioutil.WriteFile(localOldChannelMappingPath(), currentData, os.ModePerm); err != nil {
				return fmt.Errorf("write file failed %s: %s", localOldChannelMappingPath(), err)
			}
		}
	}

	tmpFile, err := ioutil.TempFile(TmpDir, "channel_mapping")
	if err != nil {
		return fmt.Errorf("create tmp file failed: %s", err)
	}

	shouldBeDeleted := true
	defer func() {
		if shouldBeDeleted {
			_ = os.Remove(tmpFile.Name())
		}
	}()

	if _, err := tmpFile.Write(newData); err != nil {
		return fmt.Errorf("write to tmp file failed %s: %s", tmpFile.Name(), err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close tmp file failed %s: %s", tmpFile.Name(), err)
	}

	if err := os.Rename(tmpFile.Name(), localChannelMappingPath()); err != nil {
		return err
	}

	shouldBeDeleted = false

	return nil
}

func newRemoteChannelMapping(channelMappingUrl string) (*ChannelMappingRemote, error) {
	resp, err := http.Get(channelMappingUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response status code %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("respBody read failed: %s", err)
	}

	channelMapping := &ChannelMappingRemote{}
	if err := json.Unmarshal(data, channelMapping); err != nil {
		return nil, fmt.Errorf("unmarshal json failed: %s", err)
	}

	return channelMapping, nil
}

func newLocalChannelMapping(channelMappingPath string) (*ChannelMappingLocal, error) {
	if exist, err := FileExists(channelMappingPath); err != nil {
		return nil, fmt.Errorf("file exists failed: %s", err)
	} else if !exist {
		return nil, LocalChannelMappingNotFoundError{error: fmt.Errorf("file %s is not found", channelMappingPath)}
	}

	data, err := ioutil.ReadFile(channelMappingPath)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %s", err)
	}

	channelMapping := &ChannelMappingLocal{}
	if err := json.Unmarshal(data, channelMapping); err != nil {
		return nil, fmt.Errorf("unmarshal json failed: %s", err)
	}

	return channelMapping, err
}

func defaultLocalChannelMappingFilePath() string {
	return filepath.Join(StorageDir, DefaultLocalChannelMappingFilename)
}

func isLocalChannelMappingFileExist() (bool, error) {
	localChannelMappingPath := defaultLocalChannelMappingFilePath()
	if app.ChannelMappingPath != "" {
		localChannelMappingPath = app.ChannelMappingPath
	}

	return FileExists(localChannelMappingPath)
}

func GetChannelMapping(messages chan ActionMessage, tryRemoteChannelMapping bool) (ChannelMapping, error) {
	if tryRemoteChannelMapping {
		channelMapping, err := newRemoteChannelMapping(app.ChannelMappingUrl)
		if err != nil {
			messages <- ActionMessage{
				msg:     fmt.Sprintf("Get remote channel mapping from %s failed: %s", app.ChannelMappingUrl, err),
				msgType: WarnMsgType,
			}
		}

		if channelMapping != nil {
			return channelMapping, nil
		}
	}

	if tryRemoteChannelMapping {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("Trying to get the local channel mapping %s ...", localChannelMappingPath()),
			msgType: WarnMsgType,
		}
	}

	channelMapping, err := newLocalChannelMapping(localChannelMappingPath())
	if err != nil {
		return nil, fmt.Errorf("get the local channel mapping failed: %s\nRun command `multiwerf update` to download the actual one", err)
	}

	return channelMapping, nil
}

func localChannelMappingPath() string {
	if app.ChannelMappingPath != "" {
		return app.ChannelMappingPath
	}

	return defaultLocalChannelMappingFilePath()
}

func localOldChannelMappingPath() string {
	return localChannelMappingPath() + ".old"
}
