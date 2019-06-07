#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

### DEPENDENCY
# make deploy

# EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)

# CLUSTERID=$(curl --progress-bar --header "Content-Type: application/json" --request POST --data @2node-111-30.json $EKSPHEMERAL_URL/create/)

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

printf "Your EKS cluster is now set up and configured:\n"
kubectl config get-contexts






 