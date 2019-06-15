#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

###############################################################################
### GLOBALS

fargateversion=0.3.0
ekspversion=v0.3.0

unameres="$(uname -s)"
case "${unameres}" in
    Linux*)     machine=Linux;;
    Darwin*)    machine=MacOS;;
    *)          echo "Sorry, not a supported platform" ; exit 1
esac


###############################################################################
### HELPER FUNCTIONS

function installFargate() {
  case "${machine}" in
    Linux*)     srcURL=https://github.com/jpignata/fargate/releases/download/v$fargateversion/fargate-$fargateversion-linux-amd64.zip ;;
    MacOS*)    srcURL=https://github.com/jpignata/fargate/releases/download/v$fargateversion/fargate-$fargateversion-darwin-amd64.zip ;;
  esac
  echo "Attempting to download $srcURL"
  curl -L $srcURL -o fargate$fargateversion
  tar xopf fargate$fargateversion
  mv ./fargate /usr/local/bin 
  rm fargate$fargateversion
}

function installEKSphemeralCLI() {
  case "${machine}" in
    Linux*)     srcURL=https://github.com/mhausenblas/eksphemeral/releases/download/$ekspversion/eksp-linux ;;
    MacOS*)    srcURL=https://github.com/mhausenblas/eksphemeral/releases/download/$ekspversion/eksp-macos ;;
  esac
  echo "Attempting to download $srcURL"
  curl -L  $srcURL -o eksp
  chmod +x eksp
  mv ./eksp /usr/local/bin
}

###############################################################################
### PRE-FLIGHT CHECKS

if ! [ -x "$(command -v jq)" ]
then
  echo "Please install jq from https://stedolan.github.io/jq/download/ and try again" >&2
  exit 1
fi

if ! [ -x "$(command -v aws)" ]
then
  echo "Please install aws via https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html and try again" >&2
  exit 1
fi

if ! [ -x "$(command -v fargate)" ]
then
  installFargate
fi

###############################################################################
### MAIN

if [[ -z "$EKSPHEMERAL_HOME" ]]
then
  echo "I don't know where to install EKSphemeral dependencies, please set the EKSPHEMERAL_HOME environment variable"
  exit 1
fi

mkdir -p $EKSPHEMERAL_HOME

git clone https://github.com/mhausenblas/eksphemeral.git $EKSPHEMERAL_HOME

cd $EKSPHEMERAL_HOME

# Install CLI
installEKSphemeralCLI

# Install EKSphemeral control plane
eksp install



