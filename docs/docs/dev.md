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
GOOS=linux GOARCH=amd64 go build -o ./proxy .
docker build --tag quay.io/mhausenblas/eksp-ui:0.2 .
Sending build context to Docker daemon  10.86MB
Step 1/17 : FROM amazonlinux:2018.03
 ---> a89f4a191d4c
Step 2/17 : LABEL maintainer="Michael Hausenblas <hausenbl@amazon.com>"
 ---> Using cache
 ---> f57e0623e5d8
Step 3/17 : ARG AWS_ACCESS_KEY_ID
 ---> Using cache
 ---> 45fc8c05256c
Step 4/17 : ARG AWS_SECRET_ACCESS_KEY
 ---> Using cache
 ---> 02a4bcc33f74
Step 5/17 : ARG AWS_DEFAULT_REGION
 ---> Using cache
 ---> 6d1562974f7c
Step 6/17 : COPY install.sh .
 ---> Using cache
 ---> 5a6d45d855ce
Step 7/17 : RUN yum install unzip jq git -y && yum clean all &&     curl -sL https://bootstrap.pypa.io/get-pip.py -o get-pip.py &&     python get-pip.py && pip install awscli --upgrade &&     export EKSPHEMERAL_HOME=/eksp &&     chmod +x install.sh && ./install.sh
 ---> Using cache
 ---> dcb28e56940f
Step 8/17 : COPY css/* /app/css/
 ---> Using cache
 ---> f2aad01c40dc
Step 9/17 : COPY img/* /app/img/
 ---> Using cache
 ---> ae2750fca6cd
Step 10/17 : COPY js/* /app/js/
 ---> Using cache
 ---> 82fec12cf16a
Step 11/17 : COPY *.html /app/
 ---> Using cache
 ---> db13d828ff73
Step 12/17 : WORKDIR /app
 ---> Using cache
 ---> 304d80268fbe
Step 13/17 : RUN chown -R 1001:1 /app
 ---> Using cache
 ---> 0c09d65e5358
Step 14/17 : USER 1001
 ---> Using cache
 ---> 6ca8e15efb82
Step 15/17 : COPY proxy .
 ---> Using cache
 ---> 397672a796b3
Step 16/17 : EXPOSE 8080
 ---> Using cache
 ---> 6508d74b4e39
Step 17/17 : CMD ["/app/proxy"]
 ---> Using cache
 ---> 2ad45f31e101
Successfully built 2ad45f31e101
Successfully tagged quay.io/mhausenblas/eksp-ui:0.2
```

Verify that the image has been built and is available, locally:

```sh
$ make verify
REPOSITORY                    TAG                 IMAGE ID            CREATED             SIZE
quay.io/mhausenblas/eksp-ui   0.2                 2ad45f31e101        About an hour ago   449MB
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