package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func handler() error {
	fmt.Printf("DEBUG:: destroy cluster start\n")
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		fmt.Println(err)
		return err
	}
	svc := s3.New(cfg)
	req := svc.ListObjectsRequest(&s3.ListObjectsInput{Bucket: aws.String("eks-cluster-meta")})
	resp, err := req.Send(context.TODO())
	if err != nil {
		fmt.Println(err)
		return err
	}
	for _, obj := range resp.Contents {
		fn := *obj.Key
		clusterID := strings.TrimSuffix(fn, ".json")
		ts := obj.LastModified
		clusterage := time.Since(*ts)
		cs, err := fetchClusterSpec("eks-cluster-meta", clusterID)
		ttl := time.Duration(cs.Timeout) * time.Minute
		headsuptime := ttl - 5*time.Minute
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
			err = deleteStack(dpstack)
			if err != nil {
				fmt.Println(err)
				return err
			}
			err = deleteStack(cpstack)
			if err != nil {
				fmt.Println(err)
				return err
			}
			// control plane tear down:
			rmClusterSpec("eks-cluster-meta", clusterID)
		case clusterage > headsuptime:
			fmt.Printf("Sending owner %v a warning concerning tear down of cluster %v\n", cs.Owner, clusterID)
			subject := fmt.Sprintf("EKS cluster %v shutting down in 5 min", cs.Name)
			body := fmt.Sprintf("Hi there!\nThis is to inform you that your EKS cluster %v (cluster ID %v) will be shut down and all associated resources destroyed in 5 min.\n Have a nice day!", cs.Name, clusterID)
			err := informOwner(cs.Owner, subject, body)
			fmt.Println(err)
			return err
		default:
			fmt.Printf("Cluster %v is %.0f min old\n", clusterID, clusterage.Minutes())
		}
	}
	fmt.Printf("DEBUG:: destroy cluster done\n")
	return nil
}

func main() {
	lambda.Start(handler)
}
