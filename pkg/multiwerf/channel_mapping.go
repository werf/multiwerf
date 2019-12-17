package multiwerf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

const LocalChannelMappingFilename = "multiwerf.json"

type ChannelMapping interface {
	GetChannelVersion(group, channel string) (string, error)
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

func (c *ChannelMappingBase) GetChannelVersion(group, channel string) (string, error) {
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

	if err := ioutil.WriteFile(localChannelMappingFilePath(), data, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func newRemoteChannelMapping(channelMappingUrl string) (*ChannelMappingRemote, error) {
	resp, err := http.Get(channelMappingUrl)
	if err != nil {
		return nil, fmt.Errorf("httpGet url failed: %s", err)
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
	if exist, err := FileExists(filepath.Dir(channelMappingPath), filepath.Base(channelMappingPath)); err != nil {
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

	return channelMapping, nil
}

func localChannelMappingFilePath() string {
	return filepath.Join(MultiwerfStorageDir, LocalChannelMappingFilename)
}

func isLocalChannelMappingExist() bool {
	exist, err := FileExists(filepath.Dir(localChannelMappingFilePath()), filepath.Base(localChannelMappingFilePath()))
	if err != nil {
		panic(fmt.Errorf("file exists failed: %s", localChannelMappingFilePath()))
	}

	return exist
}
