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
	// stub
//	return `
//{"name":"dappfile-yml","repo":"dapp","owner":"flant","desc":"","labels":[],"attribute_names":[],"licenses":["Apache-2.0"],"custom_licenses":[],"followers_count":0,"created":"2018-01-31T13:11:47.755Z","website_url":"","issue_tracker_url":"","linked_to_repos":[],"permissions":[],"versions":["0.31.24","0.32.6","0.31.23","0.32.5","0.32.4","0.31.22","0.32.3","0.31.21","0.31.20","0.32.2","0.32.1","0.32.0","0.31.19","0.31.18","0.31.17","0.31.16","0.31.15","0.31.14","0.31.13","0.31.12","0.31.11","0.31.10","0.31.9","0.31.8","0.31.7","0.31.6","0.31.5","0.31.4","0.31.3","0.30.7","0.31.2","0.30.6","0.31.1","0.31.0","0.30.5","0.30.4","0.30.3","0.30.2","0.30.1","0.30.0","0.29.0","0.27.23","0.28.12","0.28.11","0.27.22","0.27.21","0.28.10","0.28.9","0.28.8","0.28.7","0.27.20","0.28.6","0.27.19","0.28.5","0.28.4","0.27.18","0.28.3","0.27.17","0.28.2","0.27.16","0.27.15","0.28.1","0.28.0","0.27.14","0.27.13","0.27.12","0.27.11","0.27.10","0.27.9","0.26.13","0.27.8","0.27.7","0.27.6","0.27.5","0.27.4","0.26.12","0.27.3","0.27.2","0.27.1","0.27.0","0.26.11","0.26.10","0.26.9","0.26.8","0.26.7","0.26.6","0.26.5","0.26.4","0.26.3","0.26.2"],"latest_version":"0.31.24","updated":"2018-11-13T06:55:53.305Z","rating_count":0,"system_ids":[],"vcs_url":"https://github.com/flant/dapp","maturity":""}
//`
}

// GetPackageVersions returns versions field from package info json
func GetPackageVersions(packageInfo string) (versions []string) {
	res := map[string]interface{}{}
	json.Unmarshal([]byte(packageInfo), &res)

	vs, ok := res["versions"].([]interface{})
	if !ok {
		return
		//fmt.Printf("versions is not an array of strings\n")
	}

	//fmt.Printf("versions: %+v\n", vs)

	for _, v := range vs {
		versions = append(versions, fmt.Sprintf("%v", v))
	}

	//fmt.Printf("versions: %+v\n", versions)

	return versions
}

func (bc *MainBintrayClient) DownloadRelease(version string, dstDir string, files map[string]string) error {
	srcUrl := fmt.Sprintf("%s/%s/%s/%s", btDlUrl, bc.Subject, bc.Repo, version)

	for _, fileName := range files {
		fileUrl := fmt.Sprintf("%s/%s", srcUrl, fileName)
		err := http.DownloadLargeFile(fileUrl, dstDir, fileName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bc *MainBintrayClient) EventCh() chan BintrayEvent {
	return bc.eventCh
}

