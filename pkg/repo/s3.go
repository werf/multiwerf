package repo

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const DefaultS3Endpoint = "s3.yandexcloud.net"
const DefaultS3Region = "ru-central1"
const DefaultS3ReleasesFolder = "targets/releases"

type S3Client struct {
	bucket string
}

func NewS3Client(bucket string) (c S3Client) {
	return S3Client{bucket: bucket}
}

func (c S3Client) GetPackageVersions() ([]string, error) {
	awsConfig := c.awsConfig()
	sess := session.Must(session.NewSession(awsConfig))
	svc := s3.New(sess, awsConfig)

	res, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(fmt.Sprintf("%s/v", DefaultS3ReleasesFolder)),
	})
	if err != nil {
		return nil, fmt.Errorf("listing s3 bucket failed: %s", err)
	}

	var versions []string
	for _, obj := range res.Contents {
		// targets/releases/<semver>/*
		p := *obj.Key

		// trim trailing slash: targets/releases/<semver>/ => targets/releases/<semver>
		p = strings.TrimRight(p, "/")

		// skip release files
		dir, version := path.Split(p)
		if dir != "targets/releases/" {
			continue
		}

		versions = append(versions, version)
	}

	return versions, nil
}

func (c S3Client) DownloadFiles(version string, dstDir string, files map[string]string) error {
	awsConfig := c.awsConfig()
	sess := session.Must(session.NewSession(awsConfig))
	downloader := s3manager.NewDownloader(sess)

	var filesToRemove []string
	shouldBeRemoved := true
	defer func() {
		if shouldBeRemoved {
			for _, file := range filesToRemove {
				os.RemoveAll(file)
			}
		}
	}()

	for _, fileName := range files {
		dstFilePath := filepath.Join(dstDir, fileName)
		key := releaseFileKey(version, fileName)

		err := func() error {
			dstFile, err := os.Create(dstFilePath)
			if err != nil {
				return fmt.Errorf("unable to open file %q, %v", dstFilePath, err)
			}
			defer dstFile.Close()

			_, err = downloader.Download(dstFile, &s3.GetObjectInput{
				Bucket: aws.String(c.bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				return fmt.Errorf("downloading file %q failed: %s", key, err)
			}

			return nil
		}()

		if err != nil {
			return err
		}

		filesToRemove = append(filesToRemove, dstFilePath)
	}

	shouldBeRemoved = false

	return nil
}

func (c S3Client) GetFileContent(version string, fileName string) (string, error) {
	awsConfig := c.awsConfig()
	sess := session.Must(session.NewSession(awsConfig))
	downloader := s3manager.NewDownloader(sess)

	key := releaseFileKey(version, fileName)

	buff := &aws.WriteAtBuffer{}
	_, err := downloader.Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	return string(buff.Bytes()), err
}

func (c S3Client) awsConfig() *aws.Config {
	return &aws.Config{
		Endpoint:    aws.String(DefaultS3Endpoint),
		Region:      aws.String(DefaultS3Region),
		Credentials: credentials.AnonymousCredentials,
	}
}

func (c S3Client) String() string {
	return "s3"
}

func releaseFileKey(version, fileName string) string {
	return path.Join(DefaultS3ReleasesFolder, version, fileName)
}
