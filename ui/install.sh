#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

###############################################################################
### GLOBALS

fargateversion=0.3.0
ekspversion=v0.3.0

###############################################################################
### HELPER FUNCTIONS

function installFargate() {
  srcURL="https://github.com/jpignata/fargate/releases/download/v$fargateversion/fargate-$fargateversion-linux-amd64.zip"
  echo "Attempting to download $srcURL"
  curl -L $srcURL -o fargate$fargateversion.zip
  unzip fargate$fargateversion.zip
  mv ./fargate /usr/local/bin
  rm fargate$fargateversion.zip
}

function installEKSphemeralCLI() {
  srcURL="https://github.com/mhausenblas/eksphemeral/releases/download/$ekspversion/eksp-linux"
  echo "Attempting to download $srcURL"
  curl -L  $srcURL -o eksp
  chmod +x eksp
  mv ./eksp /usr/local/bin
}

###############################################################################
### MAIN

if [[ -z "$EKSPHEMERAL_HOME" ]]
then
  echo "I don't know where to install EKSphemeral dependencies, please set the EKSPHEMERAL_HOME environment variable"
  exit 1
fi

installFargate

mkdir -p $EKSPHEMERAL_HOME

git clone https://github.com/mhausenblas/eksphemeral.git $EKSPHEMERAL_HOME

cd $EKSPHEMERAL_HOME

installEKSphemeralCLI

