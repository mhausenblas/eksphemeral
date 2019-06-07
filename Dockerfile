####################################################################
## Based on Massimo's fantastic https://github.com/mreferre/eksutils
####################################################################

FROM amazonlinux:2018.03
MAINTAINER hausenbl@amazon.com

################## BEGIN INSTALLATION ######################

# setup the IAM authenticator for eksctl
RUN curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/linux/amd64/aws-iam-authenticator
RUN chmod +x ./aws-iam-authenticator
RUN mv ./aws-iam-authenticator /usr/local/bin

# set up eksctl
RUN curl --silent --location "https://github.com/weaveworks/eksctl/releases/download/latest_release/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp 
RUN mv -v /tmp/eksctl /usr/local/bin

##################### INSTALLATION END #####################

WORKDIR /

RUN adduser -D -u 10001 eksctl

USER eksctl

CMD eksctl create cluster \
    --name eksphemeral \
    --version 1.12 \
    --nodes 2 \
    --auto-kubeconfig \
    --full-ecr-access \
    --appmesh-access