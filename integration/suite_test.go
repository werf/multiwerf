package integration

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/flant/multiwerf/integration/utils"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var testDirPath string
var tmpDir string

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

var _ = BeforeEach(func() {
	var err error
	tmpDir, err = utils.GetTempDir()
	Ω(err).ShouldNot(HaveOccurred())

	testDirPath = tmpPath()

	Ω(os.Setenv("MULTIWERF_STORAGE_DIR", filepath.Join(testDirPath, "storage_dir"))).Should(Succeed())
	Ω(os.Setenv("MULTIWERF_EXPERIMENTAL", "true")).Should(Succeed())
})

var _ = AfterEach(func() {
	err := os.RemoveAll(tmpDir)
	Ω(err).ShouldNot(HaveOccurred())

	utils.ResetEnviron()
})

func tmpPath(paths ...string) string {
	pathsToJoin := append([]string{tmpDir}, paths...)
	return filepath.Join(pathsToJoin...)
}

func fixturePath(paths ...string) string {
	pathsToJoin := append([]string{"_fixtures"}, paths...)
	return filepath.Join(pathsToJoin...)
}
