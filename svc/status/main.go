package main

import (
	"context"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	fmt.Println(err.Error())
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
		Body: fmt.Sprintf("%v", err.Error()),
	}, nil
}

// fetchCluster returns the cluster spec of the cluster with given ID
func fetchCluster(bucket, id string) (string, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", err
	}
	downloader := s3manager.NewDownloader(cfg)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(id + ".json"),
	})
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("DEBUG:: status start\n")
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return serverError(err)
	}
	// validate cluster ID or list lookup in URL path:
	if _, ok := request.PathParameters["clusterid"]; !ok {
		return serverError(fmt.Errorf("Unknown cluster status query. Either specify a cluster ID or _ for listing all clusters."))
	}
	cID := request.PathParameters["clusterid"]
	// return info on specified cluster if we have an cluster ID in the URL path component:
	if cID != "*" {
		fmt.Printf("DEBUG:: cluster info lookup for ID %v start\n", cID)
		clusterspec, err := fetchCluster("eks-cluster-meta", cID)
		if err != nil {
			return serverError(err)
		}
		fmt.Printf("DEBUG:: cluster info lookup done\n")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: clusterspec,
		}, nil
	}
	// if we have no specified cluster ID in the path, list all cluster IDs:
	fmt.Printf("DEBUG:: S3 bucket listing start\n")
	svc := s3.New(cfg)
	req := svc.ListObjectsRequest(&s3.ListObjectsInput{Bucket: aws.String("eks-cluster-meta")})
	resp, err := req.Send(context.TODO())
	if err != nil {
		return serverError(err)
	}
	fmt.Printf("DEBUG:: S3 bucket listing done\n")

	fmt.Printf("DEBUG:: list cluster IDs start\n")
	clusterIDs := []string{}
	// get all objects in the bucket:
	for _, obj := range resp.Contents {
		fn := *obj.Key
		clusterIDs = append(clusterIDs, strings.TrimSuffix(fn, ".json"))
	}
	js, err := json.Marshal(clusterIDs)
	if err != nil {
		return serverError(err)
	}
	fmt.Printf("DEBUG:: list cluster IDs done\n")

	fmt.Printf("DEBUG:: status done\n")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(js),
	}, nil
}

func main() {
	lambda.Start(handler)
}
