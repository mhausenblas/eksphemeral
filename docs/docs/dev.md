# Development and testing

If you want to play around with EKSphemeral, follow these steps.

In order to build the service, clone this repo, and make sure you've got the following available, locally:

- The [jq](https://stedolan.github.io/jq/download/) tool
- The [aws](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) CLI
- The [SAM CLI](https://github.com/awslabs/aws-sam-cli)
- The [Fargate CLI](https://somanymachines.com/fargate/)
- Docker

Also, you will need access to the following services, and their implicit dependencies, such as EC2 in case of EKS: AWS Lambda, AWS Fargate, Amazon EKS. 

## The control plane

The EKSphemeral control plane is implemented in AWS Lambda and S3, see also the [architecture](/arch) for details.

In order for the local simulation, part of SAM, to work, you need to have Docker running. Note: Local testing the API is at time of writing not possible since [CORS is locally not supported](https://github.com/awslabs/aws-sam-cli/issues/323), yet.

In the `svc/` directory, do the following:

```sh
# 1. run emulation of Lambda and API Gateway locally (via Docker):
$ sam local start-api

# 2. update Go source code: add functionality, fix bugs

# 3. create binaries, automagically synced into the local SAM runtime:
$ make build
```

If you change anything in the SAM/CF [template file](https://github.com/mhausenblas/eksphemeral/blob/master/svc/template.yaml) then you need to re-start the local API emulation.

The EKSphemeral control plane has the following API:

- List the launched clusters via an HTTP `GET` to `$BASEURL/status` 
- Check status of a specific cluster via an HTTP `GET` to `$BASEURL/status/$CLUSTERID`
- Create a cluster via an HTTP `POST` to `$BASEURL/create` with following parameters (all optional):
  - `numworkers` ... number of worker nodes, defaults to `1`
  - `kubeversion` ... Kubernetes version to use, defaults to `1.12`
  - `timeout` ... timeout in minutes, after which the cluster is destroyed, defaults to `20` (and 5 minutes before that you get a warning mail)
  - `owner` ... the email address of the owner
- Auto-destruction of a cluster after the set timeout (triggered by CloudWatch events, no HTTP endpoint)

Once deployed, you can find out where the API runs via:

```sh
$ EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)
```

## The data plane

The EKSphemeral data plane consists of [eksctl](https://eksctl.io/) running in AWS Fargate, see also the [architecture](/arch) for details.

You can manually kick off the EKS cluster provisioning as described in the following.


!!! note 
    Optionally, you can build a custom container image using your own registry coordinates and customize what's in the `eksctl` image used to provision the EKS cluster via a Fargate task.

First, set the security group to use:

```sh
$ export EKSPHEMERAL_SG=XXXX
```

Note that if you don't know which default security group(s) you have available, you can use the following
command to list them:

```sh
$ aws ec2 describe-security-groups | jq '.SecurityGroups[] | select (.GroupName == "default") | .GroupId'
```

Also, you could create a dedicated security group for the data plane:

```sh
$ default_vpc=$(aws ec2 describe-vpcs --filters "Name=isDefault, Values=true" | jq .Vpcs[0].VpcId -r)
```

And:

```sh
$ aws ec2 create-security-group --group-name eksphemeral-sg --description "The security group the EKSphemeral data plane uses" --vpc-id $default_vpc
```

And:

```sh
$ aws ec2 authorize-security-group-ingress --group-name eksphemeral-sg --protocol all --port all
```

!!! warning
    That the last command, `aws ec2 authorize-security-group-ingress` apparently doesn't work, unsure but based on my research it's an AWS CLI bug.

Now you can use AWS Fargate through the Fargate CLI to provision the cluster,
using your local AWS credentials, for example like so:

```sh
$ fargate task run eksctl \
          --image quay.io/mhausenblas/eksctl:base \
          --region us-east-2 \
          --env AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id) \
          --env AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key) \
          --env AWS_DEFAULT_REGION=$(aws configure get region) \
          --env CLUSTER_NAME=test \
          --env NUM_WORKERS=3 \
          --env KUBERNETES_VERSION=1.12 \
          --security-group-id $EKSPHEMERAL_SG
```

This should take something like 10 to 15 minutes to finish.

!!! tip
    Keep an eye on the AWS console for the resources and logs.

## The UI

If you want to change or extend the UI (HTML, JS, CSS) or the [UI proxy]([main.go](https://github.com/mhausenblas/eksphemeral/tree/master/ui)) you're welcome to do so.

!!! warning

    Please make sure you're in the `ui/` directory for the following steps. 

First, export the `EKSPHEMERAL_URL` env variable like so:

```sh
$ export EKSPHEMERAL_URL=$(aws cloudformation describe-stacks --stack-name eksp | jq '.Stacks[].Outputs[] | select(.OutputKey=="EKSphemeralAPIEndpoint").OutputValue' -r)
```

Now you can build the UI container image like so:

```sh
$ make build

```

Verify that the image has been built and is available, locally:

```sh
$ make verify

```

Now you can launch it:

```sh
$ make run
docker run      --name ekspui \
                --rm \
                --detach \
                --publish 8080:8080 \
                --env EKSPHEMERAL_HOME=/eksp \
                --env AWS_ACCESS_KEY_ID=XXXX \
                --env AWS_SECRET_ACCESS_KEY=XXXX \
                --env AWS_DEFAULT_REGION=us-east-2 \
                --env EKSPHEMERAL_URL=https://nswn7lkjbk.execute-api.us-east-2.amazonaws.com/Prod \
                                quay.io/mhausenblas/eksp-ui:0.2
79a352a4b0259e0b9731d5f3cfb942f185013ac51d14c4d4710eb7cfe1c534b2
```

Keep an eye on the logs of the UI proxy:

```sh
$ docker logs --follow ekspui
2019/06/21 10:06:58 EKSPhemeral UI up and running on http://localhost:8080/
...
```

When you're done, tear down the UI proxy:

```sh
$ make stop
docker kill ekspui
ekspui
```