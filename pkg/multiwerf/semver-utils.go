package multiwerf

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
)

// CheckMajorMinor return error if string not in form MAJOR.MINOR
func CheckMajorMinor(version string) error {
	v, err := semver.NewVersion(version)
	if err != nil {
		return err
	}
	reconstructed := fmt.Sprintf("%d.%d", v.Major(), v.Minor())
	if version != reconstructed {
		return fmt.Errorf("version should be in form MAJOR.MINOR")
	}
	return nil
}


// ChooseLatestVersionSimple returns a latest version from availableVersions array
func ChooseLatestVersionSimple(availableVersions []string) (string, error) {
	vs := make([]*semver.Version, len(availableVersions))
	for i, r := range availableVersions {
		v, err := semver.NewVersion(r)
		if err != nil {
			return "", fmt.Errorf("parse version '%s' error: %v", r, err)
		}
		vs[i] = v
	}

	sort.Sort(semver.Collection(vs))

	if len(vs) > 0 {
		return vs[len(vs)-1].Original(), nil
	}
	return "", nil
}

// ChooseLatestVersion returns a latest version from availableVersions that suits version and channel constrain.
//
// version is a constrain on MAJOR and MINOR parts
//
// channel is a constrain on Prerelease part
//
// availableChannels is an array of prioritized Prerelease variants
func ChooseLatestVersion(version string, channel string, availableVersions []string, availableChannels []string) (string, error) {
	mmVersions, err := filterByMajorMinor(availableVersions, version)
	if err != nil {
		return "", err
	}
	patchesMap, patches := makePatchMap(mmVersions)

	// search for a version for channel from the end
	sort.Sort(sort.Reverse(sort.IntSlice(patches)))
	for _, patch := range patches {
		channelMap := determineChannels(patchesMap[int64(patch)], availableChannels)
		if _, ok := channelMap[channel]; ok {
			return channelMap[channel].Original(), nil
		}
	}
	return "", nil
}

// filterByMajorMinor construct array from availableVersions where each item has MAJOR.MINOR as in a version argument
func filterByMajorMinor(availableVersions []string, version string) ([]*semver.Version, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}

	vs := make([]*semver.Version, 0)
	for _, r := range availableVersions {
		nv, err := semver.NewVersion(r)
		if err != nil {
			return nil, err
		}
		if nv.Major() == v.Major() && nv.Minor() == v.Minor() {
			vs = append(vs, nv)
		}
	}
	return vs, nil
}

// makePatchMap creates a map PATCH => versions and a ordered array of available patches
func makePatchMap(versions []*semver.Version) (map[int64][]*semver.Version, []int) {
	pMap := map[int64][]*semver.Version{}
	for _, v := range versions {
		patch := v.Patch()
		if _, ok := pMap[patch]; !ok {
			pMap[patch] = make([]*semver.Version, 0)
		}
		pMap[patch] = append(pMap[patch], v)
	}

	patches := []int{}
	for p := range pMap {
		patches = append(patches, int(p))
	}
	sort.Sort(sort.IntSlice(patches))

	return pMap, patches
}

func matchChannel(version *semver.Version, channel string) bool {
	if channel == "stable" {
		return version.Prerelease() == ""
	} else {
		return strings.HasPrefix(version.Prerelease(), channel)
	}
}

// determineChannels returns the latest version for each available channel
// beta channel also suits for alpha
func determineChannels(versions []*semver.Version, availableChannels []string) map[string]*semver.Version {
	sort.Sort(semver.Collection(versions))

	res := make(map[string]*semver.Version, 0)
	for _, v := range versions {
		versionChannel := ""

		// find exact channel
		for _, channel := range availableChannels {
			if matchChannel(v, channel) {
				versionChannel = channel
			}
		}
		// propagate version to lower priority channels
		for _, channel := range availableChannels {
			res[channel] = v
			if channel == versionChannel {
				break
			}
		}
	}

	return res
}