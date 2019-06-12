package main

import (
	"context"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	clusterbucket := os.Getenv("CLUSTER_METADATA_BUCKET")
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
		cs, err := fetchClusterSpec(clusterbucket, cID)
		if err != nil {
			return serverError(err)
		}
		csjson, err := json.Marshal(cs)
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
			Body: string(csjson),
		}, nil
	}
	// if we have no specified cluster ID in the path, list all cluster IDs:
	fmt.Printf("DEBUG:: list cluster IDs start\n")
	// list all objects in the bucket:
	svc := s3.New(cfg)
	req := svc.ListObjectsRequest(&s3.ListObjectsInput{Bucket: &clusterbucket})
	resp, err := req.Send(context.TODO())
	if err != nil {
		return serverError(err)
	}
	clusterIDs := []string{}
	// get the content of all objects (cluster specs) in the bucket:
	for _, obj := range resp.Contents {
		fn := *obj.Key
		clusterIDs = append(clusterIDs, strings.TrimSuffix(fn, ".json"))
	}
	clusteridsjson, err := json.Marshal(clusterIDs)
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
		Body: string(clusteridsjson),
	}, nil
}

func main() {
	lambda.Start(handler)
}
