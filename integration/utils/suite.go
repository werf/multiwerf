package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/gomega"
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

var environ = os.Environ()

func ResetEnviron() {
	os.Clearenv()
	for _, env := range environ {
		// ignore dynamic variables (e.g. "=ExitCode" windows variable)
		if strings.HasPrefix(env, "=") {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		envName := parts[0]
		envValue := parts[1]

		Ω(os.Setenv(envName, envValue)).Should(Succeed(), env)
	}
}
