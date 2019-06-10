#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

printf "Installing the EKSphemeral control plane, this might take a few minutes ...\n"

cd svc
make install
cd ..

printf "\nControl plane should be up now, let us verify that:\n"

EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)

CONTROLPLANE_STATUS=$(curl -sL -w "%{http_code}" -o /dev/null "$EKSPHEMERAL_URL/status/*")

if [ $CONTROLPLANE_STATUS == "200" ]
then
    echo "All good, ready to launch ephemeral clusters now using the 'eksp-launch.sh' script"
else 
    echo "There was an issue setting up the EKSphemeral control plane, check the CloudFormation logs :("
fi

echo "Next, use the 'eksp-create.sh' script to launch a throwaway cluster or  'eksp-list.sh' to view them "