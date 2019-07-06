#!/usr/bin/env bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

################################################################################
# BASE install

echo "Using the follwing input: cluster $CLUSTER_NAME v$KUBERNETES_VERSION with $NUM_WORKERS workers"

# first, provision EKS control and data plane using eksctl:
eksctl create cluster \
    --name $CLUSTER_NAME \
    --version $KUBERNETES_VERSION \
    --nodes $NUM_WORKERS \
    --auto-kubeconfig \
    --full-ecr-access \
    --appmesh-access

export KUBECONFIG=/home/eksctl/.kube/eksctl/clusters/$CLUSTER_NAME

# let's wait up to 5 minutes for the nodes the get ready:
echo "Now waiting up to 5 min for cluster to be usable ..."
kubectl wait nodes --for=condition=Ready --timeout=300s --all

################################################################################
# ADDONS install

######
# install ArgoCD based off of:
# https://argoproj.github.io/argo-cd/getting_started/
echo "Now installing ArgoCD ..."
kubectl create namespace argocd
kubectl -n argocd apply -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# OPTION: expose the UI publicly via ...
# kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'

# OPTION: log in via ...
# kubectl get pods -n argocd -l app.kubernetes.io/name=argocd-server -o name | cut -d'/' -f 2

######
# install App Mesh and o11y components (Prometheus, Grafana, X-Ray) using:
# https://github.com/PaulMaddox/aws-appmesh-helm
kubectl apply -f https://raw.githubusercontent.com/PaulMaddox/aws-appmesh-helm/master/scripts/helm-rbac.yaml
helm init --service-account tiller
helm install -n aws-appmesh --namespace appmesh-system https://github.com/PaulMaddox/aws-appmesh-helm/releases/latest/download/aws-appmesh.tgz


######
# install the default Kube dashboard based off of:
# https://docs.aws.amazon.com/eks/latest/userguide/dashboard-tutorial.html
kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v1.10.1/src/deploy/recommended/kubernetes-dashboard.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/heapster/master/deploy/kube-config/influxdb/heapster.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/heapster/master/deploy/kube-config/influxdb/influxdb.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/heapster/master/deploy/kube-config/rbac/heapster-rbac.yaml

# OPTION expose the dashboard publicly via the following (NOTE: in order to log
# in, you still need to create an SA xxx and cluster admin CRB and then get the 
# auth token from executing 'kubectl -n kube-system describe secret xxx') ...
# kubectl patch svc kubernetes-dashboard -n kube-system -p '{"spec": {"type": "LoadBalancer"}}'

echo "DONE"