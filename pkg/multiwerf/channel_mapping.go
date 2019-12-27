package multiwerf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/flant/multiwerf/pkg/app"
)

const DefaultLocalChannelMappingFilename = "multiwerf.json"

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

func (c *ChannelMappingBase) Save() error {
	return nil
}

type ChannelMappingLocal struct {
	ChannelMappingBase
}

type ChannelMappingRemote struct {
	ChannelMappingBase
}

func (c *ChannelMappingRemote) Save() error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	data = append(data, []byte("\n")...)

	localChannelMappingFilePath := defaultLocalChannelMappingFilePath()
	if exist, err := FileExists(localChannelMappingFilePath); err != nil {
		return fmt.Errorf("file exists failed %s: %s", localChannelMappingFilePath, err)
	} else if exist {
		currentData, err := ioutil.ReadFile(localChannelMappingFilePath)
		if err != nil {
			return fmt.Errorf("read file failed %s: %s", localChannelMappingFilePath, err)
		}

		if bytes.Equal(currentData, data) {
			return nil
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

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("write to tmp file failed %s: %s", tmpFile.Name(), err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close tmp file failed %s: %s", tmpFile.Name(), err)
	}

	if err := os.Rename(tmpFile.Name(), localChannelMappingFilePath); err != nil {
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
		return nil, fmt.Errorf("file %s is not found", channelMappingPath)
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

func isLocalChannelMappingFilePathExist() (bool, error) {
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

	localChannelMappingPath := defaultLocalChannelMappingFilePath()
	if app.ChannelMappingPath != "" {
		localChannelMappingPath = app.ChannelMappingPath
	}

	if tryRemoteChannelMapping {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("Trying to get the local channel mapping %s ...", localChannelMappingPath),
			msgType: WarnMsgType,
		}
	}

	channelMapping, err := newLocalChannelMapping(localChannelMappingPath)
	if err != nil {
		return nil, fmt.Errorf("get the local channel mapping failed: %s\nRun command `multiwerf update %s` to download the actual one", err, strings.Join(os.Args[2:], " "))
	}

	return channelMapping, nil
}
