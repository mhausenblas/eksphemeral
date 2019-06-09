package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

// fetchClusterSpec returns the cluster spec
// in a given bucket, with a given cluster ID
func fetchClusterSpec(bucket, clusterid string) (ClusterSpec, error) {
	ccr := ClusterSpec{}
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return ccr, err
	}
	downloader := s3manager.NewDownloader(cfg)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(clusterid + ".json"),
	})
	if err != nil {
		return ccr, err
	}
	err = json.Unmarshal(buf.Bytes(), &ccr)
	if err != nil {
		return ccr, err
	}
	return ccr, nil
}

// rmClusterSpec delete the cluster spec JSON doc
// in the metadata bucket and with that effectively
// states the cluster doesn't exist anymore.
func rmClusterSpec(bucket, clusterid string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	// Create S3 service client
	svc := s3.New(cfg)
	req := svc.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(clusterid + ".json"),
	})
	_, err = req.Send(context.Background())
	if err != nil {
		return err
	}
	return nil
}
