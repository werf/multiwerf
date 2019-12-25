package multiwerf

import (
	"fmt"
	"path/filepath"

	"github.com/flant/multiwerf/pkg/app"
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
	dstPath := filepath.Join(StorageDir, version)
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
	dstPath := filepath.Join(StorageDir, version)
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
