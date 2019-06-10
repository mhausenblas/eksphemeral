package main

import (
	"context"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

// fetchClusterSpec returns the cluster spec
// in a given bucket, with a given cluster ID
func fetchClusterSpec(bucket, clusterid string) (ClusterSpec, error) {
	cs := ClusterSpec{}
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return cs, err
	}
	downloader := s3manager.NewDownloader(cfg)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
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

// getClusterAge returns the age of the cluster,
// that is, the last modified field of the JSON file
// that contains the cluster spec.
func getClusterAge(bucket, clusterid string) (time.Duration, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return 0, err
	}
	svc := s3.New(cfg)
	req := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(clusterid + ".json"),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return 0, err
	}
	clusterage := time.Since(*resp.LastModified)
	return clusterage, nil
}

// storeClusterSpec stores the cluster spec
// in a given bucket, with a given cluster ID
func storeClusterSpec(bucket, clusterid string, cs ClusterSpec) error {
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
		Bucket: aws.String(bucket),
		Key:    aws.String(clusterid + ".json"),
		Body:   strings.NewReader(string(csjson)),
	})
	return err
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	clusterbucket := os.Getenv("CLUSTER_METADATA_BUCKET")
	fmt.Printf("DEBUG:: prolong start\n")
	// validate cluster ID:
	if _, ok := request.PathParameters["clusterid"]; !ok {
		return serverError(fmt.Errorf("Unknown cluster prolong request, please specify a valid cluster ID."))
	}
	cID := request.PathParameters["clusterid"]
	// validate time to prolong cluster TTL:
	timeInMinParam := request.PathParameters["timeinmin"]
	timeInMin, err := strconv.Atoi(timeInMinParam)
	if err != nil {
		return serverError(fmt.Errorf("Invalid prolong request, please specify the time in minutes as a plain integer."))
	}
	fmt.Printf("DEBUG:: updating cluster with ID %v start\n", cID)
	clusterspec, err := fetchClusterSpec(clusterbucket, cID)
	if err != nil {
		return serverError(err)
	}
	age, err := getClusterAge(clusterbucket, cID)
	if err != nil {
		return serverError(err)
	}
	fmt.Printf("DEBUG:: cluster is %.0f min old\n", age)
	clusterspec.Timeout = clusterspec.Timeout - int(age.Minutes()) + timeInMin
	fmt.Printf("DEBUG:: new TTL is %v min starting now\n", clusterspec.Timeout)
	err = storeClusterSpec(clusterbucket, cID, clusterspec)
	if err != nil {
		return serverError(err)
	}
	fmt.Printf("DEBUG:: updating cluster done\n")
	fmt.Printf("DEBUG:: prolong done\n")
	successmsg := fmt.Sprintf("Successfully prolonged the lifetime of cluster %v for %v minutes. New TTL is %v min starting now!", cID, timeInMin, clusterspec.Timeout)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: successmsg,
	}, nil
}

func main() {
	lambda.Start(handler)
}
