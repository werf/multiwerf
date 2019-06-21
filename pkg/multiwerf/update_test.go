package multiwerf

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetBinaryInfo(t *testing.T) {
	// testdata/empty_bin_path/.multiwerf
	MultiwerfStorageDir = filepath.Join("testdata", "1.0-beta-dirs", ".multiwerf")

	msgsCh := make(chan ActionMessage, 10)

	var info BinaryInfo
	var err error

	go func() {
		info, err = GetVerifiedBinaryInfo("v1.0.1-beta.9", msgsCh)
		msgsCh <- ActionMessage{
			action: "exit",
		}
	}()

	run := true
	for {
		if !run {
			break
		}
		select {
		case msg := <-msgsCh:
			//fmt.Printf("%s %s\n", msg.msg, msg.err)

			if msg.err != nil {
				run = false
			}

			if msg.action == "exit" {
				run = false
			}
		}
	}

	assert.Equal(t, "testdata/1.0-beta-dirs/.multiwerf/v1.0.1-beta.9/werf-linux-amd64-v1.0.1-beta.9", info.BinaryPath)
	assert.Equal(t, true, info.HashVerified)
	assert.Equal(t, "v1.0.1-beta.9", info.Version)

	fmt.Printf("%+v err: %s\n", info, err)
}

func Test_LocalLatestBinaryInfo(t *testing.T) {
	// testdata/1.0-beta-dirs/.multiwerf
	MultiwerfStorageDir = filepath.Join("testdata", "1.0-beta-dirs", ".multiwerf")

	msgsCh := make(chan ActionMessage, 10)

	var info BinaryInfo
	var err error

	go func() {
		info, err = LocalLatestBinaryInfo("1.0", "beta", msgsCh)
		assert.NoError(t, err)
		msgsCh <- ActionMessage{
			action: "exit",
		}
	}()

	run := true
	for {
		if !run {
			break
		}
		select {
		case msg := <-msgsCh:
			//fmt.Printf("MSG m=%v e=%v\n", msg.msg, msg.err)

			if msg.err != nil {
				run = false
			}

			if msg.action == "exit" {
				run = false
			}
		}
	}
	assert.Equal(t, "testdata/1.0-beta-dirs/.multiwerf/v1.0.1-beta.9/werf-linux-amd64-v1.0.1-beta.9", info.BinaryPath)
	//fmt.Printf("info=%+v\nerr=%v\n", info, err)
}

// Bad semver directory in .multiwerf should not give an error
func Test_LocalLatestBinaryInfo_BadSemver(t *testing.T) {
	// testdata/empty_bin_path/.multiwerf
	MultiwerfStorageDir = filepath.Join("testdata", "bad_semver", ".multiwerf")

	msgsCh := make(chan ActionMessage, 10)

	var info BinaryInfo
	var err error

	go func() {
		info, err = LocalLatestBinaryInfo("1.0", "beta", msgsCh)
		//fmt.Printf("ERR=%v\n", err)
		msgsCh <- ActionMessage{
			action: "exit",
		}
	}()

	run := true
	for {
		if !run {
			break
		}
		select {
		case msg := <-msgsCh:
			//fmt.Printf("MSG m=%v e=%v\n", msg.msg, msg.err)

			if msg.err != nil {
				run = false
			}

			if msg.action == "exit" {
				run = false
			}
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, "testdata/bad_semver/.multiwerf/v1.0.1-beta.9/werf-linux-amd64-v1.0.1-beta.9", info.BinaryPath)
}

// no error on bad semver in .multiwerf
func Test_FindSemverDirs(t *testing.T) {
	// testdata/empty_bin_path/.multiwerf
	MultiwerfStorageDir = filepath.Join("testdata", "bad_semver", ".multiwerf")

	paths, warn, err := FindSemverDirs(MultiwerfStorageDir)

	assert.NoError(t, err)
	assert.True(t, len(paths) > 0)
	assert.NotEmpty(t, warn)

	//fmt.Printf("err: %v\npaths: %+v\n", err, paths)
}
