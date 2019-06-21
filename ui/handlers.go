package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

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
	if targetcluster != "*" {
		err = updateCache(string(body))
		if err != nil {
			perr("Can't update local cluster spec cache", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			jsonResponse(w, http.StatusInternalServerError, "Can't update local cluster spec cache")
			return
		}
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

// GetClusterConfig returns the cluster config for kubectl
func GetClusterConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`Allow: ` + "GET"))
		return
	}
	q := r.URL.Query()
	cID := q.Get("cluster")
	region, _ := os.LookupEnv("AWS_DEFAULT_REGION")
	cs, err := lookup(cID)
	if err != nil {
		perr("Can't find cluster spec in cache", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		jsonResponse(w, http.StatusInternalServerError, "Can't find cluster spec in cache")
		return
	}
	pinfo(fmt.Sprintf("Looking up config for cluster %v in region %v", cs.Name, region))
	cmd := "aws eks update-kubeconfig --region " + region + " --name " + cs.Name //+ " --dry-run"
	// config := bshellout("sh", "-c", cmd)
	plainResponse(w, http.StatusOK, string(cmd))
}
