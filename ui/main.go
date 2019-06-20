package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
var cscache map[string]ClusterSpec

func main() {
	cscache = make(map[string]ClusterSpec)
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/status", ListCluster)
	http.HandleFunc("/create", CreateCluster)
	http.HandleFunc("/prolong", ProlongCluster)
	http.HandleFunc("/configof", GetClusterConfig)
	log.Println("EKSPhemeral UI up and running")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

// ListCluster invokes the /status endpoint in the EKSphemeral control
// plane, returning the result to the caller
func ListCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`Allow: ` + "GET"))
		return
	}
	q := r.URL.Query()
	targetcluster := q.Get("cluster")

	if targetcluster != "*" { // cluster details
		cs, err := lookup(targetcluster) // try local cache
		if err == nil {
			csjson, err := json.Marshal(cs)
			if err != nil {
				perr("Can't marshal cluster spec data", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				jsonResponse(w, http.StatusInternalServerError, "Can't marshal cluster spec data")
				return
			}
			pinfo("Serving from cache")
			jsonResponse(w, http.StatusOK, string(csjson))
			return
		}
	}
	// either list all clusters or not cached yet
	_, _, _, _, ekspcp := getDefaults()
	pinfo(fmt.Sprintf("Using %v as the control plane endpoint", ekspcp))
	c := &http.Client{
		Timeout: time.Second * 30,
	}
	pres, err := c.Get(ekspcp + "/status/" + targetcluster)
	if err != nil {
		perr("Can't GET control plane for cluster status", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't GET control plane for cluster status")
		return
	}
	defer pres.Body.Close()
	body, err := ioutil.ReadAll(pres.Body)
	if err != nil {
		perr("Can't read control plane response for cluster status", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't read control plane response for cluster status")
		return
	}
	pinfo(fmt.Sprintf("Status for cluster: %v", string(body)))
	err = updateCache(string(body))
	if err != nil {
		perr("Can't update local cluster spec cache", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't update local cluster spec cache")
		return
	}
	jsonResponse(w, http.StatusOK, string(body))
}

// CreateCluster sanitizes user input, provisions the EKS cluster using the
// Fargate CLI, and invokes the /create endpoint in the EKSphemeral control
// plane, returning the result to the caller
func CreateCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`Allow: ` + "POST"))
		return
	}
	decoder := json.NewDecoder(r.Body)
	cs := ClusterSpec{}
	err := decoder.Decode(&cs)
	if err != nil {
		perr("Can't parse cluster spec from UI", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't parse cluster spec from UI")
		return
	}
	pinfo(fmt.Sprintf("From the web UI I got the following values for cluster create: %+v", cs))

	// provision cluster using Fargate CLI:
	awsAccessKeyID, awsSecretAccessKey, awsRegion, defaultSG, ekspcp := getDefaults()
	pinfo(fmt.Sprintf("Using %v as the control plane endpoint", ekspcp))
	shellout("sh", "-c", "fargate task run eksctl"+
		" --image quay.io/mhausenblas/eksctl:base"+
		" --region "+awsRegion+
		" --env AWS_ACCESS_KEY_ID="+awsAccessKeyID+
		" --env AWS_SECRET_ACCESS_KEY="+awsSecretAccessKey+
		" --env AWS_DEFAULT_REGION="+awsRegion+
		" --env CLUSTER_NAME="+cs.Name+
		" --env "+fmt.Sprintf("NUM_WORKERS=%d", cs.NumWorkers)+
		" --env KUBERNETES_VERSION="+cs.KubeVersion+
		" --security-group-id "+defaultSG)

	//create cluster spec in control plane:
	c := &http.Client{
		Timeout: time.Second * 30,
	}
	req, err := json.Marshal(cs)
	if err != nil {
		perr("Can't marshal cluster spec data", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't marshal cluster spec data")
		return
	}
	pres, err := c.Post(ekspcp+"/create/", "application/json", bytes.NewBuffer(req))
	if err != nil {
		perr("Can't POST to control plane for cluster create", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't POST to control plane for cluster create")
		return
	}
	defer pres.Body.Close()
	body, err := ioutil.ReadAll(pres.Body)
	if err != nil {
		perr("Can't read control plane response for cluster create", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't read control plane response for cluster create")
		return
	}
	// make sure to compensate for provision time, so prolong immediately for 15min:
	empty := ""
	_, err = c.Post(ekspcp+"/prolong/"+cs.ID+"/15", "application/json", bytes.NewBuffer([]byte(empty)))
	if err != nil {
		perr("Can't POST to control plane for prolonging cluster", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't POST to control plane for prolonging cluster")
		return
	}
	defer pres.Body.Close()
	jsonResponse(w, http.StatusOK, string(body))
}

// ProlongCluster prolongs the lifetime of a cluster via the /prolong endpoint
// in the EKSphemeral control plane, returning the result to the caller
func ProlongCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`Allow: ` + "POST"))
		return
	}
	///prolong/$CLUSTER_ID/$PROLONG_TIME
	type ClusterProlong struct {
		ID          string `json:"id"`
		ProlongTime int    `json:"ptime"`
	}
	decoder := json.NewDecoder(r.Body)
	cp := ClusterProlong{}
	err := decoder.Decode(&cp)
	if err != nil {
		perr("Can't parse cluster prolong values from UI", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't parse cluster prolong values from UI")
		return
	}
	pinfo(fmt.Sprintf("From the web UI I got the following values for proloning the cluster lifetime: %+v", cp))

	c := &http.Client{
		Timeout: time.Second * 30,
	}
	_, _, _, _, ekspcp := getDefaults()
	pinfo(fmt.Sprintf("Using %v as the control plane endpoint", ekspcp))
	pres, err := c.Post(ekspcp+"/prolong/"+cp.ID+"/"+strconv.Itoa(cp.ProlongTime), "application/json", r.Body)
	if err != nil {
		perr("Can't POST to control plane for prolonging cluster", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't POST to control plane for prolonging cluster")
		return
	}
	defer pres.Body.Close()
	body, err := ioutil.ReadAll(pres.Body)
	if err != nil {
		perr("Can't read control plane response for prolonging cluster", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't read control plane response for prolonging cluster")
		return
	}
	pinfo(fmt.Sprintf("Result proloning the cluster lifetime: %v", string(body)))
	// invalidate cache entry if present:
	invalidateCacheEntry(cp.ID)
	pinfo("Invalidated cache entry")

	jsonResponse(w, http.StatusOK, string(body))
}

func updateCache(csstring string) error {
	decoder := json.NewDecoder(strings.NewReader(csstring))
	cs := ClusterSpec{}
	err := decoder.Decode(&cs)
	if err != nil {
		return err
	}
	cscache[cs.ID] = cs
	return nil
}

func invalidateCacheEntry(cID string) {
	if _, ok := cscache[cID]; ok {
		delete(cscache, cID)
	}
}

func lookup(cID string) (ClusterSpec, error) {
	cs, ok := cscache[cID]
	if ok {
		return cs, nil
	}
	return cs, fmt.Errorf("not cached")
}

// jsonResponse wraps a message with a JSON header and write it out
func jsonResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprint(w, message)
}

// getKubeConfig returns the cluster config for kubectl
func GetClusterConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`Allow: ` + "GET"))
		return
	}
	q := r.URL.Query()
	clustername := q.Get("cluster")
	region, _ := os.LookupEnv("AWS_DEFAULT_REGION")
	pinfo(fmt.Sprintf("Looking up config for cluster %v in region %v", clustername, region))
	config := bshellout("sh", "-c", "aws eks update-kubeconfig --region "+
		region+" --name "+clustername+" --dry-run")
	jsonResponse(w, http.StatusOK, string(config))
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
