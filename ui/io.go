package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// jsonResponse wraps a message with a JSON header and writes it out
func jsonResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprint(w, message)
}

// plainResponse wraps a message with a text/plain header and writes it out
func plainResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	fmt.Fprint(w, message)
}

// getDefaults returns creds and default configs
func getDefaults() (awsAccessKeyID, awsSecretAccessKey, awsRegion, defaultSG, ekspcp string) {
	awsAccessKeyID, ok := os.LookupEnv("AWS_ACCESS_KEY_ID")
	if !ok {
		perr("Please set the AWS_ACCESS_KEY_ID environment variable!", nil)
		os.Exit(1)
	}
	awsSecretAccessKey, ok = os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	if !ok {
		perr("Please set the AWS_SECRET_ACCESS_KEY environment variable!", nil)
		os.Exit(1)
	}
	awsRegion, ok = os.LookupEnv("AWS_DEFAULT_REGION")
	if !ok {
		perr("Please set the AWS_DEFAULT_REGION environment variable!", nil)
		os.Exit(1)
	}
	ekspcp, ok = os.LookupEnv("EKSPHEMERAL_URL")
	if !ok {
		perr("Please set the EKSPHEMERAL_URL environment variable, pointing to the EKSphemeral control plane endpoint!", nil)
		os.Exit(1)
	}
	defaultSG, err := getDefaultSecurityGroup()
	if err != nil {
		perr("Can't start up since I'm unable to determine the default security group: %v", err)
		os.Exit(1)
	}
	// pinfo(fmt.Sprintf("Using %v as the default security group", defaultSG))
	return
}

// getDefaultSecurityGroup returns the default security group
func getDefaultSecurityGroup() (string, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", err
	}
	svc := ec2.New(cfg)
	vpcreq := svc.DescribeVpcsRequest(&ec2.DescribeVpcsInput{})
	vpcres, err := vpcreq.Send(context.TODO())
	if err != nil {
		return "", err
	}
	defaultVPC := ""
	for _, vpc := range vpcres.Vpcs {
		if *vpc.IsDefault {
			defaultVPC = *vpc.VpcId
			break
		}
	}

	sgreq := svc.DescribeSecurityGroupsRequest(&ec2.DescribeSecurityGroupsInput{})
	sgres, err := sgreq.Send(context.TODO())
	if err != nil {
		return "", err
	}
	defaultSG := ""
	for _, sg := range sgres.SecurityGroups {
		// fmt.Printf("%v\n", *sg.GroupId)
		if *sg.VpcId == defaultVPC {
			defaultSG = *sg.GroupId
			break
		}
	}
	return defaultSG, nil
}

// shellout shells out to execute a command with a variable number
// of arguments and prints the literal results from both stdout and stderr
func shellout(command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		perr("Can't shell out due to issues with stderr:", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		perr("Can't shell out due to issues with stdout:", err)
		return
	}
	err = cmd.Start()
	if err != nil {
		perr("Can't shell out due to issues with starting command:", err)
		return
	}
	go echo(stderr)
	go echo(stdout)
	err = cmd.Wait()
	if err != nil {
		perr("Something bad happened after command completed:", err)
	}
}

// bshellout shells out to execute a command with a variable number
// of arguments in a blocking manner. It returns the combined literal
// output from both stdout and stderr
func bshellout(command string, args ...string) string {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		perr("Can't shell out due to issues with stderr:", err)
		return ""
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		perr("Can't shell out due to issues with stdout:", err)
		return ""
	}
	err = cmd.Start()
	if err != nil {
		perr("Can't shell out due to issues with starting command:", err)
		return ""
	}
	go echo(stderr)
	result := slurp(stdout)
	err = cmd.Wait()
	if err != nil {
		perr("Something bad happened after command completed:", err)
	}
	return result
}

// echo prints the character stream as a set of lines
func echo(rc io.ReadCloser) {
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}

// slurp collects the character stream into one string
func slurp(rc io.ReadCloser) string {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		buf.WriteString(scanner.Text())
	}
	return buf.String()
}
