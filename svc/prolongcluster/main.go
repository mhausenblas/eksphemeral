package main

import (
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

// storeClusterSpec stores the cluster spec in a given bucket
func storeClusterSpec(bucket string, cs ClusterSpec) error {
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
		Key:    aws.String(cs.ID + ".json"),
		Body:   strings.NewReader(string(csjson)),
	})
	return err
}

// getClusterAge returns the age of the cluster
func getClusterAge(cs ClusterSpec) (time.Duration, error) {
	ct, err := strconv.ParseInt(cs.CreationTime, 10, 64)
	if err != nil {
		return 0 * time.Minute, err
	}
	clusterage := time.Since(time.Unix(ct, 0))
	return clusterage, nil
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
	cs, err := fetchClusterSpec(clusterbucket, cID)
	if err != nil {
		return serverError(err)
	}
	age, err := getClusterAge(cs)
	if err != nil {
		return serverError(err)
	}
	cs.Timeout = cs.Timeout - int(age.Minutes()) + timeInMin
	fmt.Printf("DEBUG:: new TTL is %v min starting now\n", cs.Timeout)
	err = storeClusterSpec(clusterbucket, cs)
	if err != nil {
		return serverError(err)
	}
	fmt.Printf("DEBUG:: prolong done\n")
	successmsg := fmt.Sprintf("Successfully prolonged the lifetime of cluster %v for %v minutes", cID, timeInMin)
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
