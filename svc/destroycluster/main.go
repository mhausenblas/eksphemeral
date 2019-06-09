package main

import (
	"context"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
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

// fetchClusterSpec returns the cluster spec
// in a given bucket, with a given cluster ID
func fetchClusterSpec(bucket, clusterid string) (ClusterSpec, error) {
	ccr := ClusterSpec{}
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return ccr, err
	}
	downloader := s3manager.NewDownloader(cfg)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(clusterid + ".json"),
	})
	if err != nil {
		return ccr, err
	}
	err = json.Unmarshal(buf.Bytes(), &ccr)
	if err != nil {
		return ccr, err
	}
	return ccr, nil
}

// rmClusterSpec delete the cluster spec JSON doc
// in the metadata bucket and with that effectively
// states the cluster doesn't exist anymore.
func rmClusterSpec(bucket, clusterid string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	// Create S3 service client
	svc := s3.New(cfg)
	req := svc.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(clusterid + ".json"),
	})
	_, err = req.Send(context.Background())
	if err != nil {
		return err
	}
	return nil
}

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

// lookupStacks returns the control plane stack name
// and the dataplane stack name via matching labels.
func lookupStack(clustername string) (string, string, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", "", err
	}
	svc := cloudformation.New(cfg)
	var activeStacks = []cloudformation.StackStatus{"CREATE_COMPLETE"}
	lsreq := svc.ListStacksRequest(&cloudformation.ListStacksInput{StackStatusFilter: activeStacks})
	if err != nil {
		return "", "", err
	}
	lsresp, err := lsreq.Send(context.TODO())
	if err != nil {
		return "", "", err
	}
	// iterate over active stacks to find the two eksctl created, by label
	cpstack, dpstack := "", ""
	for _, stack := range lsresp.StackSummaries {
		dsreq := svc.DescribeStacksRequest(&cloudformation.DescribeStacksInput{StackName: stack.StackName})
		if err != nil {
			return "", "", err
		}
		dsresp, err := dsreq.Send(context.TODO())
		if err != nil {
			return "", "", err
		}
		// fmt.Printf("DEBUG:: checking stack %v if it has label with cluster name %v\n", *dsresp.Stacks[0].StackName, clustername)
		cnofstack := tagValueOf(dsresp.Stacks[0], "eksctl.cluster.k8s.io/v1alpha1/cluster-name")
		if cnofstack != "" && cnofstack == clustername {
			switch {
			case tagValueOf(dsresp.Stacks[0], "alpha.eksctl.io/nodegroup-name") != "":
				dpstack = *dsresp.Stacks[0].StackName
			default:
				cpstack = *dsresp.Stacks[0].StackName
			}
		}
	}
	fmt.Printf("DEBUG:: found control plane stack %v and data plane stack %v for cluster %v\n", cpstack, dpstack, clustername)
	return cpstack, dpstack, nil
}

// tagValueOf searches through the tags of a CF stack and
// returns the value for the provided key
func tagValueOf(stack cloudformation.Stack, key string) string {
	for _, tag := range stack.Tags {
		tagk := *tag.Key
		if tagk == key {
			return *tag.Value
		}
	}
	return ""
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
		ttl := time.Since(*ts)
		cs, err := fetchClusterSpec("eks-cluster-meta", clusterID)
		t0 := time.Duration(cs.Timeout) * time.Minute
		switch {
		case ttl > t0:
			fmt.Printf("DEBUG:: tearing down cluster %v\n", clusterID)
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
		case ttl > t0-5*time.Minute && ttl <= t0-10*time.Minute:
			fmt.Printf("DEBUG:: sending owner XXX a warning re tear down of cluster %v\n", clusterID)
		default:
			fmt.Printf("DEBUG:: cluster %v is %v min old\n", clusterID, ttl.Minutes())
		}
	}
	fmt.Printf("DEBUG:: destroy cluster done\n")
	return nil
}

func main() {
	lambda.Start(handler)
}
