package multiwerf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/Masterminds/semver"
)

func Test_CheckMajorMinor(t *testing.T) {
	assert.Nil(t, CheckMajorMinor("1.1"))
	assert.NotNil(t, CheckMajorMinor("1.1a"))
	assert.NotNil(t, CheckMajorMinor("1.a1"))
	assert.NotNil(t, CheckMajorMinor(".a1"))
}

func Test_ChooseLatestVersion(t *testing.T) {
	input := []string{
		"1.1.0-alpha.1",
		"1.1.0-beta.1",
		"1.1.0-rc.1",
		"1.1.0",
		"1.1.1-alpha.1",
		"1.1.1-rc.1",
		"1.1.2",
		"1.1.2+20180910.3",
		"1.1.2-beta.2",
		"1.1.2-beta.1",
		"1.1.2-alpha.1",
		"1.1.3-alpha.2+revert.1",
		"1.1.3-rc.1",
		"1.1.4-alpha.1",
		"1.0",
		"1.2",
		"1.5.1-beta.2+20180103.1",
		"1.10.1",
		"3.1",
		"1.0.0-alpha.1",
	}

	var res string
	var err error

	res, err = ChooseLatestVersion("1.1", "alpha", input, AvailableChannels)
	assert.Nil(t, err)
	assert.Equal(t, "1.1.4-alpha.1", res)

	res, err = ChooseLatestVersion("1.1", "beta", input, AvailableChannels)
	assert.Nil(t, err)
	assert.Equal(t, "1.1.3-rc.1", res)

	res, err = ChooseLatestVersion("1.1", "rc", input, AvailableChannels)
	assert.Nil(t, err)
	assert.Equal(t, "1.1.3-rc.1", res)

	res, err = ChooseLatestVersion("1.1", "stable", input, AvailableChannels)
	assert.Nil(t, err)
	assert.Equal(t, "1.1.2+20180910.3", res)
}

func Test_filterByMajorMinor(t *testing.T) {
	versions := []string{
		"1.1",
		"1.0",
		"1.2",
		"1.1.1",
		"1.10.1",
		"3.1",
		"1.0.0-alpha.1",
	}
	res, err := filterByMajorMinor(versions, "1.1")

	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
}

func Test_makePatchMap(t *testing.T) {
	input := []string{
		"1.1.0-alpha.1",
		"1.1.0-beta.1",
		"1.1.0-rc.1",
		"1.1.0",
		"1.1.1-alpha.1",
		"1.1.1-rc.1",
		"1.1.2",
		"1.1.2-beta.2",
		"1.1.2-beta.1",
		"1.1.2-alpha.1",
		"1.1.3-alpha.2+revert.1",
		"1.1.3-rc.1",
		"1.1.4-alpha.1",
		"1.1.4",
		"1.0",
		"1.2",
		"1.5.1-beta.2+20180103.1",
		"1.10.1",
		"3.1",
		"1.0.0-alpha.1",
	}
	versions, err := filterByMajorMinor(input, "1.1")
	assert.Nil(t, err)

	patchesMap, patches := makePatchMap(versions)
	assert.Equal(t, 5, len(patches))
	assert.Equal(t, 5, len(patchesMap))
	assert.Equal(t, 4, len(patchesMap[2]))
}

// Some tests
func Test_matchChannel(t *testing.T) {
	trueInput := map[string]string{
		"1.1.0-alpha.1":                        "alpha",
		"1.1.0-beta.1":                         "beta",
		"1.1.0-rc.1":                           "rc",
		"1.1.0":                                "stable",
		"1.1.1-alpha.1":                        "alpha",
		"1.1.1-rc.1":                           "rc",
		"1.1.2":                                "stable",
		"1.1.2-beta.2":                         "beta",
		"1.1.3-alpha.2+revert.1":               "alpha",
		"1.1.3-rc.1123213+123123123qweuf532fd": "rc",
		"1.0+hotfix.321":                       "stable",
		"1.5.1-beta.2+20180103.1":              "beta",
	}

	falseInput:= map[string]string {
		"1.10.1-dev": "alpha",
		"3.1-1": "stable", // Not stable because of prerelease part. Hot fixes are versioned with metatada
		"1.0.0-alpha.1": "beta",
	}

	for version, channel := range trueInput {
		res := matchChannel(semver.MustParse(version), channel)
		assert.True(t, res)
	}
	for version, channel := range falseInput {
		res := matchChannel(semver.MustParse(version), channel)
		assert.False(t, res)
	}
}

// If PATCH has stable release — this version is available for all channels
func Test_determineChannels(t *testing.T) {
	input := []string{
		"1.1.2",
		"1.1.2-beta.2",
		"1.1.2-beta.1",
		"1.1.2-alpha.1",
	}
	versions, err := filterByMajorMinor(input, "1.1")
	assert.Nil(t, err)

	res := determineChannels(versions, AvailableChannels)

	assert.Equal(t, 4, len(res))
	assert.Equal(t, "1.1.2", res["alpha"].String())
	assert.Equal(t, "1.1.2", res["beta"].String())
	assert.Equal(t, "1.1.2", res["rc"].String())
	assert.Equal(t, "1.1.2", res["stable"].String())
}

// If PATCH has beta release — this version is available for alpha
// channel and not available for rc and stable
func Test_determineChannels_noStable(t *testing.T) {
	input := []string{
		"1.1.2-beta.2",
		"1.1.2-beta.1",
		"1.1.2-alpha.1",
	}
	versions, err := filterByMajorMinor(input, "1.1")
	assert.Nil(t, err)

	res := determineChannels(versions, AvailableChannels)

	assert.Equal(t, 2, len(res))
	assert.Equal(t, "1.1.2-beta.2", res["alpha"].String())
	assert.Equal(t, "1.1.2-beta.2", res["beta"].String())
	assert.Nil(t, res["rc"])
	assert.Nil(t, res["stable"])
}

// If PATCH has rc release — this version is not available for stable
// but available for alpha, beta, rc
func Test_determineChannels_hasRc(t *testing.T) {
	input := []string{
		"1.1.2-beta.2",
		"1.1.2-beta.1",
		"1.1.2-alpha.1+1q121",
		"1.1.2-rc",
		"1.1.2-rc.0",
	}
	versions, err := filterByMajorMinor(input, "1.1")
	assert.Nil(t, err)

	res := determineChannels(versions, AvailableChannels)

	assert.Equal(t, 3, len(res))
	assert.Equal(t, "1.1.2-rc.0", res["alpha"].String())
	assert.Equal(t, "1.1.2-rc.0", res["beta"].String())
	assert.Equal(t, "1.1.2-rc.0", res["rc"].String())
	assert.Nil(t, res["stable"])
}