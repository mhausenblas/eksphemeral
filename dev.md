# Development and testing

If you want to play around with EKSphemeral, follow these steps.

In order to build the service, clone this repo, and make sure you've got the following available, locally:

- `jq`
- `aws` CLI
- [SAM CLI](https://github.com/awslabs/aws-sam-cli)
- [Fargate CLI](https://somanymachines.com/fargate/)
- Docker

Also, you will need access to the following services (and their implicit dependencies, such as EC2 in case of EKS): AWS Lambda, AWS Fargate, Amazon EKS. 

## The control plane in AWS Lambda and S3

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

## The data plane in AWS Fargate

You can manually kick off the EKS cluster provisioning as described in the following.

Note that, optionally, you can build a custom container image using your own registry coordinates and customize what's in the `eksctl` image used to provision the EKS cluster via a Fargate task like so:

```sh
$ docker build -t quay.io/mhausenblas/eksctl:base .
$ docker push quay.io/mhausenblas/eksctl:base
```

Back to the provisioning. First, set the security group to use:

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

$ aws ec2 create-security-group --group-name eksphemeral-sg --description "The security group the EKSphemeral data plane uses" --vpc-id $default_vpc

$ aws ec2 authorize-security-group-ingress --group-name eksphemeral-sg --protocol all --port all
```

Note that the last command apparently doesn't work, unsure but based on my research it's an AWS CLI bug.

Anyways, now you can use AWS Fargate through the Fargate CLI to provision the cluster,
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

## The CLI

To manually install the binary CLI, for example on macOS, do:

```sh
$ curl -sL https://github.com/mhausenblas/eksphemeral/releases/latest/download/eksp-macos -o eksp
$ chmod +x eksp
$ sudo mv ./eksp /usr/local/bin
```

## The UI

Check out instructions in the [ui](ui/) directory.

Please create issues if anything doesn't work as described in here.