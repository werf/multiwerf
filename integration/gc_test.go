package integration

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/multiwerf/pkg/util_test"
)

var _ = Describe("gc command", func() {
	It("should do nothing", func() {
		output := util_test.SucceedCommandOutputString(
			testDirPath,
			multiwerfBinPath,
			multiwerfArgs("gc")...,
		)

		for _, substr := range []string{
			"GC: Actual versions: []",
			"GC: Local versions:  []",
			"GC: Nothing to clean",
		} {
			Ω(output).Should(ContainSubstring(substr))
		}
	})

	When("local versions exist", func() {
		BeforeEach(func() {
			util_test.CopyIn(fixturePath("gc", "local_versions_exist"), storageDir)
		})

		It("should remove all versions", func() {
			output := util_test.SucceedCommandOutputString(
				testDirPath,
				multiwerfBinPath,
				multiwerfArgs("gc")...,
			)

			for _, substr := range []string{
				"GC: Actual versions: []",
				"GC: Local versions:  [v0.0.0 v0.0.1 v0.1.0]",
				"GC: Removing version v0.0.0 ...",
				"GC: Removing version v0.0.1 ...",
				"GC: Removing version v0.1.0 ...",
			} {
				Ω(output).Should(ContainSubstring(substr))
			}
		})

		When("multiwerf.json exists", func() {
			BeforeEach(func() {
				util_test.CopyIn(fixturePath("gc", "multiwerf_json_exist"), storageDir)
			})

			It("should remove non actual versions", func() {
				output := util_test.SucceedCommandOutputString(
					testDirPath,
					multiwerfBinPath,
					multiwerfArgs("gc")...,
				)

				for _, substr := range []string{
					"GC: Actual versions: [v0.1.0]",
					"GC: Local versions:  [v0.0.0 v0.0.1 v0.1.0]",
					"GC: Removing version v0.0.0 ...",
					"GC: Removing version v0.0.1 ...",
				} {
					Ω(output).Should(ContainSubstring(substr))
				}
			})

			When("multiwerf.json.old exists", func() {
				BeforeEach(func() {
					util_test.CopyIn(fixturePath("gc", "multiwerf_json_old_exist"), storageDir)
				})

				It("should remove non actual versions", func() {
					output := util_test.SucceedCommandOutputString(
						testDirPath,
						multiwerfBinPath,
						multiwerfArgs("gc")...,
					)

					for _, substr := range []string{
						"GC: Actual versions: [v0.0.1 v0.1.0]",
						"GC: Local versions:  [v0.0.0 v0.0.1 v0.1.0]",
						"GC: Removing version v0.0.0 ...",
					} {
						Ω(output).Should(ContainSubstring(substr))
					}
				})
			})
		})
	})
})
