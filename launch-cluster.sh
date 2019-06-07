#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

###############################################################################
### PRE-FLIGHT CHECKS

if ! [ -x "$(command -v jq)" ]
then
  echo "Pre-flight check failed: jq is not installed. Yo, please install it from https://stedolan.github.io/jq/download/ and try again, cool?" >&2
  exit 1
fi

###############################################################################
### DEPENDENCIES

EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)

# Check dependency, that is, if control plane is available:
CONTROLPLANE_STATUS=$(curl -sL -w "%{http_code}" -o /dev/null "$EKSPHEMERAL_URL/status/")

if [ $CONTROLPLANE_STATUS != "200" ]
then
    echo "Pre-flight check failed: the control plane seems not to be up, are you sure you executed install.sh already, mate?" >&2
    exit 1
fi

###############################################################################
### CONTROL PLANE (METADATA) OPERATIONS

# CLUSTERID=$(curl --progress-bar --header "Content-Type: application/json" --request POST --data @2node-111-30.json $EKSPHEMERAL_URL/create/)

###############################################################################
### DATA PLANE OPERATIONS

# fargate task run eksctl \
#           --image quay.io/mhausenblas/eksctl:0.1 \
#           --region us-east-2 \
#           --env AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id) \
#           --env AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key) \
#           --env AWS_DEFAULT_REGION=$(aws configure get region)
#           --security-group-id $EKSPHEMERAL_SG

printf "Waiting for EKS cluster provisioning to complete. Allow some 15 min to complete, checking status every minute:\n"

while [ "$(aws eks describe-cluster --name eksphemeral | jq .cluster.status -r)" != "ACTIVE" ]
do
    printf "."
    sleep 60 
done

printf "\nNow moving on to configure kubectl to point to your EKS cluster:\n"
aws eks update-kubeconfig --name eksphemeral

printf "\nYour EKS cluster is now set up and configured:\n"
kubectl config get-contexts






 