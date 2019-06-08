# EKSphemeral

> Note: this is a heavy WIP, do not use in production.

A simple Amazon EKS manager for ephemeral dev/test clusters, using AWS Lambda and AWS Fargate, allowing you to launch an EKS cluster with an automatic tear-down after a given time.

In order to use EKSphemeral, clone this repo, and make sure you've got `jq`, the `aws` CLI and the [Fargate CLI](https://somanymachines.com/fargate/) installed.

## Preparation

Create an S3 bucket `eks-svc` for the Lambda functions like so:

```sh
$ aws s3api create-bucket \
      --bucket eks-svc \
      --create-bucket-configuration LocationConstraint=us-east-2 \
      --region us-east-2
```

Create an S3 bucket `eks-cluster-meta` for the cluster metadata like so:

```sh
$ aws s3api create-bucket \
      --bucket eks-cluster-meta \
      --create-bucket-configuration LocationConstraint=us-east-2 \
      --region us-east-2
```

## Usage

The following assumes that the S3 bucket as outlined above is set up and you have access to AWS configured, locally.

Do the following then to set up the EKSphemeral control plane (with `$EKSPHEMERAL_SG` being the security group you want to use for the Fargate-provisioned data plane):

```sh
$ ./eksp-up.sh $EKSPHEMERAL_SG
```

First, let's check what clusters are already managed by EKSphemeral:

```sh
$ ./eksp-list.sh
```

Now, let's create a two-node cluster, using Kubernetes version 1.11, with a 30 min timeout as defined in the example config file [2node-111-30.json](svc/2node-111-30.json):

```sh
$ cat svc/2node-111-30.json
{
    "numworkers": 2,
    "kubeversion": "1.11",
    "timeout": 30,
    "owner": "hausenbl+notif@amazon.com"
}

$ ./eksp-create.sh test-cluster 2node-111-30.json
```

## Tear down

To tear down EKSphemeral, use the following command which will remove control plane elements (Lambda functions, S3 bucket content):

```bash
$ ./eksp-down.sh
```

## Development

See the dedicated [development docs](dev.md) for how to customize and extend EKSphemeral.