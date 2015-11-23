package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mgutz/logxi/v1"
)

var (
	logger = log.New("awss3get")
)

const (
	s3Region     = "us-west-2"
	buildsBucket = "ipowow-builds"
)

var p struct {
	bucket    string
	path      string
	outputDir string
	awsRegion string
}

func init() {
	flag.StringVar(&p.path, "path", "", "Path")
	flag.StringVar(&p.outputDir, "output", "", "Output directory")
	flag.StringVar(&p.bucket, "bucket", buildsBucket, "bucket")
	flag.StringVar(&p.awsRegion, "aws-region", s3Region, "AWS region")
}

func main() {
	flag.Parse()

	if err := validateParams(); err != nil {
		logger.Fatal("Invalid parameters", "err", err)
		os.Exit(1)
	}

	if err := getFromS3(); err != nil {
		logger.Fatal("Failed to download", "err", err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func validateParams() error {
	if p.bucket == "" {
		return errors.New("no bucket provided")
	}

	if p.path == "" {
		return errors.New("no path provided")
	}

	if p.outputDir == "" {
		return errors.New("no output directory provided")
	}

	return nil
}

func getFromS3() error {

	// creds := &ec2rolecreds.EC2RoleProvider{
	// 	client: &http.Client{Timeout: 10 * time.Second},
	// 	window: 0,
	// }

	s3svc := s3.New(&aws.Config{
		Region:      aws.String(s3Region),
		Credentials: ec2rolecreds.NewCredentials(nil, 5*time.Minute),
	})

	params := &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(p.path),
	}
	resp, err := s3svc.GetObject(params)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(p.outputDir, getFilename(p.path))
	artifact, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer artifact.Close()

	if _, err := io.Copy(artifact, resp.Body); err != nil {
		return err
	}

	return nil
}

func getFilename(path string) string {
	_, file := filepath.Split(path)
	return file
}
