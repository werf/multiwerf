package multiwerf

import (
	//"fmt"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/bintray"
)

type AvailableReleasesInformer interface {
	// GetReleases retrieves information about available releases.
	GetMajorMinorReleases() ([]string, error)
	// GetllChannelsReleases retrieves infrmation about all channels for MAJOR.MINOR version
	GetAllChannelsReleases(version string) (AllChannelsReleasesInfo, error)
	// GetRelease returns exact versoin for MAJOR.MINOR and channel as use or update command will do
	GetRelease(version string, channel string) (string, error)
}

type AllChannelsReleasesInfo struct {
	Channels []string          `json:"channels"`
	Releases map[string]string `json:"releases"`
}

type MainAvailableReleasesInformer struct {
	BintrayClient bintray.BintrayClient
	Messages      chan ActionMessage
}

func NewAvailableReleasesInformer(messages chan ActionMessage) AvailableReleasesInformer {
	result := &MainAvailableReleasesInformer{}
	result.BintrayClient = bintray.NewBintrayClient(app.BintraySubject, app.BintrayRepo, app.BintrayPackage)
	result.Messages = messages
	return result
}

// TODO
func (m *MainAvailableReleasesInformer) GetMajorMinorReleases() ([]string, error) {
	return []string{}, nil
}

// TODO
func (m *MainAvailableReleasesInformer) GetAllChannelsReleases(version string) (info AllChannelsReleasesInfo, err error) {
	releases, err := RemoteLatestChannelsReleases(version, m.Messages, m.BintrayClient)
	if err != nil {
		return
	}
	return AllChannelsReleasesInfo{Channels: AvailableChannels, Releases: releases}, nil
}

func (m *MainAvailableReleasesInformer) GetRelease(version string, channel string) (string, error) {
	remoteBinInfo, err := RemoteLatestBinaryInfo(version, channel, m.Messages, m.BintrayClient)
	if err != nil {
		return "", err
	}
	return remoteBinInfo.Version, nil
}
