package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

func updateTTL(clusterbucket string, cs ClusterSpec) error {
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

// getClusterAge returns the age of the cluster
func getClusterAge(cs ClusterSpec) (time.Duration, error) {
	ct, err := strconv.ParseInt(cs.CreationTime, 10, 64)
	if err != nil {
		return 0 * time.Minute, err
	}
	clusterage := time.Since(time.Unix(ct, 0))
	return clusterage, nil
}

func handler() error {
	fmt.Printf("DEBUG:: destroy cluster start\n")
	clusterbucket := os.Getenv("CLUSTER_METADATA_BUCKET")
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		fmt.Println(err)
		return err
	}
	svc := s3.New(cfg)
	fmt.Printf("Scanning bucket %v for cluster specs\n", clusterbucket)
	req := svc.ListObjectsRequest(&s3.ListObjectsInput{Bucket: &clusterbucket})
	resp, err := req.Send(context.TODO())
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Printf("DEBUG:: iterating over cluster specs:\n")
	for _, obj := range resp.Contents {
		fn := *obj.Key
		clusterID := strings.TrimSuffix(fn, ".json")
		cs, err := fetchClusterSpec(clusterbucket, clusterID)
		clusterage, err := getClusterAge(cs)
		if err != nil {
			fmt.Println(err)
			return err
		}
		ttl := time.Duration(cs.Timeout) * time.Minute
		headsuptime := ttl - 5*time.Minute
		fmt.Printf("DEBUG:: checking TTL of cluster %v:\n", clusterID)
		switch {
		case clusterage > ttl:
			fmt.Printf("Tearing down EKS cluster %v\n", clusterID)
			if err != nil {
				fmt.Println(err)
				return err
			}
			// data plane tear down:
			cpstack, dpstack, err := lookupStack(cs.Name)
			if err != nil {
				fmt.Println(err)
				return err
			}
			switch {
			// if this time around there's a stack
			// representing the data plane, delete it:
			case dpstack != "":
				err = deleteStack(dpstack)
				if err != nil {
					fmt.Println(err)
					return err
				}
			// if this time around there's no more stack
			// representing the data plane but there's still
			// a control plane stack, delete it:
			case dpstack == "" && cpstack != "":
				err = deleteStack(cpstack)
				if err != nil {
					fmt.Println(err)
					return err
				}
			// if this time around there's neither a stack
			// representing the data plane nor a control plane
			// stack, we're ready to delete the cluster spec entry
			// from the metadata bucket:
			case dpstack == "" && cpstack == "":
				rmClusterSpec(clusterbucket, clusterID)
			default:
				fmt.Printf("DEBUG:: seems both control and data plane stacks and all cluster metadata have been deleted, so this would be a NOP.\n")
			}
		case clusterage > headsuptime:
			if cs.Owner != "" {
				fmt.Printf("Attempting to send owner %v a warning concerning tear down of cluster %v\n", cs.Owner, clusterID)
				subject := fmt.Sprintf("EKS cluster %v shutting down in 5 min", cs.Name)
				body := fmt.Sprintf("Hello there,\n\nThis is to inform you that your EKS cluster %v (cluster ID %v) will shut down and all associated resources destroyed within the next few minutes.\n\nHave a nice day,\nEKSphemeral", cs.Name, clusterID)
				err := informOwner(cs.Owner, subject, body)
				if err != nil {
					fmt.Println(err)
					return err
				}
			}
		default:
			fmt.Printf("Cluster %v is %.0f min old\n", clusterID, clusterage.Minutes())
		}
		cs.TTL = int(clusterage.Minutes())
		updateTTL(clusterbucket, cs)
	}
	fmt.Printf("DEBUG:: destroy cluster done\n")
	return nil
}

func main() {
	lambda.Start(handler)
}
