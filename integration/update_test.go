package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/werf/multiwerf/pkg/util_test"
)

var _ = Describe("update command", func() {
	When("--self-update=yes", func() {
		var multiwerfForSelfUpdateBinPath string

		BeforeEach(func() {
			testDirBinPath := filepath.Join(testDirPath, "bin")
			util_test.CopyIn(filepath.Dir(multiwerfBinPath), testDirBinPath)
			multiwerfForSelfUpdateBinPath = filepath.Join(testDirBinPath, filepath.Base(multiwerfBinPath))

			stubs.SetEnv("MULTIWERF_SELF_UPDATE", "yes")
		})

		// * multiwerf version should be changed
		// * self-update.delay file should be created
		// * .multiwerf.exe.old file should remain for windows
		// * multiwerf tmp dir should be empty for other systems
		It("should be self-updated", func() {
			version := util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfForSelfUpdateBinPath,
				multiwerfArgs("version")...,
			)

			output := util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfForSelfUpdateBinPath,
				multiwerfArgs("update", "0.0")...,
			)

			for _, substr := range []string{
				"Starting multiwerf self-update ...",
				"Self-update: Detect version",
				"Self-update: Successfully updated to",
			} {
				Ω(output).Should(ContainSubstring(substr))
			}

			selfUpdateDelayFilePath := filepath.Join(storageDir, "self-update.delay")
			Ω(selfUpdateDelayFilePath).Should(BeARegularFile(), "self-update.delay file should be created")

			multiwerfTmpDir := filepath.Join(storageDir, "tmp")
			if runtime.GOOS == "windows" {
				multiwerfOldFilePath := filepath.Join(multiwerfTmpDir, fmt.Sprintf(".%s.old", filepath.Base(multiwerfForSelfUpdateBinPath)))
				Ω(multiwerfOldFilePath).Should(BeARegularFile(), fmt.Sprintf(".%s.old file should remain for windows", filepath.Base(multiwerfForSelfUpdateBinPath)))
			} else {
				storageTmpDirShouldBeEmpty()
			}

			newVersion := util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfForSelfUpdateBinPath,
				multiwerfArgs("version")...,
			)

			Ω(newVersion).ShouldNot(BeEquivalentTo(version), "multiwerf version should be changed")
		})
	})

	When("--self-update=no", func() {
		BeforeEach(func() {
			stubs.SetEnv("MULTIWERF_SELF_UPDATE", "no")
		})

		type ItEntry struct {
			multiwerfArgs         []string
			checksAfterFirstStep  func(output string)
			checksAfterSecondStep func(output string)
			checksAfterThirdStep  func(output string)
		}

		// first step is running update with empty multiwerf storage dir
		// second step is relaunching update without any changes
		// third step is relaunching update when remote channel mapping has been changed
		threeStepsItFunc := func(e ItEntry) {
			By("first step is running update with empty multiwerf storage dir")
			stubs.SetEnv("MULTIWERF_CHANNEL_MAPPING_URL", remoteChannelMapping1Url)
			output := util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfBinPath,
				multiwerfArgs(e.multiwerfArgs...)...,
			)
			e.checksAfterFirstStep(output)

			By("second step is relaunching update without any changes")
			output = util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfBinPath,
				multiwerfArgs(e.multiwerfArgs...)...,
			)
			e.checksAfterSecondStep(output)

			By("third step is relaunching update when remote channel mapping has been changed")
			stubs.SetEnv("MULTIWERF_CHANNEL_MAPPING_URL", remoteChannelMapping2Url)
			output = util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfBinPath,
				multiwerfArgs(e.multiwerfArgs...)...,
			)
			e.checksAfterThirdStep(output)
		}

		// 1.
		// * remote channel mapping should be downloaded to multiwerf.json
		// * the release files for channel should be downloaded to v0.0.0 folder
		// * tmp dir should be empty
		// 2.
		// * local version should be used
		// * tmp dir should be empty
		// 3.
		// * multiwerf.json should be substituted with updated remote channel mapping file
		// * the actual release files for channel should be downloaded to vX.X.X folder
		// * tmp dir should be empty
		baseEntry := ItEntry{
			multiwerfArgs: []string{"update", "0.0", "stable"},
			checksAfterFirstStep: func(output string) {
				for _, substr := range []string{
					"GC: Nothing to clean",
					"The version v0.0.0 is the actual for channel 0.0/stable",
					"Downloading the version v0.0.0 ...",
					"The actual version has been successfully downloaded",
				} {
					Ω(output).Should(ContainSubstring(substr))
				}

				multiwerfJsonShouldBeEqualRemoteChannelMapping(filepath.Join(storageDir, "multiwerf.json"), remoteChannelMapping1Url)
				Ω(filepath.Join(storageDir, "multiwerf.json.old")).ShouldNot(BeAnExistingFile())
				releaseFilesShouldBeExist(actualStableVersion1)
				storageTmpDirShouldBeEmpty()
			},
			checksAfterSecondStep: func(output string) {
				for _, substr := range []string{
					"GC: Nothing to clean",
					"The version v0.0.0 is the actual for channel 0.0/stable",
					"The actual version is available locally",
				} {
					Ω(output).Should(ContainSubstring(substr))
				}

				storageTmpDirShouldBeEmpty()
			},
			checksAfterThirdStep: func(output string) {
				for _, substr := range []string{
					"GC: Nothing to clean",
					"The version v0.0.1 is the actual for channel 0.0/stable",
					"Downloading the version v0.0.1 ...",
					"The actual version has been successfully downloaded",
				} {
					Ω(output).Should(ContainSubstring(substr))
				}

				multiwerfJsonShouldBeEqualRemoteChannelMapping(filepath.Join(storageDir, "multiwerf.json"), remoteChannelMapping2Url)
				multiwerfJsonShouldBeEqualRemoteChannelMapping(filepath.Join(storageDir, "multiwerf.json.old"), remoteChannelMapping1Url)
				releaseFilesShouldBeExist(actualStableVersion2)
				storageTmpDirShouldBeEmpty()
			},
		}

		// 1.
		// * base
		// 2/3.
		// * local channel mapping should be used
		// * tmp dir should be empty
		updateNoEntry := ItEntry{
			multiwerfArgs: append(baseEntry.multiwerfArgs, "--update=no"),
			checksAfterFirstStep: func(output string) {
				baseEntry.checksAfterFirstStep(output)
			},
			checksAfterSecondStep: func(output string) {
				baseEntry.checksAfterSecondStep(output)
			},
			checksAfterThirdStep: func(output string) {
				baseEntry.checksAfterSecondStep(output)
			},
		}

		// 1.
		// * base+
		// * try-remote-channel-mapping.delay file created
		// 2/3.
		// * local channel mapping should be used
		// * tmp dir should be empty
		withCacheEntry := ItEntry{
			multiwerfArgs: append(baseEntry.multiwerfArgs, "--with-cache"),
			checksAfterFirstStep: func(output string) {
				baseEntry.checksAfterFirstStep(output)
				Ω(filepath.Join(storageDir, "try-remote-channel-mapping.delay")).Should(BeARegularFile(), "try-remote-channel-mapping.delay file created")
			},
			checksAfterSecondStep: func(output string) {
				baseEntry.checksAfterSecondStep(output)
			},
			checksAfterThirdStep: func(output string) {
				baseEntry.checksAfterSecondStep(output)
			},
		}

		DescribeTable("should work properly", threeStepsItFunc,
			Entry("base", baseEntry),
			Entry("--update=no", updateNoEntry),
			Entry("--with-cache", withCacheEntry),
		)
	})
})

func multiwerfJsonShouldBeEqualRemoteChannelMapping(multiwerfJsonFilePath, remoteChannelMappingUrl string) {
	Ω(multiwerfJsonFilePath).Should(BeARegularFile(), "remote channel mapping should be downloaded to multiwerf.json")

	data, err := ioutil.ReadFile(multiwerfJsonFilePath)
	Ω(err).ShouldNot(HaveOccurred(), "remote channel mapping should be downloaded to multiwerf.json")
	Ω(string(data)).Should(BeEquivalentTo(string(getFormatRemoteChannelMappingData(remoteChannelMappingUrl))), "remote channel mapping should be downloaded to multiwerf.json")
}
