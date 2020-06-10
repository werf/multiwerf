package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prashantv/gostub"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/werf/multiwerf/pkg/multiwerf"
	"github.com/werf/multiwerf/pkg/util_test"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var testDirPath string
var tmpDir string
var storageDir string
var multiwerfBinPath string
var stubs = gostub.New()

const (
	remoteChannelMapping1Url = "https://gist.githubusercontent.com/alexey-igrychev/a01d19ce9924761cb457fdedd83bcd6c/raw/a16a7c4b0c6adfb03e4f7a88754fc4325e320f68/multiwerf-1.json"
	remoteChannelMapping2Url = "https://gist.githubusercontent.com/alexey-igrychev/a01d19ce9924761cb457fdedd83bcd6c/raw/2a7afbd0f272dfa088887ff40c18e0468fbc16f2/multiwerf-2.json"

	actualAlphaVersion1  = "v0.0.1"
	actualStableVersion1 = "v0.0.0"
	actualStableVersion2 = "v0.0.1"
)

var _ = SynchronizedBeforeSuite(func() []byte {
	computedPathToMultiwerf := util_test.ProcessMultiwerfBinPath()
	return []byte(computedPathToMultiwerf)
}, func(computedPathToMultiwerf []byte) {
	multiwerfBinPath = string(computedPathToMultiwerf)
})

var _ = BeforeEach(func() {
	var err error
	tmpDir, err = util_test.GetTempDir()
	Ω(err).ShouldNot(HaveOccurred())

	testDirPath = tmpPath()

	storageDir = filepath.Join(testDirPath, "storage_dir")
	stubs.SetEnv("MULTIWERF_STORAGE_DIR", storageDir)

	stubs.SetEnv("MULTIWERF_SELF_BINTRAY_SUBJECT", "flant")
	stubs.SetEnv("MULTIWERF_SELF_BINTRAY_REPO", "multiwerf")
	stubs.SetEnv("MULTIWERF_SELF_BINTRAY_PACKAGE", "multiwerf-test")

	stubs.SetEnv("MULTIWERF_BINTRAY_SUBJECT", "flant")
	stubs.SetEnv("MULTIWERF_BINTRAY_REPO", "multiwerf")
	stubs.SetEnv("MULTIWERF_BINTRAY_PACKAGE", "werf-test")
})

var _ = AfterEach(func() {
	err := os.RemoveAll(tmpDir)
	Ω(err).ShouldNot(HaveOccurred())

	stubs.Reset()
})

func tmpPath(paths ...string) string {
	pathsToJoin := append([]string{tmpDir}, paths...)
	return filepath.Join(pathsToJoin...)
}

func fixturePath(paths ...string) string {
	pathsToJoin := append([]string{"_fixtures"}, paths...)
	return filepath.Join(pathsToJoin...)
}

func multiwerfArgs(userArgs ...string) []string {
	var args []string
	if os.Getenv("MULTIWERF_TEST_BINARY_PATH") != "" && os.Getenv("MULTIWERF_TEST_COVERAGE_DIR") != "" {
		coverageFilePath := filepath.Join(
			os.Getenv("MULTIWERF_TEST_COVERAGE_DIR"),
			fmt.Sprintf("%s-%s.out", strconv.FormatInt(time.Now().UTC().UnixNano(), 10), util_test.GetRandomString(10)),
		)
		args = append(args, fmt.Sprintf("-test.coverprofile=%s", coverageFilePath))
	}

	args = append(args, userArgs...)

	return args
}

func storageTmpDirShouldBeEmpty() {
	multiwerfTmpDir := filepath.Join(storageDir, "tmp")

	Ω(multiwerfTmpDir).Should(BeADirectory(), "multiwerf tmp dir should be empty")

	f, err := os.Open(multiwerfTmpDir)
	Ω(err).ShouldNot(HaveOccurred(), "multiwerf tmp dir should be empty")
	defer f.Close()

	_, err = f.Readdirnames(1)
	Ω(err).Should(MatchError(io.EOF), "multiwerf tmp dir should be empty")
}

func releaseFilesShouldBeExist(version string) {
	files := multiwerf.ReleaseFiles("werf-test", version, strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "-"))
	for _, filename := range files {
		Ω(filepath.Join(storageDir, version, filename)).Should(BeARegularFile(), fmt.Sprintf("the release files for channel should be downloaded to %s folder", version))
	}
}

func getFormatRemoteChannelMappingData(remoteChannelMappingUrl string) []byte {
	resp, err := http.Get(remoteChannelMappingUrl)
	Ω(err).ShouldNot(HaveOccurred())
	defer resp.Body.Close()

	Ω(resp.StatusCode).Should(BeEquivalentTo(200), fmt.Sprintf("unexpected response status code %d", resp.StatusCode))

	bodyData, err := ioutil.ReadAll(resp.Body)
	Ω(err).ShouldNot(HaveOccurred(), fmt.Sprintf("respBody read failed: %s", err))

	cm := multiwerf.ChannelMappingRemote{}
	err = json.Unmarshal(bodyData, &cm)
	Ω(err).ShouldNot(HaveOccurred(), fmt.Sprintf("json unmarshal failed: %s", err))

	data, err := cm.Marshal()
	Ω(err).ShouldNot(HaveOccurred(), fmt.Sprintf("json marshal failed: %s", err))

	return data
}
