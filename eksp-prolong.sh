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

if ! aws cloudformation describe-stacks --stack-name eksp > /dev/null 2>&1
then
  echo "Pre-flight check failed: the control plane seems not to be up, are you sure you executed eksp-up.sh already?" >&2
  exit 1
fi

CLUSTER_ID=${1}
PROLONG_TIME=${2}

###############################################################################
### PROLONG TTL OF AN ACTIVE CLUSTER

EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)

curl -s --request POST "$EKSPHEMERAL_URL/prolong/$CLUSTER_ID/$PROLONG_TIME"

