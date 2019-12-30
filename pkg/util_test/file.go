package util_test

import (
	"runtime"

	"github.com/otiai10/copy"

	. "github.com/onsi/gomega"
)

var LineBreak = "\n"

func init() {
	if runtime.GOOS == "windows" {
		LineBreak = "\r\n"
	}
}

func CopyIn(sourcePath, destinationPath string) {
	Ω(copy.Copy(sourcePath, destinationPath)).Should(Succeed())
}
