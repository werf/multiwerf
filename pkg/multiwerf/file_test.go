package multiwerf

import (
	"fmt"

	"testing"
	"github.com/stretchr/testify/assert"
)

func Test_LoadHashes(t *testing.T) {
	hashes := LoadHashFile(".", "SHA256SUMS")

	assert.True(t, len(hashes)>1)

	fmt.Printf("%+v\n", hashes)
}
