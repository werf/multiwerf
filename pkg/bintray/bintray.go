package bintray

import (
	"encoding/json"
	"fmt"

	"github.com/flant/multiwerf/pkg/http"
)

const DefaultBintrayApiUrl = "https://api.bintray.com"
const DefaultBintrayDlUrl = "https://dl.bintray.com"

var (
	BintrayApiUrl string
	BintrayDlUrl  string
)

type BintrayEvent struct {
	Msg   string
	Err   error
	State string
}

type BintrayClient interface {
	GetPackageInfo() (string, error)
	DownloadFiles(version string, dstDir string, files map[string]string) error
	GetFileContent(version string, fileName string) (string, error)
	EventCh() chan BintrayEvent
}

type MainBintrayClient struct {
	Subject string
	Repo    string
	Package string
	eventCh chan BintrayEvent
}

func NewBintrayClient(subject string, repo string, pkg string) (bc BintrayClient) {
	if BintrayApiUrl == "" {
		BintrayApiUrl = DefaultBintrayApiUrl
	}
	if BintrayDlUrl == "" {
		BintrayDlUrl = DefaultBintrayDlUrl
	}
	bc = &MainBintrayClient{
		Subject: subject,
		Repo:    repo,
		Package: pkg,
		eventCh: make(chan BintrayEvent, 1),
	}
	return bc
}

// GetPackageInfo returns json response from packages API
func (bc *MainBintrayClient) GetPackageInfo() (string, error) {
	apiUrl := fmt.Sprintf("%s/packages/%s/%s/%s", BintrayApiUrl, bc.Subject, bc.Repo, bc.Package)
	response, err := http.MakeRestAPICall("GET", apiUrl)
	if err != nil {
		return "", err
	}
	return response, nil
}

// GetPackageVersions returns versions field from package info json
func GetPackageVersions(packageInfo string) (versions []string) {
	res := map[string]interface{}{}
	json.Unmarshal([]byte(packageInfo), &res)

	vs, ok := res["versions"].([]interface{})
	if !ok {
		return
	}

	for _, v := range vs {
		versions = append(versions, fmt.Sprintf("%v", v))
	}

	return versions
}

func (bc *MainBintrayClient) DownloadFiles(version string, dstDir string, files map[string]string) error {
	srcUrl := fmt.Sprintf("%s/%s/%s/%s", BintrayDlUrl, bc.Subject, bc.Repo, version)

	for fileType, fileName := range files {
		// TODO implement goreleaser lifecycle and verify gpg signing
		if fileType == "sig" {
			continue
		}
		fileUrl := fmt.Sprintf("%s/%s", srcUrl, fileName)
		err := http.DownloadLargeFile(fileUrl, dstDir, fileName)
		if err != nil {
			return fmt.Errorf("%s download error: %v", fileUrl, err)
		}
	}

	return nil
}

func (bc *MainBintrayClient) GetFileContent(version string, fileName string) (string, error) {
	srcUrl := fmt.Sprintf("%s/%s/%s/%s", BintrayDlUrl, bc.Subject, bc.Repo, version)
	fileUrl := fmt.Sprintf("%s/%s", srcUrl, fileName)
	return http.MakeRestAPICall("GET", fileUrl)
}

func (bc *MainBintrayClient) EventCh() chan BintrayEvent {
	return bc.eventCh
}
