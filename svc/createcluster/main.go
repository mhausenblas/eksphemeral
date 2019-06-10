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
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

// ClusterSpec represents the parameters for eksctl,
// TTL, and ownership of a cluster.
type ClusterSpec struct {
	// Name specifies the cluster name
	Name string `json:"name"`
	// NumWorkers specifies the number of worker nodes, defaults to 1
	NumWorkers int `json:"numworkers"`
	// KubeVersion  specifies the Kubernetes version to use, defaults to `1.12`
	KubeVersion string `json:"kubeversion"`
	// Timeout specifies the timeout in minutes, after which the cluster is destroyed, defaults to 10
	Timeout int `json:"timeout"`
	// Owner specifies the email address of the owner (will be notified when cluster is created and 5 min before destruction)
	Owner string `json:"owner"`
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
	clusterbucket := os.Getenv("CLUSTER_METADATA_BUCKET")
	fmt.Println("DEBUG:: create start")
	// parse params:
	ccr := ClusterSpec{
		Name:        "unknown",
		NumWorkers:  1,
		KubeVersion: "1.12",
		Timeout:     10,
		Owner:       "nobody@example.com",
	}
	// Unmarshal the JSON payload in the POST:
	err := json.Unmarshal([]byte(request.Body), &ccr)
	if err != nil {
		return serverError(err)
	}
	fmt.Println("DEBUG:: parsing input cluster spec from HTTP POST payload done")
	fmt.Printf("Creating %v, a %v cluster with %v nodes for %v minutes which is owned by %v and adding a respective entry to bucket %v\n", ccr.Name, ccr.KubeVersion, ccr.NumWorkers, ccr.Timeout, ccr.Owner, clusterbucket)
	// create unique cluster ID:
	clusterID, err := uuid.NewV4()
	if err != nil {
		return serverError(err)
	}
	// store cluster spec in S3 bucket keyed by cluster ID:
	jsonfilename := clusterID.String() + ".json"
	err = upload(region, clusterbucket, jsonfilename, string([]byte(request.Body)))
	if err != nil {
		return serverError(err)
	}
	fmt.Println("DEBUG:: state sync done")
	// if the owner shared their mail address, let's inform them that
	// the cluster is ready now:
	if ccr.Owner != "" {
		fmt.Println("DEBUG:: begin inform owner")
		fmt.Printf("Sending owner %v an info concerning the creation of cluster %v\n", ccr.Owner, clusterID)
		subject := fmt.Sprintf("EKS cluster %v created and available", ccr.Name)
		body := fmt.Sprintf("Hello there,\n\nThis is to inform you that your EKS cluster %v (cluster ID %v) is now available for you to use.\n\nHave a nice day,\nEKSphemeral", ccr.Name, clusterID)
		err := informOwner(ccr.Owner, subject, body)
		if err != nil {
			return serverError(err)
		}
		fmt.Println("DEBUG:: inform owner done")
	}
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
