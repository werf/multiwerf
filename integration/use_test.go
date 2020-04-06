package integration

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/multiwerf/pkg/util_test"
)

var _ = Describe("use command", func() {
	It("should print only script path", func() {
		output := util_test.SucceedCommandOutputString(
			testDirPath,
			multiwerfBinPath,
			multiwerfArgs("use", "0.0", "alpha", "--as-file")...,
		)

		expectedPath := filepath.Join(
			storageDir,
			"scripts",
			"0.0-alpha",
			"werf_source",
		)

		Ω(output).Should(BeEquivalentTo(expectedPath + "\n"))
		Ω(expectedPath).Should(BeARegularFile())
	})

	It("should print only shell script", func() {
		output := util_test.SucceedCommandOutputString(
			testDirPath,
			multiwerfBinPath,
			multiwerfArgs("use", "0.0", "alpha")...,
		)

		useAsFileOutput := util_test.SucceedCommandOutputString(
			testDirPath,
			multiwerfBinPath,
			multiwerfArgs("use", "0.0", "alpha", "--as-file")...,
		)
		scriptPath := strings.TrimSpace(useAsFileOutput)
		scriptData, err := ioutil.ReadFile(scriptPath)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(bytes.Equal(scriptData, []byte(output))).Should(BeTrue())
	})
})
