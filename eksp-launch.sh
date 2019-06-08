#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

###############################################################################
### COMMAND LINE PARAMETERS
EKSPHEMERAL_SG=$1
CLUSTER_SPEC=${2:-svc/default-cc.json}

###############################################################################
### PRE-FLIGHT CHECKS

if ! [ -x "$(command -v jq)" ]
then
  echo "Pre-flight check failed: jq is not installed. Yo, please install it from https://stedolan.github.io/jq/download/ and try again, cool?" >&2
  exit 1
fi

if ! aws cloudformation describe-stacks --stack-name eksp > /dev/null 2>&1
then
  echo "Pre-flight check failed: the control plane seems not to be up, are you sure you executed eksp-up.sh already?" >&2
  exit 1
fi

# Check dependency, that is, if control plane is available:
EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)
CONTROLPLANE_STATUS=$(curl -sL -w "%{http_code}" -o /dev/null "$EKSPHEMERAL_URL/status/")

if [ $CONTROLPLANE_STATUS != "200" ]
then
  echo "Pre-flight check failed: the control plane seems not to be up, are you sure you executed eksp-up.sh already?" >&2
  exit 1
fi

###############################################################################
### DATA PLANE OPERATION

printf "I will now provision the EKS cluster using AWS Fargate:\n\n"

# provision the EKS cluster using containerized eksctl:
fargate task run eksctl \
          --image quay.io/mhausenblas/eksctl:0.1 \
          --region us-east-2 \
          --env AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id) \
          --env AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key) \
          --env AWS_DEFAULT_REGION=$(aws configure get region) \
          --security-group-id $EKSPHEMERAL_SG

printf "Waiting for EKS cluster provisioning to complete. Allow some 15 min to complete, checking status every minute:\n"

# this is necessary to make sure the EKS control plane is up and we can query the cluster status:
sleep 120

while [ "$(aws eks describe-cluster --name eksphemeral | jq .cluster.status -r)" != "ACTIVE" ]
do
    printf "."
    sleep 60 
done

# note, one could use https://docs.aws.amazon.com/cli/latest/reference/cloudformation/wait/stack-exists.html as well here, maybe?


###############################################################################
### CONTROL PLANE (METADATA) OPERATIONS

# now that the EKS cluster (our data plane) is up and running,
# let's create a cluster (metadata) entry in S3 via Lambda (our control plane):
CLUSTERID=$(curl --progress-bar --header "Content-Type: application/json" --request POST --data @$CLUSTER_SPEC $EKSPHEMERAL_URL/create/)

printf "\nCreated control plane entry for cluster %s via AWS Lambda and S3 and now moving on to provision the data plane using AWS Fargate\n\n" $CLUSTERID

###############################################################################
### CONFIG AND SMOKE TEST

printf "\nNow moving on to configure kubectl to point to your EKS cluster:\n"
aws eks update-kubeconfig --name eksphemeral

printf "\nYour EKS cluster is now set up and configured:\n"
kubectl config get-contexts






 