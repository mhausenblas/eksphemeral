package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

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
