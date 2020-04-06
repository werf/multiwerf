package multiwerf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CheckMajorMinor(t *testing.T) {
	assert.Nil(t, CheckMajorMinor("1.1"))
	assert.NotNil(t, CheckMajorMinor("1.1a"))
	assert.NotNil(t, CheckMajorMinor("1.a1"))
	assert.NotNil(t, CheckMajorMinor(".a1"))
}

func Test_HighestSemverVersion(t *testing.T) {
	input := []string{
		"0.0.1-rc.321+test.ci.6",
		"0.0.1+test.ci.4",
		"0.0.1-alpha.2+test.ci.7",
	}

	version, err := HighestSemverVersion(input)

	assert.NoError(t, err)

	assert.Equal(t, "0.0.1+test.ci.4", version)
}
