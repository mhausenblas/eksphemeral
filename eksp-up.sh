#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail


printf "Installing the EKSphemeral control plane, this might take a few minutes ...\n"

make deploy

printf "Control plane should be up now, let us verify that:\n"

curl --progress-bar $EKSPHEMERAL_URL/status/