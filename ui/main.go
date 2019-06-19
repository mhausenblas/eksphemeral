package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
	// ClusterDetails is only valid for lookup of individual clusters,
	// that is, when user does, for example, a eksp l CLUSTERID. It
	// holds info such as cluster status and config
	ClusterDetails map[string]string `json:"details"`
}

var ekspcp string

func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/create", CreateCluster)
	log.Println("EKSPhemeral UI up and running")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func init() {
	ekspcp, ok := os.LookupEnv("EKSPHEMERAL_URL")
	if !ok {
		fmt.Println("Can't start up, please set the EKSPHEMERAL_URL environment variable, pointing to the EKSphemeral control plane endpoint!")
		os.Exit(1)
	}
}

// CreateCluster sanitizes user input, provisions the EKS cluster using the
// Fargate CLI, and invokes the create/ endpoint in the EKSphemeral control
// plane, returning the result to the caller
func CreateCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`Allow: ` + "POST"))
		return
	}
	// provision cluster using Fargate CLI:
	shellout("fargate", "task", "run", "eksctl",
		"--image quay.io/mhausenblas/eksctl:base",
		"--region "+AWSRegion,
		"--env AWS_ACCESS_KEY_ID="+AWSAccessKeyID,
		"--env AWS_SECRET_ACCESS_KEY="+AWSSecretAccessKey,
		"--env AWS_DEFAULT_REGION="+AWSRegion,
		"--env CLUSTER_NAME="+csname,
		"--env NUM_WORKERS="+csnumworkers,
		"--env KUBERNETES_VERSION="+csk8sv,
		"--security-group-id "+sgID,
	)

	// create cluster spec in control plane
	c := &http.Client{
		Timeout: time.Second * 10,
	}
	clusterspec := ClusterSpec{
		Name:        csname,
		NumWorkers:  csnumworkers,
		KubeVersion: csk8sv,
		Timeout:     cstimeout,
		Owner:       csowner,
	}
	req, err := json.Marshal(clusterspec)
	c.Post(ekspcp, "application/json", bytes.NewBuffer(req))
}

func getDefaultSecurityGroup() (string, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return "", err
	}
	svc := ec2.New(cfg)

	vpcreq := svc.DescribeVpcsRequest(&ec2.DescribeVpcsInput{})
	_, err = sgreq.Send(context.TODO())
	if err != nil {
		return "", err
	}

	sgreq := svc.DescribeSecurityGroupsRequest(&ec2.DescribeSecurityGroupsInput{})
	_, err = sgreq.Send(context.TODO())
	if err != nil {
		return "", err
	}
	return "", nil

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

// pinfo writes msg in light blue to stderr
// see also https://misc.flogisoft.com/bash/tip_colors_and_formatting
func pinfo(msg string) {
	_, _ = fmt.Fprintf(os.Stdout, "\x1b[94m%v\x1b[0m\n", msg)
}

// perr writes message (and optionally error) in light red to stderr
// see also https://misc.flogisoft.com/bash/tip_colors_and_formatting
func perr(msg string, err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v: \x1b[91m%v\x1b[0m\n", msg, err)
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "\x1b[91m%v\x1b[0m\n", msg)
}
