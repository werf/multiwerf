package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/multiwerf/pkg/multiwerf"
	"github.com/flant/multiwerf/pkg/util_test"
)

var _ = Describe("other commands", func() {
	When("local channel mapping and the actual channel version exist", func() {
		BeforeEach(func() {
			// FIXME: use _fixtures instead
			stubs.SetEnv("MULTIWERF_SELF_UPDATE", "no")
			stubs.SetEnv("MULTIWERF_CHANNEL_MAPPING_URL", remoteChannelMapping1Url)
			util_test.RunSucceedCommand(
				testDirPath,
				multiwerfBinPath,
				multiwerfArgs("update", "1.0", "alpha")...,
			)
		})

		It("werf-path should print only the path", func() {
			output := util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfBinPath,
				multiwerfArgs("werf-path", "1.0", "alpha")...,
			)

			expectedPath := filepath.Join(
				storageDir,
				actualAlphaVersion1,
				multiwerf.ReleaseProgramFilename(
					"werf",
					actualAlphaVersion1,
					strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "-"),
				),
			)

			Ω(expectedPath).Should(BeARegularFile())
			Ω(output).Should(BeEquivalentTo(expectedPath + "\n"))
		})

		It("werf-exec should print only the werf version", func() {
			output := util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfBinPath,
				multiwerfArgs("werf-exec", "1.0", "alpha", "--", "version")...,
			)

			Ω(output).Should(BeEquivalentTo(actualAlphaVersion1 + "\n"))
		})
	})

	for _, cmd := range []string{"werf-path", "werf-exec"} {
		When("local channel mapping does not exist", func() {
			It(fmt.Sprintf("%s should fail with local channel mapping is not found error", cmd), func() {
				res, err := util_test.RunCommand(
					testDirPath,
					multiwerfBinPath,
					multiwerfArgs(cmd, "1.0", "alpha")...,
				)

				Ω(err).Should(HaveOccurred())

				for _, substr := range []string{
					"Error: get the local channel mapping failed",
				} {
					Ω(string(res)).Should(ContainSubstring(substr))
				}
			})
		})

		When("local channel mapping exists but the actual channel version does not", func() {
			BeforeEach(func() {
				util_test.CopyIn(fixturePath("multiwerf.json"), filepath.Join(storageDir, "multiwerf.json"))
			})

			It(fmt.Sprintf("%s should fail with the actual channel version is not found error", cmd), func() {
				res, err := util_test.RunCommand(
					testDirPath,
					multiwerfBinPath,
					multiwerfArgs(cmd, "1.0", "alpha")...,
				)

				Ω(err).Should(HaveOccurred())

				for _, substr := range []string{
					"Error: the actual channel version has not been found locally",
				} {
					Ω(string(res)).Should(ContainSubstring(substr))
				}
			})
		})
	}
})
