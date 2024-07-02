package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3 implements the Storer interface for Amazon S3 Bucket
type S3 struct {
	BasePath string
}

func NewS3Storage(basePath string) *S3 {
	return &S3{BasePath: basePath}
}

func (s *S3) Save(batchDir, filename string, bf bytes.Buffer) error {
	key := filepath.Join(s.BasePath, batchDir, filename)

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))

	svc := s3.New(sess)

	// Upload the data to S3
	_, err := svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String("myBucket"),
		Key:    aws.String(key),
		Body:   bytes.NewReader(bf.Bytes()),
		Metadata: map[string]*string{
			"encoder": aws.String("default"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to put object to S3: %v", err)
	}

	return nil
}

func (s *S3) Read(key string) (bytes.Buffer, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))

	svc := s3.New(sess)

	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String("myBucket"),
		Key:    aws.String(key),
	})

	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to get object: %v", err)
	}
	defer result.Body.Close()

	// Read the object data into a buffer
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, result.Body); err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to read object data: %v", err)
	}

	return *buf, nil
}

func (s *S3) Delete(key string) error {

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))

	svc := s3.New(sess)

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String("myBucket"),
		Key:    aws.String(key),
	})

	if err != nil {
		return err
	}

	return nil

}

func (s *S3) List(ctx context.Context, delim, blockBatch string, timeout int, returnFunc ListReturnFunc) ([]string, error) {
	return []string{}, nil
}

func (s *S3) ReadBatch(readItems []ReadItem) (map[string][]string, error) {
	// Implement the ReadBatch method
	return nil, nil
}
