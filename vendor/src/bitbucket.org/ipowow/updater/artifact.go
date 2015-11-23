package updater

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/asaskevich/govalidator"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	cr "github.com/coreroller/coreroller/updaters/lib/go"
)

const (
	s3Region    = "us-west-2"
	s3Bucket    = "ipowow-builds"
	s3AccessKey = "AKIAJFAMYE5XXMV4BASA"
	s3SecretKey = "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"
)

type Artifact struct {
	ExecutableDir    string `valid:"required"`
	ExecutablePrefix string `valid:"required"`
	Version          string `valid:"required"`
	OmahaURL         string
	InstanceID       string
	AppID            string
	GroupID          string
}

func (a Artifact) Download(update *cr.Update) (string, error) {
	s3svc := s3.New(session.New(&aws.Config{
		Region:      aws.String(s3Region),
		Credentials: credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, ""),
	}))

	key := fmt.Sprintf("/%s/%s", a.ExecutablePrefix, update.Filename)
	params := &s3.GetObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(key),
	}
	resp, err := s3svc.GetObject(params)
	if err != nil {
		return "", err
	}

	artifactPath := filepath.Join(a.ExecutableDir, a.VersionedArtifact(update.Version))
	artifact, err := os.Create(artifactPath)
	if err != nil {
		return "", err
	}
	artifact.Chmod(0755)
	defer artifact.Close()

	if _, err := io.Copy(artifact, resp.Body); err != nil {
		return "", err
	}

	return artifactPath, nil
}

func (a Artifact) Install(artifactPath string) error {
	linkPath := filepath.Join(a.ExecutableDir, a.ExecutablePrefix)
	_ = os.Remove(linkPath)

	return os.Symlink(artifactPath, linkPath)
}

func (a Artifact) Validate() error {
	_, err := govalidator.ValidateStruct(a)
	return err
}

func (a Artifact) VersionedArtifact(version string) string {
	if version == "latest" {
		return fmt.Sprintf("%s_latest", a.ExecutablePrefix)
	}
	return fmt.Sprintf("%s_v%s", a.ExecutablePrefix, version)
}

func (a Artifact) PackageURL() string {
	return fmt.Sprintf("https://s3.amazonaws.com/%s/%s/", s3Bucket, a.ExecutablePrefix)
}
