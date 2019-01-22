package bintray

import (
	"fmt"
	"encoding/json"

	"github.com/flant/multiwerf/pkg/http"
)

const btApiUrl = "https://api.bintray.com"
const btDlUrl = "https://dl.bintray.com"


type BintrayEvent struct {
	Msg string
	Err error
	State string
}

type BintrayClient interface {
	GetPackage() (string, error)
	DownloadRelease(version string, dstDir string, files map[string]string) error
	EventCh() chan BintrayEvent
}

type MainBintrayClient struct {
	Subject string
	Repo string
	Package string
	eventCh chan BintrayEvent
}

func NewBintrayClient(subject string, repo string, pkg string) (bc BintrayClient) {
	bc = &MainBintrayClient{
		Subject: subject,
		Repo: repo,
		Package: pkg,
		eventCh: make(chan BintrayEvent, 1),
	}
	return bc
}

// GetPackage returns json response from packages API
func (bc *MainBintrayClient) GetPackage() (string, error) {
	apiUrl := fmt.Sprintf("%s/packages/%s/%s/%s", btApiUrl, bc.Subject, bc.Repo, bc.Package)
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

func (bc *MainBintrayClient) DownloadRelease(version string, dstDir string, files map[string]string) error {
	srcUrl := fmt.Sprintf("%s/%s/%s/%s", btDlUrl, bc.Subject, bc.Repo, version)

	for fileType, fileName := range files {
		// TODO implement goreleaser lifecycle and verify gpg signing
		if fileType == "sig" {
			continue
		}
		fileUrl := fmt.Sprintf("%s/%s", srcUrl, fileName)
		err := http.DownloadLargeFile(fileUrl, dstDir, fileName)
		if err != nil {
			return fmt.Errorf("GET %s error: %v", fileUrl, err)
		}
	}

	return nil
}

func (bc *MainBintrayClient) EventCh() chan BintrayEvent {
	return bc.eventCh
}
