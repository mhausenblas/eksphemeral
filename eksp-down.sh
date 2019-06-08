#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail


printf "Taking down the EKSphemeral control plane, this might take a few minutes ...\n"

make destroy

# TBD: delete all objects in the eks-cluster-meta bucket
# TBD: Fargate clean-up

printf "Thanks for using EKSphemeral and hope to see ya soon ;)\n"