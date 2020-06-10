package multiwerf

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/werf/multiwerf/pkg/app"
)

type BinaryInfo struct {
	BinaryPath   string
	Version      string
	HashVerified bool
}

// verifiedLocalBinaryInfo returns BinaryInfo object for the version if it is
// stored in StorageDir and valid. Empty object is returned if no binary found.
// Hash of binary is verified with SHA256SUMS files.
func verifiedLocalBinaryInfo(messages chan ActionMessage, version string) (*BinaryInfo, error) {
	dstPath := localVersionDirPath(version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is %s, files: %+v", dstPath, files),
		debug: true,
	}

	if exist, err := DirExists(dstPath); err != nil {
		return nil, err
	} else if !exist {
		return nil, nil
	}

	binInfo := &BinaryInfo{}
	binInfo.Version = version
	binInfo.BinaryPath = filepath.Join(dstPath, files["program"])
	binInfo.HashVerified = false

	if exist, err := IsReleaseFilesExist(dstPath, files); err != nil {
		return nil, err
	} else if !exist {
		return binInfo, nil
	}

	// check hash of local binary
	match, err := VerifyReleaseFileHash(messages, dstPath, files["hash"], files["program"])
	if err != nil {
		return nil, err
	}

	binInfo.HashVerified = match

	return binInfo, nil
}

// localBinaryInfo returns BinaryInfo object for the version if it is
// stored in StorageDir. Empty object is returned if no binary found.
func localBinaryInfo(messages chan ActionMessage, version string) (*BinaryInfo, error) {
	dstPath := localVersionDirPath(version)
	files := ReleaseFiles(app.BintrayPackage, version, app.OsArch)
	messages <- ActionMessage{
		msg:   fmt.Sprintf("dstPath is %s, files: %+v", dstPath, files),
		debug: true,
	}

	if exist, err := FileExists(filepath.Join(dstPath, files["program"])); err != nil {
		return nil, err
	} else if !exist {
		return nil, nil
	}

	binInfo := &BinaryInfo{}
	binInfo.Version = version
	binInfo.BinaryPath = filepath.Join(dstPath, files["program"])
	binInfo.HashVerified = false

	return binInfo, nil
}

func localVersions() ([]string, error) {
	var versions []string

	exist, err := DirExists(StorageDir)
	if err != nil {
		return nil, fmt.Errorf("dir exists failed %s: %s", StorageDir, err)
	} else if !exist {
		return []string{}, nil
	}

	files, err := ioutil.ReadDir(StorageDir)
	if err != nil {
		return nil, fmt.Errorf("read dir failed %s: %s", StorageDir, err)
	}

	versionGlob, err := regexp.Compile("v[0-9]*\\.[0-9]*\\.[0-9]*.*")
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}

		if versionGlob.MatchString(f.Name()) {
			versions = append(versions, f.Name())
		}
	}

	return versions, nil
}

func localVersionDirPath(version string) string {
	return filepath.Join(StorageDir, version)
}
