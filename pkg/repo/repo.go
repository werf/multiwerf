package repo

import "os"

type Repo interface {
	GetPackageVersions() ([]string, error)
	DownloadFiles(version string, dstDir string, files map[string]string) error
	GetFileContent(version string, fileName string) (string, error)
	String() string
}

func debug() bool {
	return os.Getenv("MULTIWERF_DEBUG_REPO") == "1"
}
