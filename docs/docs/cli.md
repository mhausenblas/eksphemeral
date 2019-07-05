# The EKSphemeral CLI

!!! note
    Currently, the CLI binaries are available for both macOS and Linux platforms.

You can create, inspect, and prolong the lifetime of a cluster with the CLI as 
shown in the following.

## Manual install

!!! note
    You usually don't need to install the CLI manually, it should have been set up with the overall install. However, in cases where you want to access EKSphemeral from a machine other than the one you set it up originally or the CLI has been removed by someone or something, follow the steps here.


To manually install the binary CLI, for example on macOS, do:

```sh
$ curl -sL https://github.com/mhausenblas/eksphemeral/releases/latest/download/eksp-macos -o eksp
$ chmod +x eksp
$ sudo mv ./eksp /usr/local/bin
```

Now, let's check if there are already clusters are managed by EKSphemeral:

```sh
$ eksp list
No clusters found
```

Since we just installed EKSphemeral, there are no clusters, yet. Let's change that.

## Create clusters

Let's create a cluster named `mh9-eksp`, with three worker nodes, 
using Kubernetes version 1.12, with a 150 min timeout. 

First, create a file `cluster-spec.json` with the following content:

```json
{
    "id": "",
    "name": "mh9-eksp",
    "numworkers": 3,
    "kubeversion": "1.21",
    "timeout": 150,
    "ttl": 150,
    "owner": "hausenbl+notif@amazon.com",
    "created": ""
}
```

Now you can use the `create` command like so:

```sh
$ eksp create luster-spec.json
Trying to create a new ephemeral cluster ...
... using cluster spec cluster-spec.json
Seems you have set 'us-east-2' as the target region, using this for all following operations
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

Note that if no cluster spec is provided, a default cluster spec will be
used along with first security group of the default VPC.

Once the cluster is ready and you've verified your email addresses you should
get a notification that looks something like the following:

![EKSphemeral mail notification on cluster create](img/mail-notif-example.png)

The same is true at least five minutes before the cluster shuts down.

## List clusters

Next, let's check what clusters are managed by EKSphemeral:

```sh
$ eksp list
NAME       ID                                     KUBERNETES   NUM WORKERS   TIMEOUT   TTL      OWNER
mh9-eksp   e90379cf-ee0a-49c7-8f82-1660760d6bb5   v1.12        2             45 min    42 min   hausenbl+notif@amazon.com
```

Here, we get an array of cluster IDs back. We can use such a cluster ID as 
follows to look up the spec of a particular cluster:

```sh
$ eksp list e90379cf-ee0a-49c7-8f82-1660760d6bb5
ID:             e90379cf-ee0a-49c7-8f82-1660760d6bb5
Name:           mh9-eksp
Kubernetes:     v1.12
Worker nodes:   2
Timeout:        45 min
TTL:            38 min
Owner:          hausenbl+notif@amazon.com
Details:
        Status:             ACTIVE
        Endpoint:           https://A377918A0CA6D8BE793FF8BEC88964FE.sk1.us-east-2.eks.amazonaws.com
        Platform version:   eks.2
        VPC config:         private access: false, public access: true
        IAM role:           arn:aws:iam::661776721573:role/eksctl-mh9-eksp-cluster-ServiceRole-1HT8OAOGNNY2Y
```

## Prolong cluster lifetime

When you get a notification that one of your clusters is about to shut down or 
really at any time before it shuts down, you can prolong the cluster lifetime 
using the `eksp prolong` command.

Let's say we want to keep the cluster with the ID `e90379cf-ee0a-49c7-8f82-1660760d6bb5` around 
for 13 min longer. Here's what you would do:

```sh
$ eksp prolong e90379cf-ee0a-49c7-8f82-1660760d6bb5 13

Trying to set the TTL of cluster e90379cf-ee0a-49c7-8f82-1660760d6bb5 to 13 minutes, starting now
Successfully prolonged the lifetime of cluster e90379cf-ee0a-49c7-8f82-1660760d6bb5 for 13 minutes.

$ eksp list
NAME       ID                                     KUBERNETES   NUM WORKERS   TIMEOUT   TTL      OWNER
mh9-eksp   e90379cf-ee0a-49c7-8f82-1660760d6bb5   v1.12        2             13 min    13 min   hausenbl+notif@amazon.com
```

Note that the prolong command updates the `timeout` field of your cluster spec,
that is, the cluster TTL is counted from the moment you issue the prolong command, 
taking the remaining cluster runtime into account.

# Uninstall

To uninstall EKSphemeral, use the following command. This will remove the 
control plane elements, that is, delete the Lambda functions and remove all 
cluster specs from the `EKSPHEMERAL_CLUSTERMETA_BUCKET` S3 bucket:

```bash
$ eksp uninstall
Trying to uninstall EKSphemeral ...
Taking down the EKSphemeral control plane, this might take a few minutes ...

aws s3 rm s3://eks-cluster-meta --recursive
aws cloudformation delete-stack --stack-name eksp

Tear-down will complete within some 5 min. You can check the status manually, if you like, using 'make status' in the svc/ directory.
Once you see a message saying something like 'Stack with id eksp does not exist' you know for sure it's gone :)

Thanks for using EKSphemeral and hope to see ya soon ;)
```

Note that the service code bucket and the cluster metadata bucket are still around after this. 
You can either manually delete them or keep them around, to reuse them later. 