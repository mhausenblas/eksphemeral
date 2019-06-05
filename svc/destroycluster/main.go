package main

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler() error {
	fmt.Printf("DEBUG:: destroy start\n")
	//TODO: parse params, shell out to eksctl
	fmt.Printf("DEBUG:: destroy done\n")
	return nil
}

func main() {
	lambda.Start(handler)
}
