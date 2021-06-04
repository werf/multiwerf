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

	uuid "github.com/satori/go.uuid"
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
	if debug() {
		fmt.Printf("-- S3Client.GetPackageVersions\n")
	}

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
	if debug() {
		fmt.Printf("-- S3Client.DownloadFiles version=%q dstDir=%q files=%#v\n", version, dstDir, files)
	}

	awsConfig := c.awsConfig()
	sess := session.Must(session.NewSession(awsConfig))
	downloader := s3manager.NewDownloader(sess)

	for _, fileName := range files {
		dstFilePath := filepath.Join(dstDir, fileName)
		tmpFilePath := fmt.Sprintf("%s.%s", dstFilePath, uuid.NewV4().String())
		key := releaseFileKey(version, fileName)

		err := func() error {
			dstFile, err := os.Create(tmpFilePath)
			if err != nil {
				return fmt.Errorf("unable to open file %q, %v", tmpFilePath, err)
			}
			defer func() {
				if err != nil {
					_ = dstFile.Close()
					_ = os.RemoveAll(tmpFilePath)
				}
			}()

			_, err = downloader.Download(dstFile, &s3.GetObjectInput{
				Bucket: aws.String(c.bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				return fmt.Errorf("downloading file %q failed: %s", tmpFilePath, err)
			}

			if err = dstFile.Close(); err != nil {
				return fmt.Errorf("unable to close file %q: %s", tmpFilePath, err)
			}

			if err = os.Rename(tmpFilePath, dstFilePath); err != nil {
				return fmt.Errorf("unable to rename %q to %q: %s", tmpFilePath, dstFilePath, err)
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c S3Client) GetFileContent(version string, fileName string) (string, error) {
	if debug() {
		fmt.Printf("-- S3Client.GetFileContent version=%q fileName=%q\n", version, fileName)
	}

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
