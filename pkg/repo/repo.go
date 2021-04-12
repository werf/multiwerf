package repo

type Repo interface {
	GetPackageVersions() ([]string, error)
	DownloadFiles(version string, dstDir string, files map[string]string) error
	GetFileContent(version string, fileName string) (string, error)
	String() string
}
