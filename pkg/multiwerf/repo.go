package multiwerf

import (
	"github.com/werf/multiwerf/pkg/app"
	"github.com/werf/multiwerf/pkg/repo"
)

func NewSelfBtClient() (bc repo.Repo) {
	var repoName string
	if app.Experimental {
		repoName = app.SelfExperimentalBintrayRepo
	} else {
		repoName = app.SelfBintrayRepo
	}

	return repo.NewBintrayClient(app.SelfBintraySubject, repoName, app.SelfBintrayPackage)
}

func NewSelfS3Client() (s3c repo.Repo) {
	return repo.NewS3Client(app.SelfPackageName)
}

func NewAppBtClient() (bc repo.Repo) {
	return repo.NewBintrayClient(app.BintraySubject, app.BintrayRepo, app.BintrayPackage)
}

func NewAppS3Client() (s3c repo.Repo) {
	return repo.NewS3Client(app.AppPackageName)
}
