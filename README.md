# EKSphemeral: The EKS Ephemeral Cluster Manager

> Do not use in production. This is a service for development and test environments. Also, this is not an official AWS offering but something MH9 cooked up.

Managing EKS clusters for dev/test environments manually is boring. You have to wait until they're up and available and have to remember to tear them down again to minimize costs.

How about automate these steps? Meet EKSphemeral :)

EKSphemeral is a simple Amazon EKS manager for ephemeral dev/test clusters, allowing you to launch an EKS cluster with an automatic tear-down after a given time.

0. [Architecture](#architecture)
1. [Install](#install)
2. [Use](#use)
   - [Create clusters](#create-clusters)
   - [List clusters](#list-clusters)
   - [Prolong cluster lifetime](#prolong-cluster-lifetime)
3. [Uninstall](#uninstall)
4. [Development](#development)

## Architecture

EKSphemeral has a control plane implemented in an AWS Lambda/Amazon S3 combo, and as its data plane it is using [eksctl](https://eksctl.io) running in AWS Fargate. There are four scripts, `eksp-*.sh` allowing you to install/uninstall EKSphemeral and to create and query clusters. Overall, the architecture looks as follows:  

![EKSphemeral architecture](img/architecture.png)

1. The `eksp-up.sh` script provisions EKSphemeral's control plane (Lambda+S3). This is a one-time action, think of it as installing EKSphemeral in your AWS environment.
2. Whenever you want to provision a throwaway EKS cluster, use `eksp-create.sh`. It will do two things: 
3. Provision the cluster using `eksctl` running in Fargate (what we call the EKSphemeral data plane), and when that is completed,
4. Create an cluster spec entry in S3, via the `/create` endpoint of EKSphemeral's HTTP API.
5. Once that is done, you can use `eksp-list.sh` to list all managed clusters or, should you wish to gather more information on a specific cluster, use `eksp-list.sh $CLUSTERID` to retrieve it. This script uses the `/status` endpoint of EKSphemeral's HTTP API.
6. Every 5 minutes, there is a CloudWatch event that triggers the execution of another Lambda function called `DestroyClusterFunc`, which notifies the owners of clusters that are about to expire (send an email up to 5 minutes before the cluster is destroyed), and when the time comes, it tears the cluster down. 
7. Last but not least, if you want to get rid of EKSphemeral, use the `eksp-down.sh` script, removing all cluster specs in the S3 bucket and deleting all Lambda functions.

If you like, you can have a look at a [4 min video walkthrough](https://www.youtube.com/watch?v=2A8olhYL9iI), before you try it out yourself.
Since the minimal time for an end-to-end provisioning and usage cycle is ca. 40min, the video walkthrough is showing a 1:10 time compression, roughly.

If you want to try it out yourself, follow the steps below.


## Install

In order to use EKSphemeral, clone this repo, and make sure you've got `jq`, the `aws` CLI 
and the [Fargate CLI](https://somanymachines.com/fargate/) installed.

Make sure to set the respective environment variables before you proceed. 
This is so that the install process knows which S3 bucket to use for the control 
plane's Lambda functions (`EKSPHEMERAL_SVC_BUCKET`) and where to put the cluster 
metadata (`EKSPHEMERAL_CLUSTERMETA_BUCKET`):

```sh
$ export EKSPHEMERAL_SVC_BUCKET=eks-svc
$ export EKSPHEMERAL_CLUSTERMETA_BUCKET=eks-cluster-meta
```

Optionally, in order to receive email notifications about cluster creation and destruction,
you need to set the following environment variable: 

```sh
$ export EKSPHEMERAL_EMAIL_FROM=hausenbl+eksphemeral@amazon.com
```

Note, that you in addition to set the `EKSPHEMERAL_EMAIL_FROM` environment variable, you
MUST [verify](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-email-addresses.html) 
both the source email, that is, the address you provide in `EKSPHEMERAL_EMAIL_FROM` as well as the
target email address (in the `owner` field of the cluster spec, see below for details) in the 
[EU (Ireland)](https://docs.aws.amazon.com/general/latest/gr/rande.html) `eu-west-1` region. 

Now we're in the position to install the EKSphemeral control plane, that is, to create S3 buckets if they don't exist yet 
and deploy the Lambda functions:

```sh
$ ./eksp-up.sh
Installing the EKSphemeral control plane, this might take a few minutes ...
Using eks-svc as the control plane service code bucket
Using eks-cluster-meta as the bucket to store cluster
metadata
mkdir -p bin
curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.1.0/createcluster -o bin/createcluster
curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.1.0/destroycluster -o bin/destroycluster
curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.1.0/prolongcluster -o bin/prolongcluster
curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.1.0/status -o bin/status
chmod +x bin/*
sam package --template-file template.yaml --output-template-file eksp-stack.yaml --s3-bucket eks-svc
Uploading to 226fe5d95508b95aa57845beffffc654  18278955 / 18278955.0  (100.00%)
Successfully packaged artifacts and wrote output template to file eksp-stack.yaml.
Execute the following command to deploy the packaged template
aws cloudformation deploy --template-file /Users/hausenbl/go/src/github.com/mhausenblas/eksphemeral/svc/eksp-stack.yaml --stack-name <YOUR STACK NAME>
sam deploy --template-file eksp-stack.yaml --stack-name eksp --capabilities CAPABILITY_IAM --parameter-overrides ClusterMetadataBucketName="eks-cluster-meta" NotificationFromEmailAddress="hausenbl+eksphemeral@amazon.com"

Waiting for changeset to be created..
Waiting for stack create/update to complete
Successfully created/updated stack - eksp

Control plane should be up now, let us verify that:

All good, ready to launch ephemeral clusters now using the 'eksp-launch.sh' script

Next, use the 'eksp-create.sh' script to launch a throwaway cluster or 'eksp-list.sh' to view them
```

Now, let's check if there are already clusters are managed by EKSphemeral:

```sh
$ ./eksp-list.sh
[]
```

Since we just installed EKSphemeral, there are no clusters, yet. Let's change that.

## Use

### Create clusters

Let's start off by creating a throwaway EKS cluster with the [default](svc/default-cc.json) values:

```sh
$ ./eksp-create.sh
```

Now, let's create a  cluster named `2node-111-30`, using the `EKSPHEMERAL_SG` security group, with two worker nodes, using Kubernetes version 1.11, with a 30 min timeout as defined in the example cluster spec file [2node-111-30.json](svc/2node-111-30.json):

```sh
$ cat svc/2node-111-30.json
{
    "name": "2node-111-30",
    "numworkers": 2,
    "kubeversion": "1.11",
    "timeout": 30,
    "owner": "hausenbl+notif@amazon.com"
}

$ ./eksp-create.sh 2node-111-30.json $EKSPHEMERAL_SG
```

Note that both the security group and the cluster spec file are optional. If not present, the first security group of the default VPC and `default-cc.json` will be used, as we had it in the first example.

Further, note that, if you want to receive notification emails, you must [verify](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-email-addresses.html) both the source and target email address in the Ireland region.

### List clusters

Next, let's check what clusters are managed by EKSphemeral:

```sh
$ ./eksp-list.sh
["9be65bee-3baa-4fd0-aa3e-032325d5390c","dd72f73a-3457-4d4b-b997-08a2b376160b"]
```

Here, we get an array of cluster IDs back. We can use such a cluster ID as follows to look up the spec of a particular cluster:

```sh
$ ./eksp-list.sh dd72f73a-3457-4d4b-b997-08a2b376160b | jq
{
  "name": "default-eksp",
  "numworkers": 1,
  "kubeversion": "1.12",
  "timeout": 20,
  "owner": "nobody@example.com"
}
```

### Prolong cluster lifetime

When you get a notification that one of your clusters is about to shut down or really at any time 
before it shuts down, you can prolong the cluster lifetime using the `eksp-prolong.sh` script.

Let's say we want to keep the cluster with the ID `7a4aa952-9582-4d99-98a0-0ab1a4e56337` around 
for 40 min longer (with a remaining cluster runtime of 2 min). Here's what you would do:

```sh
$ ./eksp-prolong.sh 7a4aa952-9582-4d99-98a0-0ab1a4e56337 40

Trying to set the TTL of cluster 7a4aa952-9582-4d99-98a0-0ab1a4e56337 to 42 minutes, starting now
Successfully prolonged the lifetime of cluster 7a4aa952-9582-4d99-98a0-0ab1a4e56337 for 40 minutes. New TTL is 42 min starting now!

$ ./eksp-list.sh 7a4aa952-9582-4d99-98a0-0ab1a4e56337 | jq
{
  "name": "1node-112-10",
  "numworkers": 1,
  "kubeversion": "1.12",
  "timeout": 42,
  "owner": "hausenbl+notif@amazon.com"
}
```

Note that the prolong command updates the `timeout` field of your cluster spec, that is, the cluster TTL is 
counted from the moment you issue the prolong command, taking the remaining cluster runtime into account.

## Uninstall

To uninstall EKSphemeral, use the following command. This will remove the control plane elements, that is, delete the Lambda functions and remove all cluster specs from the `EKSPHEMERAL_CLUSTERMETA_BUCKET` S3 bucket:

```bash
$ ./eksp-down.sh
```

Note that the service code bucket and the cluster metadata bucket are still around after this. You can either manually delete them or keep them around, to reuse them later. 

## Development

To learn how to customize and extend EKSphemeral or simply toy around with it,see the dedicated [development docs](dev.md).