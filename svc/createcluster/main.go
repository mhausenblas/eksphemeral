package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

// ClusterSpec represents the parameters for eksctl,
// as cluster metadata including owner and how long the cluster
// still has to live.
type ClusterSpec struct {
	// ID is a unique identifier for the cluster
	ID string `json:"id"`
	// Name specifies the cluster name
	Name string `json:"name"`
	// NumWorkers specifies the number of worker nodes, defaults to 1
	NumWorkers int `json:"numworkers"`
	// KubeVersion  specifies the Kubernetes version to use, defaults to `1.12`
	KubeVersion string `json:"kubeversion"`
	// Timeout specifies the timeout in minutes, after which the cluster
	// is destroyed, defaults to 10
	Timeout int `json:"timeout"`
	// Timeout specifies the cluster time to live in minutes.
	// In other words: the remaining time the cluster has before it is destroyed
	TTL int `json:"ttl"`
	// Owner specifies the email address of the owner (will be notified when cluster is created and 5 min before destruction)
	Owner string `json:"owner"`
	// CreationTime is the UTC timestamp of when the cluster was created
	// which equals the point in time of the creation of the respective
	// JSON representation of the cluster spec as an object in the metadata
	// bucket
	CreationTime string `json:"created"`
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
	cs := ClusterSpec{
		ID:           "",
		Name:         "unknown",
		NumWorkers:   1,
		KubeVersion:  "1.12",
		Timeout:      10,
		TTL:          10,
		Owner:        "nobody@example.com",
		CreationTime: "",
	}
	// Unmarshal the JSON payload in the POST:
	err := json.Unmarshal([]byte(request.Body), &cs)
	if err != nil {
		return serverError(err)
	}
	fmt.Println("DEBUG:: parsing input cluster spec from HTTP POST payload done")
	fmt.Printf("Creating %v, a %v cluster with %v nodes for %v minutes which is owned by %v and adding a respective entry to bucket %v\n", cs.Name, cs.KubeVersion, cs.NumWorkers, cs.Timeout, cs.Owner, clusterbucket)
	// create unique cluster ID and assign:
	clusterID, err := uuid.NewV4()
	if err != nil {
		return serverError(err)
	}
	cs.ID = clusterID.String()
	cs.CreationTime = fmt.Sprintf("%v", time.Now().Unix())
	fmt.Printf("DEBUG:: created cluster spec %v", cs)
	// store cluster spec in S3 bucket keyed by cluster ID:
	err = upload(region, clusterbucket, clusterID.String()+".json", string([]byte(request.Body)))
	if err != nil {
		return serverError(err)
	}
	fmt.Println("DEBUG:: state sync done")
	// if the owner shared their mail address, let's inform them that
	// the cluster is ready now:
	if cs.Owner != "" {
		fmt.Println("DEBUG:: begin inform owner")
		fmt.Printf("Attempting to send owner %v an info concerning the creation of cluster %v\n", cs.Owner, cs.ID)
		subject := fmt.Sprintf("EKS cluster %v created and available", cs.Name)
		body := fmt.Sprintf("Hello there,\n\nThis is to inform you that your EKS cluster %v (cluster ID %v) is now available for you to use.\n\nHave a nice day,\nEKSphemeral", cs.Name, cs.ID)
		err := informOwner(cs.Owner, subject, body)
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
		Body: cs.ID,
	}, nil
}

func main() {
	lambda.Start(handler)
}
