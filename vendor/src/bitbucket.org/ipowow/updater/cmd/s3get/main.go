package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mgutz/logxi/v1"
)

var (
	logger = log.New("s3get")
)

const (
	s3Region     = "us-west-2"
	buildsBucket = "ipowow-builds"
)

var p struct {
	bucket       string
	path         string
	outputDir    string
	awsAccessKey string
	awsSecretKey string
	awsRegion    string
}

func init() {
	flag.StringVar(&p.path, "path", "", "Path")
	flag.StringVar(&p.outputDir, "output", "", "Output directory")
	flag.StringVar(&p.awsAccessKey, "aws-access-key", os.Getenv("S3GET_ACCESS_KEY"), "AWS access key")
	flag.StringVar(&p.awsSecretKey, "aws-secret-key", os.Getenv("S3GET_SECRET_KEY"), "AWS secret key")
	flag.StringVar(&p.awsRegion, "aws-region", s3Region, "AWS region")
	flag.StringVar(&p.bucket, "bucket", buildsBucket, "bucket")
}

func main() {
	flag.Parse()

	if err := validateParams(); err != nil {
		logger.Fatal("Invalid parameters", "err", err.Error())
		os.Exit(1)
	}

	if err := getFromS3(); err != nil {
		logger.Fatal("Failed to download", "err", err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func validateParams() error {

	if p.awsAccessKey == "" || p.awsSecretKey == "" {
		return errors.New("AWS credentials not provided")
	}

	if p.bucket == "" {
		return errors.New("bucket not provided")
	}

	if p.path == "" {
		return errors.New("path not provided")
	}

	if p.outputDir == "" {
		return errors.New("output directory not provided")
	}

	return nil
}

func getFromS3() error {

	s3svc := s3.New(&aws.Config{
		Region:      aws.String(p.awsRegion),
		Credentials: credentials.NewStaticCredentials(p.awsAccessKey, p.awsSecretKey, ""),
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

	logger.Info("File copied!", "destination", outputPath)
	return nil
}

func getFilename(path string) string {
	_, file := filepath.Split(path)
	return file
}
