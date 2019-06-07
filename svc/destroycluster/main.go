package main

import (
	"context"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// deleteStack deletes the respective CF stack
func deleteStack(name string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	svc := cloudformation.New(cfg)
	dsreq := svc.DeleteStackRequest(&cloudformation.DeleteStackInput{StackName: aws.String(name)})
	resp, err := dsreq.Send(context.TODO())
	if err != nil {
		return err
	}
	_ = resp
	// fmt.Printf("%v\n", resp.String())
	return nil
}

func handler() error {
	fmt.Printf("DEBUG:: destroy cluster start\n")
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		fmt.Println(err)
	}
	svc := s3.New(cfg)
	req := svc.ListObjectsRequest(&s3.ListObjectsInput{Bucket: aws.String("eks-cluster-meta")})
	resp, err := req.Send(context.TODO())
	if err != nil {
		fmt.Println(err)
	}
	for _, obj := range resp.Contents {
		fn := *obj.Key
		clusterIDs := strings.TrimSuffix(fn, ".json")
		ts := obj.LastModified
		ttl := time.Since(ts)
		switch {
		case ttl > 1*time.Minute:
			fmt.Printf("DEBUG:: tearing down cluster %v\n", clusterIDs)
			// err := deleteStack("eksctl-eksphemeral-nodegroup-ng-df9fe94e")
			// if err != nil {
			// 	fmt.Println(err)
			// }
			// err = deleteStack("eksctl-eksphemeral-cluster")
			// if err != nil {
			// 	fmt.Println(err)
			// }
		case ttl > 5*time.Minute:
			fmt.Printf("DEBUG:: sending owner XXX a warning re tear down of cluster %v\n", clusterIDs)
		default:
			fmt.Printf("DEBUG:: cluster %v still has %v min to live\n", clusterIDs, ttl)
		}
	}
	fmt.Printf("DEBUG:: destroy cluster done\n")
	return nil
}

func main() {
	lambda.Start(handler)
}
