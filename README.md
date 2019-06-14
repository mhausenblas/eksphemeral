# EKSphemeral: The EKS Ephemeral Cluster Manager

> Do not use in production. This is a service for development and test environments.
> Also, this is not an official AWS offering but something I cooked up, so use at your own risk.

Managing EKS clusters for dev/test environments manually is boring. 
You have to wait until they're up and available and have to remember 
to tear them down again to minimize costs.

How about automate these steps? Meet EKSphemeral :)

EKSphemeral is a simple Amazon EKS manager for ephemeral dev/test clusters,
 allowing you to launch EKS clusters that auto-tear down after a time period,
 and you can also prolong their lifetime if you want to continue to use them.

0. [Architecture](#architecture)
1. [Install](#install)
2. [Use](#use)
   - [Create clusters](#create-clusters)
   - [List clusters](#list-clusters)
   - [Prolong cluster lifetime](#prolong-cluster-lifetime)
3. [Uninstall](#uninstall)
4. [Development](#development)

## Architecture

EKSphemeral has a control plane implemented in an AWS Lambda/Amazon S3 combo, 
and as its data plane it is using [eksctl](https://eksctl.io) running in AWS 
Fargate. There are four scripts, `eksp-*.sh` allowing you to install/uninstall
 EKSphemeral and to create and query clusters. Overall, the architecture looks 
 as follows:  

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

First off, install the binary CLI, for example for macOS:

```sh
$ curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.2.0/eksp-macos -o eksp
$ chmod +x eksp
$ sudo mv ./eksp /usr/local/bin
```

Make sure to set the respective environment variables before you proceed. 
This is so that the install process knows which S3 bucket to use for the control 
plane's Lambda functions (`EKSPHEMERAL_SVC_BUCKET`) and where to put the cluster 
metadata (`EKSPHEMERAL_CLUSTERMETA_BUCKET`). For example:

```sh
$ export EKSPHEMERAL_SVC_BUCKET=eks-svc
$ export EKSPHEMERAL_CLUSTERMETA_BUCKET=eks-cluster-meta
```

Optionally, in order to receive email notifications about cluster creation and destruction,
you need to set the following environment variable, for example:

```sh
$ export EKSPHEMERAL_EMAIL_FROM=hausenbl+eksphemeral@amazon.com
```

In addition to setting the `EKSPHEMERAL_EMAIL_FROM` environment variable, you
MUST [verify](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-email-addresses.html) 
both the source email, that is, the address you provide in `EKSPHEMERAL_EMAIL_FROM` as well as the
target email address (in the `owner` field of the cluster spec, see below for details) in the 
[EU (Ireland)](https://docs.aws.amazon.com/general/latest/gr/rande.html) `eu-west-1` region. 

Now we're in the position to install the EKSphemeral control plane, that is, to create S3 buckets if they don't exist yet 
and deploy the Lambda functions:

```sh
$ eksp install
Installing the EKSphemeral control plane, this might take a few minutes ...
Using eks-svc as the control plane service code bucket
Using eks-cluster-meta as the bucket to store cluster
metadata
mkdir -p bin
curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.2.0/createcluster -o bin/createcluster
curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.2.0/destroycluster -o bin/destroycluster
curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.2.0/prolongcluster -o bin/prolongcluster
curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/v0.2.0/status -o bin/status
chmod +x bin/*
sam package --template-file template.yaml --output-template-file eksp-stack.yaml --s3-bucket eks-svc
Uploading to 226fe5d95508b95aa57845beffffc654  18278955 / 18278955.0  (100.00%)
Successfully packaged artifacts and wrote output template to file eksp-stack.yaml.
sam deploy --template-file eksp-stack.yaml --stack-name eksp --capabilities CAPABILITY_IAM --parameter-overrides ClusterMetadataBucketName="eks-cluster-meta" NotificationFromEmailAddress="hausenbl+eksphemeral@amazon.com"

Waiting for changeset to be created..
Waiting for stack create/update to complete
Successfully created/updated stack - eksp

Control plane should be up now, let us verify that:

All good, ready to launch ephemeral clusters now!
```

Now, let's check if there are already clusters are managed by EKSphemeral:

```sh
$ eksp list

```

Since we just installed EKSphemeral, there are no clusters, yet. Let's change that.

## Use

### Create clusters

Let's create a cluster named `mh9-eksp`, with three worker nodes, 
using Kubernetes version 1.11, with a 15 min timeout as defined in the example 
cluster spec file [mh9-test-other.json](svc/dev/mh9-test-other.json):

```sh
$ cat svc/dev/mh9-test-other.json
{
    "id": "",
    "name": "mh9-eksp",
    "numworkers": 3,
    "kubeversion": "1.11",
    "timeout": 15,
    "ttl": 15,
    "owner": "hausenbl+notif@amazon.com",
    "created": ""
}

$ eksp create svc/dev/mh9-test-other.json
Trying to create a new ephemeral cluster ...
... using cluster spec svc/dev/mh9-test.json
Seems you've set 'us-east-2' as the target region, using this for all following operations
I will now provision the EKS cluster mh9-eksp using AWS Fargate:

[i] Running task eksctl
Waiting for EKS cluster provisioning to complete. Allow some 15 min to complete, checking status every minute:
.........
Successfully created data plane for cluster mh9-eksp using AWS Fargate and now moving on to the control plane in AWS Lambda and S3 ...

Successfully created control plane entry for cluster mh9-eksp via AWS Lambda and Amazon S3 ...

Now moving on to configure kubectl to point to your EKS cluster:
Updated context arn:aws:eks:us-east-2:661776721573:cluster/mh9-eksp in /Users/hausenbl/.kube/config

Your EKS cluster is now set up and configured:
CURRENT   NAME                                                  CLUSTER                                               AUTHINFO                                              NAMESPACE
*         arn:aws:eks:us-east-2:661776721573:cluster/mh9-eksp   arn:aws:eks:us-east-2:661776721573:cluster/mh9-eksp   arn:aws:eks:us-east-2:661776721573:cluster/mh9-eksp

Note that it still can take up to 5 min until the worker nodes are available, check with the following command until you don't see the 'No resources found.' message anymore:
kubectl get nodes
```

Note that if no cluster spec is provided, [default](svc/default-cc.json) will be
used along with first security group of the default VPC.

Further, note that, if you want to receive notification emails, you must
 [verify](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-email-addresses.html) 
 both the source and target email address in the Ireland (`eu-west-1`) region.

### List clusters

Next, let's check what clusters are managed by EKSphemeral:

```sh
$ eksp list
NAME       ID                                     KUBERNETES   NUM WORKERS   TIMEOUT   TTL      OWNER
mh9-eksp   e90379cf-ee0a-49c7-8f82-1660760d6bb5   v1.12        2             45 min    42 min   hausenbl+notif@amazon.com
```

Here, we get an array of cluster IDs back. We can use such a cluster ID as follows to look up the spec of a particular cluster:

```sh
$ eksp list e90379cf-ee0a-49c7-8f82-1660760d6bb5
ID:             e90379cf-ee0a-49c7-8f82-1660760d6bb5
Name:           mh9-eksp
Kubernetes:     v1.12
Worker nodes:   2
Timeout:        45 min
TTL:            37 min
Owner:          hausenbl+notif@amazon.com
```

### Prolong cluster lifetime

When you get a notification that one of your clusters is about to shut down or really at any time 
before it shuts down, you can prolong the cluster lifetime using the `eksp-prolong.sh` script.

Let's say we want to keep the cluster with the ID `e90379cf-ee0a-49c7-8f82-1660760d6bb5` around 
for 13 min longer. Here's what you would do:

```sh
$ eksp prolong e90379cf-ee0a-49c7-8f82-1660760d6bb5 13

Trying to set the TTL of cluster e90379cf-ee0a-49c7-8f82-1660760d6bb5 to 13 minutes, starting now
Successfully prolonged the lifetime of cluster e90379cf-ee0a-49c7-8f82-1660760d6bb5 for 13 minutes.

$ eksp list e90379cf-ee0a-49c7-8f82-1660760d6bb5
ID:             e90379cf-ee0a-49c7-8f82-1660760d6bb5
Name:           mh9-eksp
Kubernetes:     v1.12
Worker nodes:   2
Timeout:        48 min
TTL:            37 min
Owner:          hausenbl+notif@amazon.com

```

Note that the prolong command updates the `timeout` field of your cluster spec, that is, the cluster TTL is 
counted from the moment you issue the prolong command, taking the remaining cluster runtime into account.

## Uninstall

To uninstall EKSphemeral, use the following command. This will remove the 
control plane elements, that is, delete the Lambda functions and remove all 
cluster specs from the `EKSPHEMERAL_CLUSTERMETA_BUCKET` S3 bucket:

```bash
$ eksp uninstall
```

Note that the service code bucket and the cluster metadata bucket are still around after this. 
You can either manually delete them or keep them around, to reuse them later. 

## Development

To learn how to customize and extend EKSphemeral or simply toy around with it, 
see the dedicated [development docs](dev.md).