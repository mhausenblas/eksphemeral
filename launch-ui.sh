#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

###############################################################################
### DEPENDENCIES CHECKS

EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)

CONTROLPLANE_STATUS=$(curl -sL -w "%{http_code}" -o /dev/null "$EKSPHEMERAL_URL/status/*")

if [ ! $CONTROLPLANE_STATUS == "200" ]
then
  echo "I don't know which EKSphemeral control plane to use, please set the EKSPHEMERAL_URL environment variable"
  exit 1
else
  export EKSPHEMERAL_URL=$EKSPHEMERAL_URL
fi

if [[ $(docker info >/dev/null 2>&1) -ne 0 ]]
then
  echo "Pre-flight check failed: Docker is not running" >&2
  exit 1
fi

###############################################################################
### MAIN


cd ui/

# if the container image is not yet available locally, build it:
if [[ $(make verify > /dev/null 2>&1) -ne 0 ]]
then
  make build
fi

# make sure to stop the already running one and otherwise launch the UI proxy:

if [[ $(docker ps | grep ekspui >/dev/null 2>&1) -eq 0 ]]
then
  make stop > /dev/null
fi

make run > /dev/null

# show user the logs:
docker logs --follow ekspui

cd -