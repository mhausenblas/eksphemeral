package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

// fetchClusterSpec returns the cluster spec
// in a given bucket, with a given cluster ID
func fetchClusterSpec(clusterbucket, clusterid string) (ClusterSpec, error) {
	cs := ClusterSpec{}
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return cs, err
	}
	downloader := s3manager.NewDownloader(cfg)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(clusterbucket),
		Key:    aws.String(clusterid + ".json"),
	})
	if err != nil {
		return cs, err
	}
	err = json.Unmarshal(buf.Bytes(), &cs)
	if err != nil {
		return cs, err
	}
	return cs, nil
}

// storeClusterSpec stores the cluster spec in a given bucket
func storeClusterSpec(clusterbucket string, cs ClusterSpec) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	csjson, err := json.Marshal(cs)
	if err != nil {
		return err
	}
	uploader := s3manager.NewUploader(cfg)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(clusterbucket),
		Key:    aws.String(cs.ID + ".json"),
		Body:   strings.NewReader(string(csjson)),
	})
	return err
}

// rmClusterSpec delete the cluster spec JSON doc
// in the metadata bucket and with that effectively
// states the cluster doesn't exist anymore
func rmClusterSpec(clusterbucket, clusterid string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	svc := s3.New(cfg)
	req := svc.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(clusterbucket),
		Key:    aws.String(clusterid + ".json"),
	})
	_, err = req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("DEBUG:: removed cluster spec for cluster with ID %v", clusterid)
	return nil
}
