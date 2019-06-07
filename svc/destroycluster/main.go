package main

import (
	"context"
	"fmt"
	_ "image/jpeg"
	_ "image/png"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

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
	fmt.Printf("%v\n", resp.String())
	return nil
}

func handler() error {
	fmt.Printf("DEBUG:: destroy cluster start\n")
	err := deleteStack("eksctl-eksphemeral-nodegroup-ng-df9fe94e")
	if err != nil {
		fmt.Println(err)
	}
	err = deleteStack("eksctl-eksphemeral-cluster")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("DEBUG:: destroy cluster done\n")
	return nil
}

func main() {
	lambda.Start(handler)
}
