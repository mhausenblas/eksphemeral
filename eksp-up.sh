#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

###############################################################################
### PRE-FLIGHT CHECK

if [[ -z "$EKSPHEMERAL_HOME" ]]
then
  echo "I don't know where to install EKSphemeral dependencies, please set the EKSPHEMERAL_HOME environment variable"
  exit 1
fi

if aws cloudformation describe-stacks --stack-name eksp > /dev/null 2>&1
then
    EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)
    printf "Pre-flight check failed: the control plane is already up and available at %s\n... are you trying to install it again?" $EKSPHEMERAL_URL >&2
    exit 1
fi

printf "Installing the EKSphemeral control plane, this might take a few minutes\n"

###############################################################################
### S3 BUCKET OPERATIONS

if [[ $(aws s3 ls | grep $EKSPHEMERAL_SVC_BUCKET) ]]; then
    echo "Using existing S3 bucket $EKSPHEMERAL_SVC_BUCKET for the control plane service code"
else
    aws s3api create-bucket \
        --bucket $EKSPHEMERAL_SVC_BUCKET \
        --create-bucket-configuration LocationConstraint=$(aws configure get region) \
        --region $(aws configure get region)
    echo "Created S3 bucket $EKSPHEMERAL_SVC_BUCKET and using it for the control plane service code"
fi

if [[ $(aws s3 ls | grep $EKSPHEMERAL_CLUSTERMETA_BUCKET) ]]; then
    echo "Using existing S3 bucket $EKSPHEMERAL_CLUSTERMETA_BUCKET to store cluster specifications"
else   
    aws s3api create-bucket \
      --bucket $EKSPHEMERAL_CLUSTERMETA_BUCKET \
      --create-bucket-configuration LocationConstraint=$(aws configure get region) \
      --region $(aws configure get region)
    echo "Created S3 bucket $EKSPHEMERAL_CLUSTERMETA_BUCKET and using it to store cluster specifications"
fi

###############################################################################
### INSTALL CONTROL PLANE

cd $EKSPHEMERAL_HOME/svc
make install EKSPHEMERAL_SVC_BUCKET=$EKSPHEMERAL_SVC_BUCKET EKSPHEMERAL_CLUSTERMETA_BUCKET=$EKSPHEMERAL_CLUSTERMETA_BUCKET EKSPHEMERAL_EMAIL_FROM=$EKSPHEMERAL_EMAIL_FROM
cd -

printf "\nControl plane should be up now, let us verify that: "

EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)

CONTROLPLANE_STATUS=$(curl -sL -w "%{http_code}" -o /dev/null "$EKSPHEMERAL_URL/status/*")

if [ $CONTROLPLANE_STATUS == "200" ]
then
    printf " All good, ready to launch ephemeral clusters now!\n"
else 
    printf " There was an issue setting up the EKSphemeral control plane, check the CloudFormation logs :(\n"
    exit 1
fi
