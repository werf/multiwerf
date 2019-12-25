package multiwerf

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetBinaryInfo(t *testing.T) {
	// testdata/empty_bin_path/.multiwerf
	StorageDir = filepath.Join("testdata", "1.0-beta-dirs", ".multiwerf")

	msgsCh := make(chan ActionMessage, 10)

	var info *BinaryInfo
	var err error

	go func() {
		info, err = verifiedLocalBinaryInfo(msgsCh, "v1.0.1-beta.9")
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
