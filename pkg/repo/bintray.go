package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	uuid "github.com/satori/go.uuid"
	"github.com/werf/multiwerf/pkg/http"
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
	if debug() {
		fmt.Printf("-- BintrayClient.GetPackageVersions\n")
	}

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
	if debug() {
		fmt.Printf("-- BintrayClient.DownloadFiles version=%q dstDir=%q files=%#v\n", version, dstDir, files)
	}

	srcUrl := fmt.Sprintf("%s/%s/%s/%s", BintrayDlUrl, bc.Subject, bc.Repo, version)

	for _, fileName := range files {
		dstFilePath := filepath.Join(dstDir, fileName)
		tmpFileName := fmt.Sprintf("%s.%s", fileName, uuid.NewV4().String())
		tmpFilePath := filepath.Join(dstDir, tmpFileName)

		fileUrl := fmt.Sprintf("%s/%s", srcUrl, fileName)

		err := func() error {
			defer os.RemoveAll(tmpFilePath)

			if err := http.DownloadLargeFile(fileUrl, dstDir, fileName); err != nil {
				return fmt.Errorf("%s download error: %v", fileUrl, err)
			}

			if err := os.Rename(tmpFilePath, dstFilePath); err != nil {
				return fmt.Errorf("unable to rename %q to %q: %s", tmpFilePath, dstFilePath, err)
			}

			return nil
		}()

		if err != nil {
			return err
		}
	}

	return nil
}

func (bc *BintrayClient) GetFileContent(version string, fileName string) (string, error) {
	if debug() {
		fmt.Printf("-- BintrayClient.GetFileContent version=%q fileName=%q\n", version, fileName)
	}

	srcUrl := fmt.Sprintf("%s/%s/%s/%s", BintrayDlUrl, bc.Subject, bc.Repo, version)
	fileUrl := fmt.Sprintf("%s/%s", srcUrl, fileName)
	return http.MakeRestAPICall("GET", fileUrl)
}

func (bc *BintrayClient) String() string {
	return "bintray"
}
