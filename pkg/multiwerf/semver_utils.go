package multiwerf

import (
	"fmt"
	"sort"

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

// HighestSemverVersion returns the latest version from availableVersions array
func HighestSemverVersion(versions []string) (string, error) {
	vs := make([]*semver.Version, len(versions))
	for i, r := range versions {
		v, err := semver.NewVersion(r)
		if err != nil {
			return "", fmt.Errorf("parse version %s error: %v", r, err)
		}
		vs[i] = v
	}

	sort.Sort(semver.Collection(vs))

	if len(vs) > 0 {
		return vs[len(vs)-1].Original(), nil
	}
	return "", nil
}
