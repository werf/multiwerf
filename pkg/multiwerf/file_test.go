package multiwerf

import (
	"fmt"

	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_LoadHashes(t *testing.T) {
	hashes := LoadHashFile("testdata", "SHA256SUMS")

	assert.True(t, len(hashes) > 1)

	fmt.Printf("%+v\n", hashes)
}

func Test_VerifyReleaseFileHash(t *testing.T) {
	verified, err := VerifyReleaseFileHash("testdata", "SHA256SUMS", "testfile")
	assert.NoError(t, err)
	assert.True(t, verified)

	verified, err = VerifyReleaseFileHash("testdata", "SHA256SUMS", "testfile2")
	assert.NoError(t, err)
	assert.True(t, verified)

	verified, err = VerifyReleaseFileHash("testdata", "SHA256SUMS", "non-existent")
	assert.Error(t, err)
	assert.False(t, verified)
}
