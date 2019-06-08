#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail


printf "Taking down the EKSphemeral control plane, this might take a few minutes ...\n"

cd svc
make destroy
cd ..

printf "\nTear-down will complete within some 5 min. You can check the status manually, if you like, using 'make status' in the svc/ directory.\nOnce you see a message saying something like 'Stack with id eksp does not exist' you know for sure it's gone :)\n"

# TBD: delete all objects in the eks-cluster-meta bucket
# TBD: Fargate clean-up

printf "\n\nThanks for using EKSphemeral and hope to see ya soon ;)\n"