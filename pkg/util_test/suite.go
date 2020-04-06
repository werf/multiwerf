package util_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func GetTempDir() (string, error) {
	dir, err := ioutil.TempDir("", "multiwerf-integration-tests-")
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "darwin" {
		dir, err = filepath.EvalSymlinks(dir)
		if err != nil {
			return "", fmt.Errorf("eval symlinks of path %s failed: %s", dir, err)
		}
	}

	return dir, nil
}

func ProcessMultiwerfBinPath() string {
	path := os.Getenv("MULTIWERF_TEST_BINARY_PATH")
	if path != "" {
		return path
	}

	return BuildMultiwerfBinPath()
}

func BuildMultiwerfBinPath() string {
	path, err := gexec.Build("github.com/flant/multiwerf/cmd/multiwerf")
	Ω(err).ShouldNot(HaveOccurred())
	return path
}
