package integration

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/flant/multiwerf/integration/utils"
)

var _ = It("self-update", func() {
	multiwerfBinPath, err := gexec.Build("github.com/flant/multiwerf/cmd/multiwerf")
	Ω(err).ShouldNot(HaveOccurred())

	version := utils.SucceedCommandOutputString(
		testDirPath,
		multiwerfBinPath,
		"version",
	)

	output := utils.SucceedCommandOutputString(
		testDirPath,
		multiwerfBinPath,
		[]string{"update", "1.0", "alpha"}...,
	)

	for _, substr := range []string{
		"multiwerf dev self-update: detect version",
		"multiwerf dev self-update: download release",
	} {
		Ω(output).Should(ContainSubstring(substr))
	}

	newVersion := utils.SucceedCommandOutputString(
		testDirPath,
		multiwerfBinPath,
		"version",
	)

	Ω(newVersion).ShouldNot(BeEquivalentTo(version))
})
