# EKSphemeral

> Note: this is a heavy WIP, do not use in production.

A simple Amazon EKS manager for ephemeral clusters, using AWS Lambda and sporting the following features:

- Create a cluster via a HTTP `POST` to `$BASEURL/create` with following parameters (all optional):
  - `numworkers` ... number of worker nodes, defaults to `1`
  - `kubeversion` ... Kubernetes version to use, defaults to `1.12`
  - `timeout` ... timeout in minutes, after which the cluster is destroyed, defaults to `10`
  - `owner` ... the email address of the owner
- Check cluster status via a HTTP `GET` to `$BASEURL/status/$CLUSTERID`
- Auto-destruction of a cluster after the set timeout

In order to build the service, clone this repo, and make sure you've got the `aws` CLI and the [SAM CLI](https://github.com/awslabs/aws-sam-cli) installed.

Dependencies: AWS Lambda, Amazon EKS, Docker, `aws, `sam`, and `jq`.

## Preparation

Create an S3 bucket `eks-svc` for the Lambda functions like so:

```sh
$ aws s3api create-bucket \
      --bucket eks-svc \
      --create-bucket-configuration LocationConstraint=us-east-2 \
      --region us-east-2
```

## Local development

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

## Deployment

The following assumes that the S3 bucket as outlined above is set up and you have access to AWS configured.

In the `svc/` directory, do the following:

```sh
make deploy
```

After above command you can get the HTTP API endpoint like this (requires `jq` to be installed, locally):

```sh
$ EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)
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

