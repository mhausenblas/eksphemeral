package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	uuid "github.com/satori/go.uuid"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

// ClusterCreateRequest has the params for eksctl
type ClusterCreateRequest struct {
	// NumWorkers specifies the number of worker nodes, defaults to 1
	NumWorkers int `json:"numworkers"`
	// KubeVersion  specifies the Kubernetes version to use, defaults to `1.12`
	KubeVersion string `json:"kubeversion"`
	// Timeout specifies the timeout in minutes, after which the cluster is destroyed, defaults to 10
	Timeout int `json:"timeout"`
	// Owner specifies the email address of the owner (will be notified when cluster is created and 5 min before destruction)
	Owner string `json:"owner"`
}

func fetcheksctl() error {
	eksctlbin, err := os.Create("/tmp/eksctl")
	if err != nil {
		return err
	}
	defer eksctlbin.Close()
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	downloader := s3manager.NewDownloader(cfg)
	_, err = downloader.Download(eksctlbin, &s3.GetObjectInput{
		Bucket: aws.String("eks-cluster-meta"),
		Key:    aws.String("eksctl"),
	})
	if err != nil {
		return err
	}
	return nil
}

func upload(region, bucket, jsonfilename, content string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	uploader := s3manager.NewUploader(cfg)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(jsonfilename),
		Body:   strings.NewReader(content),
	})
	return err
}

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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	region := os.Getenv("AWS_REGION")
	clusterbucket := "eks-cluster-meta"
	fmt.Println("DEBUG:: create start")
	// parse params:
	ccr := ClusterCreateRequest{
		NumWorkers:  1,
		KubeVersion: "1.12",
		Timeout:     10,
		Owner:       "nobody@example.com",
	}

	// Unmarshal the json, return 404 if error
	err := json.Unmarshal([]byte(request.Body), &ccr)
	if err != nil {
		return serverError(err)
	}
	fmt.Printf("Creating a %v cluster with %v nodes for %v minutes which is owned by %v\n", ccr.KubeVersion, ccr.NumWorkers, ccr.Timeout, ccr.Owner)
	fmt.Println("DEBUG:: parse done")

	// create unique cluster ID:
	clusterID, err := uuid.NewV4()
	if err != nil {
		return serverError(err)
	}
	// shell out to eksctl with above params:
	err = fetcheksctl()
	if err != nil {
		return serverError(err)
	}
	fmt.Println("DEBUG:: eksctl done")

	// store cluster spec in S3 bucket keyed by cluster ID:
	jsonfilename := clusterID.String() + ".json"
	err = upload(region, clusterbucket, jsonfilename, string([]byte(request.Body)))
	if err != nil {
		return serverError(err)
	}
	fmt.Println("DEBUG:: state sync done")

	fmt.Println("DEBUG:: create done")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: clusterID.String(),
	}, nil
}

func main() {
	lambda.Start(handler)
}
