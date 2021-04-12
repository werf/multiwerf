package repo

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/werf/multiwerf/pkg/http"

	uuid "github.com/satori/go.uuid"
)

const DefaultBintrayApiUrl = "https://api.bintray.com"
const DefaultBintrayDlUrl = "https://dl.bintray.com"

var (
	BintrayApiUrl string
	BintrayDlUrl  string
)

type BintrayClient struct {
	Subject string
	Repo    string
	Package string
}

func NewBintrayClient(subject string, repo string, pkg string) (bc Repo) {
	if BintrayApiUrl == "" {
		BintrayApiUrl = DefaultBintrayApiUrl
	}
	if BintrayDlUrl == "" {
		BintrayDlUrl = DefaultBintrayDlUrl
	}
	bc = &BintrayClient{
		Subject: subject,
		Repo:    repo,
		Package: pkg,
	}
	return bc
}

func (bc *BintrayClient) GetPackageVersions() ([]string, error) {
	pkgInfo, err := bc.getPackageInfo()
	if err != nil {
		return nil, fmt.Errorf("package %s GET info error: %v", bc.Package, err)
	}

	versions := GetPackageVersions(pkgInfo)

	return versions, nil
}

// getPackageInfo returns json response from packages API
func (bc *BintrayClient) getPackageInfo() (string, error) {
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
	_ = json.Unmarshal([]byte(packageInfo), &res)

	vs, ok := res["versions"].([]interface{})
	if !ok {
		return
	}

	for _, v := range vs {
		versions = append(versions, fmt.Sprintf("%v", v))
	}

	return versions
}

func (bc *BintrayClient) DownloadFiles(version string, dstDir string, files map[string]string) error {
	srcUrl := fmt.Sprintf("%s/%s/%s/%s", BintrayDlUrl, bc.Subject, bc.Repo, version)

	tmpDstDir := fmt.Sprintf("%s.tmp.%s", dstDir, uuid.NewV4().String())
	if err := os.MkdirAll(tmpDstDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating tmp dir %q: %s", tmpDstDir, err)
	}
	defer func() {
		os.RemoveAll(tmpDstDir)
	}()

	for _, fileName := range files {
		// TODO implement goreleaser lifecycle and verify gpg signing
		//if fileType == "sig" {
		//	continue
		//}
		fileUrl := fmt.Sprintf("%s/%s", srcUrl, fileName)

		if err := http.DownloadLargeFile(fileUrl, tmpDstDir, fileName); err != nil {
			return fmt.Errorf("%s download error: %v", fileUrl, err)
		}
	}

	if err := os.Rename(tmpDstDir, dstDir); err != nil {
		return fmt.Errorf("unable to rename %q to %q: %s", tmpDstDir, dstDir, err)
	}

	return nil
}

func (bc *BintrayClient) GetFileContent(version string, fileName string) (string, error) {
	srcUrl := fmt.Sprintf("%s/%s/%s/%s", BintrayDlUrl, bc.Subject, bc.Repo, version)
	fileUrl := fmt.Sprintf("%s/%s", srcUrl, fileName)
	return http.MakeRestAPICall("GET", fileUrl)
}

func (bc *BintrayClient) String() string {
	return "bintray"
}
