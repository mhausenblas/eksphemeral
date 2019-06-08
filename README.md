# EKSphemeral

> Note: this is a heavy WIP, do not use in production.

A simple Amazon EKS manager for ephemeral dev/test clusters, using AWS Lambda and AWS Fargate, allowing you to launch an EKS cluster with an automatic tear-down after a given time.

In order to build the service, clone this repo, and make sure you've got the `aws` CLI, [SAM CLI](https://github.com/awslabs/aws-sam-cli), and the [Fargate CLI](https://somanymachines.com/fargate/) installed.

All the dependencies: AWS Lambda, AWS Fargate, Amazon EKS, Docker, `aws`, `sam`, `fargate` and `jq`.

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

Optionally, you can build a custom container image using your own registry coordinates and customize what's in the `eksctl` image used to provision the EKS cluster via a Fargate task:

```sh
$ docker build -t quay.io/mhausenblas/eksctl:0.1 .
$ docker push quay.io/mhausenblas/eksctl:0.1
```

## Local development

### Control plane in AWS Lambda

In order for the local simulation, part of SAM, to work, you need to have Docker running. Note: Local testing the API is at time of writing not possible since [CORS is locally not supported](https://github.com/awslabs/aws-sam-cli/issues/323), yet.

In the `svc/` directory, do the following:

```sh
# 1. run emulation of Lambda and API Gateway locally (via Docker):
$ sam local start-api

# 2. update Go source code: add functionality, fix bugs

# 3. create binaries, automagically synced into the local SAM runtime:
$ make build
```

If you change anything in the SAM/CF [template file](svc/template.yaml) then you need to re-start the local API emulation.

The EKSphemeral control plane has the following API

- List the launched clusters via an HTTP `GET` to `$BASEURL/status` 
- Check status of a specific cluster via an HTTP `GET` to `$BASEURL/status/$CLUSTERID`
- Create a cluster via an HTTP `POST` to `$BASEURL/create` with following parameters (all optional):
  - `numworkers` ... number of worker nodes, defaults to `1`
  - `kubeversion` ... Kubernetes version to use, defaults to `1.12`
  - `timeout` ... timeout in minutes, after which the cluster is destroyed, defaults to `10`
  - `owner` ... the email address of the owner
- Auto-destruction of a cluster after the set timeout (triggered by CloudWatch events, no HTTP endpoint)

### Data plane in AWS Fargate

You can manually kick off the EKS cluster provisioning as follows.

First, set the security group to use:

```sh
$ export EKSPHEMERAL_SG=XXXX
```

Note that if you don't know which default security group(s) you have available, you can use the following
command to list them:

```sh
$ aws ec2 describe-security-groups | jq '.SecurityGroups[] | select (.GroupName == "default") | .GroupId'
```

Now you can use Fargate to provision the cluster (using your local AWS credentials):

```sh
$ fargate task run eksctl \
          --image quay.io/mhausenblas/eksctl:0.1 \
          --region us-east-2 \
          --env AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id) \
          --env AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key) \
          --env AWS_DEFAULT_REGION=$(aws configure get region)
          --security-group-id $EKSPHEMERAL_SG
```

## Usage

The following assumes that the S3 bucket as outlined above is set up and you have access to AWS configured, locally.

In the `svc/` directory, do the following:

```sh
make deploy
```

After above command you can get the HTTP API endpoint like this (requires `jq` to be installed, locally):

```sh
$ EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)
```

First, let's check what clusters are already managed by EKSphemeral:

```sh
$ curl $EKSPHEMERAL_URL/status/
```

Now, let's create a 2-node cluster, using Kubernetes version 1.11, with a 30 min timeout:

```sh
$ curl --progress-bar \
       --header "Content-Type: application/json" \
       --request POST \
       --data @2node-111-30.json \
       $EKSPHEMERAL_URL/create/
```


## Clean up

```bash
$ aws cloudformation delete-stack --stack-name eksp
```

TBD: Fargate clean-up